package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	ServerName    = "grok-image-mcp"
	ServerVersion = "0.1.0"
	XAIBaseURL    = "https://api.x.ai/v1"
)

type Config struct {
	XAIAPIKey string `json:"xaiApiKey"`
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type                 string                 `json:"type"`
	Properties           map[string]interface{} `json:"properties"`
	Required             []string               `json:"required,omitempty"`
	AdditionalProperties bool                   `json:"additionalProperties,omitempty"`
}

var (
	lastImagePath string
	httpClient    = &http.Client{Timeout: 120 * time.Second}
	logFile       *os.File
	globalCtx     context.Context
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--setup" || os.Args[1] == "-setup") {
		runSetupWizard()
		return
	}

	rand.Seed(time.Now().UnixNano())

	logPath := os.Getenv("GROK_IMAGE_LOG_FILE")
	if logPath != "" {
		var err error
		// #nosec - log path is user-configured from environment variable, permission restricted to owner-only
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening log file:", err)
		} else {
			logMessage("Server started, logging enabled (version: %s)", ServerVersion)
		}
	}

	var cancel context.CancelFunc
	globalCtx, cancel = context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logMessage("Received signal %v, shutting down...", sig)
		cancel()
		if logFile != nil {
			_ = logFile.Close()
		}
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Bytes()
		logMessage("Received request raw: %s", string(line))
		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			logMessage("JSON Parse error: %v", err)
			sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		handleRequest(&req)
	}

	if err := scanner.Err(); err != nil {
		logMessage("Error reading stdin: %v", err)
		fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
		os.Exit(1)
	}
}

func logMessage(format string, args ...interface{}) {
	if logFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		msg := fmt.Sprintf("[%s] %s\n", timestamp, fmt.Sprintf(format, args...))
		_, _ = logFile.WriteString(msg)
	}
}

func sendResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		logMessage("Error marshaling response: %v", err)
		fmt.Fprintln(os.Stderr, "Error marshaling response:", err)
		return
	}
	logMessage("Sending response: %s", string(data))
	_, _ = os.Stdout.Write(data)
	_, _ = os.Stdout.Write([]byte("\n"))
}

func sendError(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
	respData, err := json.Marshal(resp)
	if err != nil {
		logMessage("Error marshaling error response: %v", err)
		fmt.Fprintln(os.Stderr, "Error marshaling error response:", err)
		return
	}
	logMessage("Sending error response: %s", string(respData))
	_, _ = os.Stdout.Write(respData)
	_, _ = os.Stdout.Write([]byte("\n"))
}

func handleRequest(req *JSONRPCRequest) {
	logMessage("Handling method: %s", req.Method)
	switch req.Method {
	case "initialize":
		result := map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    ServerName,
				"version": ServerVersion,
			},
		}
		sendResponse(req.ID, result)

	case "notifications/initialized":

	case "tools/list":
		tools := getToolsList()
		sendResponse(req.ID, map[string]interface{}{
			"tools": tools,
		})

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.ID, -32602, "Invalid params", err.Error())
			return
		}
		handleToolCall(req.ID, params.Name, params.Arguments)

	default:
		if req.ID != nil {
			sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method), nil)
		}
	}
}

