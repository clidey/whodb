#!/bin/bash

# macOS Code Signing Script for WhoDB
# This script signs and notarizes the macOS app to prevent Gatekeeper warnings
# Prerequisites:
# - Xcode Command Line Tools
# - Valid Apple Developer ID certificate
# - Apple Developer account credentials for notarization

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to display usage
usage() {
    echo "Usage: $0 -a APP_PATH -i IDENTITY [-t TEAM_ID] [-u APPLE_ID] [-p APP_PASSWORD] [-n]"
    echo "  -a APP_PATH      Path to the .app bundle"
    echo "  -i IDENTITY      Developer ID certificate identity (e.g., 'Developer ID Application: Your Name (TEAMID)')"
    echo "  -t TEAM_ID       Apple Developer Team ID (required for notarization)"
    echo "  -u APPLE_ID      Apple ID email (required for notarization)"
    echo "  -p APP_PASSWORD  App-specific password for notarization"
    echo "  -n               Skip notarization (only sign)"
    echo "  -h               Show this help message"
    exit 1
}

# Parse command line arguments
SKIP_NOTARIZATION=false

while getopts "a:i:t:u:p:nh" opt; do
    case $opt in
        a) APP_PATH="$OPTARG" ;;
        i) IDENTITY="$OPTARG" ;;
        t) TEAM_ID="$OPTARG" ;;
        u) APPLE_ID="$OPTARG" ;;
        p) APP_PASSWORD="$OPTARG" ;;
        n) SKIP_NOTARIZATION=true ;;
        h) usage ;;
        *) usage ;;
    esac
done

# Validate required arguments
if [ -z "$APP_PATH" ] || [ -z "$IDENTITY" ]; then
    echo -e "${RED}Error: APP_PATH and IDENTITY are required${NC}"
    usage
fi

# Check if app exists
if [ ! -d "$APP_PATH" ]; then
    echo -e "${RED}Error: App not found at $APP_PATH${NC}"
    exit 1
fi

echo -e "${GREEN}macOS Code Signing Script for WhoDB${NC}"
echo "===================================="
echo -e "${YELLOW}App Path: $APP_PATH${NC}"
echo -e "${YELLOW}Identity: $IDENTITY${NC}"

# Step 1: Sign the app bundle
echo -e "\n${GREEN}Step 1: Signing app bundle...${NC}"

# Sign all frameworks and dylibs first
find "$APP_PATH" -name "*.dylib" -o -name "*.framework" | while read -r item; do
    echo -e "${YELLOW}Signing: $item${NC}"
    codesign --force --timestamp --options runtime --sign "$IDENTITY" "$item"
done

# Sign the main executable
EXECUTABLE_NAME=$(basename "$APP_PATH" .app)
EXECUTABLE_PATH="$APP_PATH/Contents/MacOS/$EXECUTABLE_NAME"

if [ -f "$EXECUTABLE_PATH" ]; then
    echo -e "${YELLOW}Signing main executable: $EXECUTABLE_PATH${NC}"
    codesign --force --timestamp --options runtime --sign "$IDENTITY" "$EXECUTABLE_PATH"
fi

# Sign the entire app bundle
echo -e "${YELLOW}Signing app bundle: $APP_PATH${NC}"
codesign --force --timestamp --options runtime --sign "$IDENTITY" --deep "$APP_PATH"

# Verify the signature
echo -e "\n${GREEN}Verifying signature...${NC}"
codesign --verify --deep --strict --verbose=2 "$APP_PATH"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ App signed successfully${NC}"
else
    echo -e "${RED}✗ Signature verification failed${NC}"
    exit 1
fi

# Step 2: Notarization (if not skipped)
if [ "$SKIP_NOTARIZATION" = false ]; then
    # Validate notarization requirements
    if [ -z "$TEAM_ID" ] || [ -z "$APPLE_ID" ] || [ -z "$APP_PASSWORD" ]; then
        echo -e "${YELLOW}Warning: Skipping notarization - missing credentials${NC}"
        echo "To notarize, provide TEAM_ID, APPLE_ID, and APP_PASSWORD"
        exit 0
    fi
    
    echo -e "\n${GREEN}Step 2: Notarizing app...${NC}"
    
    # Create a ZIP file for notarization
    ZIP_PATH="${APP_PATH}.zip"
    echo -e "${YELLOW}Creating ZIP for notarization...${NC}"
    ditto -c -k --keepParent "$APP_PATH" "$ZIP_PATH"
    
    # Submit for notarization
    echo -e "${YELLOW}Submitting for notarization...${NC}"
    
    xcrun notarytool submit "$ZIP_PATH" \
        --apple-id "$APPLE_ID" \
        --password "$APP_PASSWORD" \
        --team-id "$TEAM_ID" \
        --wait \
        --verbose
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Notarization successful${NC}"
        
        # Staple the notarization ticket to the app
        echo -e "${YELLOW}Stapling notarization ticket...${NC}"
        xcrun stapler staple "$APP_PATH"
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ Notarization ticket stapled${NC}"
        else
            echo -e "${RED}✗ Failed to staple notarization ticket${NC}"
        fi
    else
        echo -e "${RED}✗ Notarization failed${NC}"
        exit 1
    fi
    
    # Clean up ZIP file
    rm -f "$ZIP_PATH"
else
    echo -e "${YELLOW}Skipping notarization as requested${NC}"
fi

echo -e "\n${GREEN}Code signing completed!${NC}"

# Display final verification
echo -e "\n${GREEN}Final verification:${NC}"
spctl -a -v "$APP_PATH"