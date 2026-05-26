package auth

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryI is the data-access contract satisfied by *Repository in
// production and by test fakes in unit tests.
type RepositoryI interface {
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByID(ctx context.Context, userID uuid.UUID) (*User, error)
	FindOrgByID(ctx context.Context, orgID uuid.UUID) (*Organization, error)
	FindAPIKeyByPrefix(ctx context.Context, prefix string) (*APIKey, error)
	TouchAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID) error
	CreateOrgAndUser(ctx context.Context, org OrgInput, user UserInput, key *APIKey) (*Organization, *User, error)
	UpdateOrg(ctx context.Context, orgID uuid.UUID, name string, qualityThreshold float32, energyBudgetKwh *float32) error
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error)
	InviteMember(ctx context.Context, orgID uuid.UUID, email, passwordHash, name, role string) (*User, error)
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
	ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]APIKey, error)
	CreateAPIKey(ctx context.Context, k *APIKey) error
	RevokeAPIKey(ctx context.Context, keyID, orgID uuid.UUID) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}
