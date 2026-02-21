package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadTokenMissingFile(t *testing.T) {
	TokenPathOverride = filepath.Join(t.TempDir(), "nonexistent.json")
	defer func() { TokenPathOverride = "" }()

	_, err := LoadToken()
	if err == nil {
		t.Fatal("expected error when token file missing")
	}
	if !contains(err.Error(), "not authenticated") {
		t.Errorf("expected helpful error, got: %s", err.Error())
	}
}

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	TokenPathOverride = filepath.Join(dir, "token.json")
	defer func() { TokenPathOverride = "" }()

	token := &Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		TokenType:    "Bearer",
	}

	if err := SaveToken(token); err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(TokenPathOverride)
	if err != nil {
		t.Fatalf("could not stat token file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected 0600 permissions, got %o", perm)
	}

	// Load and verify
	loaded, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}
	if loaded.AccessToken != "test-access-token" {
		t.Errorf("access token mismatch: %q", loaded.AccessToken)
	}
	if loaded.RefreshToken != "test-refresh-token" {
		t.Errorf("refresh token mismatch: %q", loaded.RefreshToken)
	}
}

func TestTokenIsExpired(t *testing.T) {
	expired := &Token{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	if !expired.IsExpired() {
		t.Error("expected expired token to report IsExpired=true")
	}

	valid := &Token{ExpiresAt: time.Now().Add(1 * time.Hour)}
	if valid.IsExpired() {
		t.Error("expected valid token to report IsExpired=false")
	}
}

func TestTokenNeedsRefresh(t *testing.T) {
	// Expires in 3 minutes — needs refresh
	soon := &Token{ExpiresAt: time.Now().Add(3 * time.Minute)}
	if !soon.NeedsRefresh() {
		t.Error("expected token expiring in 3 min to need refresh")
	}

	// Expires in 10 minutes — does not need refresh
	later := &Token{ExpiresAt: time.Now().Add(10 * time.Minute)}
	if later.NeedsRefresh() {
		t.Error("expected token expiring in 10 min to NOT need refresh")
	}
}

func TestRefreshIfNeededSkipsWhenValid(t *testing.T) {
	token := &Token{
		AccessToken:  "still-valid",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}

	ctx := context.Background()
	result, err := RefreshIfNeeded(ctx, token, "test-client-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken != "still-valid" {
		t.Errorf("expected original token, got %q", result.AccessToken)
	}
}

func TestDeleteToken(t *testing.T) {
	dir := t.TempDir()
	TokenPathOverride = filepath.Join(dir, "token.json")
	defer func() { TokenPathOverride = "" }()

	if err := SaveToken(&Token{AccessToken: "x", ExpiresAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	if err := DeleteToken(); err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	if _, err := os.Stat(TokenPathOverride); !os.IsNotExist(err) {
		t.Error("expected token file to be deleted")
	}
}

func TestDeleteTokenNonExistent(t *testing.T) {
	TokenPathOverride = filepath.Join(t.TempDir(), "nope.json")
	defer func() { TokenPathOverride = "" }()

	if err := DeleteToken(); err != nil {
		t.Errorf("DeleteToken on nonexistent file should not error: %v", err)
	}
}

func TestRequireAuthNoToken(t *testing.T) {
	TokenPathOverride = filepath.Join(t.TempDir(), "nope.json")
	defer func() { TokenPathOverride = "" }()

	ctx := context.Background()
	_, err := RequireAuth(ctx)
	if err == nil {
		t.Fatal("expected error when no token")
	}
	if !contains(err.Error(), "not authenticated") {
		t.Errorf("expected helpful error, got: %s", err.Error())
	}
}

func TestRequireAuthNoClientID(t *testing.T) {
	dir := t.TempDir()
	TokenPathOverride = filepath.Join(dir, "token.json")
	defer func() { TokenPathOverride = "" }()

	if err := SaveToken(&Token{
		AccessToken: "test",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	t.Setenv("KIT_AZURE_CLIENT_ID", "")

	ctx := context.Background()
	_, err := RequireAuth(ctx)
	if err == nil {
		t.Fatal("expected error when no client ID")
	}
	if !contains(err.Error(), "KIT_AZURE_CLIENT_ID") {
		t.Errorf("expected client ID error, got: %s", err.Error())
	}
}

func TestDeviceCodeFlowNoClientID(t *testing.T) {
	ctx := context.Background()
	_, err := DeviceCodeFlow(ctx, "")
	if err == nil {
		t.Fatal("expected error with empty client ID")
	}
	if !contains(err.Error(), "KIT_AZURE_CLIENT_ID") {
		t.Errorf("expected helpful error, got: %s", err.Error())
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
