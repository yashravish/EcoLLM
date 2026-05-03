package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ecollm/inference-gateway/internal/health"
	"github.com/ecollm/inference-gateway/internal/proxy"
)

func main() {
	// Logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logLevel := getEnv("LOG_LEVEL", "info")
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	port := getEnv("PORT", "8090")
	ollamaURL := getEnv("OLLAMA_BASE_URL", "http://localhost:11434")

	// Per-model vLLM URLs; fall back to Ollama routes if not set
	endpoints := proxy.ModelEndpoints{
		Phi3URL:     getEnv("PHI3_VLLM_URL", ollamaURL+"/api"),
		MistralURL:  getEnv("MISTRAL_VLLM_URL", ollamaURL+"/api"),
		Llama13BURL: getEnv("LLAMA13B_VLLM_URL", ollamaURL+"/api"),
		Llama70BURL: getEnv("LLAMA70B_VLLM_URL", ollamaURL+"/api"),
	}

	p := proxy.New(endpoints)
	h := health.New(endpoints)

	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		AppName:      "ecollm-inference-gateway",
	})

	app.Get("/health", h.Check)
	app.Get("/health/:model", h.CheckModel)

	// OpenAI-compatible per-model routes
	app.Post("/phi3/v1/chat/completions", p.Forward("phi3"))
	app.Post("/mistral/v1/chat/completions", p.Forward("mistral"))
	app.Post("/llama13b/v1/chat/completions", p.Forward("llama13b"))
	app.Post("/llama70b/v1/chat/completions", p.Forward("llama70b"))

	// Model list passthrough
	app.Get("/:model/v1/models", p.ListModels)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("shutting down inference-gateway")
		if err := app.Shutdown(); err != nil {
			log.Error().Err(err).Msg("shutdown error")
		}
	}()

	log.Info().Str("port", port).Msg("inference-gateway listening")
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
