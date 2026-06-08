#!/bin/bash
# Save version-tagged proof images for a release (OAuth or API key).
# Usage:
#   ./scripts/generate_release_proof.sh              # generate fresh proof for current --version
#   ./scripts/generate_release_proof.sh v0.2.0-beta.1
#   ./scripts/generate_release_proof.sh --reuse-latest   # copy latest test outputs (no API calls)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

REUSE_LATEST=0
TAG=""

for arg in "$@"; do
  case "$arg" in
    --reuse-latest) REUSE_LATEST=1 ;;
    -h|--help)
      echo "Usage: $0 [vX.Y.Z] [--reuse-latest]"
      exit 0
      ;;
    *)
      if [ -z "$TAG" ]; then
        TAG="$arg"
      fi
      ;;
  esac
done

if [ -z "${XAI_API_KEY:-}" ] && [ ! -f "${GROK_AUTH_JSON:-$HOME/.grok/auth.json}" ]; then
  echo "❌ Error: no live credentials found."
  echo "   Set XAI_API_KEY or run 'grok login' (SuperGrok / X Premium+ OAuth)."
  exit 1
fi

unset GROK_IMAGE_MOCK || true

echo "🚀 Building server binary..."
go build -o grok-image-mcp .

VERSION=$(./grok-image-mcp --version)
if [ -z "$TAG" ]; then
  TAG="v${VERSION}"
fi

OUT_DIR="assets/releases/${TAG}"
mkdir -p "$OUT_DIR"

detect_auth_mode() {
  local status
  status=$(printf '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_configuration_status","arguments":{}},"id":1}' | ./grok-image-mcp 2>/dev/null)
  if echo "$status" | grep -q 'Grok subscription OAuth is active'; then
    echo "grok_oauth"
  elif [ -n "${XAI_API_KEY:-}" ]; then
    echo "api_key"
  else
    echo "config_file"
  fi
}

ext_for() {
  local path="$1"
  case "${path##*.}" in
    jpg|jpeg|png|webp) echo ".${path##*.}" ;;
    *) echo ".jpg" ;;
  esac
}

copy_proof() {
  local generated_src="$1"
  local edited_src="$2"
  local gen_ext edited_ext
  gen_ext=$(ext_for "$generated_src")
  edited_ext=$(ext_for "$edited_src")

  cp "$generated_src" "$OUT_DIR/generated${gen_ext}"
  cp "$edited_src" "$OUT_DIR/edited${edited_ext}"

  cat > "$OUT_DIR/manifest.json" <<EOF
{
  "tag": "${TAG}",
  "version": "${VERSION}",
  "generated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "platform": "$(uname -s)/$(uname -m)",
  "auth_mode": "$(detect_auth_mode)",
  "files": {
    "generated": "generated${gen_ext}",
    "edited": "edited${edited_ext}"
  }
}
EOF

  echo "✅ Release proof saved to ${OUT_DIR}/"
  echo "   - generated${gen_ext}"
  echo "   - edited${edited_ext}"
  echo "   - manifest.json"
}

if [ "$REUSE_LATEST" -eq 1 ]; then
  GENERATED=$(ls -t generated_imgs/generated-* 2>/dev/null | head -1 || true)
  EDITED=$(ls -t generated_imgs/edited-* 2>/dev/null | head -1 || true)
  if [ -z "$GENERATED" ] || [ -z "$EDITED" ]; then
    echo "❌ No recent generated/edited images in generated_imgs/ — run ./scripts/test_all.sh first"
    exit 1
  fi
  copy_proof "$GENERATED" "$EDITED"
  exit 0
fi

AUTH_MODE=$(detect_auth_mode)
GEN_PROMPT="Release validation image for grok-image-mcp ${TAG}: a luminous picture frame with flowing electric-blue light trails on deep black, clean minimal vector style, premium tech branding visual, no text or letters"
EDIT_PROMPT="Add a subtle soft violet outer glow around the frame while keeping the same composition"

echo "🎨 Generating release proof for ${TAG} (${AUTH_MODE})..."
SESSION_OUTPUT=$(./grok-image-mcp 2>/dev/null <<EOF
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"generate_image","arguments":{"prompt":"${GEN_PROMPT}","model":"grok-imagine-image","aspectRatio":"1:1","resolution":"1k"}},"id":10}
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"continue_editing","arguments":{"prompt":"${EDIT_PROMPT}","model":"grok-imagine-image"}},"id":11}
EOF
)

GEN_LINE=$(echo "$SESSION_OUTPUT" | sed -n '1p')
EDIT_LINE=$(echo "$SESSION_OUTPUT" | sed -n '2p')

if echo "$GEN_LINE" | grep -q '"error"'; then
  echo "❌ generate_image failed:"
  echo "$GEN_LINE"
  exit 1
fi
if echo "$EDIT_LINE" | grep -q '"error"'; then
  echo "❌ continue_editing failed:"
  echo "$EDIT_LINE"
  exit 1
fi

GENERATED=$(ls -t generated_imgs/generated-* 2>/dev/null | head -1)
EDITED=$(ls -t generated_imgs/edited-* 2>/dev/null | head -1)
copy_proof "$GENERATED" "$EDITED"