func getToolsList() []Tool {
	return []Tool{
		{
			Name:        "configure_xai_token",
			Description: "Configure your xAI API token for Grok Imagine image generation",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"apiKey": map[string]interface{}{
						"type":        "string",
						"description": "Your xAI API key from https://console.x.ai",
					},
				},
				Required: []string{"apiKey"},
			},
		},
		{
			Name:        "generate_image",
			Description: "Generate a NEW image from a text prompt using Grok Imagine models.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "Text prompt describing the NEW image to create from scratch",
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "Optional model name. Defaults to GROK_IMAGE_MODEL environment variable or 'grok-imagine-image-quality'.",
						"enum":        []string{"grok-imagine-image-quality", "grok-imagine-image"},
					},
					"aspectRatio": map[string]interface{}{
						"type":        "string",
						"description": "Optional aspect ratio for the generated image. Defaults to '1:1'.",
						"enum":        []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20", "auto"},
					},
					"resolution": map[string]interface{}{
						"type":        "string",
						"description": "Optional output resolution. Defaults to '1k'.",
						"enum":        []string{"1k", "2k"},
					},
					"numberOfImages": map[string]interface{}{
						"type":        "integer",
						"description": "Optional number of images to generate (1-10). Defaults to 1.",
						"minimum":     1,
						"maximum":     10,
					},
				},
				Required: []string{"prompt"},
			},
		},
		{
			Name:        "edit_image",
			Description: "Edit a SPECIFIC existing image file, optionally using up to 2 additional reference images. Use this when you have the exact file path of an image to modify.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"imagePath": map[string]interface{}{
						"type":        "string",
						"description": "Full file path to the main image file to edit",
					},
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "Text describing the modifications to make to the existing image",
					},
					"referenceImages": map[string]interface{}{
						"type":        "array",
						"description": "Optional array of file paths to additional reference images (up to 2, for a total of 3 images)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "Optional model name. Defaults to GROK_IMAGE_MODEL environment variable or 'grok-imagine-image-quality'.",
						"enum":        []string{"grok-imagine-image-quality", "grok-imagine-image"},
					},
					"aspectRatio": map[string]interface{}{
						"type":        "string",
						"description": "Optional aspect ratio for the edited image. Defaults to the source image ratio.",
						"enum":        []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20", "auto"},
					},
					"resolution": map[string]interface{}{
						"type":        "string",
						"description": "Optional output resolution. Defaults to '1k'.",
						"enum":        []string{"1k", "2k"},
					},
				},
				Required: []string{"imagePath", "prompt"},
			},
		},
		{
			Name:        "continue_editing",
			Description: "Continue editing the LAST image that was generated or edited in this session, optionally using additional reference images. This automatically uses the previous image without needing a file path.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "Text describing the modifications/changes/improvements to make to the last image",
					},
					"referenceImages": map[string]interface{}{
						"type":        "array",
						"description": "Optional array of file paths to additional reference images to use during editing",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "Optional model name. Defaults to GROK_IMAGE_MODEL environment variable or 'grok-imagine-image-quality'.",
						"enum":        []string{"grok-imagine-image-quality", "grok-imagine-image"},
					},
					"aspectRatio": map[string]interface{}{
						"type":        "string",
						"description": "Optional aspect ratio for the edited image.",
						"enum":        []string{"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20", "auto"},
					},
					"resolution": map[string]interface{}{
						"type":        "string",
						"description": "Optional output resolution. Defaults to '1k'.",
						"enum":        []string{"1k", "2k"},
					},
				},
				Required: []string{"prompt"},
			},
		},
		{
			Name:        "get_configuration_status",
			Description: "Check if xAI API token is configured",
			InputSchema: InputSchema{
				Type:                 "object",
				Properties:           map[string]interface{}{},
				AdditionalProperties: false,
			},
		},
		{
			Name:        "get_last_image_info",
			Description: "Get information about the last generated/edited image in this session (file path, size, etc.)",
			InputSchema: InputSchema{
				Type:                 "object",
				Properties:           map[string]interface{}{},
				AdditionalProperties: false,
			},
		},
	}
}

