package usage

import (
	"context"

	"github.com/ecollm/api/pkg/apierror"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FeedbackRepository persists feedback events for routing quality improvement.
type FeedbackRepository struct {
	db *pgxpool.Pool
}

func NewFeedbackRepository(db *pgxpool.Pool) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

type feedbackInput struct {
	Rating         int    `json:"rating"`          // 1–5
	Comment        string `json:"comment"`
	IssueType      string `json:"issue_type"`
	ExpectedOutput string `json:"expected_output"`
}

// InsertFeedback writes a feedback row, joining request metadata for denormalization.
func (r *FeedbackRepository) InsertFeedback(ctx context.Context, orgID, requestID string, in feedbackInput) error {
	var internalID uuid.UUID
	var modelUsed, taskType string
	err := r.db.QueryRow(ctx,
		`SELECT id, COALESCE(model_selected,''), COALESCE(task_type,'')
		 FROM requests WHERE request_id = $1 AND org_id = $2`,
		requestID, orgID,
	).Scan(&internalID, &modelUsed, &taskType)
	if err != nil {
		return err
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO feedback_events
		    (id, request_id, org_id, rating, comment, issue_type, expected_output, model_used, task_type)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		uuid.New(), internalID, orgUUID,
		in.Rating,
		nullFeedbackStr(in.Comment),
		nullFeedbackStr(in.IssueType),
		nullFeedbackStr(in.ExpectedOutput),
		modelUsed,
		taskType,
	)
	return err
}

func nullFeedbackStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// FeedbackHandler handles POST /v1/requests/:id/feedback.
type FeedbackHandler struct {
	repo *FeedbackRepository
}

func NewFeedbackHandler(repo *FeedbackRepository) *FeedbackHandler {
	return &FeedbackHandler{repo: repo}
}

// SubmitFeedback handles POST /v1/requests/:id/feedback.
func (h *FeedbackHandler) SubmitFeedback(c *fiber.Ctx) error {
	orgID, _ := c.Locals("org_id").(string)
	if orgID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(apierror.ErrUnauthorized)
	}

	requestID := c.Params("id")
	if requestID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("id", "required"))
	}

	var in feedbackInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.ErrInvalidRequest)
	}
	if in.Rating < 1 || in.Rating > 5 {
		return c.Status(fiber.StatusBadRequest).JSON(apierror.Validation("rating", "must be between 1 and 5"))
	}

	err := h.repo.InsertFeedback(c.UserContext(), orgID, requestID, in)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apierror.ErrNotFound)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(apierror.ErrInternal)
	}

	return c.SendStatus(fiber.StatusCreated)
}
