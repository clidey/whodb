#!/bin/bash

# Test script for merge-schema.sh
# This script tests both CE and EE modes and verifies the output

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MERGE_SCRIPT="$SCRIPT_DIR/merge-schema.sh"
OUTPUT_SCHEMA="$PROJECT_ROOT/core/graph/schema.merged.graphqls"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Testing merge-schema.sh...${NC}"
echo "========================================"

# Test 1: CE mode
echo -e "\n${YELLOW}Test 1: Community Edition mode${NC}"
$MERGE_SCRIPT ce

if [ -f "$OUTPUT_SCHEMA" ]; then
    echo -e "${GREEN}✓ Output file created${NC}"
    
    # Check that it doesn't contain EE types
    if grep -q "MSSQL\|DynamoDB\|Oracle" "$OUTPUT_SCHEMA"; then
        echo -e "${RED}✗ CE schema contains EE types (should not)${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ CE schema does not contain EE types${NC}"
    fi
    
    # Check that it contains core types
    if grep -q "Postgres" "$OUTPUT_SCHEMA" && grep -q "MySQL" "$OUTPUT_SCHEMA"; then
        echo -e "${GREEN}✓ CE schema contains core database types${NC}"
    else
        echo -e "${RED}✗ CE schema missing core database types${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Output file not created${NC}"
    exit 1
fi

# Test 2: EE mode
echo -e "\n${YELLOW}Test 2: Enterprise Edition mode${NC}"
$MERGE_SCRIPT ee

if [ -f "$OUTPUT_SCHEMA" ]; then
    echo -e "${GREEN}✓ Output file created${NC}"
    
    # Check that it contains EE types
    if grep -q "MSSQL" "$OUTPUT_SCHEMA" && grep -q "DynamoDB" "$OUTPUT_SCHEMA" && grep -q "Oracle" "$OUTPUT_SCHEMA"; then
        echo -e "${GREEN}✓ EE schema contains enterprise database types${NC}"
    else
        echo -e "${RED}✗ EE schema missing enterprise database types${NC}"
        exit 1
    fi
    
    # Check that it still contains core types
    if grep -q "Postgres" "$OUTPUT_SCHEMA" && grep -q "MySQL" "$OUTPUT_SCHEMA"; then
        echo -e "${GREEN}✓ EE schema contains core database types${NC}"
    else
        echo -e "${RED}✗ EE schema missing core database types${NC}"
        exit 1
    fi
    
    # Verify the enum structure is valid
    echo -e "\n${YELLOW}Checking DatabaseType enum structure:${NC}"
    awk '/^enum DatabaseType/,/^}/' "$OUTPUT_SCHEMA" | grep -E "^\s+\w+" | while read -r line; do
        echo "  - $line"
    done
    
    # Check for proper comma placement
    if awk '/^enum DatabaseType/,/^}/' "$OUTPUT_SCHEMA" | grep -E "^\s+\w+\s*$" | tail -n 1 | grep -q ","; then
        echo -e "${RED}✗ Last enum value has trailing comma${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ Enum formatting is correct (no trailing comma)${NC}"
    fi
    
else
    echo -e "${RED}✗ Output file not created${NC}"
    exit 1
fi

# Test 3: Invalid mode
echo -e "\n${YELLOW}Test 3: Invalid mode handling${NC}"
if $MERGE_SCRIPT invalid 2>/dev/null; then
    echo -e "${RED}✗ Script should fail with invalid mode${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Script correctly rejects invalid mode${NC}"
fi

# Test 4: No arguments
echo -e "\n${YELLOW}Test 4: No arguments handling${NC}"
if $MERGE_SCRIPT 2>/dev/null; then
    echo -e "${RED}✗ Script should fail without arguments${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Script correctly requires arguments${NC}"
fi

echo -e "\n${GREEN}========================================"
echo -e "All tests passed! ✓${NC}"

# Show the final merged EE schema enum
echo -e "\n${YELLOW}Final merged DatabaseType enum:${NC}"
awk '/^enum DatabaseType/,/^}/' "$OUTPUT_SCHEMA"