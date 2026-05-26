package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecollm/api/internal/admin"
	"github.com/joho/godotenv"
	"github.com/ecollm/api/internal/audit"
	"github.com/ecollm/api/internal/auth"
	"github.com/ecollm/api/internal/billing"
	"github.com/ecollm/api/internal/carbon"
	"github.com/ecollm/api/internal/chat"
	"github.com/ecollm/api/internal/config"
	"github.com/ecollm/api/internal/database"
	"github.com/ecollm/api/internal/inference"
	"github.com/ecollm/api/internal/middleware"
	"github.com/ecollm/api/internal/prompt"
	"github.com/ecollm/api/internal/router"
	"github.com/ecollm/api/internal/telemetry"
	"github.com/ecollm/api/internal/usage"
	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// usageBillingAdapter adapts *usage.Repository to billing.UsageReader.
// The two packages can't share a type directly without an import cycle,
// so main bridges them here.
type usageBillingAdapter struct{ repo *usage.Repository }

func (a *usageBillingAdapter) GetUsageForBilling(ctx context.Context, orgID string, day time.Time) (*billing.UsageSummaryDTO, error) {
	s, err := a.repo.GetUsageForBilling(ctx, orgID, day)
	if err != nil {
		return nil, err
	}
	return &billing.UsageSummaryDTO{
		TotalRequests:  s.TotalRequests,
		TotalTokens:    s.TotalTokens,
		TotalEnergyKwh: s.TotalEnergyKwh,
		TotalCO2eGrams: s.TotalCO2eGrams,
		TotalCostUSD:   s.TotalCostUSD,
	}, nil
}

func (a *usageBillingAdapter) GetMonthlyRequestCount(ctx context.Context, orgID string, month time.Time) (int64, error) {
	return a.repo.GetMonthlyRequestCount(ctx, orgID, month)
}

func (a *usageBillingAdapter) ListOrgsWithUsage(ctx context.Context, day time.Time) ([]string, error) {
	return a.repo.ListOrgsWithUsage(ctx, day)
}

// carbonGridAdapter wraps *carbon.Estimator to satisfy router.GridReader.
// The selector calls GridCarbonIntensity() once per routing decision so it can
// score models on actual gCO2 per request rather than a fixed US-average default.
type carbonGridAdapter struct{ est *carbon.Estimator }

func (a *carbonGridAdapter) GridCarbonIntensity() float64 {
	intensity, _ := a.est.GridIntensity()
	return intensity
}