func handleToolCall(id interface{}, toolName string, arguments json.RawMessage) {
	apiKey, _ := loadConfig()

	if toolName == "configure_xai_token" {
		var args struct {
			APIKey string `json:"apiKey"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			sendError(id, -32602, "Invalid arguments", err.Error())
			return
		}
		if args.APIKey == "" {
			sendError(id, -32602, "API key is required", nil)
			return
		}
		if err := saveConfig(args.APIKey); err != nil {
			sendError(id, -32603, "Failed to save configuration", err.Error())
			return
		}
		sendResponse(id, map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "✅ xAI API token configured successfully! You can now use Grok Imagine image generation features.",
				},
			},
		})
		return
	}

	if toolName == "get_configuration_status" {
		isConfigured := apiKey != ""
		statusText := "❌ xAI API token is not configured"
		sourceInfo := "\n\n📝 Configuration options:\n1. Environment variable: XAI_API_KEY\n2. Use configure_xai_token tool"
		if isConfigured {
			_, source := loadConfig()
			statusText = "✅ xAI API token is configured and ready to use"
			if source == "environment" {
				sourceInfo = "\n📍 Source: Environment variable (XAI_API_KEY)"
			} else {
				sourceInfo = "\n📍 Source: Configuration file (~/.grok-image-config.json)"
			}
		}
		sendResponse(id, map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": statusText + sourceInfo,
				},
			},
		})
		return
	}

	if apiKey == "" {
		sendError(id, -32603, "xAI API token not configured. Use configure_xai_token first.", nil)
		return
	}

	switch toolName {
	case "generate_image":
		var args struct {
			Prompt         string  `json:"prompt"`
			Model          *string `json:"model"`
			AspectRatio    *string `json:"aspectRatio"`
			Resolution     *string `json:"resolution"`
			NumberOfImages *int    `json:"numberOfImages"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			sendError(id, -32602, "Invalid arguments", err.Error())
			return
		}
		handleGenerateImage(id, apiKey, args.Prompt, args.Model, args.AspectRatio, args.Resolution, args.NumberOfImages)

	case "edit_image":
		var args struct {
			ImagePath       string   `json:"imagePath"`
			Prompt          string   `json:"prompt"`
			ReferenceImages []string `json:"referenceImages"`
			Model           *string  `json:"model"`
			AspectRatio     *string  `json:"aspectRatio"`
			Resolution      *string  `json:"resolution"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			sendError(id, -32602, "Invalid arguments", err.Error())
			return
		}
		handleEditImage(id, apiKey, args.ImagePath, args.Prompt, args.ReferenceImages, args.Model, args.AspectRatio, args.Resolution)

	case "continue_editing":
		var args struct {
			Prompt          string   `json:"prompt"`
			ReferenceImages []string `json:"referenceImages"`
			Model           *string  `json:"model"`
			AspectRatio     *string  `json:"aspectRatio"`
			Resolution      *string  `json:"resolution"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			sendError(id, -32602, "Invalid arguments", err.Error())
			return
		}
		if lastImagePath == "" {
			sendError(id, -32603, "No previous image found. Please generate or edit an image first.", nil)
			return
		}
		if _, err := os.Stat(lastImagePath); os.IsNotExist(err) {
			sendError(id, -32603, fmt.Sprintf("Last image file not found at: %s. Please generate a new image.", lastImagePath), nil)
			return
		}
		handleEditImage(id, apiKey, lastImagePath, args.Prompt, args.ReferenceImages, args.Model, args.AspectRatio, args.Resolution)

	case "get_last_image_info":
		if lastImagePath == "" {
			sendResponse(id, map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "📷 No previous image found.",
					},
				},
			})
			return
		}
		info, err := os.Stat(lastImagePath)
		if err != nil {
			sendResponse(id, map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("📷 Last Image Path: %s\nStatus: ❌ File not found", lastImagePath),
					},
				},
			})
			return
		}
		sendResponse(id, map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("📷 Last Image Information:\n\nPath: %s\nFile Size: %d KB\nLast Modified: %s\n\n💡 Use continue_editing to modify this image.", lastImagePath, info.Size()/1024, info.ModTime().Format(time.RFC1123)),
				},
			},
		})

	default:
		sendError(id, -32601, fmt.Sprintf("Unknown tool: %s", toolName), nil)
	}
}

