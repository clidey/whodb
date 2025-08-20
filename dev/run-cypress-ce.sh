#!/bin/bash
# Wrapper script to ensure proper cleanup of frontend and backend on any exit

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HEADLESS="${1:-false}"

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up test environment..."
    
    # Kill frontend gracefully first
    echo "   Stopping frontend..."
    pkill -TERM -f 'vite --port 3000' 2>/dev/null || true
    sleep 1
    
    # Run cleanup script
    echo "   Running cleanup script..."
    bash "$SCRIPT_DIR/cleanup-e2e.sh"
    
    # Kill anything still on port 3000 (just in case)
    echo "   Ensuring port 3000 is free..."
    if lsof -ti:3000 >/dev/null 2>&1; then
        lsof -ti:3000 | xargs kill -9 2>/dev/null || true
    fi
    
    echo "✅ Cleanup complete"
}

# Set trap to cleanup on any exit
trap 'cleanup; exit ${EXIT_CODE:-0}' EXIT INT TERM

# Main test execution with error handling
run_tests() {
    # Setup backend
    echo "🚀 Setting up CE test environment..."
    bash "$SCRIPT_DIR/setup-e2e.sh" || { echo "❌ Backend setup failed"; return 1; }

    # Start frontend
    echo "🚀 Starting frontend..."
    cd "$PROJECT_ROOT/frontend"
    NODE_ENV=test vite --port 3000 &
    FRONTEND_PID=$!

    # Wait for services
    echo "⏳ Waiting for services..."
    bash "$SCRIPT_DIR/wait-for-services.sh" || { echo "❌ Services failed to start"; return 1; }

    # Run Cypress
    echo "🧪 Running Cypress tests..."
    cd "$PROJECT_ROOT/frontend"
    if [ "$HEADLESS" = "true" ]; then
        NODE_ENV=test npx cypress run || { echo "❌ Cypress tests failed"; return 1; }
    else
        NODE_ENV=test pnpm cypress open || { echo "❌ Cypress failed to open"; return 1; }
    fi

    echo "✅ Test run complete"
    return 0
}

# Execute tests and capture exit code
run_tests
EXIT_CODE=$?

# The trap will handle cleanup and exit with the correct code