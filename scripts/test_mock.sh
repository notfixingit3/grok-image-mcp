#!/bin/bash
# Full offline integration test using mock mode (no xAI API key or credits required)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

export GROK_IMAGE_MOCK=1
export GROK_IMAGES_DIR="$ROOT_DIR/.test-mock-output"
rm -rf "$GROK_IMAGES_DIR"
mkdir -p "$GROK_IMAGES_DIR"
unset XAI_API_KEY || true

call_tool() {
  local name="$1"
  local args="$2"
  local id="${3:-1}"
  local request
  request=$(printf '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"%s","arguments":%s},"id":%s}' "$name" "$args" "$id")
  echo "$request" | ./grok-image-mcp
}

assert_no_error() {
  local response="$1"
  local label="$2"
  if echo "$response" | grep -q '"error"'; then
    echo "❌ $label failed:"
    echo "$response"
    exit 1
  fi
  echo "✅ $label"
}

echo "🚀 Building server binary..."
go build -o grok-image-mcp .

echo "🔧 Mock configuration status"
RESP=$(call_tool "get_configuration_status" '{}' 1)
assert_no_error "$RESP" "get_configuration_status"
echo "$RESP" | grep -q 'Mock mode is active' || exit 1

echo "🎨 Mock generate_image"
GEN_RESPONSE=$(call_tool "generate_image" '{"prompt":"A test image in mock mode","model":"grok-imagine-image","aspectRatio":"1:1","numberOfImages":2}' 2)
assert_no_error "$GEN_RESPONSE" "generate_image"
echo "$GEN_RESPONSE" | grep -q '\[MOCK\]' || exit 1

GEN_COUNT=$(ls "$GROK_IMAGES_DIR"/generated-* 2>/dev/null | wc -l | tr -d ' ')
if [ "$GEN_COUNT" -lt 2 ]; then
  echo "❌ Expected at least 2 mock generated files, found $GEN_COUNT"
  exit 1
fi
echo "✅ Mock generated $GEN_COUNT file(s)"

echo "🎨 Mock continue_editing (single server session)"
SESSION_OUTPUT=$(./grok-image-mcp 2>/dev/null <<'EOF'
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"generate_image","arguments":{"prompt":"session seed image","model":"grok-imagine-image"}},"id":10}
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"continue_editing","arguments":{"prompt":"Add a mock glow effect"}},"id":11}
EOF
)
EDIT_LINE=$(echo "$SESSION_OUTPUT" | sed -n '2p')
if echo "$EDIT_LINE" | grep -q '"error"'; then
  echo "❌ continue_editing session test failed"
  exit 1
fi
echo "$EDIT_LINE" | grep -q '\[MOCK\].*edited' || exit 1
echo "✅ continue_editing in persistent session"

echo "🎨 Mock edit_image with explicit path"
LAST_IMAGE=$(ls -t "$GROK_IMAGES_DIR"/generated-* 2>/dev/null | head -1)
PATH_EDIT=$(call_tool "edit_image" "{\"imagePath\":\"$LAST_IMAGE\",\"prompt\":\"Mock recolor\"}" 5)
assert_no_error "$PATH_EDIT" "edit_image"
echo "$PATH_EDIT" | grep -q 'Edited image saved' || exit 1

echo "🔧 Mock get_last_image_info (session)"
INFO_OUTPUT=$(./grok-image-mcp 2>/dev/null <<'EOF'
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"generate_image","arguments":{"prompt":"info test"}},"id":20}
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_last_image_info","arguments":{}},"id":21}
EOF
)
if echo "$INFO_OUTPUT" | grep -q '"error"'; then
  echo "❌ get_last_image_info session test failed"
  exit 1
fi
echo "$INFO_OUTPUT" | grep -q 'Last Image Information' || exit 1
echo "✅ get_last_image_info in persistent session"

echo ""
echo "🎉 Mock integration tests passed — no API key or credits needed."