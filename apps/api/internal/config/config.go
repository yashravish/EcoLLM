package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all environment-driven configuration for the API server.
// All values are loaded at startup via Load(). No global state; caller owns the Config.
type Config struct {
	Port               string
	DatabaseURL        string
	RedisURL           string
	JWTSecret          string
	LogLevel           string
	AllowedOrigins     []string
	Phi3SidecarURL     string
	GridRegion         string
	RateLimitPerMinute int
	RequestTimeout     time.Duration

	// Inference endpoints per model
	InferencePhi3URL     string
	InferenceMistralURL  string
	InferenceLlama13BURL string
	InferenceLlama70BURL string

	// External model names sent to upstream API (leave empty for local vLLM/Ollama)
	InferenceAPIKey          string
	InferencePhi3Model       string
	InferenceMistralModel    string
	InferenceLlama13BModel   string
	InferenceLlama70BModel   string

	// Carbon / energy
	GridAPIKey    string  // Electricity Maps API key (optional; static fallback when empty)
	PUEMultiplier float64 // Power Usage Effectiveness multiplier (default 1.3)

	// Observability
	OTelEndpoint string // OTLP HTTP exporter endpoint (optional; stdout when empty)

	// Feature flags
	EnablePromptOptimization bool
	EnableCarbonTracking     bool
	EnableCache              bool
	EnableFallback           bool

	// OAuth
	GitHubClientID     string
	GitHubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string
	APIBaseURL         string // backend origin used to build OAuth redirect URLs
	FrontendURL        string // frontend origin for post-auth redirects
}

// Load reads configuration from environment variables, applying defaults for
// optional fields and returning an error for any required field that is absent.
func Load() (*Config, error) {
	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		Phi3SidecarURL: getEnv("PHI3_SIDECAR_URL", "http://prompt-optimizer:8081"),
		GridRegion:     getEnv("GRID_REGION", "US-EAST"),

		InferencePhi3URL:     getEnv("INFERENCE_PHI3_URL", "http://vllm-phi3:8000"),
		InferenceMistralURL:  getEnv("INFERENCE_MISTRAL_URL", "http://vllm-mistral:8001"),
		InferenceLlama13BURL: getEnv("INFERENCE_LLAMA13B_URL", "http://vllm-llama13b:8002"),
		InferenceLlama70BURL: getEnv("INFERENCE_LLAMA70B_URL", "http://vllm-llama70b:8003"),

		InferenceAPIKey:        getEnv("INFERENCE_API_KEY", ""),
		InferencePhi3Model:     getEnv("INFERENCE_PHI3_MODEL", ""),
		InferenceMistralModel:  getEnv("INFERENCE_MISTRAL_MODEL", ""),
		InferenceLlama13BModel: getEnv("INFERENCE_LLAMA13B_MODEL", ""),
		InferenceLlama70BModel: getEnv("INFERENCE_LLAMA70B_MODEL", ""),

		GridAPIKey:   getEnv("GRID_API_KEY", ""),
		OTelEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),

		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		APIBaseURL:         getEnv("API_BASE_URL", "http://localhost:8080"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),

		EnablePromptOptimization: getEnvBool("ENABLE_PROMPT_OPTIMIZATION", true),
		EnableCarbonTracking:     getEnvBool("ENABLE_CARBON_TRACKING", true),
		EnableCache:              getEnvBool("ENABLE_CACHE", true),
		EnableFallback:           getEnvBool("ENABLE_FALLBACK", true),
	}

	pueStr := getEnv("PUE_MULTIPLIER", "1.3")
	pue, err := strconv.ParseFloat(pueStr, 64)
	if err != nil || pue < 1.0 {
		pue = 1.3
	}
	cfg.PUEMultiplier = pue

	// Required fields
	var missing []string
	cfg.DatabaseURL = requireEnv("DATABASE_URL", &missing)
	cfg.RedisURL = requireEnv("REDIS_URL", &missing)
	cfg.JWTSecret = requireEnv("JWT_SECRET", &missing)
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	// Parsed fields
	rateLimitStr := getEnv("RATE_LIMIT_PER_MIN", "60")
	rateLimit, err := strconv.Atoi(rateLimitStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_PER_MIN %q: %w", rateLimitStr, err)
	}
	cfg.RateLimitPerMinute = rateLimit

	timeoutStr := getEnv("REQUEST_TIMEOUT", "30s")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REQUEST_TIMEOUT %q: %w", timeoutStr, err)
	}
	cfg.RequestTimeout = timeout

	originsStr := getEnv("ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.AllowedOrigins = strings.Split(originsStr, ",")
	for i := range cfg.AllowedOrigins {
		cfg.AllowedOrigins[i] = strings.TrimSpace(cfg.AllowedOrigins[i])
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func requireEnv(key string, missing *[]string) string {
	v := os.Getenv(key)
	if v == "" {
		*missing = append(*missing, key)
	}
	return v
}

func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}
