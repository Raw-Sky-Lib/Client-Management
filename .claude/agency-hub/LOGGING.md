# Logging Reference

This document is the canonical reference for logging in Golang featuring my projects. It covers every log level, when and how to use each one, all slog attribute functions, what not to log, and how dev and
production output differ.

---

## Setup

The logger lives in `pkg/logger/` — two files:

| File | Responsibility |
|------|---------------|
| `pkg/logger/logger.go` | Initializes the handler based on environment |
| `pkg/logger/levels.go` | Custom level constants (`LevelTrace`, `LevelFatal`) and helper functions (`Trace`, `Fatal`) |

**Initialize once at startup** (already done in `cmd/app/main.go`):

```go
logger.InitLogger(cfg.Environment) // "production" or anything else (dev)
```

> **Before `InitLogger` is called** — `logger.Log` is `nil`, so you cannot use it yet.
> Use the stdlib `log` package for any fatal errors that occur before initialization:
> ```go
> cfg, err := config.LoadConfig()
> if err != nil {
>     log.Fatalf("could not load config: %v", err) // stdlib log — logger not yet ready
> }
> logger.InitLogger(cfg.Environment) // from here on, use logger.Log / logger.Fatal
> ```

**Import in any package:**

```go
import (
    "log/slog"
    "github.com/DagMT/Goguard.git/pkg/logger"
)
```

### One logger, one init

`InitLogger` is called **once** in `main.go` — never inside individual packages. Every other file just imports `pkg/logger` and uses `logger.Log` directly. The global is shared across the entire process.

```go
// Any package — no InitLogger call, just use the global
logger.Log.Info("something happened", slog.String("Key", "value"))
```

### The pre-init zone (config loading)

There is a chicken-and-egg constraint: `LoadConfig()` must run before `InitLogger()` because the logger needs the environment value (`"production"` / `"development"`) that comes from config. This means any logging that happens inside `LoadConfig` — or any package it calls — cannot use `logger.Log` yet, because it is still `nil`.

Use the stdlib `log` package for anything in this zone:

```go
// internal/config/config.go — logger.Log is nil here, stdlib log is correct
if err := godotenv.Load(); err != nil {
    log.Printf("No .env file found, using environment variables: %v", err)
}
```

Calling `InitLogger` inside `config.go` to fix the nil would break the environment detection — the logger would always initialize with the wrong or default settings before the environment variable is even read.

**The boundary is clear:**

```
log.Printf / log.Fatalf   →  config loading and anything before InitLogger
logger.Log.*              →  everything after InitLogger in main.go
```

---

## Level Hierarchy

Levels are integers. A handler with minimum level `X` silently drops anything below `X`.

```
TRACE  -8   dev only    ← logger.LevelTrace  (custom)
DEBUG  -4   dev only    ← slog.LevelDebug
INFO    0   dev + prod  ← slog.LevelInfo
WARN    4   dev + prod  ← slog.LevelWarn
ERROR   8   dev + prod  ← slog.LevelError
FATAL  12   dev + prod  ← logger.LevelFatal  (custom, exits after logging)
```

| Level | Constant | Visible in | One-line rule |
|-------|----------|------------|---------------|
| `TRACE` | `logger.LevelTrace` | dev only | deep internals: loops, connection steps, parser state |
| `DEBUG` | `slog.LevelDebug` | dev only | function-level flow, config values, cache hits/misses |
| `INFO` | `slog.LevelInfo` | dev + prod | normal lifecycle events, request completed, server started |
| `WARN` | `slog.LevelWarn` | dev + prod | recoverable problems, degraded state, auto-corrected config |
| `ERROR` | `slog.LevelError` | dev + prod | failure the app survives — needs attention but keeps running |
| `FATAL` | `logger.LevelFatal` | dev + prod | unrecoverable — logs the error then calls `os.Exit(1)` |

---

## When and How to Use Each Level

### TRACE — dev only, ultra-verbose

