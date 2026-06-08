<p align="center">
  <img src="assets/logo.png" alt="Grok Image MCP Logo" width="300" />
</p>

<p align="center">
  <sub><i>Logo generated using Grok Imagine</i></sub>
</p>

<p align="center">
  <a href="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/ci.yml"><img src="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
  <a href="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/release.yml"><img src="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/release.yml/badge.svg" alt="Release Workflow Status" /></a>
  <a href="https://github.com/notfixingit3/grok-image-mcp/releases"><img src="https://img.shields.io/github/v/release/notfixingit3/grok-image-mcp?include_prereleases" alt="Latest Release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/notfixingit3/grok-image-mcp" alt="License" /></a>
</p>

# Grok Image MCP Server

An MCP (Model Context Protocol) server that provides AI image generation and editing using xAI's Grok Imagine API (`grok-imagine-image` / `grok-imagine-image-quality`).

This is a Grok/xAI adaptation of [nano-banana-mcpv2](https://github.com/notfixingit3/nano-banana-mcpv2), which uses Google's Gemini image APIs.

---

### Sample Output (Grok Imagine)

Preview image generated with `grok-imagine-image-quality`:

<p align="center">
  <img src="assets/sample_output.png" alt="Grok Imagine Sample Output" width="400" />
</p>

For conversion verification, protocol tests, and API diagnostics, see [TEST_REPORT.md](TEST_REPORT.md).

---

## Free Development (No API Key Required)

**No xAI credits? No problem.** Enable mock mode to exercise the full MCP workflow offline:

```bash
export GROK_IMAGE_MOCK=1
go build -o grok-image-mcp .
./scripts/test_mock.sh
```

Mock mode:
- Runs `generate_image`, `edit_image`, and `continue_editing` without calling xAI
- Saves real image files to disk (uses `assets/sample_output.png` as a stand-in)
- Works in MCP clients — see [examples/cursor-mcp.json](examples/cursor-mcp.json)

When you eventually have credits, unset `GROK_IMAGE_MOCK` and add `XAI_API_KEY`.

---

## Features

- **Generate Images**: Create new images from text descriptions.
- **Edit Images**: Modify existing images using text prompts and optional reference images (up to 3 total).
- **Iterative Editing**: Refine the last generated or edited image sequentially.
- **Dynamic Model Selection**: Choose models via tool parameters or environment variables.
- **Cross-Platform Auto-Saving**: Automatically saves generated images locally.
- **Zero-Publish Install**: Build locally or install from GitHub releases.

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
   - `GROK_IMAGES_DIR`: Custom directory for saved images (overrides platform defaults).
   - `GROK_IMAGE_MOCK`: Set to `1` for free offline development without an API key.
   - `GROK_IMAGE_MOCK_ASSET`: Optional image file to use as mock output (defaults to `assets/sample_output.png`).
   - `GROK_IMAGE_LOG_FILE`: Optional path for request/response logging.
3. **Global Configuration**: `~/.grok-image-config.json` generated via the `configure_xai_token` tool.

---

## Getting Your API Key & xAI Credits

### How to Get Your API Key
1. Go to [console.x.ai](https://console.x.ai).
2. Create an API key for your team.
3. Set it via `XAI_API_KEY` or the `configure_xai_token` tool.

### Credits Required
Image generation and editing require an xAI team with active credits or licenses. Without credits, authenticated requests return HTTP 403. Purchase credits at [console.x.ai](https://console.x.ai).

### Pricing (as of 2026)
- **`grok-imagine-image`**: $0.02 per image
- **`grok-imagine-image-quality`**: $0.05 per image
- Image edits are billed for both input and output images

---

## Installation & Client Integration

### Method A: Build & Run Locally (Recommended for Development)

```bash
go build -o grok-image-mcp .
```

Offline tests (no API key or credits needed):

```bash
./scripts/test_protocol.sh   # MCP protocol checks
./scripts/test_mock.sh       # full generate/edit flow in mock mode
go test -v ./...             # unit tests
```

Live integration test (requires xAI credits):

```bash
export XAI_API_KEY="your-api-key-here"
./scripts/test_all.sh
```

Add this to your MCP settings file (e.g., Cursor, Claude Desktop, or Claude Code config):

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "/path/to/grok-image-mcp",
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
go build -o grok-image-mcp .
./grok-image-mcp --setup
```

### Method C: Download Pre-compiled Binary

Download binaries for macOS ARM64/AMD64, Linux AMD64, or Windows from [GitHub Releases](https://github.com/notfixingit3/grok-image-mcp/releases).

### Method D: Run via Docker

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
- **`serviceTier`** (optional): `default` or `priority` for faster processing.

### `edit_image`
Modify a specific existing image file.
- **`imagePath`** (required): Full local file path of the base image.
- **`prompt`** (required): Description of modifications.
- **`referenceImages`** (optional): Up to 2 additional reference image paths (3 images total).
- **`model`** (optional): Custom model name.
- **`aspectRatio`** (optional): Output aspect ratio.
- **`resolution`** (optional): `1k` or `2k`.
- **`numberOfImages`** (optional): Number of edited variations (1-10).
- **`serviceTier`** (optional): `default` or `priority`.

### `continue_editing`
Refine the last image generated/edited in the active session.
- **`prompt`** (required): Description of modification.
- **`referenceImages`** (optional): Array of reference image file paths.
- **`model`** (optional): Custom model name.
- **`aspectRatio`** (optional): Output aspect ratio.
- **`resolution`** (optional): `1k` or `2k`.
- **`numberOfImages`** (optional): Number of edited variations (1-10).
- **`serviceTier`** (optional): `default` or `priority`.

### `get_last_image_info`
Check details of the last generated/edited image (file path, file size, last modified timestamp).

### `get_configuration_status`
Verify if the xAI token is configured and see its origin source.

### `configure_xai_token`
Configure your xAI API key:
- **`apiKey`** (required): Your xAI API key.

---

## Differences from nano-banana-mcpv2

| Feature | nano-banana-mcpv2 | grok-image-mcp |
|---|---|---|
| API Provider | Google Gemini / Imagen | xAI Grok Imagine |
| Auth | Query param `?key=` | `Authorization: Bearer` header |
| `generate_imagen` tool | Yes (Imagen 4) | No (not applicable) |
| Max reference images | Unlimited in schema | 3 (xAI API limit) |
| Batch generation | Up to 4 (Imagen) | Up to 10 (xAI) |
| Resolution control | Via aspect ratio | `1k` / `2k` explicit |

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