func loadConfig() (string, string) {
	if key := os.Getenv("XAI_API_KEY"); key != "" {
		return key, "environment"
	}

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
			return config.XAIAPIKey, "config_file"
		}
	}

	home, _ := os.UserHomeDir()
	globalPath := filepath.Join(home, ".grok-image-config.json")
	// #nosec G304 - global config file read is intentional
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := json.Unmarshal(data, &config); err == nil && config.XAIAPIKey != "" {
			return config.XAIAPIKey, "config_file"
		}
	}

	return "", "not_configured"
}

func saveConfig(key string) error {
	config := Config{XAIAPIKey: key}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	globalPath := filepath.Join(home, ".grok-image-config.json")

	// #nosec G301 - global config dir must be owner-only (0700)
	_ = os.MkdirAll(filepath.Dir(globalPath), 0700)
	// #nosec G306 - config file must be owner-only (0600)
	return os.WriteFile(globalPath, data, 0600)
}

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

type GenerateRequest struct {
	Model          string  `json:"model"`
	Prompt         string  `json:"prompt"`
	AspectRatio    *string `json:"aspect_ratio,omitempty"`
	N              *int    `json:"n,omitempty"`
	Resolution     *string `json:"resolution,omitempty"`
	ResponseFormat string  `json:"response_format"`
}

type ImageRef struct {
	URL  string `json:"url"`
	Type string `json:"type,omitempty"`
}

type EditRequest struct {
	Model          string     `json:"model"`
	Prompt         string     `json:"prompt"`
	Image          *ImageRef  `json:"image,omitempty"`
	Images         []ImageRef `json:"images,omitempty"`
	AspectRatio    *string    `json:"aspect_ratio,omitempty"`
	Resolution     *string    `json:"resolution,omitempty"`
	ResponseFormat string     `json:"response_format"`
}

type ImageData struct {
	URL      *string `json:"url"`
	B64JSON  *string `json:"b64_json"`
	MimeType *string `json:"mime_type"`
}

type ImageResponse struct {
	Data []ImageData `json:"data"`
}

func encodeImageAsDataURI(imagePath string) (string, error) {
	// #nosec G304 - reading image path is intentional and requested by user/client
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	mimeType := getMimeType(imagePath)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(imgData)), nil
}

func doXAIRequest(ctx context.Context, apiKey, endpoint string, payload interface{}) (*ImageResponse, int, []byte, error) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("payload formatting error: %w", err)
	}

	url := XAIBaseURL + endpoint
	logMessage("Sending request to URL: %s", url)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadData))
	if err != nil {
		return nil, 0, nil, fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logMessage("xAI API call failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		return nil, resp.StatusCode, bodyBytes, fmt.Errorf("xAI API call failed with status %d", resp.StatusCode)
	}

	logMessage("xAI API call succeeded with status %d", resp.StatusCode)

	var imageResp ImageResponse
	if err := json.Unmarshal(bodyBytes, &imageResp); err != nil {
		return nil, resp.StatusCode, bodyBytes, fmt.Errorf("failed to parse API response: %w", err)
	}

	return &imageResp, resp.StatusCode, bodyBytes, nil
}

func downloadImage(ctx context.Context, imageURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = mimeType[:idx]
	}

	return data, mimeType, nil
}

func saveImageResult(ctx context.Context, imageData ImageData, prefix string) (string, string, error) {
	var imageBytes []byte
	mimeType := "image/jpeg"

	if imageData.MimeType != nil && *imageData.MimeType != "" {
		mimeType = *imageData.MimeType
	}

	if imageData.B64JSON != nil && *imageData.B64JSON != "" {
		decoded, err := base64.StdEncoding.DecodeString(*imageData.B64JSON)
		if err != nil {
			return "", "", err
		}
		imageBytes = decoded
	} else if imageData.URL != nil && *imageData.URL != "" {
		var err error
		imageBytes, mimeType, err = downloadImage(ctx, *imageData.URL)
		if err != nil {
			return "", "", err
		}
	} else {
		return "", "", fmt.Errorf("no image data in response")
	}

	imagesDir := getImagesDirectory()
	// #nosec G301 - generated images folder should be user-browsable (0755)
	_ = os.MkdirAll(imagesDir, 0755)

	timestamp := time.Now().Format("20060102-150405")
	// #nosec G404 - weak random is sufficient for filename random identifier
	randomID := fmt.Sprintf("%06d", rand.Intn(1000000))
	ext := extensionForMimeType(mimeType)
	fileName := fmt.Sprintf("%s-%s-%s%s", prefix, timestamp, randomID, ext)
	filePath := filepath.Join(imagesDir, fileName)

	// #nosec G306 - generated image files must be user-viewable (0644)
	if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
		return "", "", err
	}

	b64 := base64.StdEncoding.EncodeToString(imageBytes)
	return filePath, b64, nil
}