Use when you need to trace individual steps inside a process: connection negotiation,
loop iterations, internal state transitions. Never shows in production.

```go
logger.Trace("checking Redis ping", slog.String("Host", cfg.RedisHost))
logger.Trace("route matched",
    slog.String("Pattern", "/api/users"),
    slog.String("Method", "GET"),
)
logger.Trace("db row scanned", slog.Int("Row", i), slog.String("ID", id))
```

### DEBUG — dev only, function-level flow

Use when you want to confirm a code path was taken, inspect a value, or track a cache decision.
Not as granular as Trace — one log per meaningful step, not per iteration.

```go
logger.Log.Debug("config loaded",
    slog.String("Environment", cfg.Environment),
    slog.String("Port", cfg.Port),
)
logger.Log.Debug("cache miss", slog.String("Key", "session:abc123"))
logger.Log.Debug("token validated", slog.String("UserID", claims.Subject))
```

### INFO — normal operational events

Use for anything that represents a healthy, expected event you want a record of in production.
Server lifecycle, completed requests, successful user actions.

```go
logger.Log.Info("Starting Goguard API",
    slog.String("Port", cfg.Port),
    slog.String("Environment", cfg.Environment),
)
logger.Log.Info("Server listening on", slog.String("Addr", srv.Addr))
logger.Log.Info("request completed",
    slog.String("RequestID", reqID),
    slog.String("Method", "POST"),
    slog.String("Path", "/api/users"),
    slog.Int("Status", 201),
    slog.Duration("Elapsed", time.Since(start)),
)
logger.Log.Info("Server exited gracefully")
```

### WARN — recoverable, something is off

Use when the app auto-corrected a problem, a retry is happening, or a deprecated path was used.
The system is degraded but still functional.

```go
logger.Log.Warn("DB_SSLMODE is set to 'disable' in production. Forcing 'require'.")
logger.Log.Warn("Redis retry attempt",
    slog.Int("Attempt", 2),
    slog.Duration("Backoff", 500*time.Millisecond),
)
logger.Log.Warn("slow query detected",
    slog.String("Query", "SELECT ..."),
    slog.Duration("Elapsed", 3*time.Second),
)
```

### ERROR — failure, app keeps running

Use when an operation failed and the caller needs to know, but the server is not going down.
Always include the error string and enough context to reproduce the issue.

```go
logger.Log.Error("failed to process request",
    slog.String("RequestID", reqID),
    slog.String("Error", err.Error()),
    slog.Int("Status", 500),
)
logger.Log.Error("failed to write to database",
    slog.String("Table", "users"),
    slog.String("Operation", "INSERT"),
    slog.String("Error", err.Error()),
)
```

> Do NOT use `Fatal` inside request handlers — a single bad request would kill the entire server.
> Use `Error`, log it, and return an appropriate HTTP response.

### FATAL — unrecoverable, exits immediately

Use only during application startup or initialization, when the process cannot continue.
After logging, `os.Exit(1)` is called — **nothing after `logger.Fatal(...)` will run**.

```go
// At startup — if we can't connect, there's no point continuing
logger.Fatal("could not load config", slog.String("Error", err.Error()))
logger.Fatal("Could not ping database", slog.String("Error", err.Error()))
logger.Fatal("Couldn't Connect to Redis", slog.String("Error", err.Error()))
```

```
Fatal is NOT appropriate for:
  - Request handler errors         → use Error
  - Business logic failures        → use Error or Warn
  - Anything inside a goroutine that can be retried
```

---

## slog Attribute Functions

Always use the strongly-typed functions. Avoid bare string key/value pairs — they are
untyped and produce no compile-time safety.

```go
slog.String("Key", "value")
slog.Int("Count", 42)
slog.Int64("UserID", 9223372036854775807)
slog.Bool("Active", true)
slog.Float64("CpuUsage", 45.67)
slog.Duration("Elapsed", 245*time.Millisecond) // auto-formats: "245ms"
slog.Time("CreatedAt", time.Now())             // ISO 8601 in JSON
slog.Any("Custom", someStruct)                 // fallback for anything else
slog.Err(err)                                  // shorthand for slog.Any("err", err)
```

