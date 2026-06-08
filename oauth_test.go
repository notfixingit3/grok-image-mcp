package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTestGrokAuthFile(t *testing.T, dir string, entry grokAuthEntry) string {
	t.Helper()
	path := filepath.Join(dir, "auth.json")
	payload := map[string]grokAuthEntry{
		"https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828": entry,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSelectGrokAuthEntryPrefersKnownScope(t *testing.T) {
	store := map[string]grokAuthEntry{
		"other": {Key: "other-token"},
		"https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828": {Key: "preferred-token"},
	}
	scope, entry := selectGrokAuthEntry(store)
	if scope != "https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828" {
		t.Fatalf("unexpected scope: %q", scope)
	}
	if entry.Key != "preferred-token" {
		t.Fatalf("unexpected token: %q", entry.Key)
	}
}

func TestGrokTokenNeedsRefresh(t *testing.T) {
	future := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339Nano)
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339Nano)
	soon := time.Now().UTC().Add(2 * time.Minute).Format(time.RFC3339Nano)

	if grokTokenNeedsRefresh(grokAuthEntry{ExpiresAt: future}) {
		t.Fatal("future token should not need refresh")
	}
	if !grokTokenNeedsRefresh(grokAuthEntry{ExpiresAt: past}) {
		t.Fatal("expired token should need refresh")
	}
	if !grokTokenNeedsRefresh(grokAuthEntry{ExpiresAt: soon}) {
		t.Fatal("soon-to-expire token should need refresh")
	}
}

func TestResolveCredentialPrefersOAuthInAutoMode(t *testing.T) {
	dir := t.TempDir()
	path := writeTestGrokAuthFile(t, dir, grokAuthEntry{
		Key:          "oauth-token",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339Nano),
		OIDCClientID: "test-client",
		Email:        "user@example.com",
	})

	t.Setenv("GROK_AUTH_JSON", path)
	t.Setenv("GROK_IMAGE_AUTH", "auto")
	t.Setenv("XAI_API_KEY", "xai-should-not-win")

	token, source := resolveCredential(context.Background())
	if token != "oauth-token" {
		t.Fatalf("token = %q, want oauth-token", token)
	}
	if !strings.HasPrefix(source, "grok_oauth") {
		t.Fatalf("source = %q, want grok_oauth prefix", source)
	}
}

func TestResolveCredentialAPIKeyModeIgnoresOAuth(t *testing.T) {
	dir := t.TempDir()
	path := writeTestGrokAuthFile(t, dir, grokAuthEntry{
		Key:       "oauth-token",
		ExpiresAt: time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339Nano),
	})

	t.Setenv("GROK_AUTH_JSON", path)
	t.Setenv("GROK_IMAGE_AUTH", "api_key")
	t.Setenv("XAI_API_KEY", "xai-env-key")

	token, source := resolveCredential(context.Background())
	if token != "xai-env-key" {
		t.Fatalf("token = %q, want xai-env-key", token)
	}
	if source != "environment" {
		t.Fatalf("source = %q, want environment", source)
	}
}

func TestRefreshGrokOAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Fatalf("grant_type = %q", r.Form.Get("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	originalEndpoint := grokOAuthTokenEndpoint
	grokOAuthTokenEndpoint = server.URL
	t.Cleanup(func() { grokOAuthTokenEndpoint = originalEndpoint })

	entry := grokAuthEntry{
		Key:          "old-access",
		RefreshToken: "old-refresh",
		OIDCClientID: "test-client",
	}

	refreshed, err := refreshGrokOAuthToken(context.Background(), entry)
	if err != nil {
		t.Fatalf("refreshGrokOAuthToken failed: %v", err)
	}
	if refreshed.Key != "new-access" {
		t.Fatalf("refreshed key = %q", refreshed.Key)
	}
	if refreshed.RefreshToken != "new-refresh" {
		t.Fatalf("refreshed refresh token = %q", refreshed.RefreshToken)
	}
	if refreshed.ExpiresAt == "" {
		t.Fatal("expected expires_at to be set")
	}
}

func TestCredentialSourceDescription(t *testing.T) {
	if got := credentialSourceDescription("grok_oauth (user@example.com)"); !strings.Contains(got, "subscription OAuth") {
		t.Fatalf("unexpected description: %q", got)
	}
	if got := credentialSourceDescription("environment"); !strings.Contains(got, "XAI_API_KEY") {
		t.Fatalf("unexpected description: %q", got)
	}
}