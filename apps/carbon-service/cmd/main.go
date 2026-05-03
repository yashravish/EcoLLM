package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ecollm/carbon-service/internal/calculator"
	"github.com/ecollm/carbon-service/internal/grid"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, _ := zerolog.ParseLevel(getEnv("LOG_LEVEL", "info"))
	zerolog.SetGlobalLevel(level)

	port := getEnv("PORT", "8091")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	electricityMapsKey := getEnv("ELECTRICITY_MAPS_API_KEY", "")
	defaultRegion := getEnv("GRID_REGION", "US-EAST")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid REDIS_URL")
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal().Err(err).Msg("redis ping failed")
	}
	log.Info().Str("redis_url", redisURL).Msg("redis connected")

	gridClient := grid.New(rdb, electricityMapsKey, defaultRegion)
	calc := calculator.New(gridClient)

	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		AppName:      "ecollm-carbon-service",
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "carbon-service"})
	})

	app.Get("/v1/intensity", func(c *fiber.Ctx) error {
		region := c.Query("region", defaultRegion)
		intensity, source, err := gridClient.Intensity(c.Context(), region)
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{
			"region":    region,
			"intensity": intensity,
			"source":    source,
			"unit":      "gCO2/kWh",
		})
	})

	app.Post("/v1/estimate", func(c *fiber.Ctx) error {
		var req calculator.EstimateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
		result, err := calc.Estimate(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(result)
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("shutting down carbon-service")
		_ = app.Shutdown()
	}()

	log.Info().Str("port", port).Msg("carbon-service listening")
	if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatal().Err(err).Msg("listen failed")
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
