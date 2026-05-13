package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/ecollm/api/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

const oauthStateTTL = 5 * time.Minute

// profileFetcher is the signature for provider-specific profile-fetch functions.
type profileFetcher func(ctx context.Context, token *oauth2.Token, conf *oauth2.Config) (providerID, email, name string, err error)

// OAuthHandler handles the OAuth 2.0 begin and callback flows for GitHub and Google.
type OAuthHandler struct {
	svc         *Service
	redis       *redis.Client
	frontendURL string // e.g., http://localhost:3000
	github      *oauth2.Config
	google      *oauth2.Config
}

// NewOAuthHandler constructs OAuthHandler from the app config.
func NewOAuthHandler(svc *Service, redisClient *redis.Client, cfg *config.Config) *OAuthHandler {
	return &OAuthHandler{
		svc:         svc,
		redis:       redisClient,
		frontendURL: cfg.FrontendURL,
		github:      newGitHubConfig(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.APIBaseURL+"/auth/github/callback"),
		google:      newGoogleConfig(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.APIBaseURL+"/auth/google/callback"),
	}
}

// BeginGitHub handles GET /auth/github/begin.
func (h *OAuthHandler) BeginGitHub(c *fiber.Ctx) error {
	return h.beginOAuth(c, h.github)
}

// BeginGoogle handles GET /auth/google/begin.
func (h *OAuthHandler) BeginGoogle(c *fiber.Ctx) error {
	return h.beginOAuth(c, h.google)
}

// CallbackGitHub handles GET /auth/github/callback.
func (h *OAuthHandler) CallbackGitHub(c *fiber.Ctx) error {
	return h.handleCallback(c, "github", h.github, FetchGitHubProfile)
}

// CallbackGoogle handles GET /auth/google/callback.
func (h *OAuthHandler) CallbackGoogle(c *fiber.Ctx) error {
	return h.handleCallback(c, "google", h.google, FetchGoogleProfile)
}

// beginOAuth generates a CSRF state, stores it in Redis, and redirects to the provider.
func (h *OAuthHandler) beginOAuth(c *fiber.Ctx, conf *oauth2.Config) error {
	state, err := generateOAuthState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to generate state")
	}
	if err := h.redis.Set(c.UserContext(), "oauth_state:"+state, "1", oauthStateTTL).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to store state")
	}
	return c.Redirect(conf.AuthCodeURL(state, oauth2.AccessTypeOnline), fiber.StatusTemporaryRedirect)
}

// handleCallback validates the CSRF state, exchanges the code, fetches the profile,
// and runs account-linking logic before redirecting to the frontend.
func (h *OAuthHandler) handleCallback(
	c *fiber.Ctx,
	provider string,
	conf *oauth2.Config,
	fetch profileFetcher,
) error {
	// User cancelled or provider returned an error.
	if c.Query("error") != "" {
		return c.Redirect(h.frontendURL+"/register", fiber.StatusTemporaryRedirect)
	}

	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		return c.Redirect(h.frontendURL+"/register", fiber.StatusTemporaryRedirect)
	}

	// Validate and consume the CSRF state.
	stateKey := "oauth_state:" + state
	if h.redis.Exists(c.UserContext(), stateKey).Val() == 0 {
		return c.Redirect(h.frontendURL+"/callback?error=oauth_failed", fiber.StatusTemporaryRedirect)
	}
	h.redis.Del(c.UserContext(), stateKey)

	// Exchange authorisation code for access token.
	token, err := conf.Exchange(c.UserContext(), code)
	if err != nil {
		return c.Redirect(h.frontendURL+"/callback?error=oauth_failed", fiber.StatusTemporaryRedirect)
	}

	// Fetch provider profile.
	providerID, email, name, err := fetch(c.UserContext(), token, conf)
	if err != nil {
		return c.Redirect(h.frontendURL+"/callback?error=oauth_failed", fiber.StatusTemporaryRedirect)
	}

	if email == "" {
		return c.Redirect(h.frontendURL+"/callback?error=oauth_no_email", fiber.StatusTemporaryRedirect)
	}

	// Account-linking and JWT issuance.
	jwt, nextURL, err := h.svc.HandleOAuthCallback(c.UserContext(), provider, providerID, email, name)
	if err != nil {
		return c.Redirect(h.frontendURL+"/callback?error=oauth_failed", fiber.StatusTemporaryRedirect)
	}

	return c.Redirect(
		h.frontendURL+"/callback?token="+jwt+"&next="+nextURL,
		fiber.StatusTemporaryRedirect,
	)
}

// generateOAuthState returns a cryptographically random 32-hex-char string.
func generateOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
