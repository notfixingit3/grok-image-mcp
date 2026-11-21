# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-11-20

### Added
- MCP `ping` request handler and graceful handling for `notifications/cancelled`.
- MCP protocol version negotiation supporting `2024-11-05`, `2025-03-26`, and `2025-06-18` (defaulting to `2025-06-18` for unknown versions).
- Embedded default mock sample asset (`assets/sample_output.png`) via `go:embed` so mock mode works regardless of working directory.

### Changed
- Raised stdin `bufio.Scanner` buffer limit to 10 MiB to prevent crashes on large JSON-RPC payloads.
- Tool-level validation, filesystem, and xAI API failures now return MCP tool results with `isError: true` instead of JSON-RPC protocol errors.
- Image save failures in `generate_image` and `edit_image` now surface explicit disk errors instead of claiming no image was returned by xAI.
- Model validation allow-list now honors custom models configured via `GROK_IMAGE_MODEL` consistently across tool schemas, default resolution, and explicit tool arguments.
- Bumped Go version in `go.mod` and `Dockerfile` to `1.27` (`golang:1.27-alpine3.24`) and Alpine runtime base image to `3.24` (`alpine:3.24`).

### Security
- Redacted `apiKey`, `token`, and other credential fields before writing raw JSON-RPC requests to `GROK_IMAGE_LOG_FILE`.
- Removed automatic reading and silent migration of `.grok-image-config.json` from the current working directory into `$HOME`.
- Atomic file writes (temp file + chmod + sync + rename) for `~/.grok/auth.json` and `~/.grok-image-config.json`, with explicit `os.UserHomeDir()` error checks.
- Bounded `downloadImage` response reads to 20 MiB using `io.LimitReader`.
- Isolated `HOME` across all tests in `scripts/test_protocol.sh` so running tests never modifies real user configuration files.

## [0.2.0] - 2026-06-08

Stable release promoting `v0.2.0-beta.1` from `dev`.

### Added
- **Hermes Agent** client integration docs and examples (`hermes-oauth-live.yaml`, `hermes-mock.yaml`).
- Project [`opencode.jsonc`](opencode.jsonc) for live OAuth; `opencode-oauth-live.jsonc` and `claude-desktop-oauth-live.json` examples.

## [0.2.0-beta.1] - 2026-06-08

### Added
- **Release proof images** — `scripts/generate_release_proof.sh` saves live generate/edit output to `assets/releases/<tag>/` for `TEST_REPORT.md` and GitHub Releases.
- **Grok subscription OAuth** — auto-reads `~/.grok/auth.json` from `grok login` (SuperGrok / X Premium+).
- OAuth token refresh via `https://auth.x.ai/oauth2/token` when the session is near expiry.
- `GROK_IMAGE_AUTH` (`auto` / `oauth` / `api_key`) and `GROK_AUTH_JSON` environment variables.
- Example config [examples/grok-oauth-live.toml](examples/grok-oauth-live.toml) for keyless live mode.

### Changed
- Credential priority in `auto` mode: Grok OAuth → `XAI_API_KEY` → config file.
- `get_configuration_status` and setup wizard report OAuth subscription state.
- Error messages mention `grok login` as an alternative to API keys.

## [0.2.0-beta.0] - 2026-06-08

### Added
- **Client Integration** docs for Grok Build, Cursor, Claude Desktop, Claude Code, OpenCode, VS Code, and Windsurf.
- Per-client example configs in `examples/` (TOML, JSON, JSONC).
- Project-level auto-config: `.grok/config.toml`, `.mcp.json`, `.cursor/mcp.json` (mock mode via `go run .`).
- CLI `--version` and `--help` flags.
- Input validation for tool arguments (empty prompts, invalid enums, `numberOfImages` range).
- Mock-mode protocol tests in `scripts/test_protocol.sh`.

### Changed
- Setup wizard (`--setup`) now offers mock-only, live key, or skip paths.
- Docker docs include mock-mode example alongside live config.

## [0.1.2] - 2026-06-08

### Added
- **Mock mode** (`GROK_IMAGE_MOCK=1` or `--mock`) for free offline development without xAI credits.
- `scripts/test_mock.sh` — full generate/edit/continue integration test using mock mode.
- Example MCP configs in `examples/` for Cursor (mock and live).

### Changed
- Polished README with dual-mode table, quick start, and centered logo hero.
- Resolved all `gosec` findings with documented `#nosec` annotations.
- Image tools skip API key requirement when mock mode is active.
- `configure_xai_token` skips live validation in mock mode.
- CI now runs mock integration tests on every push.

## [0.1.1] - 2026-06-08

### Added
- `GROK_IMAGES_DIR` environment variable for custom image output directory.
- `serviceTier` parameter on `generate_image`, `edit_image`, and `continue_editing`.
- `numberOfImages` parameter on `edit_image` and `continue_editing`.
- Automatic retry with backoff on xAI HTTP 429 rate limits.
- Image file validation before upload (20 MiB max, JPEG/PNG/WebP only).
- Warnings when reference images are skipped or fail to load.
- API key validation in `configure_xai_token` (matches `--setup` behavior).
- Go unit tests (`main_test.go`) and CI workflow (tests, protocol tests, gosec).
- Expanded protocol tests for format/size validation.

### Changed
- Build all scripts and Docker/release workflows with `go build .` (multi-file package).
- Replaced deprecated `math/rand` seeding with `crypto/rand` filename suffixes.
- Generic README MCP config path (`/path/to/grok-image-mcp`).

## [0.1.0] - 2026-06-08

### Changed
- Improved xAI API error messages when credits/licenses are missing.
- `get_last_image_info` and `continue_editing` guard checks no longer require an API key.

### Added
- Grok Imagine-generated logo and sample output in `assets/`.
- `TEST_REPORT.md` with conversion checklist and test results.
- `scripts/test_protocol.sh` for MCP protocol tests without API credits.
- `scripts/test_all.sh` for full integration testing.
- `scripts/generate_assets.sh` to dogfood asset generation through the MCP server.
- Initial release of `grok-image-mcp`, adapted from [nano-banana-mcpv2](https://github.com/notfixingit3/nano-banana-mcpv2).
- MCP tools: `generate_image`, `edit_image`, `continue_editing`, `get_last_image_info`, `get_configuration_status`, `configure_xai_token`.
- xAI Grok Imagine API integration via `POST /v1/images/generations` and `POST /v1/images/edits`.
- Support for `grok-imagine-image-quality` and `grok-imagine-image` models.
- Configurable aspect ratio, resolution (`1k`/`2k`), and batch image generation (up to 10).
- Multi-image editing with up to 3 source images (main + 2 references).
- Global configuration at `~/.grok-image-config.json` with `XAI_API_KEY` environment variable override.
- Interactive setup wizard (`--setup`) with API key validation.
- Cross-platform image auto-saving to `generated_imgs/` or `~/grok-images/`.
- Multi-stage Dockerfile and GitHub Actions release workflow for cross-compiled binaries.