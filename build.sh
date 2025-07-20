#!/bin/bash

# Simple build script - defaults to CE, use --ee for Enterprise

set -e

if [[ "$1" == "--ee" ]]; then
    echo "🏢 Building WhoDB Enterprise Edition..."
    
    # Validate EE requirements first
    if [ -f "./scripts/validate-ee.sh" ]; then
        ./scripts/validate-ee.sh
    else
        # Basic check if scripts don't exist
        if [ ! -d "./ee" ]; then
            echo "❌ Error: Enterprise Edition directory 'ee' not found"
            exit 1
        fi
    fi
    
    cd core && go build -tags ee -o whodb-ee
    echo "✅ Built: core/whodb-ee"
else
    echo "🚀 Building WhoDB Community Edition..."
    cd core && go build -o whodb
    echo "✅ Built: core/whodb"
fi