#!/bin/bash
# Generate README assets (logo + sample output) using grok-image-mcp

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ -z "${XAI_API_KEY:-}" ]; then
  echo "❌ Error: XAI_API_KEY environment variable is not set."
  exit 1
fi

mkdir -p assets generated_imgs

go build -o grok-image-mcp main.go

call_tool() {
  local name="$1"
  local args="$2"
  local id="$3"
  local request
  request=$(printf '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"%s","arguments":%s},"id":%s}' "$name" "$args" "$id")
  echo "$request" | ./grok-image-mcp
}

copy_latest() {
  local prefix="$1"
  local dest="$2"
  local latest
  latest=$(ls -t generated_imgs/${prefix}-* 2>/dev/null | head -1)
  if [ -z "$latest" ]; then
    echo "❌ No generated file with prefix $prefix"
    exit 1
  fi
  cp "$latest" "$dest"
  echo "✅ Saved $dest"
}

echo "🎨 Generating logo..."
LOGO_PROMPT='A premium app icon logo for an AI image MCP server called Grok Image. A stylized luminous picture frame containing flowing abstract light waves, electric blue and white on a deep black background, minimal flat vector design, centered composition, modern tech branding, no text, no letters'
LOGO_RESPONSE=$(call_tool "generate_image" "$(jq -nc --arg p "$LOGO_PROMPT" '{prompt:$p, model:"grok-imagine-image-quality", aspectRatio:"1:1", resolution:"1k"}')" 101)
echo "$LOGO_RESPONSE" | grep -q '"error"' && { echo "$LOGO_RESPONSE"; exit 1; }
copy_latest "generated" "assets/logo.png"

echo "🎨 Generating sample output..."
SAMPLE_PROMPT='A sleek high-fidelity digital art illustration of a luminous neural image waveform emerging from a picture frame in deep space, neon cosmic dust and modern geometric lines, electric blue and violet palette, ultra detailed cinematic lighting, representing premium Grok Imagine image generation quality'
SAMPLE_RESPONSE=$(call_tool "generate_image" "$(jq -nc --arg p "$SAMPLE_PROMPT" '{prompt:$p, model:"grok-imagine-image-quality", aspectRatio:"1:1", resolution:"1k"}')" 102)
echo "$SAMPLE_RESPONSE" | grep -q '"error"' && { echo "$SAMPLE_RESPONSE"; exit 1; }
copy_latest "generated" "assets/sample_output.png"

echo "🎉 Asset generation complete."