func main() {
	// Load .env from project root (../../.env when run from apps/api/) or CWD.
	if err := godotenv.Load("../../.env"); err != nil {
		_ = godotenv.Load(".env")
	}

	// ── Config ──────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	telemetry.InitLogger(cfg.LogLevel)

	tracerShutdown, err := telemetry.InitTracer("ecollm-api", cfg.OTelEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init tracer")
	}
	defer func() {
		if err := tracerShutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("tracer shutdown error")
		}
	}()

	// ── Data Layer ───────────────────────────────────────────────────────────
	pgPool, err := database.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pgPool.Close()

	redisClient, err := database.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer redisClient.Close()

	// ── Repositories ─────────────────────────────────────────────────────────
	authRepo := auth.NewRepository(pgPool)
	requestRepo := chat.NewRequestRepository(pgPool)
	usageRepo := usage.NewRepository(pgPool)
	billingRepo := billing.NewRepository(pgPool)
	auditRepo := audit.New(pgPool)
	feedbackRepo := usage.NewFeedbackRepository(pgPool)

	// ── Services ─────────────────────────────────────────────────────────────
	authService := auth.NewService(authRepo, redisClient, cfg.JWTSecret)
	gridCache := database.NewGridCacheAdapter(redisClient)
	gridSvc := carbon.NewGridService(gridCache, cfg.GridAPIKey)
	carbonEstimator := carbon.NewEstimatorWithGrid(cfg.GridRegion, gridSvc)
	promptOptimizer := prompt.NewOptimizer(cfg.Phi3SidecarURL)
	taskClassifier := router.NewClassifier()
	modelScorer := router.NewScorer(pgPool)
	modelSelector := router.NewSelector(modelScorer).
		WithDB(pgPool).
		WithGridReader(&carbonGridAdapter{est: carbonEstimator})

	inferenceGateway := inference.NewGateway(inference.InferenceEndpoints{
		Phi3URL:     cfg.InferencePhi3URL,
		MistralURL:  cfg.InferenceMistralURL,
		Llama13BURL: cfg.InferenceLlama13BURL,
		Llama70BURL: cfg.InferenceLlama70BURL,

		APIKey:                cfg.InferenceAPIKey,
		Phi3ExternalModel:     cfg.InferencePhi3Model,
		MistralExternalModel:  cfg.InferenceMistralModel,
		Llama13BExternalModel: cfg.InferenceLlama13BModel,
		Llama70BExternalModel: cfg.InferenceLlama70BModel,
	}, cfg.RequestTimeout)

	energyRepo := carbon.NewEnergyRepository(pgPool)
	carbonRepo := carbon.NewCarbonRepository(pgPool)

	chatService := chat.NewService(
		promptOptimizer,
		taskClassifier,
		modelSelector,
		inferenceGateway,
		carbonEstimator,
		requestRepo,
		energyRepo,
		carbonRepo,
		redisClient,
		chat.ServiceConfig{
			EnableCache:              cfg.EnableCache,
			EnableFallback:           cfg.EnableFallback,
			EnablePromptOptimization: cfg.EnablePromptOptimization,
			EnableCarbonTracking:     cfg.EnableCarbonTracking,
		},
		pgPool,
	)

	usageService := usage.NewService(usageRepo)
	adminService := admin.NewService(pgPool)
	billingService := billing.NewService(billingRepo)

	// ── Handlers ─────────────────────────────────────────────────────────────
	authHandler := auth.NewHandler(authService).WithAudit(auditRepo)
	chatHandler := chat.NewHandler(chatService)
	usageHandler := usage.NewHandler(usageService)
	feedbackHandler := usage.NewFeedbackHandler(feedbackRepo)
	adminHandler := admin.NewHandler(adminService)
	billingHandler := billing.NewHandler(billingService)

	// ── Fiber App ─────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: cfg.RequestTimeout,
		BodyLimit:    1 * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if e, ok := err.(*fiber.Error); ok {
				switch e.Code {
				case fiber.StatusNotFound:
					return c.Status(fiber.StatusNotFound).JSON(apierror.ErrNotFound)
				case fiber.StatusRequestEntityTooLarge:
					return c.Status(fiber.StatusRequestEntityTooLarge).JSON(apierror.ErrRequestTooLarge)
				}
			}
			return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
		},
	})

	// ── Global Middleware ─────────────────────────────────────────────────────
	app.Use(middleware.Recovery())
	app.Use(middleware.RequestID())
	app.Use(middleware.Logging(cfg.LogLevel))
	app.Use(middleware.CORS(cfg.AllowedOrigins))
	app.Use(middleware.MaxBody())

	// ── Health / Readiness ───────────────────────────────────────────────────
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "ecollm-api"})
	})

	// /ready is the K8s readiness probe — checks that both data stores are up.
	app.Get("/ready", func(c *fiber.Ctx) error {
		if err := pgPool.Ping(c.UserContext()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(
				fiber.Map{"status": "not ready", "reason": "postgres: " + err.Error()},
			)
		}
		if err := redisClient.Ping(c.UserContext()).Err(); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(
				fiber.Map{"status": "not ready", "reason": "redis: " + err.Error()},
			)
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// /me (root-level) — used by the dashboard (GET /me is the canonical user-info endpoint).
	app.Get("/me", middleware.JWTAuth(cfg.JWTSecret, redisClient), authHandler.Me)

	// ── Public Inference API (/v1) ────────────────────────────────────────────
	api := app.Group("/v1",
		middleware.Auth(authService, cfg.JWTSecret, redisClient),
		middleware.RateLimit(redisClient, cfg.RateLimitPerMinute, time.Minute),
	)
	api.Post("/chat/completions", chatHandler.CreateCompletion)
	api.Post("/completions", chatHandler.CreateCompletion)
	api.Post("/route/preview", chatHandler.PreviewRoute)
	api.Get("/models", chatHandler.ListModels)
	api.Get("/usage", usageHandler.GetUsage)
	api.Get("/carbon", usageHandler.GetCarbon)
	api.Get("/requests", usageHandler.GetRequests)
	api.Get("/requests/:id", usageHandler.GetRequest)
	api.Post("/requests/:id/feedback", feedbackHandler.SubmitFeedback)
	api.Get("/billing", billingHandler.GetBilling)
	api.Get("/billing/:id", billingHandler.GetBillingEvent)

	// ── API Key Routes ────────────────────────────────────────────────────────
	keyGroup := app.Group("/api-keys",
		middleware.JWTAuth(cfg.JWTSecret, redisClient),
	)
	keyGroup.Get("", authHandler.ListAPIKeys)
	keyGroup.Post("", authHandler.CreateAPIKey)
	keyGroup.Delete("/:id", authHandler.RevokeAPIKey)

	// ── Auth Routes ───────────────────────────────────────────────────────────
	authGroup := app.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/logout", middleware.JWTAuth(cfg.JWTSecret, redisClient), authHandler.Logout)
	authGroup.Get("/me", middleware.JWTAuth(cfg.JWTSecret, redisClient), authHandler.Me)
	authGroup.Delete("/me", middleware.JWTAuth(cfg.JWTSecret, redisClient), authHandler.DeleteMe)

	// ── Organization Routes ───────────────────────────────────────────────────
	orgGroup := app.Group("/organizations",
		middleware.JWTAuth(cfg.JWTSecret, redisClient),
	)
	orgGroup.Get("/:id", authHandler.GetOrg)
	orgGroup.Patch("/:id", middleware.RequireRole("admin"), authHandler.UpdateOrg)
	orgGroup.Get("/:id/members", authHandler.ListMembers)
	orgGroup.Post("/:id/members", middleware.RequireRole("admin"), authHandler.InviteMember)
	orgGroup.Patch("/:id/members/:userID", middleware.RequireRole("admin"), authHandler.UpdateMemberRole)
	orgGroup.Delete("/:id/members/:userID", middleware.RequireRole("admin"), authHandler.RemoveMember)

	// ── Admin Routes ──────────────────────────────────────────────────────────
	adminGroup := app.Group("/admin",
		middleware.JWTAuth(cfg.JWTSecret, redisClient),
		middleware.RequireRole("admin"),
	)
	adminGroup.Get("/metrics", adminHandler.GetMetrics)
	adminGroup.Get("/models", adminHandler.ListModels)
	adminGroup.Post("/models", adminHandler.CreateModel)
	adminGroup.Patch("/models/:id", adminHandler.UpdateModel)
	adminGroup.Get("/routes", adminHandler.GetRoutes)
	adminGroup.Patch("/routes", adminHandler.UpdateRoutes)
	adminGroup.Get("/carbon", adminHandler.GetCarbonMetrics)

	// ── Background Workers ────────────────────────────────────────────────────
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	go usage.StartAggregationWorker(workerCtx, usageRepo, time.Hour)
	go billing.StartBillingWorker(workerCtx, billingRepo, &usageBillingAdapter{repo: usageRepo}, 24*time.Hour)
	// Feedback learning loop: aggregates feedback_events → model_quality_scores weekly.
	go router.StartLearningWorker(workerCtx, pgPool, 7*24*time.Hour)
	// Reload learned quality scores into the selector every hour.
	modelSelector.StartScoreRefresh(workerCtx, time.Hour)

	// ── Prometheus Metrics Endpoint (separate port) ───────────────────────────
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		metricsServer := &http.Server{Addr: ":9091", Handler: mux}
		log.Info().Str("addr", ":9091").Msg("prometheus metrics server started")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("metrics server error")
		}
	}()

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-quit
		log.Info().Msg("shutdown signal received, draining connections")
		cancelWorkers()
		if err := app.Shutdown(); err != nil {
			log.Error().Err(err).Msg("server shutdown error")
		}
	}()

	// ── Start Server ─────────────────────────────────────────────────────────
	addr := ":" + cfg.Port
	log.Info().Str("addr", addr).Msg("ecollm api server starting")

	if err := app.Listen(addr); err != nil {
		log.Fatal().Err(err).Msg("server exited")
	}
}