func handleGenerateImage(id interface{}, apiKey, prompt string, customModel, aspectRatio, resolution *string, numberOfImages *int) {
	model := resolveModel(customModel)

	reqPayload := GenerateRequest{
		Model:          model,
		Prompt:         prompt,
		AspectRatio:    aspectRatio,
		N:              numberOfImages,
		Resolution:     resolution,
		ResponseFormat: "b64_json",
	}

	ctx := globalCtx
	if ctx == nil {
		ctx = context.Background()
	}

	imageResp, statusCode, bodyBytes, err := doXAIRequest(ctx, apiKey, "/images/generations", reqPayload)
	if err != nil {
		if statusCode != 0 {
			sendError(id, -32603, err.Error(), string(bodyBytes))
		} else {
			sendError(id, -32603, err.Error(), nil)
		}
		return
	}

	content := []map[string]interface{}{}
	savedFiles := []string{}

	for _, item := range imageResp.Data {
		filePath, b64, err := saveImageResult(ctx, item, "generated")
		if err != nil {
			logMessage("Failed to save image: %v", err)
			continue
		}
		savedFiles = append(savedFiles, filePath)
		lastImagePath = filePath

		mimeType := "image/jpeg"
		if item.MimeType != nil && *item.MimeType != "" {
			mimeType = *item.MimeType
		}

		content = append(content, map[string]interface{}{
			"type":     "image",
			"data":     b64,
			"mimeType": mimeType,
		})
	}

	statusText := fmt.Sprintf("🎨 Image generated with Grok Imagine (%s)!\n\nPrompt: \"%s\"", model, prompt)
	if aspectRatio != nil && *aspectRatio != "" {
		statusText += fmt.Sprintf("\nAspect Ratio: %s", *aspectRatio)
	}
	if resolution != nil && *resolution != "" {
		statusText += fmt.Sprintf("\nResolution: %s", *resolution)
	}
	if numberOfImages != nil && *numberOfImages > 1 {
		statusText += fmt.Sprintf("\nImages Requested: %d", *numberOfImages)
	}

	if len(savedFiles) > 0 {
		statusText += "\n\n📁 Image saved to:\n"
		for _, f := range savedFiles {
			statusText += fmt.Sprintf("- %s\n", f)
		}
		statusText += "\n🔄 To modify this image, use: continue_editing"
	} else {
		statusText += "\n\nNote: No image was returned by the xAI API."
	}

	textPart := map[string]interface{}{
		"type": "text",
		"text": statusText,
	}
	content = append([]map[string]interface{}{textPart}, content...)

	sendResponse(id, map[string]interface{}{
		"content": content,
	})
}

