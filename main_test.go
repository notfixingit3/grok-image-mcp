package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatXAIErrorCredits(t *testing.T) {
	body := []byte(`{"error":"Your newly created team doesn't have any credits or licenses yet."}`)
	msg := formatXAIError(http.StatusForbidden, body)
	if !strings.Contains(msg, "console.x.ai") {
		t.Fatalf("expected credits guidance, got: %s", msg)
	}
}

func TestFormatXAIErrorRateLimit(t *testing.T) {
	msg := formatXAIError(http.StatusTooManyRequests, []byte(`{"error":"rate limit"}`))
	if !strings.Contains(strings.ToLower(msg), "rate limit") {
		t.Fatalf("expected rate limit message, got: %s", msg)
	}
}

func TestGetMimeTypeAndExtension(t *testing.T) {
	cases := []struct {
		path string
		mime string
		ext  string
	}{
		{"photo.jpg", "image/jpeg", ".jpg"},
		{"photo.jpeg", "image/jpeg", ".jpg"},
		{"photo.png", "image/png", ".png"},
		{"photo.webp", "image/webp", ".webp"},
	}

	for _, tc := range cases {
		mime := getMimeType(tc.path)
		if mime != tc.mime {
			t.Fatalf("getMimeType(%q) = %q, want %q", tc.path, mime, tc.mime)
		}
		ext := extensionForMimeType(mime)
		if ext != tc.ext {
			t.Fatalf("extensionForMimeType(%q) = %q, want %q", mime, ext, tc.ext)
		}
	}
}

func TestResolveModel(t *testing.T) {
	t.Setenv("GROK_IMAGE_MODEL", "grok-imagine-image")
	custom := "custom-model"
	got := resolveModel(&custom)
	if got != "custom-model" {
		t.Fatalf("custom model not respected: %q", got)
	}
	got = resolveModel(nil)
	if got != "grok-imagine-image" {
		t.Fatalf("env model not respected: %q", got)
	}
	t.Setenv("GROK_IMAGE_MODEL", "")
	got = resolveModel(nil)
	if got != "grok-imagine-image-quality" {
		t.Fatalf("default model wrong: %q", got)
	}
}

func TestGetImagesDirectoryCustomEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GROK_IMAGES_DIR", dir)
	if got := getImagesDirectory(); got != dir {
		t.Fatalf("getImagesDirectory() = %q, want %q", got, dir)
	}
}

func TestValidateImageFile(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "ok.png")
	if err := os.WriteFile(valid, []byte("png"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateImageFile(valid); err != nil {
		t.Fatalf("valid image rejected: %v", err)
	}

	invalidExt := filepath.Join(dir, "bad.gif")
	if err := os.WriteFile(invalidExt, []byte("gif"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateImageFile(invalidExt); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported format error, got: %v", err)
	}

	tooLarge := filepath.Join(dir, "big.jpg")
	if err := os.WriteFile(tooLarge, make([]byte, maxImageFileBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateImageFile(tooLarge); err == nil || !strings.Contains(err.Error(), "20 MiB") {
		t.Fatalf("expected size limit error, got: %v", err)
	}
}

func TestBuildEditImageRefsWarnings(t *testing.T) {
	dir := t.TempDir()
	mainImage := filepath.Join(dir, "main.png")
	if err := os.WriteFile(mainImage, []byte("main"), 0600); err != nil {
		t.Fatal(err)
	}

	refs := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		path := filepath.Join(dir, fmt.Sprintf("ref%d.png", i))
		if err := os.WriteFile(path, []byte("ref"), 0600); err != nil {
			t.Fatal(err)
		}
		refs = append(refs, path)
	}

	built, warning, err := buildEditImageRefs(mainImage, refs)
	if err != nil {
		t.Fatalf("buildEditImageRefs failed: %v", err)
	}
	if len(built) != maxEditImages {
		t.Fatalf("expected %d images, got %d", maxEditImages, len(built))
	}
	if !strings.Contains(warning, "skipped") {
		t.Fatalf("expected skipped warning, got: %q", warning)
	}
}

func TestValidateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	originalBase := xaiBaseURL
	xaiBaseURL = server.URL
	t.Cleanup(func() { xaiBaseURL = originalBase })

	status, body, err := validateAPIKey(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("validateAPIKey failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, string(body))
	}
}

func TestValidateAPIKeyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
	}))
	defer server.Close()

	originalBase := xaiBaseURL
	xaiBaseURL = server.URL
	t.Cleanup(func() { xaiBaseURL = originalBase })

	_, _, err := validateAPIKey(context.Background(), "bad-key")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDoXAIRequestRetriesOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"aGVsbG8=","mime_type":"image/png"}]}`))
	}))
	defer server.Close()

	originalBase := xaiBaseURL
	xaiBaseURL = server.URL
	t.Cleanup(func() { xaiBaseURL = originalBase })

	resp, status, _, err := doXAIRequest(context.Background(), "test-key", "/images/generations", GenerateRequest{
		Model:          "grok-imagine-image",
		Prompt:         "test",
		ResponseFormat: "b64_json",
	})
	if err != nil {
		t.Fatalf("doXAIRequest failed after retry: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected one image, got %+v", resp.Data)
	}
	if attempts < 2 {
		t.Fatalf("expected retry, attempts = %d", attempts)
	}
}

func TestIsMockMode(t *testing.T) {
	t.Setenv("GROK_IMAGE_MOCK", "")
	mockModeEnabled = false
	if isMockMode() {
		t.Fatal("expected mock mode disabled")
	}

	t.Setenv("GROK_IMAGE_MOCK", "1")
	if !isMockMode() {
		t.Fatal("expected mock mode enabled via env")
	}

	mockModeEnabled = true
	t.Setenv("GROK_IMAGE_MOCK", "")
	if !isMockMode() {
		t.Fatal("expected mock mode enabled via flag")
	}
	mockModeEnabled = false
}

func TestMockSampleImageBytes(t *testing.T) {
	data, mime, err := mockSampleImageBytes()
	if err != nil {
		t.Fatalf("mockSampleImageBytes failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected image bytes")
	}
	if mime == "" {
		t.Fatal("expected mime type")
	}
}

func TestToolsListJSON(t *testing.T) {
	tools := getToolsList()
	if len(tools) != 6 {
		t.Fatalf("expected 6 tools, got %d", len(tools))
	}
	data, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("marshal tools: %v", err)
	}
	if !strings.Contains(string(data), "configure_xai_token") {
		t.Fatalf("tools JSON missing configure_xai_token")
	}
	if strings.Contains(string(data), "generate_imagen") {
		t.Fatalf("tools JSON should not include generate_imagen")
	}
}