package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// newGitHubConfig returns the oauth2.Config for GitHub with user:email scope.
func newGitHubConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     github.Endpoint,
		Scopes:       []string{"user:email"},
		RedirectURL:  redirectURL,
	}
}

// newGoogleConfig returns the oauth2.Config for Google with openid+email+profile scope.
func newGoogleConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"openid", "email", "profile"},
		RedirectURL:  redirectURL,
	}
}

// FetchGitHubProfile exchanges the token for the GitHub user's providerID, email, and name.
// providerID is the stable numeric GitHub user ID converted to a string.
// If the profile email is empty it falls back to the /user/emails API.
func FetchGitHubProfile(ctx context.Context, token *oauth2.Token, conf *oauth2.Config) (providerID, email, name string, err error) {
	client := conf.Client(ctx, token)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", "", "", fmt.Errorf("build github /user request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("github /user: %w", err)
	}
	defer resp.Body.Close()

	var profile struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return "", "", "", fmt.Errorf("decode github profile: %w", err)
	}

	displayName := profile.Name
	if displayName == "" {
		displayName = profile.Login
	}

	emailAddr := profile.Email
	if emailAddr == "" {
		emailAddr, err = fetchGitHubPrimaryEmail(ctx, client)
		if err != nil {
			return "", "", "", err
		}
	}

	return fmt.Sprintf("%d", profile.ID), emailAddr, displayName, nil
}

// fetchGitHubPrimaryEmail fetches the primary verified email from /user/emails.
func fetchGitHubPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("build github /user/emails request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github /user/emails: %w", err)
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decode github emails: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}
	return "", nil
}

// FetchGoogleProfile fetches the Google user's providerID, email, and name via userinfo.
// providerID is the stable "sub" claim from Google's OpenID Connect response.
func FetchGoogleProfile(ctx context.Context, token *oauth2.Token, conf *oauth2.Config) (providerID, email, name string, err error) {
	client := conf.Client(ctx, token)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return "", "", "", fmt.Errorf("build google userinfo request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()

	var profile struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return "", "", "", fmt.Errorf("decode google profile: %w", err)
	}

	return profile.Sub, profile.Email, profile.Name, nil
}
