#!/bin/bash

# WhoDB Backend Build Script
# Builds the Go backend for either CE or EE edition
# Usage: ./scripts/build-backend.sh [ce|ee]

set -e

# Parse arguments
EDITION=${1:-ce}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Validate edition
if [[ "$EDITION" != "ce" && "$EDITION" != "ee" ]]; then
    echo "❌ Error: Invalid edition '$EDITION'. Must be 'ce' or 'ee'"
    exit 1
fi

echo "🔨 Building WhoDB backend ($EDITION edition)..."

# Step 1: Generate GraphQL code
echo "📊 Generating GraphQL code..."
if [ "$EDITION" = "ee" ]; then
    echo "   Generating EE GraphQL code with native schema extension..."
    cd "$PROJECT_ROOT/ee"
    GOWORK="$PROJECT_ROOT/go.work.ee" go generate .
    if [ $? -ne 0 ]; then
        echo "❌ EE GraphQL generation failed"
        exit 1
    fi
    cd "$PROJECT_ROOT/core"
else
    echo "   Generating CE GraphQL code..."
    cd "$PROJECT_ROOT/core"
    go generate ./...
    if [ $? -ne 0 ]; then
        echo "❌ CE GraphQL generation failed"
        exit 1
    fi
fi

# Step 2: Download Go dependencies
echo "📦 Downloading Go dependencies..."
cd "$PROJECT_ROOT/core"
go mod download
if [ $? -ne 0 ]; then
    echo "❌ Failed to download Go dependencies"
    exit 1
fi

# For EE, also handle EE module dependencies
if [ "$EDITION" = "ee" ]; then
    if [ -d "$PROJECT_ROOT/ee" ]; then
        echo "📦 Downloading EE Go dependencies..."
        cd "$PROJECT_ROOT/ee"
        go mod download
        if [ $? -ne 0 ]; then
            echo "❌ Failed to download EE Go dependencies"
            exit 1
        fi
        cd "$PROJECT_ROOT/core"
    fi
fi

# Step 3: Run go generate if needed
if grep -r "//go:generate" . --include="*.go" > /dev/null 2>&1; then
    echo "🔧 Running go generate..."
    go generate ./...
fi

# Step 4: Build the binary
echo "🏗️  Compiling backend..."
BUILD_FLAGS=""
OUTPUT_NAME="whodb"

# Set build flags and output name based on edition
if [ "$EDITION" = "ee" ]; then
    BUILD_FLAGS="-tags ee"
    OUTPUT_NAME="whodb-ee"
fi

# Add version information if git is available
if command -v git &> /dev/null && [ -d "$PROJECT_ROOT/.git" ]; then
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    LDFLAGS="-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME"
    BUILD_FLAGS="$BUILD_FLAGS -ldflags \"$LDFLAGS\""
fi

# Build the binary
# For EE builds, use go.work.ee
if [ "$EDITION" = "ee" ]; then
    echo "   Using go.work.ee for EE build"
    eval GOWORK="$PROJECT_ROOT/go.work.ee" go build $BUILD_FLAGS -o "$OUTPUT_NAME"
else
    eval go build $BUILD_FLAGS -o "$OUTPUT_NAME"
fi

if [ $? -eq 0 ]; then
    echo "✅ Backend built successfully: core/$OUTPUT_NAME"
    
    # Make the binary executable
    chmod +x "$OUTPUT_NAME"
    
    # Show binary info
    echo ""
    echo "📋 Binary information:"
    echo "   Path: $PROJECT_ROOT/core/$OUTPUT_NAME"
    echo "   Size: $(du -h "$OUTPUT_NAME" | cut -f1)"
    if [ "$EDITION" = "ee" ]; then
        echo "   Edition: Enterprise"
        echo "   EE Features: ✓ Enabled"
    else
        echo "   Edition: Community"
    fi
    
    # Verify the binary works
    echo ""
    echo "🔍 Verifying binary..."
    if ./"$OUTPUT_NAME" --version > /dev/null 2>&1; then
        echo "✅ Binary verification passed"
    else
        echo "⚠️  Warning: Binary built but --version flag not implemented"
    fi
else
    echo "❌ Backend build failed"
    exit 1
fi