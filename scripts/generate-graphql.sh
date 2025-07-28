#!/bin/bash

# Script to generate GraphQL code based on build edition
# Usage: ./scripts/generate-graphql.sh [community|ee]

EDITION=${1:-community}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "Generating GraphQL code for $EDITION edition..."

if [ "$EDITION" = "ee" ]; then
    # For EE, first run the merge-schema.sh script to prepare the merged schema
    echo "Merging schema for EE mode..."
    "$SCRIPT_DIR/merge-schema.sh" "ee"
    if [ $? -ne 0 ]; then
        echo "Schema merge failed"
        exit 1
    fi
    
    # Generate EE code in ee directory using the merged schema
    cd ee
    echo "Generating EE GraphQL code using merged schema..."
    go run github.com/99designs/gqlgen generate --config gqlgen.ee.yml
    RESULT=$?
    
    # Clean up the merged schema file after generation
    rm -f "$PROJECT_ROOT/core/graph/schema.merged.graphqls"
else
    # For CE, just use the standard generation with the original schema
    cd core
    echo "Generating CE GraphQL code..."
    # Make sure no merged schema file exists to avoid conflicts
    rm -f graph/schema.merged.graphqls
    go run github.com/99designs/gqlgen generate
    RESULT=$?
fi

if [ $RESULT -eq 0 ]; then
    echo "GraphQL code generation completed successfully"
else
    echo "GraphQL code generation failed"
    exit $RESULT
fi