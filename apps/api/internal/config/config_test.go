package config

import (
	"strings"
	"testing"
	"time"
)

// setRequired sets the three required env vars for all tests that need them.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/testdb")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "test-secret")
}

func TestLoad_Success(t *testing.T) {
	setRequired(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
}

func TestLoad_RequiredVars_AllMissing(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error when all required vars missing, got nil")
	}
	for _, key := range []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error message missing %q; got: %v", key, err)
		}
	}
}

func TestLoad_RequiredVars_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "secret")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("expected error containing DATABASE_URL, got: %v", err)
	}
}

func TestLoad_RequiredVars_MissingRedisURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "secret")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "REDIS_URL") {
		t.Errorf("expected error containing REDIS_URL, got: %v", err)
	}
}

func TestLoad_RequiredVars_MissingJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("expected error containing JWT_SECRET, got: %v", err)
	}
}

func TestLoad_Defaults(t *testing.T) {
	setRequired(t)
	// Clear optional vars so defaults are used.
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("RATE_LIMIT_PER_MIN", "")
	t.Setenv("REQUEST_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.RateLimitPerMinute != 60 {
		t.Errorf("RateLimitPerMinute = %d, want 60", cfg.RateLimitPerMinute)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("RequestTimeout = %v, want 30s", cfg.RequestTimeout)
	}
	if cfg.PUEMultiplier != 1.3 {
		t.Errorf("PUEMultiplier = %f, want 1.3", cfg.PUEMultiplier)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	setRequired(t)
	t.Setenv("PORT", "9090")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
}

func TestLoad_InvalidRateLimit(t *testing.T) {
	setRequired(t)
	t.Setenv("RATE_LIMIT_PER_MIN", "not-a-number")
	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid RATE_LIMIT_PER_MIN, got nil")
	}
	if !strings.Contains(err.Error(), "RATE_LIMIT_PER_MIN") {
		t.Errorf("error should mention RATE_LIMIT_PER_MIN, got: %v", err)
	}
}

func TestLoad_InvalidRequestTimeout(t *testing.T) {
	setRequired(t)
	t.Setenv("REQUEST_TIMEOUT", "not-a-duration")
	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid REQUEST_TIMEOUT, got nil")
	}
	if !strings.Contains(err.Error(), "REQUEST_TIMEOUT") {
		t.Errorf("error should mention REQUEST_TIMEOUT, got: %v", err)
	}
}

func TestLoad_PUEMultiplier_BelowOne_Defaults(t *testing.T) {
	setRequired(t)
	t.Setenv("PUE_MULTIPLIER", "0.5") // < 1.0 → must default to 1.3
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.PUEMultiplier != 1.3 {
		t.Errorf("PUEMultiplier = %f, want 1.3 for invalid < 1.0 input", cfg.PUEMultiplier)
	}
}

func TestLoad_PUEMultiplier_InvalidString_Defaults(t *testing.T) {
	setRequired(t)
	t.Setenv("PUE_MULTIPLIER", "bad")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.PUEMultiplier != 1.3 {
		t.Errorf("PUEMultiplier = %f, want 1.3 for unparsable input", cfg.PUEMultiplier)
	}
}

func TestLoad_PUEMultiplier_ValidValue(t *testing.T) {
	setRequired(t)
	t.Setenv("PUE_MULTIPLIER", "1.5")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.PUEMultiplier != 1.5 {
		t.Errorf("PUEMultiplier = %f, want 1.5", cfg.PUEMultiplier)
	}
}

func TestLoad_AllowedOrigins_Split(t *testing.T) {
	setRequired(t)
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com , https://local.dev")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	want := []string{"https://app.example.com", "https://admin.example.com", "https://local.dev"}
	if len(cfg.AllowedOrigins) != len(want) {
		t.Fatalf("AllowedOrigins len = %d, want %d: %v", len(cfg.AllowedOrigins), len(want), cfg.AllowedOrigins)
	}
	for i, w := range want {
		if cfg.AllowedOrigins[i] != w {
			t.Errorf("AllowedOrigins[%d] = %q, want %q", i, cfg.AllowedOrigins[i], w)
		}
	}
}

func TestLoad_AllowedOrigins_Default(t *testing.T) {
	setRequired(t)
	t.Setenv("ALLOWED_ORIGINS", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins = %v, want [http://localhost:3000]", cfg.AllowedOrigins)
	}
}

func TestLoad_FeatureFlags_DefaultTrue(t *testing.T) {
	setRequired(t)
	for _, key := range []string{
		"ENABLE_PROMPT_OPTIMIZATION",
		"ENABLE_CARBON_TRACKING",
		"ENABLE_CACHE",
		"ENABLE_FALLBACK",
	} {
		t.Setenv(key, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.EnablePromptOptimization {
		t.Error("EnablePromptOptimization should default to true")
	}
	if !cfg.EnableCarbonTracking {
		t.Error("EnableCarbonTracking should default to true")
	}
	if !cfg.EnableCache {
		t.Error("EnableCache should default to true")
	}
	if !cfg.EnableFallback {
		t.Error("EnableFallback should default to true")
	}
}

func TestLoad_FeatureFlags_Disabled(t *testing.T) {
	setRequired(t)
	t.Setenv("ENABLE_PROMPT_OPTIMIZATION", "false")
	t.Setenv("ENABLE_CARBON_TRACKING", "false")
	t.Setenv("ENABLE_CACHE", "false")
	t.Setenv("ENABLE_FALLBACK", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.EnablePromptOptimization {
		t.Error("EnablePromptOptimization should be false")
	}
	if cfg.EnableCarbonTracking {
		t.Error("EnableCarbonTracking should be false")
	}
	if cfg.EnableCache {
		t.Error("EnableCache should be false")
	}
	if cfg.EnableFallback {
		t.Error("EnableFallback should be false")
	}
}

func TestLoad_RequiredFieldValues(t *testing.T) {
	const dbURL = "postgres://user:pass@localhost:5432/mydb"
	const redisURL = "redis://localhost:6379/1"
	const jwtSecret = "super-secret-key"

	t.Setenv("DATABASE_URL", dbURL)
	t.Setenv("REDIS_URL", redisURL)
	t.Setenv("JWT_SECRET", jwtSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DatabaseURL != dbURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, dbURL)
	}
	if cfg.RedisURL != redisURL {
		t.Errorf("RedisURL = %q, want %q", cfg.RedisURL, redisURL)
	}
	if cfg.JWTSecret != jwtSecret {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, jwtSecret)
	}
}
