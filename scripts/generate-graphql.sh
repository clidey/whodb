#!/bin/bash

# Script to generate GraphQL code based on build edition
# Usage: ./scripts/generate-graphql.sh [community|ee]

EDITION=${1:-community}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "Generating GraphQL code for $EDITION edition..."

if [ "$EDITION" = "ee" ]; then
    # Merge schemas for EE build
    ./scripts/merge-schema.sh ee
    
    # Generate code using EE configuration
    cd core
    go run github.com/99designs/gqlgen generate --config gqlgen.ee.yml
    RESULT=$?
    
    # Clean up merged schema after generation (optional)
    # rm -f graph/schema.merged.graphqls
else
    # Generate code using standard configuration
    cd core
    go run github.com/99designs/gqlgen generate
    RESULT=$?
fi

if [ $RESULT -eq 0 ]; then
    echo "GraphQL code generation completed successfully"
else
    echo "GraphQL code generation failed"
    exit $RESULT
fi