<p align="center">
  <img src="assets/logo.png" alt="Grok Image MCP Logo" width="280" />
</p>

<p align="center">
  <strong>Grok Image MCP</strong><br />
  <sub>Image generation &amp; editing for AI coding agents · xAI Grok Imagine · MCP stdio</sub>
</p>

<p align="center">
  <a href="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/ci.yml"><img src="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/release.yml"><img src="https://github.com/notfixingit3/grok-image-mcp/actions/workflows/release.yml/badge.svg" alt="Release" /></a>
  <a href="https://github.com/notfixingit3/grok-image-mcp/releases"><img src="https://img.shields.io/github/v/release/notfixingit3/grok-image-mcp?include_prereleases&label=release" alt="Latest Release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/notfixingit3/grok-image-mcp?label=license" alt="License" /></a>
  <a href="https://github.com/notfixingit3/grok-image-mcp"><img src="https://img.shields.io/github/stars/notfixingit3/grok-image-mcp?style=flat&label=stars" alt="Stars" /></a>
</p>

<p align="center">
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" alt="Go 1.22+" /></a>
  <a href="https://modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-stdio-8B5CF6" alt="MCP" /></a>
  <a href="https://docs.x.ai/developers/model-capabilities/imagine"><img src="https://img.shields.io/badge/xAI-Grok%20Imagine-000000" alt="xAI Grok Imagine" /></a>
  <a href="TEST_REPORT.md"><img src="https://img.shields.io/badge/tests-mock%20%2B%20protocol-brightgreen" alt="Tests" /></a>
  <a href="examples/cursor-mcp.json"><img src="https://img.shields.io/badge/mock%20mode-free-blue" alt="Mock Mode" /></a>
  <a href="https://www.cursor.com"><img src="https://img.shields.io/badge/Cursor-compatible-222222" alt="Cursor" /></a>
</p>

# Grok Image MCP Server

An MCP (Model Context Protocol) server that brings **Grok Imagine** image generation and editing to **Grok Build**, **Cursor**, **Claude Desktop**, **Claude Code**, **OpenCode**, **VS Code**, **Windsurf**, and any MCP-compatible client.

