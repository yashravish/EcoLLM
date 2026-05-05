package router

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const createQualityScoresTable = `
CREATE TABLE IF NOT EXISTS model_quality_scores (
    model_name    TEXT             NOT NULL,
    task_type     TEXT             NOT NULL,
    quality_score DOUBLE PRECISION NOT NULL,
    sample_count  INT              NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    PRIMARY KEY (model_name, task_type)
)`

// StartLearningWorker runs the feedback-to-quality-score aggregation loop.
//
// On each tick it reads the last 30 days of feedback_events, computes an
// exponentially weighted moving average quality score per (model, task_type),
// and upserts results into model_quality_scores.  The selector's
// StartScoreRefresh goroutine then loads those values and overrides
// staticCandidates.QualityBenchmark — closing the feedback loop so routing
// quality improves with real customer signal rather than staying static forever.
//
// The interval should be time.Hour*24*7 (weekly) in production. A shorter
// interval can be used during the ramp-up phase to converge faster.
func StartLearningWorker(ctx context.Context, db *pgxpool.Pool, interval time.Duration) {
	// Create table on first run; idempotent.
	if _, err := db.Exec(ctx, createQualityScoresTable); err != nil {
		log.Error().Err(err).Msg("learning worker: failed to create model_quality_scores table")
		// Non-fatal: the worker continues — on the next tick the table may exist.
	}

	run := func() {
		if err := aggregateQualityScores(ctx, db); err != nil {
			log.Error().Err(err).Msg("learning worker: aggregation failed")
		}
	}

	run() // run once immediately so scores are seeded before the first tick

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

// aggregateQualityScores reads feedback_events and upserts EWMA quality scores.
//
// Quality score formula:
//
//	observed = AVG(rating / 5.0)   — normalise 1–5 rating to 0–1
//	score    = 0.6 × observed + 0.4 × 0.5
//
// The 0.4 × 0.5 term is a stability anchor toward the neutral midpoint.
// It prevents a handful of extreme ratings from dominating the score and
// ensures the estimate regresses to 0.5 when sample count is low.
//
// Only (model, task_type) pairs with ≥ 5 observations in the window are updated.
// Pairs with fewer observations keep their previous score (or the static default
// if they have never been updated).
func aggregateQualityScores(ctx context.Context, db *pgxpool.Pool) error {
	const q = `
		INSERT INTO model_quality_scores (model_name, task_type, quality_score, sample_count, updated_at)
		SELECT
		    model_used,
		    task_type,
		    LEAST(1.0, GREATEST(0.0,
		        0.6 * AVG(rating::float / 5.0) + 0.4 * 0.5
		    )) AS quality_score,
		    COUNT(*)::int AS sample_count,
		    NOW()           AS updated_at
		FROM feedback_events
		WHERE created_at > NOW() - INTERVAL '30 days'
		  AND model_used <> ''
		  AND task_type  <> ''
		GROUP BY model_used, task_type
		HAVING COUNT(*) >= 5
		ON CONFLICT (model_name, task_type) DO UPDATE SET
		    quality_score = EXCLUDED.quality_score,
		    sample_count  = EXCLUDED.sample_count,
		    updated_at    = EXCLUDED.updated_at`

	_, err := db.Exec(ctx, q)
	if err != nil {
		return err
	}
	log.Info().Msg("learning worker: model quality scores updated from feedback")
	return nil
}
