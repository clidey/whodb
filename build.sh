#!/bin/bash

# WhoDB Build Script - Comprehensive build system for CE and EE editions
# Usage: ./build.sh [options]
# Options:
#   --ee              Build Enterprise Edition
#   --frontend-only   Build only frontend
#   --backend-only    Build only backend
#   --skip-validate   Skip EE validation (for EE builds)
#   --clean           Clean build artifacts before building
#   --help            Show this help message

set -e

# Parse command line arguments
EDITION="ce"
BUILD_FRONTEND=true
BUILD_BACKEND=true
SKIP_VALIDATE=false
CLEAN_BUILD=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --ee)
            EDITION="ee"
            shift
            ;;
        --frontend-only)
            BUILD_BACKEND=false
            shift
            ;;
        --backend-only)
            BUILD_FRONTEND=false
            shift
            ;;
        --skip-validate)
            SKIP_VALIDATE=true
            shift
            ;;
        --clean)
            CLEAN_BUILD=true
            shift
            ;;
        --help)
            echo "WhoDB Build Script"
            echo "Usage: ./build.sh [options]"
            echo ""
            echo "Options:"
            echo "  --ee              Build Enterprise Edition (default: Community Edition)"
            echo "  --frontend-only   Build only frontend"
            echo "  --backend-only    Build only backend"
            echo "  --skip-validate   Skip EE validation (for EE builds)"
            echo "  --clean           Clean build artifacts before building"
            echo "  --help            Show this help message"
            echo ""
            echo "Examples:"
            echo "  ./build.sh                    # Build CE (both frontend and backend)"
            echo "  ./build.sh --ee               # Build EE (both frontend and backend)"
            echo "  ./build.sh --ee --backend-only # Build only EE backend"
            echo "  ./build.sh --clean --ee       # Clean build of EE"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Set up paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"

# Clean build artifacts if requested
if [ "$CLEAN_BUILD" = true ]; then
    echo "🧹 Cleaning build artifacts..."
    rm -f "$PROJECT_ROOT/core/whodb"
    rm -f "$PROJECT_ROOT/core/whodb-ee"
    rm -rf "$PROJECT_ROOT/core/build"
    rm -rf "$PROJECT_ROOT/frontend/dist"
    rm -rf "$PROJECT_ROOT/frontend/build"
    rm -rf "$PROJECT_ROOT/frontend/node_modules/.vite"
    echo "✅ Clean complete"
fi

# Determine edition name for display
if [ "$EDITION" = "ee" ]; then
    EDITION_NAME="Enterprise Edition"
else
    EDITION_NAME="Community Edition"
fi

echo "🚀 Building WhoDB $EDITION_NAME"
echo "   Frontend: $BUILD_FRONTEND"
echo "   Backend: $BUILD_BACKEND"
echo ""


# Validate EE requirements if building EE
if [ "$EDITION" = "ee" ] && [ "$SKIP_VALIDATE" = false ]; then
    if [ -f "$PROJECT_ROOT/scripts/validate-ee.sh" ]; then
        "$PROJECT_ROOT/scripts/validate-ee.sh"
        if [ $? -ne 0 ]; then
            echo "❌ EE validation failed. Use --skip-validate to bypass (not recommended)"
            exit 1
        fi
    else
        echo "⚠️  Warning: validate-ee.sh not found, skipping validation"
    fi
fi

# Build frontend
if [ "$BUILD_FRONTEND" = true ]; then
    echo "🔨 Building frontend..."
    if [ -f "$PROJECT_ROOT/scripts/build-frontend.sh" ]; then
        "$PROJECT_ROOT/scripts/build-frontend.sh" "$EDITION"
    else
        # Fallback to inline build if script doesn't exist
        echo "⚠️  build-frontend.sh not found, using fallback build process"
        
        cd "$PROJECT_ROOT/frontend"
        
        # Install dependencies if needed
        if [ ! -d "node_modules" ]; then
            echo "📦 Installing frontend dependencies..."
            pnpm install
        fi
        
        # Build frontend
        if [ "$EDITION" = "ee" ]; then
            pnpm run build:ee
        else
            pnpm run build
        fi
        
        echo "✅ Built: frontend/dist"
        
        # Copy frontend build to core for embedding
        echo "📋 Copying frontend build to core/build..."
        rm -rf "$PROJECT_ROOT/core/build"
        cp -r "$PROJECT_ROOT/frontend/build" "$PROJECT_ROOT/core/build"
        echo "✅ Frontend copied to core/build"
        
        cd "$PROJECT_ROOT"
    fi
fi

# Build backend
if [ "$BUILD_BACKEND" = true ]; then
    echo "🔨 Building backend..."
    if [ -f "$PROJECT_ROOT/scripts/build-backend.sh" ]; then
        "$PROJECT_ROOT/scripts/build-backend.sh" "$EDITION"
    else
        # Fallback to inline build if script doesn't exist
        echo "⚠️  build-backend.sh not found, using fallback build process"
        
        # Generate GraphQL code
        "$PROJECT_ROOT/scripts/generate-graphql.sh" "$EDITION"
        
        # Build backend
        cd "$PROJECT_ROOT/core"
        if [ "$EDITION" = "ee" ]; then
            # Use go.work.ee for EE builds
            GOWORK="$PROJECT_ROOT/go.work.ee" go build -tags ee -o whodb-ee
            echo "✅ Built: core/whodb-ee"
        else
            go build -o whodb
            echo "✅ Built: core/whodb"
        fi
        cd "$PROJECT_ROOT"
    fi
fi

echo ""
echo "✅ Build complete!"
echo ""

# Provide next steps based on what was built
if [ "$BUILD_BACKEND" = true ] && [ "$BUILD_FRONTEND" = true ]; then
    if [ "$EDITION" = "ee" ]; then
        echo "To run WhoDB Enterprise Edition:"
        echo "  cd core && ./whodb-ee"
    else
        echo "To run WhoDB Community Edition:"
        echo "  cd core && ./whodb"
    fi
elif [ "$BUILD_BACKEND" = true ]; then
    if [ "$EDITION" = "ee" ]; then
        echo "Backend built: core/whodb-ee"
    else
        echo "Backend built: core/whodb"
    fi
elif [ "$BUILD_FRONTEND" = true ]; then
    echo "Frontend built: frontend/dist"
    echo "Note: You'll need to run the backend separately to serve the frontend"
fi