### slog.Group — namespace related fields

Nest related attributes under a key. Avoids collision between `status` in HTTP and `status` in DB.

```go
logger.Log.Info("request_completed",
    slog.String("RequestID", reqID),
    slog.Group("HTTP",
        slog.String("Method", "GET"),
        slog.String("Path", "/api/users"),
        slog.Int("Status", 200),
        slog.Duration("Duration", 120*time.Millisecond),
    ),
    slog.Group("DB",
        slog.Int("Queries", 3),
        slog.Duration("Duration", 18*time.Millisecond),
    ),
)
```

**JSON output:**
```json
{"level":"INFO","msg":"request_completed","RequestID":"abc","HTTP":{"Method":"GET","Path":"/api/users","Status":200},"DB":{"Queries":3}}
```

**Tint (dev) output:**
```
3:04PM INFO request_completed RequestID=abc HTTP.Method=GET HTTP.Path=/api/users HTTP.Status=200 DB.Queries=3
```

### logger.Log.With() — attach persistent context to a logger

Create a child logger that carries fixed attributes on every call. Ideal for request handlers
and background workers where you want `request_id` or `user_id` on every line automatically.

```go
func handleCreateUser(w http.ResponseWriter, r *http.Request) {
    reqLogger := logger.Log.With(
        slog.String("RequestID", reqID),
        slog.String("Handler", "create_user"),
    )

    reqLogger.Info("request started")

    user, err := createUser(r.Context(), input)
    if err != nil {
        reqLogger.Error("user creation failed", slog.String("Error", err.Error()))
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    reqLogger.Info("user created", slog.String("UserID", user.ID))
}
// Every log above automatically includes RequestID and Handler — no repetition.
```

### logger.Log.LogAttrs() — zero-allocation hot path

When logging inside tight loops or high-frequency middleware, use `LogAttrs` to avoid
the allocations that the variadic `...any` form causes.

```go
logger.Log.LogAttrs(ctx, slog.LevelInfo, "request_completed",
    slog.String("RequestID", reqID),
    slog.Duration("Elapsed", time.Since(start)),
    slog.Int("Status", 200),
)
```

---

## What NOT to Log

| Category | Examples | Rule |
|----------|----------|------|
| Credentials | passwords, API keys, JWT tokens, session cookies | never |
| Payment data | full card numbers, CVV, bank account numbers | never |
| Auth secrets | HMAC keys, private keys, client secrets | never |
| PII | email, phone, full name, IP address | avoid unless required by law/compliance and documented |
| Raw bodies | full request/response payloads | Trace/Debug only in dev; never in prod |
| Stack traces | `fmt.Sprintf("%+v", err)` | use `err.Error()` — expose the message, not the trace |

### Redacting sensitive fields with `slog.LogValuer`

If a struct contains sensitive data but you need to log the struct, implement `LogValue()`:

```go
type AuthToken struct {
    Value     string
    ExpiresAt time.Time
}

func (t AuthToken) LogValue() slog.Value {
    return slog.GroupValue(
        slog.Time("ExpiresAt", t.ExpiresAt),
        slog.String("Value", "[REDACTED]"),
    )
}

// Usage — Value is automatically redacted
logger.Log.Info("token issued", slog.Any("Token", token))
// Output: Token.ExpiresAt=2026-05-01T00:00:00Z Token.Value=[REDACTED]
```

---

## Dev vs Production Behavior

| | Development | Production |
|---|---|---|
| **Handler** | `tint.NewHandler` (colored text) | `slog.NewJSONHandler` |
| **Format** | Human-readable, colorized | JSON — one object per line |
| **Min level** | TRACE (-8) — all levels visible | INFO (0) — Trace and Debug silenced |
| **TRACE** | Yellow label | Not emitted |
| **FATAL** | Red label | `"FATAL"` string in JSON |
| **Timestamp** | `3:04PM` (Kitchen) | RFC3339 nano (machine-parseable) |
| **Best for** | Local iteration, debugging | Log aggregators: Datadog, Loki, CloudWatch |

