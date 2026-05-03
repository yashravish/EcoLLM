package chat

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles SQL for the chat/requests domain.
type Repository struct {
	db *pgxpool.Pool
}

func NewRequestRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Insert writes a completed request record to the requests table.
func (r *Repository) Insert(ctx context.Context, rec *RequestRecord) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO requests (
			id, org_id, request_id,
			prompt_original, prompt_optimized,
			task_type, complexity,
			model_selected, model_fallback,
			routing_score, routing_confidence,
			used_fallback, cache_hit,
			response_text, finish_reason,
			prompt_tokens, completion_tokens, total_tokens,
			latency_ms, status, error_message
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21
		)`,
		rec.ID, rec.OrgID, rec.RequestID,
		rec.PromptOriginal, rec.PromptOptimized,
		rec.TaskType, rec.Complexity,
		rec.ModelSelected, rec.ModelFallback,
		rec.RoutingScore, rec.RoutingConfidence,
		rec.UsedFallback, rec.CacheHit,
		rec.ResponseText, rec.FinishReason,
		rec.PromptTokens, rec.CompletionTokens, rec.TotalTokens,
		rec.LatencyMs, rec.Status, rec.ErrorMessage,
	)
	return err
}

// FindByID returns a single request by its external request_id, scoped to orgID.
func (r *Repository) FindByID(ctx context.Context, requestID, orgID string) (*RequestRecord, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, org_id, request_id, prompt_original, prompt_optimized,
		        task_type, complexity, model_selected, model_fallback,
		        routing_score, routing_confidence, used_fallback, cache_hit,
		        response_text, finish_reason,
		        prompt_tokens, completion_tokens, total_tokens,
		        latency_ms, status, error_message, created_at
		 FROM requests
		 WHERE request_id = $1 AND org_id = $2`,
		requestID, orgID,
	)

	rec := &RequestRecord{}
	err := row.Scan(
		&rec.ID, &rec.OrgID, &rec.RequestID,
		&rec.PromptOriginal, &rec.PromptOptimized,
		&rec.TaskType, &rec.Complexity,
		&rec.ModelSelected, &rec.ModelFallback,
		&rec.RoutingScore, &rec.RoutingConfidence,
		&rec.UsedFallback, &rec.CacheHit,
		&rec.ResponseText, &rec.FinishReason,
		&rec.PromptTokens, &rec.CompletionTokens, &rec.TotalTokens,
		&rec.LatencyMs, &rec.Status, &rec.ErrorMessage,
		&rec.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rec, nil
}
