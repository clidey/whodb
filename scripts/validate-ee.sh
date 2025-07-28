#!/bin/bash

# WhoDB Enterprise Edition Build Validation Script
# This script validates that EE modules are available before building

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🔍 Validating Enterprise Edition build requirements..."

# Check if EE directory exists
if [ ! -d "$PROJECT_ROOT/ee" ]; then
    echo "❌ Error: Enterprise Edition directory 'ee' not found"
    echo ""
    echo "The Enterprise Edition requires additional modules that are not included"
    echo "in the open-source distribution. To build the Enterprise Edition:"
    echo ""
    echo "1. Ensure you have access to the EE modules"
    echo "2. Place the 'ee' directory in the project root: $PROJECT_ROOT/ee"
    echo "3. Run this script again"
    echo ""
    echo "For more information, see: https://github.com/clidey/whodb/blob/main/ee/README.md"
    exit 1
fi

# Generic check that EE directory has content
if [ -z "$(ls -A "$PROJECT_ROOT/ee" 2>/dev/null)" ]; then
    echo "❌ Error: Enterprise Edition directory is empty"
    exit 1
fi

# Check for EE go.mod
if [ ! -f "$PROJECT_ROOT/ee/go.mod" ]; then
    echo "❌ Error: Enterprise Edition appears to be incomplete"
    echo "   Missing required module files"
    exit 1
fi

echo "✓ Enterprise Edition modules found"

# Check for required tools
echo "✓ Checking for required build tools..."
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed"
    echo "   Install Go from: https://golang.org/dl/"
    exit 1
fi
echo "  ✓ Go $(go version | awk '{print $3}')"

if ! command -v pnpm &> /dev/null; then
    echo "❌ Error: pnpm is not installed"
    echo "   Install pnpm with: npm install -g pnpm"
    exit 1
fi
echo "  ✓ pnpm $(pnpm --version)"

if ! command -v node &> /dev/null; then
    echo "❌ Error: Node.js is not installed"
    echo "   Install Node.js from: https://nodejs.org/"
    exit 1
fi
echo "  ✓ Node.js $(node --version)"

echo ""
echo "✅ All Enterprise Edition requirements validated!"
echo "   You can now build the Enterprise Edition with:"
echo "   - ./build.sh --ee"
echo ""