**Dev sample output:**
```
3:04PM INFO  Starting Goguard API Port=8080 Environment=development
3:04PM INFO  Server listening on Addr=:8080
3:04PM WARN  DB_SSLMODE is set to 'disable' in production. Forcing 'require'.
```

**Production JSON sample output:**
```json
{"time":"2026-04-02T14:30:00Z","level":"INFO","msg":"Starting Goguard API","Port":"8080","Environment":"production"}
{"time":"2026-04-02T14:30:00Z","level":"INFO","msg":"Server listening on","Addr":":8080"}
{"time":"2026-04-02T14:30:01Z","level":"FATAL","msg":"Could not ping database","Error":"connection refused"}
```

---

## Common Mistakes

### Using unstructured messages

```go
// Bad — the user ID is buried in a string, unqueryable
logger.Log.Info("User 123 logged in from 192.168.1.1")

// Good
logger.Log.Info("user_login",
    slog.String("UserID", "123"),
    slog.String("IP", "192.168.1.1"),
)
```

### Inconsistent key names

```go
// Bad — three names for the same concept
logger.Log.Info("event", slog.String("userId", "1"))
logger.Log.Info("event", slog.String("user_id", "2"))
logger.Log.Info("event", slog.String("uid", "3"))

// Good — this project uses PascalCase throughout
logger.Log.Info("event", slog.String("UserID", "1"))
```

### Using Fatal inside a handler

```go
// Bad — kills the entire server for a single bad request
func handler(w http.ResponseWriter, r *http.Request) {
    data, err := parseBody(r)
    if err != nil {
        logger.Fatal("bad input", slog.String("Error", err.Error())) // ← wrong
    }
}

// Good — log the error and return an HTTP response
func handler(w http.ResponseWriter, r *http.Request) {
    data, err := parseBody(r)
    if err != nil {
        logger.Log.Error("failed to parse request body", slog.String("Error", err.Error()))
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
}
```

### Flooding logs in a loop

```go
// Bad — one log per item at high volume
for _, item := range items {
    logger.Log.Debug("processing", slog.Any("Item", item))
}

// Good — log the summary
logger.Log.Debug("batch started", slog.Int("Count", len(items)))
// ... process ...
logger.Log.Debug("batch completed",
    slog.Int("Processed", processed),
    slog.Int("Failed", failed),
    slog.Duration("Elapsed", time.Since(start)),
)
```

### Losing context in goroutines

```go
// Bad — the goroutine has no request context
go func() {
    logger.Log.Info("async task started") // no RequestID
}()

// Good — capture the context before launching the goroutine
reqLogger := logger.Log.With(slog.String("RequestID", reqID))
go func() {
    reqLogger.Info("async task started") // includes RequestID
}()
```

---

## Quick Reference Card

```go
// Custom helpers (levels.go)
logger.Trace("msg", slog.String("Key", "value"))   // dev only, -8
logger.Fatal("msg", slog.String("Key", "value"))   // logs + os.Exit(1), level 12

// Built-in slog levels via logger.Log
logger.Log.Debug("msg", slog.String("Key", "value"))
logger.Log.Info("msg", slog.String("Key", "value"))
logger.Log.Warn("msg", slog.String("Key", "value"))
logger.Log.Error("msg", slog.String("Key", "value"))

// Attribute types
slog.String / slog.Int / slog.Int64 / slog.Bool
slog.Float64 / slog.Duration / slog.Time / slog.Any / slog.Err

// Grouping — PascalCase on group name and all keys inside
slog.Group("Namespace", slog.String("Key", "value"), ...)

// Child logger with fixed context
child := logger.Log.With(slog.String("RequestID", id))

// Zero-alloc hot path
logger.Log.LogAttrs(ctx, slog.LevelInfo, "msg", slog.String("Key", "value"))
```
