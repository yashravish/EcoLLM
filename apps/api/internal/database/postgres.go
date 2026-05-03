package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// NewPostgresPool creates a pgx connection pool, verifies connectivity, and
// logs the result. Returns an error if the pool cannot be established.
func NewPostgresPool(databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	// Pool sizing: enough headroom for concurrent requests without over-allocating
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}

	stat := pool.Stat()
	log.Info().
		Int32("total_conns", stat.TotalConns()).
		Int32("idle_conns", stat.IdleConns()).
		Str("database_url", maskURL(databaseURL)).
		Msg("postgres pool connected")

	return pool, nil
}

// maskURL replaces the password segment of a DSN with "***" for safe logging.
func maskURL(dsn string) string {
	// Naive masking: hide anything between :// user:pass@ boundary
	const placeholder = "***:***@"
	if i := indexOf(dsn, "@"); i >= 0 {
		if j := indexOf(dsn, "://"); j >= 0 {
			return dsn[:j+3] + placeholder + dsn[i+1:]
		}
	}
	return "***"
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
