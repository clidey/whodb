#!/bin/bash

# Icon generation script for Tauri
# Requires ImageMagick (convert command)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if ImageMagick is installed
if ! command -v convert &> /dev/null; then
    echo -e "${RED}Error: ImageMagick is not installed${NC}"
    echo "Install with:"
    echo "  macOS: brew install imagemagick"
    echo "  Ubuntu/Debian: sudo apt-get install imagemagick"
    echo "  Fedora: sudo dnf install ImageMagick"
    exit 1
fi

# Source icon (should be at least 512x512)
SOURCE_ICON="../frontend/public/images/logo.png"

# Check if source exists
if [ ! -f "$SOURCE_ICON" ]; then
    echo -e "${RED}Error: Source icon not found at $SOURCE_ICON${NC}"
    exit 1
fi

# Create icons directory
mkdir -p src-tauri/icons

echo -e "${GREEN}Generating Tauri icons from $SOURCE_ICON${NC}"

# Generate PNG icons for Tauri
echo -e "${YELLOW}Generating PNG icons...${NC}"

# Required sizes for Tauri
sizes=(32 128 256 512)
for size in "${sizes[@]}"; do
    output="src-tauri/icons/${size}x${size}.png"
    convert "$SOURCE_ICON" -resize ${size}x${size} "$output"
    echo -e "  Created: $output"
done

# Create @2x version for macOS
convert "$SOURCE_ICON" -resize 256x256 "src-tauri/icons/128x128@2x.png"
echo -e "  Created: src-tauri/icons/128x128@2x.png"

# Copy as main icon.png (required by some Tauri configurations)
cp "$SOURCE_ICON" "src-tauri/icons/icon.png"
echo -e "  Created: src-tauri/icons/icon.png"

# Generate .ico for Windows
echo -e "\n${YELLOW}Generating Windows icon...${NC}"
convert "$SOURCE_ICON" -resize 256x256 -define icon:auto-resize=256,128,64,48,32,16 "src-tauri/icons/icon.ico"
echo -e "  Created: src-tauri/icons/icon.ico"

# Generate .icns for macOS
echo -e "\n${YELLOW}Generating macOS icon...${NC}"

# Create temporary directory for iconset
ICONSET="src-tauri/icons/icon.iconset"
mkdir -p "$ICONSET"

# Generate required sizes for macOS
convert "$SOURCE_ICON" -resize 16x16     "$ICONSET/icon_16x16.png"
convert "$SOURCE_ICON" -resize 32x32     "$ICONSET/icon_16x16@2x.png"
convert "$SOURCE_ICON" -resize 32x32     "$ICONSET/icon_32x32.png"
convert "$SOURCE_ICON" -resize 64x64     "$ICONSET/icon_32x32@2x.png"
convert "$SOURCE_ICON" -resize 128x128   "$ICONSET/icon_128x128.png"
convert "$SOURCE_ICON" -resize 256x256   "$ICONSET/icon_128x128@2x.png"
convert "$SOURCE_ICON" -resize 256x256   "$ICONSET/icon_256x256.png"
convert "$SOURCE_ICON" -resize 512x512   "$ICONSET/icon_256x256@2x.png"
convert "$SOURCE_ICON" -resize 512x512   "$ICONSET/icon_512x512.png"
convert "$SOURCE_ICON" -resize 1024x1024 "$ICONSET/icon_512x512@2x.png"

# Convert to .icns (macOS only)
if [[ "$OSTYPE" == "darwin"* ]]; then
    iconutil -c icns "$ICONSET" -o "src-tauri/icons/icon.icns"
    echo -e "  Created: src-tauri/icons/icon.icns"
else
    echo -e "${YELLOW}  Note: .icns creation requires macOS. Skipping.${NC}"
fi

# Clean up iconset
rm -rf "$ICONSET"

echo -e "\n${GREEN}Icon generation complete!${NC}"
echo "Icons created in: src-tauri/icons/"
ls -la src-tauri/icons/