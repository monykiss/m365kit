// Package auth provides Microsoft 365 OAuth 2.0 authentication via device code flow.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	graphBaseURL   = "https://graph.microsoft.com/v1.0"
	authorityBase  = "https://login.microsoftonline.com/common/oauth2/v2.0"
	defaultScopes  = "Files.ReadWrite Sites.ReadWrite.All User.Read Chat.ReadWrite ChannelMessage.Send Team.ReadBasic.All offline_access"
	tokenFileName  = "token.json"
	refreshWindow  = 5 * time.Minute
	pollInterval   = 5 * time.Second
	deviceTimeout  = 5 * time.Minute
)

// Token holds the OAuth 2.0 tokens from Microsoft.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// IsExpired returns true if the token has expired.
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// ExpiresIn returns the duration until the token expires.
func (t *Token) ExpiresIn() time.Duration {
	return time.Until(t.ExpiresAt)
}

// NeedsRefresh returns true if the token expires within the refresh window.
func (t *Token) NeedsRefresh() bool {
	return t.ExpiresIn() < refreshWindow
}

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// DeviceCodeFlow initiates the OAuth device code flow.
func DeviceCodeFlow(ctx context.Context, clientID string) (*Token, error) {
	if clientID == "" {
		return nil, fmt.Errorf("KIT_AZURE_CLIENT_ID is not set — register an Azure AD app and set this environment variable\nSee: kit auth --help")
	}

	// Step 1: Request device code
	resp, err := http.PostForm(authorityBase+"/devicecode", url.Values{
		"client_id": {clientID},
		"scope":     {defaultScopes},
	})
	if err != nil {
		return nil, fmt.Errorf("could not contact Microsoft login service: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var dcResp deviceCodeResponse
	if err := json.Unmarshal(body, &dcResp); err != nil {
		return nil, fmt.Errorf("could not parse device code response: %w", err)
	}

	// Step 2: Display instructions
	fmt.Printf("Open %s and enter code: %s\n", dcResp.VerificationURI, dcResp.UserCode)
	fmt.Println("Waiting for authorization...")

	// Step 3: Poll for token
	interval := pollInterval
	if dcResp.Interval > 0 {
		interval = time.Duration(dcResp.Interval) * time.Second
	}

	deadline := time.Now().Add(deviceTimeout)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("device code authorization timed out — run: kit auth login to try again")
		}

		token, err := pollToken(clientID, dcResp.DeviceCode)
		if err != nil {
			if err.Error() == "authorization_pending" {
				continue
			}
			if err.Error() == "slow_down" {
				interval += 5 * time.Second
				continue
			}
			return nil, err
		}

		if err := SaveToken(token); err != nil {
			return nil, fmt.Errorf("authenticated but could not save token: %w", err)
		}

		return token, nil
	}
}

func pollToken(clientID, deviceCode string) (*Token, error) {
	resp, err := http.PostForm(authorityBase+"/token", url.Values{
		"client_id":   {clientID},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {deviceCode},
	})
	if err != nil {
		return nil, fmt.Errorf("token poll failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("could not parse token response: %w", err)
	}

	if tr.Error != "" {
		if tr.Error == "authorization_pending" || tr.Error == "slow_down" {
			return nil, fmt.Errorf(tr.Error)
		}
		if tr.Error == "expired_token" {
			return nil, fmt.Errorf("authorization code expired — run: kit auth login to try again")
		}
		return nil, fmt.Errorf("authentication failed: %s — %s", tr.Error, tr.ErrorDesc)
	}

	return &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		TokenType:    tr.TokenType,
	}, nil
}

// RefreshIfNeeded refreshes the token if it expires within 5 minutes.
func RefreshIfNeeded(ctx context.Context, t *Token, clientID string) (*Token, error) {
	if !t.NeedsRefresh() {
		return t, nil
	}
	if t.RefreshToken == "" {
		return nil, fmt.Errorf("token expired and no refresh token available — run: kit auth login")
	}

	resp, err := http.PostForm(authorityBase+"/token", url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {t.RefreshToken},
		"scope":         {defaultScopes},
	})
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("could not parse refresh response: %w", err)
	}

	if tr.Error != "" {
		return nil, fmt.Errorf("token refresh failed: %s — run: kit auth login", tr.ErrorDesc)
	}

	newToken := &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second),
		TokenType:    tr.TokenType,
	}

	if err := SaveToken(newToken); err != nil {
		return nil, fmt.Errorf("refreshed but could not save token: %w", err)
	}

	return newToken, nil
}

// tokenDir returns the path to ~/.kit/ directory.
func tokenDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".kit"), nil
}

// tokenPath returns the full path to the token file.
func tokenPath() (string, error) {
	dir, err := tokenDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, tokenFileName), nil
}

// TokenPathOverride allows tests to override the token path.
var TokenPathOverride string

func resolvedTokenPath() (string, error) {
	if TokenPathOverride != "" {
		return TokenPathOverride, nil
	}
	return tokenPath()
}

// LoadToken loads the saved token from ~/.kit/token.json.
func LoadToken() (*Token, error) {
	path, err := resolvedTokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not authenticated — run: kit auth login")
		}
		return nil, fmt.Errorf("could not read token file: %w", err)
	}

	var t Token
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("token file is corrupted — run: kit auth login")
	}

	return &t, nil
}

// SaveToken persists the token to ~/.kit/token.json with 0600 permissions.
func SaveToken(t *Token) error {
	path, err := resolvedTokenPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("could not create token directory: %w", err)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal token: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("could not write token file: %w", err)
	}

	return nil
}

// DeleteToken removes the token file.
func DeleteToken() error {
	path, err := resolvedTokenPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not delete token: %w", err)
	}
	return nil
}

// WhoAmI returns the display name and email of the authenticated user.
func WhoAmI(ctx context.Context, client *http.Client) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", graphBaseURL+"/me", nil)
	if err != nil {
		return "", "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("Graph API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Graph API returned %d: %s", resp.StatusCode, string(body))
	}

	var user struct {
		DisplayName       string `json:"displayName"`
		UserPrincipalName string `json:"userPrincipalName"`
		Mail              string `json:"mail"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return "", "", fmt.Errorf("could not parse user info: %w", err)
	}

	email := user.Mail
	if email == "" {
		email = user.UserPrincipalName
	}

	return user.DisplayName, email, nil
}

// GraphBaseURL returns the base URL for Graph API calls.
func GraphBaseURL() string {
	return graphBaseURL
}

// Scopes returns the OAuth scopes as a display-friendly slice.
func Scopes() []string {
	return strings.Split(defaultScopes, " ")
}
