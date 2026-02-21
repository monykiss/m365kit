package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

// BearerTransport injects the Bearer token into every HTTP request.
type BearerTransport struct {
	Token string
	Base  http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t *BearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.Token)
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}

// RequireAuth loads and validates the auth token, returning an authenticated HTTP client.
func RequireAuth(ctx context.Context) (*http.Client, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated — run: kit auth login\n(requires KIT_AZURE_CLIENT_ID environment variable)")
	}

	clientID := os.Getenv("KIT_AZURE_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("KIT_AZURE_CLIENT_ID not set — see: kit auth --help")
	}

	token, err = RefreshIfNeeded(ctx, token, clientID)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w\nRun: kit auth login", err)
	}

	client := &http.Client{
		Transport: &BearerTransport{Token: token.AccessToken},
	}

	return client, nil
}
