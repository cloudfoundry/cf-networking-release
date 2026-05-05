#!/bin/bash

set -e

# Script to generate OpenAPI/Swagger documentation for Policy Server API

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
POLICY_SERVER_DIR="$PROJECT_ROOT/src/code.cloudfoundry.org/policy-server"
OUTPUT_DIR="$PROJECT_ROOT/docs/swagger"

echo "🚀 Generating OpenAPI specification for CF Networking API..."

# Check if swag is installed
SWAG_CMD="swag"
if ! command -v swag &> /dev/null; then
    # Try GOPATH/bin
    GOPATH=$(go env GOPATH)
    if [ -f "$GOPATH/bin/swag" ]; then
        SWAG_CMD="$GOPATH/bin/swag"
        echo "✅ Found swag at $SWAG_CMD"
    else
        echo "❌ swag command not found. Installing swaggo/swag..."
        go install github.com/swaggo/swag/cmd/swag@latest
        SWAG_CMD="$GOPATH/bin/swag"
        echo "✅ swag installed successfully"
    fi
fi

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Change to policy server directory
cd "$POLICY_SERVER_DIR"

echo "📝 Running swag init..."

# Generate swagger docs
$SWAG_CMD init \
    --dir ./ \
    --generalInfo cmd/policy-server/main.go \
    --output "$OUTPUT_DIR" \
    --outputTypes go,json,yaml \
    --parseDependency \
    --parseInternal \
    --parseDepth 2

if [ $? -eq 0 ]; then
    echo "✅ OpenAPI specification generated successfully!"
    echo "📁 Output files:"
    echo "   • YAML: $OUTPUT_DIR/swagger.yaml"
    echo "   • JSON: $OUTPUT_DIR/swagger.json" 
    echo "   • Go:   $OUTPUT_DIR/docs.go"
    echo ""
    echo "🌐 You can view the API documentation by:"
    echo "   1. Opening swagger.yaml in Swagger Editor (https://editor.swagger.io/)"
    echo "   2. Using swagger-ui with the generated files"
    echo "   3. Integrating the docs.go file into your application"
else
    echo "❌ Failed to generate OpenAPI specification"
    exit 1
fi

echo "🎉 Done!"