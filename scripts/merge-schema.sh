#!/bin/bash

# Script to merge GraphQL schemas for CE and EE modes
# Usage: ./merge-schema.sh [ce|ee]

set -e

# Check if argument is provided
if [ $# -ne 1 ]; then
    echo "Usage: $0 [ce|ee]"
    echo "  ce - Community Edition (core schema only)"
    echo "  ee - Enterprise Edition (core + EE extensions)"
    exit 1
fi

MODE=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CORE_SCHEMA="$PROJECT_ROOT/core/graph/schema.graphqls"
EE_EXTENSION="$PROJECT_ROOT/ee/core/graph/schema.extension.graphqls"
OUTPUT_SCHEMA="$PROJECT_ROOT/core/graph/schema.merged.graphqls"

# Validate mode
if [ "$MODE" != "ce" ] && [ "$MODE" != "ee" ]; then
    echo "Error: Invalid mode. Use 'ce' or 'ee'"
    exit 1
fi

# Check if core schema exists
if [ ! -f "$CORE_SCHEMA" ]; then
    echo "Error: Core schema not found at $CORE_SCHEMA"
    exit 1
fi

# For CE mode, just copy the core schema
if [ "$MODE" = "ce" ]; then
    echo "Building Community Edition schema..."
    cp "$CORE_SCHEMA" "$OUTPUT_SCHEMA"
    echo "CE schema written to $OUTPUT_SCHEMA"
    exit 0
fi

# For EE mode, merge the schemas
echo "Building Enterprise Edition schema..."

# Check if EE extension exists
if [ ! -f "$EE_EXTENSION" ]; then
    echo "Error: EE extension not found at $EE_EXTENSION"
    exit 1
fi

# Create a temporary file for the merged schema
TEMP_FILE=$(mktemp)

# Process the core schema and insert EE types into the DatabaseType enum
awk '
BEGIN {
    in_db_enum = 0
    enum_lines_count = 0
}
/^enum DatabaseType/ {
    in_db_enum = 1
    print
    next
}
in_db_enum && /^}/ {
    # Add comma to last core value and add EE types
    for (i = 1; i <= enum_lines_count; i++) {
        if (i == enum_lines_count) {
            # Last core enum value - add comma
            gsub(/,$/, "", enum_lines[i])  # Remove any existing comma
            print enum_lines[i] ","
        } else {
            print enum_lines[i]
        }
    }
    # Add EE types
    print "  MSSQL,"
    print "  DynamoDB,"
    print "  Oracle"
    in_db_enum = 0
    print
    next
}
in_db_enum {
    enum_lines_count++
    enum_lines[enum_lines_count] = $0
    next
}
{
    print
}
' "$CORE_SCHEMA" > "$TEMP_FILE"

# Move the temporary file to the output location
mv "$TEMP_FILE" "$OUTPUT_SCHEMA"

echo "EE schema written to $OUTPUT_SCHEMA"
echo "Added enterprise database types: MSSQL, DynamoDB, Oracle"