Adapted from [nano-banana-mcpv2](https://github.com/notfixingit3/nano-banana-mcpv2) (Gemini/Imagen) for xAI's image API.

---

## Two Modes — Both Supported

| | **Mock mode** (free) | **Live mode** (xAI credits) |
|---|---|---|
| **Enable** | `GROK_IMAGE_MOCK=1` | `XAI_API_KEY=...` |
| **API calls** | None | xAI Grok Imagine |
| **Cost** | Free | ~$0.02–$0.05 / image |
| **Use case** | Dev, testing, MCP wiring | Real image generation |
| **Config examples** | [examples/](examples/) (per-client) | [examples/cursor-mcp-live.json](examples/cursor-mcp-live.json) |

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

### 2. Configure your MCP client

See **[Client Integration](#client-integration)** below for Grok Build, Cursor, Claude, OpenCode, VS Code, Windsurf, and Docker.

**Clone-and-go:** this repo ships with project-level mock configs — [`.grok/config.toml`](.grok/config.toml) (Grok Build), [`.mcp.json`](.mcp.json) (Claude Code), and [`.cursor/mcp.json`](.cursor/mcp.json) (Cursor). Open the repo in any of these clients and the tools load automatically (uses `go run .`, no pre-build needed).

### 3. Verify

```bash
./scripts/test_protocol.sh   # protocol checks (no key)
./scripts/test_mock.sh       # full offline flow (no key)
go test -v ./...             # unit tests
```

---

## Client Integration

Replace `/path/to/grok-image-mcp` with your built binary (`go build -o grok-image-mcp .`) or a [release binary](https://github.com/notfixingit3/grok-image-mcp/releases). All examples below use **mock mode**; for live mode, swap `GROK_IMAGE_MOCK` for `XAI_API_KEY`.

| Client | Config file | Example |
|---|---|---|
| **Grok Build** | `~/.grok/config.toml` or `.grok/config.toml` | [grok-build-mock.toml](examples/grok-build-mock.toml) |
| **Cursor** | `~/.cursor/mcp.json` or `.cursor/mcp.json` | [cursor-mcp.json](examples/cursor-mcp.json) |
| **Claude Desktop** | `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) | [claude-desktop-mock.json](examples/claude-desktop-mock.json) |
| **Claude Code** | `.mcp.json` in project root (or `~/.claude.json`) | [claude-code-mock.json](examples/claude-code-mock.json) |
| **OpenCode** | `opencode.jsonc` in project root (or `~/.config/opencode/opencode.jsonc`) | [opencode-mock.jsonc](examples/opencode-mock.jsonc) |
| **VS Code** (Copilot MCP) | `~/Library/Application Support/Code/User/mcp.json` (macOS) | [vscode-mcp.json](examples/vscode-mcp.json) |
| **Windsurf** | `~/.codeium/windsurf/mcp_config.json` | [windsurf-mcp.json](examples/windsurf-mcp.json) |
| **Docker** | any client's `mcpServers` block | see [Installation Options](#installation-options) |

> **Grok Build** also auto-loads `.mcp.json`, `.cursor/mcp.json`, and `~/.claude.json` for compatibility. Run `grok mcp list` or `grok mcp doctor grok-image-mcp` to verify.

---

### Grok Build

**Global** (`~/.grok/config.toml`) or **project** (`.grok/config.toml`):

```toml
[mcp_servers.grok-image-mcp]
command = "/path/to/grok-image-mcp"
env = { GROK_IMAGE_MOCK = "1", GROK_IMAGE_MODEL = "grok-imagine-image-quality" }
enabled = true
```

**Live mode** — reference an env var instead of hardcoding the key:

```toml
[mcp_servers.grok-image-mcp]
command = "/path/to/grok-image-mcp"
env = { XAI_API_KEY = "${XAI_API_KEY}", GROK_IMAGE_MODEL = "grok-imagine-image-quality" }
enabled = true
```

**CLI:**

```bash
grok mcp add grok-image-mcp --command /path/to/grok-image-mcp \
  --env GROK_IMAGE_MOCK=1 GROK_IMAGE_MODEL=grok-imagine-image-quality
grok mcp doctor grok-image-mcp
```

Full examples: [grok-build-mock.toml](examples/grok-build-mock.toml) · [grok-build-live.toml](examples/grok-build-live.toml)

---

### Cursor

Edit `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project):

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "/path/to/grok-image-mcp",
      "env": {
        "GROK_IMAGE_MOCK": "1",
        "GROK_IMAGE_MODEL": "grok-imagine-image-quality"
      }
    }
  }
}
```

Restart Cursor or reload MCP servers from settings. Example: [cursor-mcp.json](examples/cursor-mcp.json) · live: [cursor-mcp-live.json](examples/cursor-mcp-live.json)

---

### Claude Desktop

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`  
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "/path/to/grok-image-mcp",
      "env": {
        "GROK_IMAGE_MOCK": "1",
        "GROK_IMAGE_MODEL": "grok-imagine-image-quality"
      }
    }
  }
}
```

Restart Claude Desktop after saving. Example: [claude-desktop-mock.json](examples/claude-desktop-mock.json)

---

### Claude Code

Place `.mcp.json` in your project root (Claude Code walks up to the git root):

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "/path/to/grok-image-mcp",
      "env": {
        "GROK_IMAGE_MOCK": "1",
        "GROK_IMAGE_MODEL": "grok-imagine-image-quality"
      }
    }
  }
}
```

This repo includes a ready-to-use [`.mcp.json`](.mcp.json) (mock mode via `go run .`). Example: [claude-code-mock.json](examples/claude-code-mock.json)

