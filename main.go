package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	ServerName    = "grok-image-mcp"
	ServerVersion = "0.2.0-beta.1"
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
	lastImagePath    string
	httpClient       = &http.Client{Timeout: 120 * time.Second}
	logFile          *os.File
	globalCtx        context.Context
	mockModeEnabled  bool
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--setup", "-setup":
			runSetupWizard()
			return
		case "--version", "-version", "-v":
			fmt.Println(ServerVersion)
			return
		case "--help", "-help", "-h":
			printHelp()
			return
		case "--mock", "-mock":
			mockModeEnabled = true
			os.Args = append(os.Args[:1], os.Args[2:]...)
		}
	}

	if isMockMode() {
		fmt.Fprintln(os.Stderr, "[grok-image-mcp] Mock mode enabled — image tools run offline without xAI credits")
	}

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
					"serviceTier": map[string]interface{}{
						"type":        "string",
						"description": "Optional service tier. Use 'priority' for faster processing.",
						"enum":        []string{"default", "priority"},
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
					"numberOfImages": map[string]interface{}{
						"type":        "integer",
						"description": "Optional number of edited variations to generate (1-10). Defaults to 1.",
						"minimum":     1,
						"maximum":     10,
					},
					"serviceTier": map[string]interface{}{
						"type":        "string",
						"description": "Optional service tier. Use 'priority' for faster processing.",
						"enum":        []string{"default", "priority"},
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
					"numberOfImages": map[string]interface{}{
						"type":        "integer",
						"description": "Optional number of edited variations to generate (1-10). Defaults to 1.",
						"minimum":     1,
						"maximum":     10,
					},
					"serviceTier": map[string]interface{}{
						"type":        "string",
						"description": "Optional service tier. Use 'priority' for faster processing.",
						"enum":        []string{"default", "priority"},
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
		ctx := globalCtx
		if ctx == nil {
			ctx = context.Background()
		}
		if !isMockMode() {
			statusCode, bodyBytes, err := validateAPIKey(ctx, args.APIKey)
			if err != nil {
				sendError(id, -32603, formatValidationFailure(statusCode, bodyBytes), string(bodyBytes))
				return
			}
		}
		if err := saveConfig(args.APIKey); err != nil {
			sendError(id, -32603, "Failed to save configuration", err.Error())
			return
		}
		sendResponse(id, map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "✅ xAI API token configured successfully! You can now use Grok Imagine image generation features." + mockModeSuffix(),
				},
			},
		})
		return
	}

	if toolName == "get_configuration_status" {
		_, source := loadConfig()
		isConfigured := apiKey != ""
		statusText := "❌ xAI credentials are not configured"
		sourceInfo := "\n\n📝 Configuration options:\n1. Grok subscription: run `grok login` (uses ~/.grok/auth.json)\n2. Environment variable: XAI_API_KEY\n3. Use configure_xai_token tool"
		if isMockMode() {
			statusText = "🧪 Mock mode is active — image tools work offline without xAI credits"
			sourceInfo = "\n📍 Set GROK_IMAGE_MOCK=1 or run with --mock to enable"
			if isConfigured {
				sourceInfo += "\n📍 Live credentials are also available for when you disable mock mode"
				if desc := credentialSourceDescription(source); desc != "" {
					sourceInfo += "\n📍 " + desc
				}
			}
		} else if isConfigured {
			if strings.HasPrefix(source, "grok_oauth") {
				statusText = "✅ Grok subscription OAuth is active — image tools use your SuperGrok / X Premium+ session"
			} else {
				statusText = "✅ xAI API credentials are configured and ready to use"
			}
			if desc := credentialSourceDescription(source); desc != "" {
				sourceInfo = "\n📍 Source: " + desc
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

	if toolName == "get_last_image_info" {
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
		return
	}

	if toolName == "continue_editing" {
		var args struct {
			Prompt          string   `json:"prompt"`
			ReferenceImages []string `json:"referenceImages"`
			Model           *string  `json:"model"`
			AspectRatio     *string  `json:"aspectRatio"`
			Resolution      *string  `json:"resolution"`
			NumberOfImages  *int     `json:"numberOfImages"`
			ServiceTier     *string  `json:"serviceTier"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			sendError(id, -32602, "Invalid arguments", err.Error())
			return
		}
		if err := validateGenerateImageArgs(args.Prompt, args.Model, args.AspectRatio, args.Resolution, args.ServiceTier, args.NumberOfImages); err != nil {
			sendError(id, -32602, err.Error(), nil)
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
		if !isMockMode() && apiKey == "" {
			sendError(id, -32603, "xAI credentials not configured. Run `grok login` for subscription OAuth, or use configure_xai_token.", nil)
			return
		}
		if isMockMode() {
			handleMockEditImage(id, lastImagePath, args.Prompt, args.ReferenceImages, args.Model, args.AspectRatio, args.Resolution, args.NumberOfImages, args.ServiceTier)
		} else {
			handleEditImage(id, apiKey, lastImagePath, args.Prompt, args.ReferenceImages, args.Model, args.AspectRatio, args.Resolution, args.NumberOfImages, args.ServiceTier)
		}
		return
	}

	switch toolName {
	case "generate_image":
		if !isMockMode() && apiKey == "" {
			sendError(id, -32603, "xAI credentials not configured. Run `grok login` for subscription OAuth, or use configure_xai_token.", nil)
			return
		}
		var args struct {
			Prompt         string  `json:"prompt"`
			Model          *string `json:"model"`
			AspectRatio    *string `json:"aspectRatio"`
			Resolution     *string `json:"resolution"`
			NumberOfImages *int    `json:"numberOfImages"`
			ServiceTier    *string `json:"serviceTier"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			sendError(id, -32602, "Invalid arguments", err.Error())
			return
		}
		if err := validateGenerateImageArgs(args.Prompt, args.Model, args.AspectRatio, args.Resolution, args.ServiceTier, args.NumberOfImages); err != nil {
			sendError(id, -32602, err.Error(), nil)
			return
		}
		if isMockMode() {
			handleMockGenerateImage(id, args.Prompt, args.Model, args.AspectRatio, args.Resolution, args.NumberOfImages, args.ServiceTier)
		} else {
			handleGenerateImage(id, apiKey, args.Prompt, args.Model, args.AspectRatio, args.Resolution, args.NumberOfImages, args.ServiceTier)
		}

	case "edit_image":
		if !isMockMode() && apiKey == "" {
			sendError(id, -32603, "xAI credentials not configured. Run `grok login` for subscription OAuth, or use configure_xai_token.", nil)
			return
		}
		var args struct {
			ImagePath       string   `json:"imagePath"`
			Prompt          string   `json:"prompt"`
			ReferenceImages []string `json:"referenceImages"`
			Model           *string  `json:"model"`
			AspectRatio     *string  `json:"aspectRatio"`
			Resolution      *string  `json:"resolution"`
			NumberOfImages  *int     `json:"numberOfImages"`
			ServiceTier     *string  `json:"serviceTier"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			sendError(id, -32602, "Invalid arguments", err.Error())
			return
		}
		if err := validateEditImageArgs(args.ImagePath, args.Prompt, args.Model, args.AspectRatio, args.Resolution, args.ServiceTier, args.NumberOfImages); err != nil {
			sendError(id, -32602, err.Error(), nil)
			return
		}
		if isMockMode() {
			handleMockEditImage(id, args.ImagePath, args.Prompt, args.ReferenceImages, args.Model, args.AspectRatio, args.Resolution, args.NumberOfImages, args.ServiceTier)
		} else {
			handleEditImage(id, apiKey, args.ImagePath, args.Prompt, args.ReferenceImages, args.Model, args.AspectRatio, args.Resolution, args.NumberOfImages, args.ServiceTier)
		}

	default:
		sendError(id, -32601, fmt.Sprintf("Unknown tool: %s", toolName), nil)
	}
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

type GenerateRequest struct {
	Model          string  `json:"model"`
	Prompt         string  `json:"prompt"`
	AspectRatio    *string `json:"aspect_ratio,omitempty"`
	N              *int    `json:"n,omitempty"`
	Resolution     *string `json:"resolution,omitempty"`
	ServiceTier    *string `json:"service_tier,omitempty"`
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
	N              *int       `json:"n,omitempty"`
	Resolution     *string    `json:"resolution,omitempty"`
	ServiceTier    *string    `json:"service_tier,omitempty"`
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

func doXAIRequest(ctx context.Context, apiKey, endpoint string, payload interface{}) (*ImageResponse, int, []byte, error) {
	retryDelays := []time.Duration{0, 2 * time.Second, 4 * time.Second}
	var lastStatus int
	var lastBody []byte
	var lastErr error

	for attempt, delay := range retryDelays {
		if delay > 0 {
			logMessage("Retrying xAI request after %v (attempt %d)", delay, attempt+1)
			select {
			case <-ctx.Done():
				return nil, 0, nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		imageResp, statusCode, bodyBytes, err := doXAIRequestOnce(ctx, apiKey, endpoint, payload)
		if err == nil {
			return imageResp, statusCode, bodyBytes, nil
		}
		lastStatus = statusCode
		lastBody = bodyBytes
		lastErr = err
		if statusCode != http.StatusTooManyRequests {
			return nil, statusCode, bodyBytes, err
		}
	}

	return nil, lastStatus, lastBody, lastErr
}

func doXAIRequestOnce(ctx context.Context, apiKey, endpoint string, payload interface{}) (*ImageResponse, int, []byte, error) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("payload formatting error: %w", err)
	}

	url := xaiBaseURL + endpoint
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
		return nil, resp.StatusCode, bodyBytes, fmt.Errorf("%s", formatXAIError(resp.StatusCode, bodyBytes))
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
	randomID := randomSuffix()
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

func handleGenerateImage(id interface{}, apiKey, prompt string, customModel, aspectRatio, resolution *string, numberOfImages *int, serviceTier *string) {
	model := resolveModel(customModel)

	reqPayload := GenerateRequest{
		Model:          model,
		Prompt:         prompt,
		AspectRatio:    aspectRatio,
		N:              numberOfImages,
		Resolution:     resolution,
		ServiceTier:    serviceTier,
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
	if serviceTier != nil && *serviceTier != "" {
		statusText += fmt.Sprintf("\nService Tier: %s", *serviceTier)
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

func handleEditImage(id interface{}, apiKey, imagePath, prompt string, referenceImages []string, customModel, aspectRatio, resolution *string, numberOfImages *int, serviceTier *string) {
	model := resolveModel(customModel)

	allImages, warning, err := buildEditImageRefs(imagePath, referenceImages)
	if err != nil {
		sendError(id, -32603, fmt.Sprintf("Failed to prepare images for editing: %s", imagePath), err.Error())
		return
	}

	reqPayload := EditRequest{
		Model:          model,
		Prompt:         prompt,
		AspectRatio:    aspectRatio,
		N:              numberOfImages,
		Resolution:     resolution,
		ServiceTier:    serviceTier,
		ResponseFormat: "b64_json",
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
	if warning != "" {
		statusText += fmt.Sprintf("\n\n⚠️ %s", warning)
	}
	if len(referenceImages) > 0 {
		statusText += fmt.Sprintf("\nReference images provided: %d", len(referenceImages))
	}
	if numberOfImages != nil && *numberOfImages > 1 {
		statusText += fmt.Sprintf("\nVariations requested: %d", *numberOfImages)
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

func printHelp() {
	fmt.Printf(`Grok Image MCP Server v%s

Usage:
  grok-image-mcp              Start MCP server (stdio JSON-RPC)
  grok-image-mcp --mock       Start in mock mode (no xAI credits)
  grok-image-mcp --setup      Interactive setup wizard
  grok-image-mcp --version    Print version
  grok-image-mcp --help       Show this help

Environment:
  XAI_API_KEY           xAI API key for live image generation (pay-as-you-go)
  GROK_IMAGE_AUTH       Credential priority: auto (default), oauth, api_key
  GROK_AUTH_JSON        Path to Grok OAuth store (default: ~/.grok/auth.json)
  GROK_IMAGE_MOCK=1     Enable offline mock mode
  GROK_IMAGE_MODEL      Default model (grok-imagine-image-quality)
  GROK_IMAGES_DIR       Custom output directory for saved images
  GROK_IMAGE_MOCK_ASSET Image file used as mock output
  GROK_IMAGE_LOG_FILE   Optional request/response log path

Docs: https://github.com/notfixingit3/grok-image-mcp
`, ServerVersion)
}

func runSetupWizard() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("🖼️  Grok Image MCP - Interactive Setup Wizard")
	fmt.Println("==============================================")
	fmt.Println()
	fmt.Println("Choose a setup path:")
	fmt.Println("  1) Mock mode only (free — no API key or credits)")
	fmt.Println("  2) Live mode — Grok subscription OAuth or xAI API key")
	fmt.Println("  3) Skip")
	fmt.Println()

	var choice string
	for {
		fmt.Print("Enter choice [1-3]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Error reading input: %v\n", err)
			os.Exit(1)
		}
		choice = strings.TrimSpace(input)
		if choice == "1" || choice == "2" || choice == "3" {
			break
		}
		fmt.Println("❌ Please enter 1, 2, or 3.")
	}

	switch choice {
	case "1":
		fmt.Println()
		fmt.Println("🧪 Mock mode setup")
		fmt.Println("------------------")
		fmt.Println("Add this to your MCP client config:")
		fmt.Println()
		fmt.Println(`  "env": { "GROK_IMAGE_MOCK": "1" }`)
		fmt.Println()
		fmt.Println("Or run the server with:  grok-image-mcp --mock")
		fmt.Println()
		fmt.Println("See README Client Integration for Grok Build, Cursor, Claude, OpenCode, and more.")
		fmt.Println("Project examples: .grok/config.toml, .mcp.json, .cursor/mcp.json")
		return
	case "3":
		fmt.Println("Setup skipped.")
		return
	}

	fmt.Println()
	if token, src, err := loadGrokOAuthToken(context.Background()); err == nil && token != "" {
		fmt.Println("✅ Detected Grok subscription OAuth in ~/.grok/auth.json")
		if strings.HasPrefix(src, "grok_oauth (") {
			fmt.Println("   Account:", strings.TrimSuffix(strings.TrimPrefix(src, "grok_oauth ("), ")"))
		}
		fmt.Println("   No API key needed — image tools will use your subscription session.")
		fmt.Println("   Add this MCP server to your client and start generating.")
		return
	}

	fmt.Println("No Grok OAuth session found. This wizard will configure an xAI API key for live mode.")
	fmt.Println("Tip: SuperGrok / X Premium+ users can run `grok login` instead — no API key required.")
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

	statusCode, bodyBytes, err := validateAPIKey(context.Background(), apiKey)
	if err != nil {
		fmt.Printf("❌ %s\n", formatValidationFailure(statusCode, bodyBytes))
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