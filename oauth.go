package main

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

const grokOAuthRefreshBuffer = 5 * time.Minute

var grokOAuthTokenEndpoint = "https://auth.x.ai/oauth2/token"

type grokAuthEntry struct {
	Key          string `json:"key"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	OIDCIssuer   string `json:"oidc_issuer"`
	OIDCClientID string `json:"oidc_client_id"`
	AuthMode     string `json:"auth_mode"`
	Email        string `json:"email"`
}

func grokAuthFilePath() string {
	if custom := strings.TrimSpace(os.Getenv("GROK_AUTH_JSON")); custom != "" {
		return custom
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".grok", "auth.json")
}

func authModePreference() string {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("GROK_IMAGE_AUTH")))
	switch mode {
	case "oauth", "api_key", "auto":
		return mode
	default:
		return "auto"
	}
}

func loadGrokAuthStore() (map[string]grokAuthEntry, string, error) {
	path := grokAuthFilePath()
	if path == "" {
		return nil, "", fmt.Errorf("home directory unavailable")
	}
	// #nosec G304 - path is user home or explicit env override
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, path, fmt.Errorf("invalid auth.json: %w", err)
	}

	store := make(map[string]grokAuthEntry, len(raw))
	for scope, entryRaw := range raw {
		var entry grokAuthEntry
		if err := json.Unmarshal(entryRaw, &entry); err != nil {
			continue
		}
		if strings.TrimSpace(entry.Key) == "" {
			continue
		}
		store[scope] = entry
	}
	if len(store) == 0 {
		return nil, path, fmt.Errorf("no OAuth entries with access tokens found")
	}
	return store, path, nil
}

func selectGrokAuthEntry(store map[string]grokAuthEntry) (string, grokAuthEntry) {
	preferredScopes := []string{
		"https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828",
		"https://accounts.x.ai/sign-in",
	}
	for _, scope := range preferredScopes {
		if entry, ok := store[scope]; ok {
			return scope, entry
		}
	}
	for scope, entry := range store {
		if strings.Contains(scope, "auth.x.ai") || strings.Contains(scope, "accounts.x.ai") {
			return scope, entry
		}
	}
	for scope, entry := range store {
		return scope, entry
	}
	return "", grokAuthEntry{}
}

func parseGrokExpiresAt(expiresAt string) (time.Time, bool) {
	expiresAt = strings.TrimSpace(expiresAt)
	if expiresAt == "" {
		return time.Time{}, false
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, expiresAt); err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

func grokTokenNeedsRefresh(entry grokAuthEntry) bool {
	expiresAt, ok := parseGrokExpiresAt(entry.ExpiresAt)
	if !ok {
		return false
	}
	return time.Until(expiresAt) <= grokOAuthRefreshBuffer
}

type grokTokenRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func refreshGrokOAuthToken(ctx context.Context, entry grokAuthEntry) (grokAuthEntry, error) {
	if strings.TrimSpace(entry.RefreshToken) == "" {
		return entry, fmt.Errorf("no refresh token available; run `grok login` to re-authenticate")
	}
	clientID := strings.TrimSpace(entry.OIDCClientID)
	if clientID == "" {
		clientID = "b1a00492-073a-47ea-816f-4c329264a828"
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", entry.RefreshToken)
	form.Set("client_id", clientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, grokOAuthTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return entry, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return entry, fmt.Errorf("oauth refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return entry, fmt.Errorf("failed to read oauth refresh response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return entry, fmt.Errorf("oauth refresh failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var refreshed grokTokenRefreshResponse
	if err := json.Unmarshal(body, &refreshed); err != nil {
		return entry, fmt.Errorf("invalid oauth refresh response: %w", err)
	}
	if strings.TrimSpace(refreshed.AccessToken) == "" {
		return entry, fmt.Errorf("oauth refresh response missing access_token")
	}

	entry.Key = refreshed.AccessToken
	if refreshed.RefreshToken != "" {
		entry.RefreshToken = refreshed.RefreshToken
	}
	if refreshed.ExpiresIn > 0 {
		entry.ExpiresAt = time.Now().UTC().Add(time.Duration(refreshed.ExpiresIn) * time.Second).Format("2006-01-02T15:04:05.000000Z")
	}
	return entry, nil
}

func persistGrokAuthEntry(path, scope string, entry grokAuthEntry) error {
	// #nosec G304 - path is user home or explicit env override
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	updated, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	raw[scope] = updated
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	// #nosec G306 - mirror Grok auth.json permissions
	return os.WriteFile(path, out, 0600)
}

func loadGrokOAuthToken(ctx context.Context) (string, string, error) {
	store, path, err := loadGrokAuthStore()
	if err != nil {
		return "", "", err
	}
	scope, entry := selectGrokAuthEntry(store)
	if scope == "" {
		return "", "", fmt.Errorf("no usable OAuth entry found")
	}

	if grokTokenNeedsRefresh(entry) {
		refreshed, refreshErr := refreshGrokOAuthToken(ctx, entry)
		if refreshErr != nil {
			if expiresAt, ok := parseGrokExpiresAt(entry.ExpiresAt); ok && time.Now().After(expiresAt) {
				return "", "", refreshErr
			}
			fmt.Fprintln(os.Stderr, "[grok-image-mcp] OAuth refresh failed, using existing token:", refreshErr)
		} else {
			entry = refreshed
			if err := persistGrokAuthEntry(path, scope, entry); err != nil {
				fmt.Fprintln(os.Stderr, "[grok-image-mcp] OAuth token refreshed but failed to update auth.json:", err)
			}
		}
	}

	source := "grok_oauth"
	if entry.Email != "" {
		source = "grok_oauth (" + entry.Email + ")"
	}
	return entry.Key, source, nil
}

func loadAPIKeyFromConfigFiles() (string, bool) {
	var config Config

	// #nosec G304 - local config file read is intentional
	if data, err := os.ReadFile(".grok-image-config.json"); err == nil {
		if err := json.Unmarshal(data, &config); err == nil && config.XAIAPIKey != "" {
			home, _ := os.UserHomeDir()
			globalPath := filepath.Join(home, ".grok-image-config.json")
			if _, err := os.Stat(globalPath); os.IsNotExist(err) {
				// #nosec - global path is home-dir relative, migration is safe and intentional
				_ = os.WriteFile(globalPath, data, 0600)
				fmt.Fprintln(os.Stderr, "[grok-image-mcp] Automatically migrated local configuration to global:", globalPath)
			}
			return config.XAIAPIKey, true
		}
	}

	home, _ := os.UserHomeDir()
	globalPath := filepath.Join(home, ".grok-image-config.json")
	// #nosec G304 - global config file read is intentional
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := json.Unmarshal(data, &config); err == nil && config.XAIAPIKey != "" {
			return config.XAIAPIKey, true
		}
	}
	return "", false
}

func resolveCredential(ctx context.Context) (token string, source string) {
	mode := authModePreference()

	tryOAuth := func() (string, string, bool) {
		if mode == "api_key" {
			return "", "", false
		}
		token, src, err := loadGrokOAuthToken(ctx)
		if err != nil {
			return "", "", false
		}
		return token, src, true
	}

	tryEnv := func() (string, string, bool) {
		if mode == "oauth" {
			return "", "", false
		}
		if key := strings.TrimSpace(os.Getenv("XAI_API_KEY")); key != "" {
			return key, "environment", true
		}
		return "", "", false
	}

	tryConfig := func() (string, string, bool) {
		if mode == "oauth" {
			return "", "", false
		}
		if key, ok := loadAPIKeyFromConfigFiles(); ok {
			return key, "config_file", true
		}
		return "", "", false
	}

	if mode == "oauth" {
		if token, src, ok := tryOAuth(); ok {
			return token, src
		}
		return "", "not_configured"
	}
	if mode == "api_key" {
		if token, src, ok := tryEnv(); ok {
			return token, src
		}
		if token, src, ok := tryConfig(); ok {
			return token, src
		}
		return "", "not_configured"
	}

	// auto: subscription OAuth first, then API key paths
	if token, src, ok := tryOAuth(); ok {
		return token, src
	}
	if token, src, ok := tryEnv(); ok {
		return token, src
	}
	if token, src, ok := tryConfig(); ok {
		return token, src
	}
	return "", "not_configured"
}

func loadConfig() (string, string) {
	ctx := globalCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return resolveCredential(ctx)
}

func credentialSourceDescription(source string) string {
	switch {
	case strings.HasPrefix(source, "grok_oauth"):
		return "Grok subscription OAuth (~/.grok/auth.json from `grok login`)"
	case source == "environment":
		return "Environment variable (XAI_API_KEY)"
	case source == "config_file":
		return "Configuration file (~/.grok-image-config.json)"
	default:
		return ""
	}
}