---

### OpenCode

Add to `opencode.jsonc` (project root or `~/.config/opencode/opencode.jsonc`):

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "grok-image-mcp": {
      "type": "local",
      "command": ["/path/to/grok-image-mcp"],
      "environment": {
        "GROK_IMAGE_MOCK": "1",
        "GROK_IMAGE_MODEL": "grok-imagine-image-quality"
      },
      "enabled": true
    }
  }
}
```

OpenCode uses `environment` (not `env`) and `command` as an array. Verify with `opencode mcp list`. Examples: [opencode-mock.jsonc](examples/opencode-mock.jsonc) · [opencode-live.jsonc](examples/opencode-live.jsonc)

---

### VS Code (GitHub Copilot MCP)

Edit `~/Library/Application Support/Code/User/mcp.json` (macOS) or the equivalent on your OS:

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "/path/to/grok-image-mcp",
      "env": {
        "GROK_IMAGE_MOCK": "1",
        "GROK_IMAGE_MODEL": "grok-imagine-image-quality"
      }
    }
  }
}
```

Example: [vscode-mcp.json](examples/vscode-mcp.json)

---

### Windsurf

Edit `~/.codeium/windsurf/mcp_config.json`:

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "/path/to/grok-image-mcp",
      "env": {
        "GROK_IMAGE_MOCK": "1",
        "GROK_IMAGE_MODEL": "grok-imagine-image-quality"
      }
    }
  }
}
```

Restart Windsurf / reload MCP. Example: [windsurf-mcp.json](examples/windsurf-mcp.json)

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
| **Setup wizard** | `./grok-image-mcp --setup` (mock or live) |
| **CLI help** | `./grok-image-mcp --help` / `--version` |
| **Pre-built binary** | [GitHub Releases](https://github.com/notfixingit3/grok-image-mcp/releases) |
| **Docker** | `docker build -t grok-image-mcp .` |

Docker MCP config (live):

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

Docker MCP config (mock — no credits):

```json
{
  "mcpServers": {
    "grok-image-mcp": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-e", "GROK_IMAGE_MOCK=1", "grok-image-mcp"]
    }
  }
}
```

---

## Troubleshooting

| Issue | What to do |
|---|---|
| **HTTP 403 — no credits/licenses** | Add credits at [console.x.ai](https://console.x.ai). The server surfaces this with a direct link. |
| **HTTP 429 — rate limit** | Server retries automatically (up to 3 attempts). Wait and retry, or use `serviceTier: "default"`. |
| **Tools not visible in client** | Confirm binary path is absolute, restart/reload MCP, run `grok mcp doctor grok-image-mcp` (Grok Build). |
| **No xAI key yet** | Use mock mode: `GROK_IMAGE_MOCK=1` or `./grok-image-mcp --mock`. Run `./scripts/test_mock.sh` to verify. |
| **Empty or invalid tool args** | v0.2+ validates prompts, enums, and ranges before calling xAI — check the error message. |

**Pricing (live mode):** `grok-imagine-image` ~$0.02/image · `grok-imagine-image-quality` ~$0.05/image. See [xAI Imagine docs](https://docs.x.ai/developers/model-capabilities/imagine).

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

- **`dev`** — active development, pre-releases tagged `v*.*.*-beta.*`
- **`main`** — stable releases (`v*.*.*`)

Changes go to `dev`, then PR to `main`. Tag betas on `dev` (e.g. `v0.2.0-beta.0`), stable releases on `main`.

---

## License & Credits

MIT License — see [LICENSE](LICENSE).

- Forked concept from [nano-banana-mcpv2](https://github.com/notfixingit3/nano-banana-mcpv2)
- Image API by [xAI Grok Imagine](https://docs.x.ai/developers/model-capabilities/imagine)
- Protocol by [Anthropic MCP](https://modelcontextprotocol.io)