package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/DagMT/client-portal/internal/config"
	"github.com/DagMT/client-portal/internal/database"
	"github.com/DagMT/client-portal/internal/middleware"
	"github.com/DagMT/client-portal/internal/startup"
	"github.com/DagMT/client-portal/internal/utils"
	"github.com/DagMT/client-portal/pkg/logger"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Pre-logger zone: config must load before logger can init.
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logger.InitLogger(cfg.Environment)

	pool, err := database.Connect(cfg.SupabaseDBURL)
	if err != nil {
		logger.Fatal("could not connect to database", slog.String("Error", err.Error()))
	}
	defer pool.Close()

	redisOpt, err := redis.ParseURL(cfg.UpstashRedisURL)
	if err != nil {
		logger.Fatal("invalid UPSTASH_REDIS_URL", slog.String("Error", err.Error()))
	}
	rdb := redis.NewClient(redisOpt)
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Fatal("could not connect to Redis", slog.String("Error", err.Error()))
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	if err := startup.ValidateManagementToken(
		cfg.AgencyAPIURL,
		cfg.AgencyManagementToken,
		cfg.AgencyClientID,
		httpClient,
	); err != nil {
		logger.Fatal("startup validation failed", slog.String("Error", err.Error()))
	}

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Security)
	r.Use(chimiddleware.Recoverer)

	// Health — exempt from auth and CSRF
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// CSRF-protected routes (all browser-facing)
	r.Group(func(r chi.Router) {
		r.Use(middleware.CSRF)

		// Onboarding — rate limited by IP, no auth required
		// r.With(middleware.RateLimit(rdb, "onboard", 5, time.Minute)).
		//     Mount("/api/onboarding", onboarding.Routes(...))  // wired in CLI-9

		// Auth endpoints
		// r.Mount("/api/auth", auth.Routes(...))  // wired in CLI-10

		// Authenticated routes — 30/min per IP
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(cfg.JWTSecret))
			r.Use(middleware.RateLimit(rdb, "auth", 30, time.Minute))

			// r.Mount("/api/assistant", claude.Routes(...))  // wired in CLI-16
		})
	})

	// Admin routes — machine-to-machine, no CSRF
	// r.Mount("/api/admin", admin.Routes(...))  // wired in CLI-11

	logger.Log.Info("Starting client-portal API",
		slog.String("Port", cfg.Port),
		slog.String("Environment", cfg.Environment),
	)

	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		logger.Fatal("server stopped", slog.String("Error", err.Error()))
	}
}
