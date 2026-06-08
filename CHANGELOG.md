# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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