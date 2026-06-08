# Grok Image MCP Server

An MCP (Model Context Protocol) server that provides AI image generation and editing using xAI's Grok Imagine API (`grok-imagine-image` / `grok-imagine-image-quality`).

This project is a Grok/xAI adaptation of [nano-banana-mcpv2](https://github.com/notfixingit3/nano-banana-mcpv2), which uses Google's Gemini image APIs.

---

## Features

- **Generate Images**: Create new images from text descriptions.
- **Edit Images**: Modify existing images using text prompts and optional reference images (up to 3 total).
- **Iterative Editing**: Refine the last generated or edited image sequentially.
- **Dynamic Model Selection**: Choose models via tool parameters or environment variables.
- **Cross-Platform Auto-Saving**: Automatically saves generated images locally.

---

## Supported Models

By default, the server uses **`grok-imagine-image-quality`** ($0.05/image).

You can also use:
- **`grok-imagine-image`**: Faster, lower-cost model ($0.02/image).
- **`grok-imagine-image-quality`**: Higher-fidelity creative model ($0.05/image).

---

## Configuration & Environment Variables

The server checks configuration in the following priority:

1. **Tool Arguments**: Pass `model` explicitly inside tool calls (highest priority).
2. **Environment Variables**:
   - `XAI_API_KEY`: Your xAI API key from [console.x.ai](https://console.x.ai).
   - `GROK_IMAGE_MODEL`: Set a default model server-wide (e.g., `grok-imagine-image-quality`).
   - `GROK_IMAGE_LOG_FILE`: Optional path for request/response logging.
3. **Global Configuration**: `~/.grok-image-config.json` generated via the `configure_xai_token` tool.

---

## Getting Your API Key

1. Go to [console.x.ai](https://console.x.ai).
2. Create an API key.
3. Set it via `XAI_API_KEY` or the `configure_xai_token` tool.

---

## Installation & Client Integration

### Method A: Build & Run Locally (Recommended for Development)

```bash
go build -o grok-image-mcp main.go
```

To verify image generation over stdio:

```bash
export XAI_API_KEY="your-api-key-here"
./scripts/test_generation.sh
```

Add this to your MCP settings file (e.g., Cursor, Claude Desktop, or Claude Code config):

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "/Users/house/Documents/gitlab/grok-image-mcp/grok-image-mcp",
      "env": {
        "XAI_API_KEY": "your-xai-api-key-here",
        "GROK_IMAGE_MODEL": "grok-imagine-image-quality"
      }
    }
  }
}
```

### Method B: Interactive Setup Wizard

```bash
go build -o grok-image-mcp main.go
./grok-image-mcp --setup
```

### Method C: Run via Docker

```bash
docker build -t grok-image-mcp .
```

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e",
        "XAI_API_KEY=your-xai-api-key-here",
        "grok-image-mcp"
      ]
    }
  }
}
```

---

## Available Tools

### `generate_image`
Create a new image from a text description using Grok Imagine.
- **`prompt`** (required): Description of the image to generate.
- **`model`** (optional): `grok-imagine-image-quality` or `grok-imagine-image`.
- **`aspectRatio`** (optional): `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `2:1`, `1:2`, `19.5:9`, `9:19.5`, `20:9`, `9:20`, or `auto`.
- **`resolution`** (optional): `1k` or `2k`.
- **`numberOfImages`** (optional): Number of images to generate (1-10).

### `edit_image`
Modify a specific existing image file.
- **`imagePath`** (required): Full local file path of the base image.
- **`prompt`** (required): Description of modifications.
- **`referenceImages`** (optional): Up to 2 additional reference image paths (3 images total).
- **`model`** (optional): Custom model name.
- **`aspectRatio`** (optional): Output aspect ratio.
- **`resolution`** (optional): `1k` or `2k`.

### `continue_editing`
Refine the last image generated/edited in the active session.
- **`prompt`** (required): Description of modification.
- **`referenceImages`** (optional): Array of reference image file paths.
- **`model`** (optional): Custom model name.
- **`aspectRatio`** (optional): Output aspect ratio.
- **`resolution`** (optional): `1k` or `2k`.

### `get_last_image_info`
Check details of the last generated/edited image (file path, file size, last modified timestamp).

### `get_configuration_status`
Verify if the xAI token is configured and see its source.

### `configure_xai_token`
Configure your xAI API key:
- **`apiKey`** (required): Your xAI API key.

---

## File Storage Directories

Images are saved automatically to:
- **Windows**: `%USERPROFILE%\Documents\grok-images\`
- **macOS/Linux**: `./generated_imgs/` (or `~/grok-images/` if run from system directories).

---

## Contributing & Branches

- **`main`**: Production-ready, stable releases (tagged `v*.*.*`).
- **`dev`**: Active features, improvements, and pre-releases (tagged `v*.*.*-beta.*`).

Make changes on `dev` and open a PR to `main` for release.

---

## License & Credits

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

### Acknowledgments
- **Inspiration**: Adapted from [nano-banana-mcpv2](https://github.com/notfixingit3/nano-banana-mcpv2).
- **xAI**: For the Grok Imagine image generation API.
- **Anthropic**: For the Model Context Protocol (MCP) specification.