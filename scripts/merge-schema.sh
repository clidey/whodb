#!/bin/bash

# Script to merge GraphQL schemas based on build edition
# Usage: ./scripts/merge-schema.sh [community|ee]

EDITION=${1:-community}
CORE_SCHEMA="core/graph/schema.graphqls"
MERGED_SCHEMA="core/graph/schema.merged.graphqls"
EE_EXTENSION="ee/core/graph/schema.extension.graphqls"

# Start with the core schema
cp "$CORE_SCHEMA" "$MERGED_SCHEMA"

if [ "$EDITION" = "ee" ]; then
    echo "Building Enterprise Edition schema..."
    
    # Check if extension file exists
    if [ -f "$EE_EXTENSION" ]; then
        # Extract the DatabaseType enum from core schema
        ENUM_START=$(grep -n "enum DatabaseType {" "$CORE_SCHEMA" | cut -d: -f1)
        ENUM_END=$(awk -v start="$ENUM_START" 'NR > start && /^}/ {print NR; exit}' "$CORE_SCHEMA")
        
        # Extract enterprise database types from extension
        EE_TYPES=$(sed -n '/extend enum DatabaseType {/,/}/p' "$EE_EXTENSION" | grep -E '^\s+[A-Z]' | sed 's/^[ \t]*/  /')
        
        # Insert EE types before the closing brace of DatabaseType enum
        if [ -n "$EE_TYPES" ]; then
            # Create a temporary file with the modified enum
            head -n $((ENUM_END - 1)) "$MERGED_SCHEMA" > "$MERGED_SCHEMA.tmp"
            echo "$EE_TYPES" >> "$MERGED_SCHEMA.tmp"
            tail -n +$ENUM_END "$MERGED_SCHEMA" >> "$MERGED_SCHEMA.tmp"
            mv "$MERGED_SCHEMA.tmp" "$MERGED_SCHEMA"
            
            echo "Enterprise database types added to schema"
        fi
    else
        echo "Warning: EE extension file not found at $EE_EXTENSION"
    fi
else
    echo "Building Community Edition schema..."
fi

echo "Schema merged to: $MERGED_SCHEMA"