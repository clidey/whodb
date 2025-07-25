#!/bin/bash

# WhoDB Development Mode Helper
# Quickly switch between CE and EE development modes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Parse arguments
MODE=${1:-status}

show_help() {
    echo "WhoDB Development Mode Helper"
    echo ""
    echo "Usage: ./scripts/dev-mode.sh [command]"
    echo ""
    echo "Commands:"
    echo "  ce       - Switch to Community Edition development"
    echo "  ee       - Switch to Enterprise Edition development"
    echo "  status   - Show current development mode (default)"
    echo "  help     - Show this help message"
    echo ""
    echo "This script helps you quickly switch between CE and EE development"
    echo "by setting up the appropriate environment and starting dev servers."
}

show_status() {
    echo "🔍 Checking development environment..."
    echo ""
    
    # Check for running backend
    if pgrep -f "whodb" > /dev/null; then
        echo "✓ Backend is running"
        if pgrep -f "whodb-ee" > /dev/null; then
            echo "  Edition: Enterprise"
        else
            echo "  Edition: Community"
        fi
    else
        echo "✗ Backend is not running"
    fi
    
    # Check for running frontend dev server
    if lsof -i :3000 > /dev/null 2>&1; then
        echo "✓ Frontend dev server is running on port 3000"
    else
        echo "✗ Frontend dev server is not running"
    fi
    
    # Check which binaries exist
    echo ""
    echo "Available binaries:"
    if [ -f "$PROJECT_ROOT/core/whodb" ]; then
        echo "  ✓ core/whodb (CE)"
    fi
    if [ -f "$PROJECT_ROOT/core/whodb-ee" ]; then
        echo "  ✓ core/whodb-ee (EE)"
    fi
    
    # Check frontend build
    if [ -d "$PROJECT_ROOT/frontend/dist" ]; then
        echo "  ✓ frontend/dist (production build exists)"
    fi
}

start_ce_dev() {
    echo "🚀 Starting Community Edition development environment..."
    echo ""
    
    # Kill any existing processes
    echo "Stopping any existing services..."
    pkill -f "whodb" || true
    
    # Build CE backend if needed
    if [ ! -f "$PROJECT_ROOT/core/whodb" ]; then
        echo "Building CE backend..."
        "$PROJECT_ROOT/build.sh" --backend-only
    fi
    
    # Start backend
    echo "Starting CE backend..."
    cd "$PROJECT_ROOT/core"
    ./whodb &
    BACKEND_PID=$!
    echo "Backend started with PID: $BACKEND_PID"
    
    # Start frontend dev server
    echo ""
    echo "Starting CE frontend dev server..."
    cd "$PROJECT_ROOT/frontend"
    pnpm start
}

start_ee_dev() {
    echo "🏢 Starting Enterprise Edition development environment..."
    echo ""
    
    # Validate EE first
    if ! "$PROJECT_ROOT/scripts/validate-ee.sh" > /dev/null 2>&1; then
        echo "❌ EE validation failed. Running validation..."
        "$PROJECT_ROOT/scripts/validate-ee.sh"
        exit 1
    fi
    
    # Kill any existing processes
    echo "Stopping any existing services..."
    pkill -f "whodb" || true
    
    # Build EE backend if needed
    if [ ! -f "$PROJECT_ROOT/core/whodb-ee" ]; then
        echo "Building EE backend..."
        "$PROJECT_ROOT/build.sh" --ee --backend-only
    fi
    
    # Start backend
    echo "Starting EE backend..."
    cd "$PROJECT_ROOT/core"
    ./whodb-ee &
    BACKEND_PID=$!
    echo "Backend started with PID: $BACKEND_PID"
    
    # Start frontend dev server
    echo ""
    echo "Starting EE frontend dev server..."
    cd "$PROJECT_ROOT/frontend"
    pnpm start:ee
}

case $MODE in
    ce)
        start_ce_dev
        ;;
    ee)
        start_ee_dev
        ;;
    status)
        show_status
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Unknown command: $MODE"
        echo ""
        show_help
        exit 1
        ;;
esac