func handleEditImage(id interface{}, apiKey, imagePath, prompt string, referenceImages []string, customModel, aspectRatio, resolution *string) {
	model := resolveModel(customModel)

	mainDataURI, err := encodeImageAsDataURI(imagePath)
	if err != nil {
		sendError(id, -32603, fmt.Sprintf("Failed to read image at %s", imagePath), err.Error())
		return
	}

	reqPayload := EditRequest{
		Model:          model,
		Prompt:         prompt,
		AspectRatio:    aspectRatio,
		Resolution:     resolution,
		ResponseFormat: "b64_json",
	}

	allImages := []ImageRef{{URL: mainDataURI, Type: "image_url"}}
	for _, refPath := range referenceImages {
		if len(allImages) >= 3 {
			break
		}
		if refDataURI, err := encodeImageAsDataURI(refPath); err == nil {
			allImages = append(allImages, ImageRef{URL: refDataURI, Type: "image_url"})
		}
	}

	if len(allImages) == 1 {
		reqPayload.Image = &allImages[0]
	} else {
		reqPayload.Images = allImages
	}

	ctx := globalCtx
	if ctx == nil {
		ctx = context.Background()
	}

	imageResp, statusCode, bodyBytes, err := doXAIRequest(ctx, apiKey, "/images/edits", reqPayload)
	if err != nil {
		if statusCode != 0 {
			sendError(id, -32603, err.Error(), string(bodyBytes))
		} else {
			sendError(id, -32603, err.Error(), nil)
		}
		return
	}

	content := []map[string]interface{}{}
	savedFiles := []string{}

	for _, item := range imageResp.Data {
		filePath, b64, err := saveImageResult(ctx, item, "edited")
		if err != nil {
			logMessage("Failed to save image: %v", err)
			continue
		}
		savedFiles = append(savedFiles, filePath)
		lastImagePath = filePath

		mimeType := "image/jpeg"
		if item.MimeType != nil && *item.MimeType != "" {
			mimeType = *item.MimeType
		}

		content = append(content, map[string]interface{}{
			"type":     "image",
			"data":     b64,
			"mimeType": mimeType,
		})
	}

	statusText := fmt.Sprintf("🎨 Image edited with Grok Imagine (%s)!\n\nOriginal: %s\nEdit prompt: \"%s\"", model, imagePath, prompt)
	if len(referenceImages) > 0 {
		statusText += fmt.Sprintf("\nReference images: %d", len(referenceImages))
	}

	if len(savedFiles) > 0 {
		statusText += "\n\n📁 Edited image saved to:\n"
		for _, f := range savedFiles {
			statusText += fmt.Sprintf("- %s\n", f)
		}
		statusText += "\n🔄 To modify this image, use: continue_editing"
	} else {
		statusText += "\n\nNote: No edited image was returned by the xAI API."
	}

	textPart := map[string]interface{}{
		"type": "text",
		"text": statusText,
	}
	content = append([]map[string]interface{}{textPart}, content...)

	sendResponse(id, map[string]interface{}{
		"content": content,
	})
}

func runSetupWizard() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("🖼️  Grok Image MCP - Interactive Setup Wizard")
	fmt.Println("==============================================")
	fmt.Println("This wizard will help you configure your xAI API key.")
	fmt.Println()

	var apiKey string
	for {
		fmt.Print("Enter your xAI API key (should start with xai-...): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Error reading input: %v\n", err)
			os.Exit(1)
		}
		apiKey = strings.TrimSpace(input)
		if apiKey == "" {
			fmt.Println("❌ API key cannot be empty. Please try again.")
			continue
		}
		break
	}

	fmt.Println("\n🔍 Validating your API key against xAI API...")

	req, err := http.NewRequest("GET", XAIBaseURL+"/models", nil)
	if err != nil {
		fmt.Printf("❌ Request creation error: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Connection error while validating key: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(bodyBytes, &errResp)
		if errResp.Error.Message != "" {
			fmt.Printf("❌ API key validation failed (Status %d): %s\n", resp.StatusCode, errResp.Error.Message)
		} else {
			fmt.Printf("❌ API key validation failed with HTTP status %d\n", resp.StatusCode)
		}
		os.Exit(1)
	}

	fmt.Println("✅ API key is valid!")

	fmt.Println("\n💾 Saving configuration...")
	if err := saveConfig(apiKey); err != nil {
		fmt.Printf("❌ Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	home, _ := os.UserHomeDir()
	globalPath := filepath.Join(home, ".grok-image-config.json")
	fmt.Printf("🎉 Setup completed successfully! Key saved to: %s\n", globalPath)
}