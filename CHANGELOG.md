# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-06-08

### Added
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