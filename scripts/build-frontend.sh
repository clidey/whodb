#!/bin/bash

# WhoDB Frontend Build Script
# Builds the React frontend for either CE or EE edition
# Usage: ./scripts/build-frontend.sh [ce|ee]

set -e

# Parse arguments
EDITION=${1:-ce}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FRONTEND_DIR="$PROJECT_ROOT/frontend"

# Validate edition
if [[ "$EDITION" != "ce" && "$EDITION" != "ee" ]]; then
    echo "❌ Error: Invalid edition '$EDITION'. Must be 'ce' or 'ee'"
    exit 1
fi

echo "🔨 Building WhoDB frontend ($EDITION edition)..."

# Change to frontend directory
cd "$FRONTEND_DIR"

# Step 1: Check for package manager
if ! command -v pnpm &> /dev/null; then
    echo "❌ Error: pnpm is not installed"
    echo "   Install pnpm with: npm install -g pnpm"
    exit 1
fi

# Step 2: Install dependencies if needed
if [ ! -d "node_modules" ] || [ ! -f "node_modules/.modules.yaml" ]; then
    echo "📦 Installing frontend dependencies..."
    pnpm install
    if [ $? -ne 0 ]; then
        echo "❌ Failed to install dependencies"
        exit 1
    fi
else
    echo "✓ Dependencies already installed"
fi

# Step 3: Clean previous build artifacts
if [ -d "dist" ]; then
    echo "🧹 Cleaning previous build..."
    rm -rf dist
fi

# Step 4: Set environment variables
export NODE_ENV=production
if [ "$EDITION" = "ee" ]; then
    export VITE_BUILD_EDITION=ee
else
    export VITE_BUILD_EDITION=ce
fi

echo "📝 Build configuration:"
echo "   Edition: $EDITION"
echo "   Node environment: $NODE_ENV"
echo "   Vite build edition: $VITE_BUILD_EDITION"

# Step 5: Generate GraphQL types for frontend
echo "📊 Generating GraphQL types..."
if [ "$EDITION" = "ee" ]; then
    pnpm run generate:ee
else
    pnpm run generate:ce
fi

if [ $? -ne 0 ]; then
    echo "❌ GraphQL type generation failed"
    exit 1
fi

# Step 6: Run TypeScript compiler
echo "🔍 Type checking with TypeScript..."
pnpm exec tsc --noEmit
if [ $? -ne 0 ]; then
    echo "❌ TypeScript type checking failed"
    echo "   Fix the type errors above and try again"
    exit 1
fi

# Step 7: Build the frontend
echo "🏗️  Building frontend assets..."
if [ "$EDITION" = "ee" ]; then
    pnpm run build:ee
else
    pnpm run build
fi

if [ $? -eq 0 ]; then
    echo "✅ Frontend built successfully"
    
    # Show build info
    echo ""
    echo "📋 Build information:"
    echo "   Output directory: $FRONTEND_DIR/dist"
    echo "   Edition: $EDITION"
    
    # Calculate and show build size
    if [ -d "dist" ]; then
        TOTAL_SIZE=$(du -sh dist | cut -f1)
        JS_COUNT=$(find dist -name "*.js" | wc -l | tr -d ' ')
        CSS_COUNT=$(find dist -name "*.css" | wc -l | tr -d ' ')
        
        echo "   Total size: $TOTAL_SIZE"
        echo "   JavaScript files: $JS_COUNT"
        echo "   CSS files: $CSS_COUNT"
        
        # Check for large bundles
        LARGE_FILES=$(find dist -name "*.js" -size +500k)
        if [ ! -z "$LARGE_FILES" ]; then
            echo ""
            echo "⚠️  Warning: Large JavaScript bundles detected (>500KB):"
            echo "$LARGE_FILES" | while read file; do
                SIZE=$(du -h "$file" | cut -f1)
                echo "   - $(basename "$file"): $SIZE"
            done
            echo "   Consider code splitting or lazy loading"
        fi
    fi
    
    # Verify index.html exists
    if [ -f "dist/index.html" ]; then
        echo ""
        echo "✅ Frontend build verification passed"
    else
        echo ""
        echo "❌ Error: dist/index.html not found"
        exit 1
    fi
else
    echo "❌ Frontend build failed"
    exit 1
fi

# Step 8: Copy static assets if any
if [ -d "public" ] && [ "$(ls -A public)" ]; then
    echo "📂 Copying static assets..."
    cp -r public/* dist/ 2>/dev/null || true
fi

echo ""
echo "🎉 Frontend build complete!"
echo ""
echo "The frontend is ready to be served by the WhoDB backend."
echo "Make sure the backend is configured to serve files from: $FRONTEND_DIR/dist"