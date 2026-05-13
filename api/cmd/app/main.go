// Package main is the entry point for the client-portal API server.
//
// @title           Client Portal API
// @version         1.0
// @description     Multi-tenant CMS dashboard backend. Clients manage website content here.
// @host            localhost:8081
// @BasePath        /
// @securitydefinitions.apikey CookieAuth
// @in              cookie
// @name            access_token
// @description     Portal JWT stored in the access_token HTTP-only cookie. Issued by /api/auth/exchange or /api/onboarding/confirm.
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	_ "github.com/DagMT/client-portal/docs"
	"github.com/DagMT/client-portal/internal/auth"
	"github.com/DagMT/client-portal/internal/claude"
	"github.com/DagMT/client-portal/internal/config"
	"github.com/DagMT/client-portal/internal/database"
	"github.com/DagMT/client-portal/internal/mailer"
	"github.com/DagMT/client-portal/internal/middleware"
	"github.com/DagMT/client-portal/internal/onboarding"
	"github.com/DagMT/client-portal/internal/media"
	"github.com/DagMT/client-portal/internal/revalidate"
	"github.com/DagMT/client-portal/internal/startup"
	"github.com/DagMT/client-portal/internal/tenant"
	"github.com/DagMT/client-portal/internal/utils"
	"github.com/DagMT/client-portal/pkg/logger"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func main() {
	// Pre-logger zone: config must load before logger can init.
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger.InitLogger(cfg.Environment)
	logger.Trace("config loaded", slog.String("Environment", cfg.Environment), slog.String("Port", cfg.Port))

	logger.Trace("connecting to database")
	pool, err := database.Connect(cfg.SupabaseDBURL)
	if err != nil {
		logger.Fatal("could not connect to database", slog.String("Error", err.Error()))
	}
	defer pool.Close()
	logger.Trace("database connected")

	logger.Trace("connecting to Redis")
	redisOpt, err := redis.ParseURL(cfg.UpstashRedisURL)
	if err != nil {
		logger.Fatal("invalid UPSTASH_REDIS_URL", slog.String("Error", err.Error()))
	}
	rdb := redis.NewClient(redisOpt)
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("could not connect to Redis", slog.String("Error", err.Error()))
	}
	logger.Trace("Redis connected")

	logger.Trace("validating management token with agency-hub", slog.String("Agency", cfg.AgencyAPIURL))
	httpClient := &http.Client{Timeout: 15 * time.Second}
	if err := startup.ValidateManagementToken(
		cfg.AgencyAPIURL,
		cfg.AgencyManagementToken,
		cfg.AgencyClientID,
		httpClient,
	); err != nil {
		logger.Fatal("startup validation failed", slog.String("Error", err.Error()))
	}
	logger.Trace("agency-hub management token valid")

	encKey, err := utils.DeriveEncryptionKey(cfg.JWTSecret)
	if err != nil {
		logger.Fatal("derive encryption key", slog.String("Error", err.Error()))
	}
	logger.Trace("encryption key derived")

	secure := cfg.Environment == "production"

	m, err := mailer.New(cfg.MailerProvider, cfg.EmailFrom, cfg.ResendAPIKey, cfg.BrevoSMTPUser, cfg.BrevoSMTPKey)
	if err != nil {
		logger.Fatal("mailer init failed", slog.String("Error", err.Error()))
	}
	logger.Trace("mailer ready", slog.String("Provider", cfg.MailerProvider))

	logger.Trace("wiring auth feature")
	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(
		authRepo, httpClient, m, cfg.FrontendURL, cfg.PublicURL,
		encKey, cfg.JWTSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry, secure,
	)
	authHandler := auth.NewHandler(authSvc, secure, cfg.FrontendURL)
	logger.Trace("auth feature ready")

	logger.Trace("wiring tenant feature")
	tenantRepo := tenant.NewRepository(pool)
	tenantSvc := tenant.NewService(tenantRepo, encKey)
	logger.Trace("tenant feature ready")

	logger.Trace("wiring revalidation service")
	revalidateSvc := revalidate.NewService(httpClient)
	revalidateHandler := revalidate.NewHandler(revalidateSvc)
	logger.Trace("revalidation service ready")

	mediaHandler := media.NewHandler(httpClient)

	logger.Trace("wiring Claude assistant feature")
	claudeRL := claude.NewRateLimiter(rdb)
	claudeRepo := claude.NewRepository(httpClient, cfg.AgencyAPIURL, cfg.AgencyManagementToken, cfg.AgencyClientID)
	claudePrompt := claude.NewPromptBuilder(httpClient)
	claudeSvc := claude.NewService(claudeRL, claudeRepo, claudePrompt, cfg.AnthropicAPIKey, cfg.AnthropicDefaultModel)
	claudeHandler := claude.NewHandler(claudeSvc)
	logger.Trace("Claude assistant feature ready")

	logger.Trace("wiring onboarding feature")
	onboardRepo := onboarding.NewRepository(pool)
	onboardSvc := onboarding.NewService(
		onboardRepo, httpClient, m,
		cfg.AgencyAPIURL, cfg.AgencyManagementToken, cfg.AgencyClientID,
		encKey, cfg.PublicURL, cfg.FrontendURL,
	)
	onboardHandler := onboarding.NewHandler(onboardSvc, cfg.AgencyManagementToken, cfg.FrontendURL, authSvc)
	logger.Trace("onboarding feature ready")

	logger.Trace("building router")
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.CORS(cfg.FrontendURL)) // must be first — answers OPTIONS before auth/CSRF
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Security)
	r.Use(chimiddleware.Recoverer)

	// Swagger UI — development and staging only, never production.
	// Behaves like the real frontend: cookies sent automatically, CSRF header injected
	// from the csrf_token cookie on every mutation, auth state persisted across refreshes.
	if cfg.Environment != "production" {
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.PersistAuthorization(true),
			httpSwagger.UIConfig(map[string]string{
				// Send cookies with every request (mirrors axios withCredentials: true)
				"withCredentials": "true",
				// Automatically attach X-CSRF-Token from the csrf_token cookie —
				// the same double-submit pattern the frontend axios interceptor does.
				// Call GET /api/auth/csrf first; after that every POST just works.
				"requestInterceptor": `function(request) {
					const match = document.cookie
						.split('; ')
						.find(function(c) { return c.startsWith('csrf_token='); });
					if (match) {
						request.headers['X-CSRF-Token'] = match.split('=')[1];
					}
					return request;
				}`,
			}),
		))
		logger.Trace("swagger UI enabled", slog.String("URL", "http://localhost:"+cfg.Port+"/swagger/index.html"))
	}

	// Health — exempt from auth and CSRF
	r.Get("/health", healthCheck)

	// CSRF-protected routes (all browser-facing)
	r.Group(func(r chi.Router) {
		r.Use(middleware.CSRF)

		r.Route("/api/onboarding", onboarding.Routes(onboardHandler, rdb))
		r.Route("/api/auth", auth.Routes(
			authHandler,
			middleware.RateLimit(rdb, "magic_link", 1, 2*time.Minute,
				"Please wait 2 minutes before requesting another sign-in link."),
			middleware.Authenticate(cfg.JWTSecret),
		))

		// Authenticated routes — 30/min per IP
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(cfg.JWTSecret))
			r.Use(middleware.RateLimit(rdb, "auth", 30, time.Minute))
			r.Use(tenant.ResolveTenant(tenantSvc))

			r.Route("/api/assistant", claude.Routes(claudeHandler))
			r.Route("/api/revalidate", revalidate.Routes(revalidateHandler))
			r.Route("/api/media", media.Routes(mediaHandler))
		})
	})

	// Admin routes — machine-to-machine, no CSRF
	r.Route("/api/admin", onboarding.AdminRoutes(onboardHandler))
	logger.Trace("router ready")

	logger.Log.Info("Starting client-portal API",
		slog.String("Port", cfg.Port),
		slog.String("Environment", cfg.Environment),
	)

	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		logger.Fatal("server stopped", slog.String("Error", err.Error()))
	}
}

// healthCheck returns 200 OK when the server is running.
//
// @Summary     Health check
// @Tags        health
// @Produce     json
// @Success     200 {object} map[string]string
// @Router      /health [get]
func healthCheck(w http.ResponseWriter, r *http.Request) {
	utils.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
