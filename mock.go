package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Minimal 1x1 PNG used when no sample asset is available on disk.
var fallbackPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
	0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc, 0x33, 0x00, 0x00, 0x00,
	0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func isMockMode() bool {
	if mockModeEnabled {
		return true
	}
	value := strings.ToLower(strings.TrimSpace(os.Getenv("GROK_IMAGE_MOCK")))
	return value == "1" || value == "true" || value == "yes"
}

func mockModeSuffix() string {
	if isMockMode() {
		return "\n\n🧪 Mock mode is active — no live xAI API calls will be made."
	}
	return ""
}

func mockSampleImageBytes() ([]byte, string, error) {
	if custom := strings.TrimSpace(os.Getenv("GROK_IMAGE_MOCK_ASSET")); custom != "" {
		data, err := os.ReadFile(custom)
		if err != nil {
			return nil, "", err
		}
		return data, getMimeType(custom), nil
	}

	candidates := []string{
		"assets/sample_output.png",
		"assets/logo.png",
	}
	for _, candidate := range candidates {
		if data, err := os.ReadFile(candidate); err == nil {
			return data, getMimeType(candidate), nil
		}
	}

	return fallbackPNG, "image/png", nil
}

func saveMockImage(prefix string, source []byte, mimeType string) (string, string, error) {
	imagesDir := getImagesDirectory()
	// #nosec G301 - generated images folder should be user-browsable (0755)
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return "", "", err
	}

	timestamp := time.Now().Format("20060102-150405")
	randomID := randomSuffix()
	ext := extensionForMimeType(mimeType)
	fileName := fmt.Sprintf("%s-%s-%s%s", prefix, timestamp, randomID, ext)
	filePath := filepath.Join(imagesDir, fileName)

	// #nosec G306 - generated image files must be user-viewable (0644)
	if err := os.WriteFile(filePath, source, 0644); err != nil {
		return "", "", err
	}

	return filePath, base64.StdEncoding.EncodeToString(source), nil
}

func handleMockGenerateImage(id interface{}, prompt string, customModel, aspectRatio, resolution *string, numberOfImages *int, serviceTier *string) {
	model := resolveModel(customModel)
	count := 1
	if numberOfImages != nil && *numberOfImages > 0 {
		count = *numberOfImages
		if count > 10 {
			count = 10
		}
	}

	sample, mimeType, err := mockSampleImageBytes()
	if err != nil {
		sendError(id, -32603, "Mock mode failed to load sample image", err.Error())
		return
	}

	content := []map[string]interface{}{}
	savedFiles := []string{}

	for i := 0; i < count; i++ {
		filePath, b64, err := saveMockImage("generated", sample, mimeType)
		if err != nil {
			sendError(id, -32603, "Mock mode failed to save image", err.Error())
			return
		}
		savedFiles = append(savedFiles, filePath)
		lastImagePath = filePath
		content = append(content, map[string]interface{}{
			"type":     "image",
			"data":     b64,
			"mimeType": mimeType,
		})
	}

	statusText := fmt.Sprintf("🧪 [MOCK] Image generated with Grok Imagine (%s)!\n\nPrompt: \"%s\"", model, prompt)
	statusText += "\nMode: Offline mock — no xAI API call was made."
	if aspectRatio != nil && *aspectRatio != "" {
		statusText += fmt.Sprintf("\nAspect Ratio: %s", *aspectRatio)
	}
	if resolution != nil && *resolution != "" {
		statusText += fmt.Sprintf("\nResolution: %s", *resolution)
	}
	if count > 1 {
		statusText += fmt.Sprintf("\nImages Requested: %d", count)
	}
	if serviceTier != nil && *serviceTier != "" {
		statusText += fmt.Sprintf("\nService Tier: %s", *serviceTier)
	}

	statusText += "\n\n📁 Image saved to:\n"
	for _, f := range savedFiles {
		statusText += fmt.Sprintf("- %s\n", f)
	}
	statusText += "\n🔄 To modify this image, use: continue_editing"
	statusText += "\n\n💡 Disable mock mode (unset GROK_IMAGE_MOCK) and add xAI credits for real generation."

	content = append([]map[string]interface{}{{"type": "text", "text": statusText}}, content...)
	sendResponse(id, map[string]interface{}{"content": content})
}

func handleMockEditImage(id interface{}, imagePath, prompt string, referenceImages []string, customModel, aspectRatio, resolution *string, numberOfImages *int, serviceTier *string) {
	model := resolveModel(customModel)

	_, warning, err := buildEditImageRefs(imagePath, referenceImages)
	if err != nil {
		sendError(id, -32603, fmt.Sprintf("Failed to prepare images for editing: %s", imagePath), err.Error())
		return
	}

	source, err := os.ReadFile(imagePath)
	if err != nil {
		sendError(id, -32603, fmt.Sprintf("Failed to read image at %s", imagePath), err.Error())
		return
	}
	mimeType := getMimeType(imagePath)

	count := 1
	if numberOfImages != nil && *numberOfImages > 0 {
		count = *numberOfImages
		if count > 10 {
			count = 10
		}
	}

	content := []map[string]interface{}{}
	savedFiles := []string{}

	for i := 0; i < count; i++ {
		filePath, b64, err := saveMockImage("edited", source, mimeType)
		if err != nil {
			sendError(id, -32603, "Mock mode failed to save edited image", err.Error())
			return
		}
		savedFiles = append(savedFiles, filePath)
		lastImagePath = filePath
		content = append(content, map[string]interface{}{
			"type":     "image",
			"data":     b64,
			"mimeType": mimeType,
		})
	}

	statusText := fmt.Sprintf("🧪 [MOCK] Image edited with Grok Imagine (%s)!\n\nOriginal: %s\nEdit prompt: \"%s\"", model, imagePath, prompt)
	statusText += "\nMode: Offline mock — no xAI API call was made."
	if warning != "" {
		statusText += fmt.Sprintf("\n\n⚠️ %s", warning)
	}
	if len(referenceImages) > 0 {
		statusText += fmt.Sprintf("\nReference images provided: %d", len(referenceImages))
	}
	if count > 1 {
		statusText += fmt.Sprintf("\nVariations requested: %d", count)
	}
	if serviceTier != nil && *serviceTier != "" {
		statusText += fmt.Sprintf("\nService Tier: %s", *serviceTier)
	}

	statusText += "\n\n📁 Edited image saved to:\n"
	for _, f := range savedFiles {
		statusText += fmt.Sprintf("- %s\n", f)
	}
	statusText += "\n🔄 To modify this image, use: continue_editing"

	content = append([]map[string]interface{}{{"type": "text", "text": statusText}}, content...)
	sendResponse(id, map[string]interface{}{"content": content})
}