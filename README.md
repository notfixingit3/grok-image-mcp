<p align="center">
  <img src="assets/logo.png" alt="Grok Image MCP Logo" width="280" />
</p>

<p align="center">
  <strong>Grok Image MCP</strong><br />
  <sub>Image generation &amp; editing for AI coding agents · xAI Grok Imagine · MCP stdio</sub>
</p>

<p align="center">
  <a href="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/ci.yml"><img src="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
  <a href="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/release.yml"><img src="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/release.yml/badge.svg" alt="Release Workflow Status" /></a>
  <a href="https://github.com/notfixingit3/grok-image-mcp/releases"><img src="https://img.shields.io/github/v/release/notfixingit3/grok-image-mcp?include_prereleases" alt="Latest Release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/notfixingit3/grok-image-mcp" alt="License" /></a>
  <a href="TEST_REPORT.md"><img src="https://img.shields.io/badge/tests-mock%20%2B%20protocol-brightgreen" alt="Tests" /></a>
</p>

# Grok Image MCP Server

An MCP (Model Context Protocol) server that brings **Grok Imagine** image generation and editing to Cursor, Claude Desktop, Claude Code, and any MCP-compatible client.

Adapted from [nano-banana-mcpv2](https://github.com/notfixingit3/nano-banana-mcpv2) (Gemini/Imagen) for xAI's image API.

---

## Two Modes — Both Supported

| | **Mock mode** (free) | **Live mode** (xAI credits) |
|---|---|---|
| **Enable** | `GROK_IMAGE_MOCK=1` | `XAI_API_KEY=...` |
| **API calls** | None | xAI Grok Imagine |
| **Cost** | Free | ~$0.02–$0.05 / image |
| **Use case** | Dev, testing, MCP wiring | Real image generation |
| **Config example** | [examples/cursor-mcp.json](examples/cursor-mcp.json) | [examples/cursor-mcp-live.json](examples/cursor-mcp-live.json) |

Switch anytime: unset `GROK_IMAGE_MOCK` and add your key when you're ready for live generation.

```bash
# Free — works right now
export GROK_IMAGE_MOCK=1
go build -o grok-image-mcp .
./scripts/test_mock.sh

# Live — when you have xAI credits
export XAI_API_KEY="your-key"
unset GROK_IMAGE_MOCK
./scripts/test_all.sh
```

---

### Sample Output

<p align="center">
  <img src="assets/sample_output.png" alt="Grok Imagine sample output" width="420" />
</p>

<p align="center">
  <sub><i>Preview asset · see <a href="TEST_REPORT.md">TEST_REPORT.md</a> for test status</i></sub>
</p>

---

## Features

- **Generate images** — text-to-image via Grok Imagine (`grok-imagine-image` / `grok-imagine-image-quality`)
- **Edit images** — modify existing files with prompts and up to 2 reference images (3 total)
- **Continue editing** — iteratively refine the last image in a session
- **Mock mode** — full offline workflow without an API key or credits
- **Auto-save** — images written to disk automatically for agent follow-up
- **Cross-platform** — single Go binary, no Node.js runtime

---

## Supported Models

| Model | Speed | Cost | Best for |
|---|---|---|---|
| `grok-imagine-image-quality` *(default)* | Slower | $0.05/image | High-fidelity creative work |
| `grok-imagine-image` | Faster | $0.02/image | Quick drafts and iteration |

---

## Quick Start

### 1. Build

```bash
git clone https://github.com/notfixingit3/grok-image-mcp.git
cd grok-image-mcp
go build -o grok-image-mcp .
```

### 2. Configure MCP client

**Mock (no key needed):**

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "/path/to/grok-image-mcp",
      "env": { "GROK_IMAGE_MOCK": "1" }
    }
  }
}
```

**Live:**

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

### 3. Verify

```bash
./scripts/test_protocol.sh   # protocol checks (no key)
./scripts/test_mock.sh       # full offline flow (no key)
go test -v ./...             # unit tests
```

---

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `XAI_API_KEY` | Live mode | xAI API key from [console.x.ai](https://console.x.ai) |
| `GROK_IMAGE_MOCK` | No | Set to `1` for free offline mock mode |
| `GROK_IMAGE_MODEL` | No | Default model (`grok-imagine-image-quality`) |
| `GROK_IMAGES_DIR` | No | Custom output directory for saved images |
| `GROK_IMAGE_MOCK_ASSET` | No | Image file used as mock output |
| `GROK_IMAGE_LOG_FILE` | No | Optional request/response log path |

Config file fallback: `~/.grok-image-config.json` via the `configure_xai_token` tool.

---

## Available Tools

| Tool | Description |
|---|---|
| `generate_image` | Create a new image from a text prompt |
| `edit_image` | Edit a specific image file by path |
| `continue_editing` | Edit the last image in the current session |
| `get_last_image_info` | Path, size, and timestamp of the last image |
| `get_configuration_status` | Check API key / mock mode status |
| `configure_xai_token` | Save an xAI API key to `~/.grok-image-config.json` |

<details>
<summary><strong>Tool parameters</strong></summary>

### `generate_image`
- `prompt` *(required)*, `model`, `aspectRatio`, `resolution` (`1k`/`2k`), `numberOfImages` (1–10), `serviceTier` (`default`/`priority`)

### `edit_image` / `continue_editing`
- `prompt` *(required)*, `imagePath` *(edit only)*, `referenceImages`, `model`, `aspectRatio`, `resolution`, `numberOfImages`, `serviceTier`

</details>

---

## Installation Options

| Method | Command |
|---|---|
| **Local build** | `go build -o grok-image-mcp .` |
| **Setup wizard** | `./grok-image-mcp --setup` |
| **Pre-built binary** | [GitHub Releases](https://github.com/notfixingit3/grok-image-mcp/releases) |
| **Docker** | `docker build -t grok-image-mcp .` |

Docker MCP config:

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-e", "XAI_API_KEY=your-key", "grok-image-mcp"]
    }
  }
}
```

---

## File Storage

| Platform | Default directory |
|---|---|
| macOS / Linux (dev) | `./generated_imgs/` |
| macOS / Linux (system paths) | `~/grok-images/` |
| Windows | `%USERPROFILE%\Documents\grok-images\` |

Override with `GROK_IMAGES_DIR`.

---

## Contributing

- **`dev`** — active development
- **`main`** — stable releases (`v*.*.*`)

Changes go to `dev`, then PR to `main`.

---

## License & Credits

MIT License — see [LICENSE](LICENSE).

- Forked concept from [nano-banana-mcpv2](https://github.com/notfixingit3/nano-banana-mcpv2)
- Image API by [xAI Grok Imagine](https://docs.x.ai/developers/model-capabilities/imagine)
- Protocol by [Anthropic MCP](https://modelcontextprotocol.io)