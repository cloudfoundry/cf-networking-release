#!/bin/bash

set -e

# Simple script to generate OpenAPI 3.1 specification for CF Networking API

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
POLICY_SERVER_DIR="$PROJECT_ROOT/src/code.cloudfoundry.org/policy-server"
OUTPUT_DIR="$PROJECT_ROOT/docs/swagger"

echo "🚀 Generating OpenAPI 3.1 specification..."

# Install swag v2 if not present
GOPATH=$(go env GOPATH)
SWAG_CMD="$GOPATH/bin/swag"

if ! [ -f "$SWAG_CMD" ] || ! $SWAG_CMD --version 2>/dev/null | grep -q "v2\."; then
    echo "📦 Installing swag v2..."
    go install github.com/swaggo/swag/v2/cmd/swag@latest
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Generate OpenAPI 3.1 specification
cd "$POLICY_SERVER_DIR"
$SWAG_CMD init \
    --dir ./ \
    --generalInfo cmd/policy-server/main.go \
    --output "$OUTPUT_DIR" \
    --outputTypes json,yaml \
    --parseDependency \
    --parseInternal \
    --v3.1

# Rename to openapi.*
mv "$OUTPUT_DIR/swagger.yaml" "$OUTPUT_DIR/openapi.yaml"
mv "$OUTPUT_DIR/swagger.json" "$OUTPUT_DIR/openapi.json"

echo "✅ Generated:"
echo "   • $OUTPUT_DIR/openapi.yaml"
echo "   • $OUTPUT_DIR/openapi.json"
