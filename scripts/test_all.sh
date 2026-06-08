#!/bin/bash
# Comprehensive integration test for grok-image-mcp over stdio

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ -z "${XAI_API_KEY:-}" ] && [ ! -f "${GROK_AUTH_JSON:-$HOME/.grok/auth.json}" ]; then
  echo "❌ Error: no live credentials found."
  echo "   Set XAI_API_KEY or run 'grok login' (SuperGrok / X Premium+ OAuth)."
  exit 1
fi

unset GROK_IMAGE_MOCK || true

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

echo "🔧 Testing initialize..."
INIT_REQUEST='{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":0}'
INIT_RESPONSE=$(echo "$INIT_REQUEST" | ./grok-image-mcp)
assert_no_error "$INIT_RESPONSE" "initialize"
echo "$INIT_RESPONSE" | grep -q '"name":"grok-image-mcp"' || { echo "❌ Wrong server name"; exit 1; }

echo "🔧 Testing tools/list..."
LIST_REQUEST='{"jsonrpc":"2.0","method":"tools/list","id":2}'
LIST_RESPONSE=$(echo "$LIST_REQUEST" | ./grok-image-mcp)
assert_no_error "$LIST_RESPONSE" "tools/list"
for tool in configure_xai_token generate_image edit_image continue_editing get_configuration_status get_last_image_info; do
  echo "$LIST_RESPONSE" | grep -q "\"name\":\"$tool\"" || { echo "❌ Missing tool: $tool"; exit 1; }
done
echo "✅ All expected tools present"

echo "🔧 Testing get_configuration_status..."
STATUS_RESPONSE=$(call_tool "get_configuration_status" '{}' 3)
assert_no_error "$STATUS_RESPONSE" "get_configuration_status"
echo "$STATUS_RESPONSE" | grep -qE 'configured and ready|Grok subscription OAuth is active' || { echo "❌ Live credentials not detected"; exit 1; }

echo "🎨 Testing generate_image + get_last_image_info + continue_editing (single session)..."
SESSION_OUTPUT=$(./grok-image-mcp 2>/dev/null <<'EOF'
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"generate_image","arguments":{"prompt":"A minimal abstract logo mark: a glowing picture frame with flowing light trails, electric blue on deep black, clean vector style, no text","model":"grok-imagine-image","aspectRatio":"1:1","resolution":"1k"}},"id":4}
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_last_image_info","arguments":{}},"id":5}
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"continue_editing","arguments":{"prompt":"Add a subtle soft purple outer glow around the frame while keeping the same composition","model":"grok-imagine-image"}},"id":6}
EOF
)
GEN_RESPONSE=$(echo "$SESSION_OUTPUT" | sed -n '1p')
INFO_RESPONSE=$(echo "$SESSION_OUTPUT" | sed -n '2p')
EDIT_RESPONSE=$(echo "$SESSION_OUTPUT" | sed -n '3p')

assert_no_error "$GEN_RESPONSE" "generate_image"
echo "$GEN_RESPONSE" | grep -q 'Image generated with Grok Imagine' || { echo "❌ Unexpected generate response"; exit 1; }

LAST_IMAGE=$(ls -t generated_imgs/generated-* 2>/dev/null | head -1)
if [ -z "$LAST_IMAGE" ] || [ ! -f "$LAST_IMAGE" ]; then
  echo "❌ No generated image file found in generated_imgs/"
  exit 1
fi
echo "✅ Generated image saved: $LAST_IMAGE"

assert_no_error "$INFO_RESPONSE" "get_last_image_info"
echo "$INFO_RESPONSE" | grep -q "$LAST_IMAGE" || { echo "❌ Last image path mismatch"; exit 1; }

assert_no_error "$EDIT_RESPONSE" "continue_editing"
EDITED_IMAGE=$(ls -t generated_imgs/edited-* 2>/dev/null | head -1)
if [ -z "$EDITED_IMAGE" ] || [ ! -f "$EDITED_IMAGE" ]; then
  echo "❌ No edited image file found"
  exit 1
fi
echo "✅ Edited image saved: $EDITED_IMAGE"

echo "🎨 Testing edit_image with explicit path..."
PATH_EDIT_RESPONSE=$(call_tool "edit_image" "{\"imagePath\":\"$EDITED_IMAGE\",\"prompt\":\"Shift the palette toward warmer cyan highlights\",\"model\":\"grok-imagine-image\"}" 7)
assert_no_error "$PATH_EDIT_RESPONSE" "edit_image"
echo "$PATH_EDIT_RESPONSE" | grep -q 'Image edited with Grok Imagine' || { echo "❌ Unexpected edit response"; exit 1; }

echo ""
echo "🎉 All integration tests passed!"