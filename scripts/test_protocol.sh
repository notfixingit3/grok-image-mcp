#!/bin/bash
# Protocol-level MCP tests that do not require live xAI API credits

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

call_rpc() {
  echo "$1" | ./grok-image-mcp
}

echo "🚀 Building server binary..."
go build -o grok-image-mcp .

echo "🔧 initialize"
RESP=$(call_rpc '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":0}')
echo "$RESP" | grep -q '"name":"grok-image-mcp"' || exit 1
echo "✅ initialize"

echo "🔧 tools/list"
RESP=$(call_rpc '{"jsonrpc":"2.0","method":"tools/list","id":1}')
for tool in configure_xai_token generate_image edit_image continue_editing get_configuration_status get_last_image_info; do
  echo "$RESP" | grep -q "\"name\":\"$tool\"" || { echo "❌ missing $tool"; exit 1; }
done
echo "✅ tools/list"

echo "🔧 get_configuration_status (unconfigured)"
unset XAI_API_KEY || true
RESP=$(call_rpc '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_configuration_status","arguments":{}},"id":2}')
echo "$RESP" | grep -q 'not configured' || exit 1
echo "✅ get_configuration_status unconfigured"

echo "🔧 get_last_image_info (empty session)"
RESP=$(call_rpc '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_last_image_info","arguments":{}},"id":3}')
echo "$RESP" | grep -q 'No previous image found' || exit 1
echo "✅ get_last_image_info empty"

echo "🔧 continue_editing without prior image"
RESP=$(call_rpc '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"continue_editing","arguments":{"prompt":"test"}},"id":4}')
echo "$RESP" | grep -q 'No previous image found' || exit 1
echo "✅ continue_editing guard"

echo "🔧 unknown tool"
RESP=$(call_rpc '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"generate_imagen","arguments":{}},"id":5}')
echo "$RESP" | grep -q 'Unknown tool' || exit 1
echo "✅ unknown tool rejected (Gemini tool removed)"

echo "🔧 edit_image rejects unsupported format without API call"
FAKE_KEY="xai-test-key-not-real"
export XAI_API_KEY="$FAKE_KEY"
BAD_FORMAT_FILE="/tmp/grok-image-mcp-test.gif"
printf 'GIF' > "$BAD_FORMAT_FILE"
RESP=$(call_rpc "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"edit_image\",\"arguments\":{\"imagePath\":\"$BAD_FORMAT_FILE\",\"prompt\":\"test\"}},\"id\":6}")
echo "$RESP" | grep -q 'unsupported image format' || exit 1
echo "✅ edit_image format validation"

echo "🔧 edit_image rejects oversized file without API call"
BIG_FILE="/tmp/grok-image-mcp-oversized.jpg"
dd if=/dev/zero of="$BIG_FILE" bs=1m count=21 status=none 2>/dev/null || dd if=/dev/zero of="$BIG_FILE" bs=1048576 count=21 2>/dev/null
trap 'rm -f "$BIG_FILE" "$BAD_FORMAT_FILE"' EXIT
RESP=$(call_rpc "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"edit_image\",\"arguments\":{\"imagePath\":\"$BIG_FILE\",\"prompt\":\"test\"}},\"id\":7}")
echo "$RESP" | grep -q '20 MiB' || exit 1
echo "✅ edit_image size validation"

echo "🔧 GROK_IMAGES_DIR respected"
CUSTOM_DIR="$ROOT_DIR/.test-output-dir"
mkdir -p "$CUSTOM_DIR"
export GROK_IMAGES_DIR="$CUSTOM_DIR"
RESP=$(call_rpc '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_last_image_info","arguments":{}},"id":8}')
echo "$RESP" | grep -q 'No previous image found' || exit 1
echo "✅ GROK_IMAGES_DIR env accepted"

unset XAI_API_KEY || true

echo ""
echo "🎉 Protocol tests passed."