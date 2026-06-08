package main

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	maxImageFileBytes = 20 << 20 // 20 MiB per xAI API limit
	maxEditImages     = 3
)

var xaiBaseURL = XAIBaseURL

func resolveModel(customModel *string) string {
	if customModel != nil && strings.TrimSpace(*customModel) != "" {
		return strings.TrimSpace(*customModel)
	}
	if envModel := os.Getenv("GROK_IMAGE_MODEL"); envModel != "" {
		return strings.TrimSpace(envModel)
	}
	return "grok-imagine-image-quality"
}

func getImagesDirectory() string {
	if customDir := strings.TrimSpace(os.Getenv("GROK_IMAGES_DIR")); customDir != "" {
		return customDir
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "Documents", "grok-images")
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	if strings.HasPrefix(cwd, "/usr/") || strings.HasPrefix(cwd, "/opt/") || strings.HasPrefix(cwd, "/var/") {
		return filepath.Join(home, "grok-images")
	}
	return filepath.Join(cwd, "generated_imgs")
}

func getMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func extensionForMimeType(mimeType string) string {
	switch mimeType {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func formatXAIError(statusCode int, bodyBytes []byte) string {
	message := fmt.Sprintf("xAI API call failed with status %d", statusCode)
	var errResp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &errResp); err == nil {
		if errResp.Error != "" {
			message = errResp.Error
		} else if errResp.Code != "" {
			message = errResp.Code
		}
	}
	if statusCode == http.StatusForbidden && strings.Contains(strings.ToLower(message), "credit") {
		message += " Add credits or licenses at https://console.x.ai before using image generation."
	}
	if statusCode == http.StatusTooManyRequests {
		message += " Rate limit exceeded; the server will retry automatically, but you may need to wait and try again."
	}
	return message
}

func validateImageFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > maxImageFileBytes {
		return fmt.Errorf("image file exceeds 20 MiB limit (%d bytes)", info.Size())
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return nil
	default:
		return fmt.Errorf("unsupported image format %q (use JPEG, PNG, or WebP)", ext)
	}
}

func encodeImageAsDataURI(imagePath string) (string, error) {
	if err := validateImageFile(imagePath); err != nil {
		return "", err
	}
	// #nosec G304 - reading image path is intentional and requested by user/client
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	mimeType := getMimeType(imagePath)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imgData)), nil
}

func buildEditImageRefs(mainPath string, referenceImages []string) ([]ImageRef, string, error) {
	mainURI, err := encodeImageAsDataURI(mainPath)
	if err != nil {
		return nil, "", fmt.Errorf("main image: %w", err)
	}

	refs := []ImageRef{{URL: mainURI, Type: "image_url"}}
	var warnings []string
	skippedRefs := 0
	failedRefs := 0

	for i, refPath := range referenceImages {
		if len(refs) >= maxEditImages {
			skippedRefs = len(referenceImages) - i
			break
		}
		refURI, err := encodeImageAsDataURI(refPath)
		if err != nil {
			failedRefs++
			continue
		}
		refs = append(refs, ImageRef{URL: refURI, Type: "image_url"})
	}

	if skippedRefs > 0 {
		warnings = append(warnings, fmt.Sprintf("%d reference image(s) skipped (xAI supports up to %d images total)", skippedRefs, maxEditImages))
	}
	if failedRefs > 0 {
		warnings = append(warnings, fmt.Sprintf("%d reference image(s) could not be loaded (missing, too large, or unsupported format)", failedRefs))
	}

	return refs, strings.Join(warnings, "; "), nil
}

func randomSuffix() string {
	b := make([]byte, 3)
	if _, err := crand.Read(b); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06x", int(b[0])<<16|int(b[1])<<8|int(b[2]))
}

func validateAPIKey(ctx context.Context, apiKey string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xaiBaseURL+"/models", nil)
	if err != nil {
		return 0, nil, fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("connection error while validating key: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read validation response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, bodyBytes, fmt.Errorf("%s", formatXAIError(resp.StatusCode, bodyBytes))
	}
	return resp.StatusCode, bodyBytes, nil
}

func formatValidationFailure(statusCode int, bodyBytes []byte) string {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Sprintf("API key validation failed (status %d): %s", statusCode, errResp.Error.Message)
	}
	return fmt.Sprintf("API key validation failed with HTTP status %d", statusCode)
}