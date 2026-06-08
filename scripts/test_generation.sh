#!/bin/bash
# Helper script to test grok-image-mcp image generation

set -e

if [ -z "$XAI_API_KEY" ]; then
  echo "❌ Error: XAI_API_KEY environment variable is not set."
  echo "Please set it before running this script: export XAI_API_KEY='your-key'"
  exit 1
fi

echo "🚀 Compiling server binary..."
go build -o grok-image-mcp .

echo "🎨 Sending generate_image tool request over stdio..."
REQUEST='{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "generate_image", "arguments": {"prompt": "A simple abstract logo with flowing curves, minimal vector style", "model": "grok-imagine-image", "aspectRatio": "1:1"}}, "id": 1}'

RESPONSE=$(echo "$REQUEST" | ./grok-image-mcp)

if echo "$RESPONSE" | grep -q '"error"'; then
  echo "❌ Error returned from server:"
  echo "$RESPONSE" | grep -o '"message":[^,]*' || echo "$RESPONSE"
  exit 1
fi

echo "✅ Success! Image generated successfully."
echo "Response content:"
echo "$RESPONSE" | grep -o '"text":[^,]*' || echo "$RESPONSE"
echo "📁 Check the 'generated_imgs' directory for the saved image file."