#!/bin/bash

# WhoDB Enterprise Edition Build Validation Script
# This script validates that EE modules are available before building

set -e

echo "🔍 Validating Enterprise Edition build requirements..."

# Check if EE directory exists
if [ ! -d "./ee" ]; then
    echo "❌ Error: Enterprise Edition directory './ee' not found"
    echo ""
    echo "The Enterprise Edition requires additional modules that are not included"
    echo "in the open-source distribution. To build the Enterprise Edition:"
    echo ""
    echo "1. Ensure you have access to the EE modules"
    echo "2. Place the 'ee' directory in the project root"
    echo "3. Run this script again"
    echo ""
    echo "For more information, see: https://github.com/clidey/whodb/blob/main/ee/README.md"
    exit 1
fi

# Check if EE backend modules exist
if [ ! -d "./ee/core/src/plugins" ]; then
    echo "❌ Error: EE backend plugins not found at './ee/core/src/plugins'"
    echo "   The EE directory structure appears to be incomplete"
    exit 1
fi

# Check if EE frontend modules exist
if [ ! -d "./ee/frontend/src" ]; then
    echo "❌ Error: EE frontend modules not found at './ee/frontend/src'"
    echo "   The EE directory structure appears to be incomplete"
    exit 1
fi

# Check for required EE plugins
echo "✓ Checking for EE database plugins..."
required_plugins=("dynamodb" "mssql" "oracle")
for plugin in "${required_plugins[@]}"; do
    if [ ! -d "./ee/core/src/plugins/$plugin" ]; then
        echo "❌ Error: Required EE plugin '$plugin' not found"
        exit 1
    fi
    echo "  ✓ Found $plugin plugin"
done

# Check for EE go.mod
if [ ! -f "./ee/go.mod" ]; then
    echo "❌ Error: EE go.mod not found at './ee/go.mod'"
    exit 1
fi

echo "✓ EE go.mod found"

# Check for EE frontend components
echo "✓ Checking for EE frontend components..."
if [ ! -d "./ee/frontend/src/components/charts" ]; then
    echo "❌ Error: EE charts components not found"
    exit 1
fi
echo "  ✓ Found charts components"

if [ ! -d "./ee/frontend/src/components/theme" ]; then
    echo "❌ Error: EE theme components not found"
    exit 1
fi
echo "  ✓ Found theme components"

echo ""
echo "✅ All Enterprise Edition requirements validated!"
echo "   You can now build the Enterprise Edition with:"
echo "   - make build-ee"
echo "   - ./build.sh --ee"
echo ""