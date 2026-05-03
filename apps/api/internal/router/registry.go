package router

import "github.com/jackc/pgx/v5/pgxpool"

// Scorer wraps scoring logic with a reference to the model registry.
type Scorer struct {
	db *pgxpool.Pool
}

func NewScorer(db *pgxpool.Pool) *Scorer {
	return &Scorer{db: db}
}