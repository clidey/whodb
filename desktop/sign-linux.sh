#!/bin/bash

# Linux Code Signing Script for WhoDB
# This script signs Linux executables and packages
# Note: Linux doesn't have mandatory signing like Windows/macOS, but GPG signing
# can be used for package repositories and to ensure authenticity

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to display usage
usage() {
    echo "Usage: $0 -f FILE -k GPG_KEY_ID [-d]"
    echo "  -f FILE         Path to the file to sign (executable, .deb, .AppImage, etc.)"
    echo "  -k GPG_KEY_ID   GPG key ID or email to use for signing"
    echo "  -d              Create detached signature (.sig file)"
    echo "  -h              Show this help message"
    exit 1
}

# Parse command line arguments
DETACHED=false

while getopts "f:k:dh" opt; do
    case $opt in
        f) FILE_PATH="$OPTARG" ;;
        k) GPG_KEY_ID="$OPTARG" ;;
        d) DETACHED=true ;;
        h) usage ;;
        *) usage ;;
    esac
done

# Validate required arguments
if [ -z "$FILE_PATH" ] || [ -z "$GPG_KEY_ID" ]; then
    echo -e "${RED}Error: FILE_PATH and GPG_KEY_ID are required${NC}"
    usage
fi

# Check if file exists
if [ ! -f "$FILE_PATH" ]; then
    echo -e "${RED}Error: File not found at $FILE_PATH${NC}"
    exit 1
fi

# Check if GPG is installed
if ! command -v gpg &> /dev/null; then
    echo -e "${RED}Error: GPG is not installed${NC}"
    echo "Install with: sudo apt-get install gnupg (Debian/Ubuntu) or sudo dnf install gnupg (Fedora)"
    exit 1
fi

echo -e "${GREEN}Linux Code Signing Script for WhoDB${NC}"
echo "===================================="
echo -e "${YELLOW}File: $FILE_PATH${NC}"
echo -e "${YELLOW}GPG Key: $GPG_KEY_ID${NC}"

# Check if the GPG key exists
echo -e "\n${GREEN}Checking GPG key...${NC}"
if ! gpg --list-secret-keys "$GPG_KEY_ID" &> /dev/null; then
    echo -e "${RED}Error: GPG key not found: $GPG_KEY_ID${NC}"
    echo "Available keys:"
    gpg --list-secret-keys
    exit 1
fi

FILE_TYPE=$(basename "$FILE_PATH")

# Special handling for different file types
case "$FILE_TYPE" in
    *.deb)
        echo -e "\n${GREEN}Signing Debian package...${NC}"
        
        # Check if dpkg-sig is installed
        if ! command -v dpkg-sig &> /dev/null; then
            echo -e "${YELLOW}dpkg-sig not found. Installing...${NC}"
            sudo apt-get update && sudo apt-get install -y dpkg-sig
        fi
        
        # Sign the .deb package
        dpkg-sig --sign builder -k "$GPG_KEY_ID" "$FILE_PATH"
        
        # Verify the signature
        echo -e "\n${GREEN}Verifying .deb signature...${NC}"
        dpkg-sig --verify "$FILE_PATH"
        ;;
        
    *.AppImage)
        echo -e "\n${GREEN}Signing AppImage...${NC}"
        
        if [ "$DETACHED" = true ]; then
            # Create detached signature
            gpg --armor --detach-sign --local-user "$GPG_KEY_ID" "$FILE_PATH"
            echo -e "${GREEN}✓ Created detached signature: ${FILE_PATH}.asc${NC}"
        else
            echo -e "${YELLOW}Note: AppImages typically use detached signatures${NC}"
            echo "Consider using -d flag for detached signature"
            
            # Sign the file inline (less common for AppImages)
            gpg --armor --sign --local-user "$GPG_KEY_ID" "$FILE_PATH"
        fi
        ;;
        
    *)
        # Generic file signing (executables, archives, etc.)
        echo -e "\n${GREEN}Signing file...${NC}"
        
        if [ "$DETACHED" = true ]; then
            # Create detached signature
            gpg --armor --detach-sign --local-user "$GPG_KEY_ID" "$FILE_PATH"
            echo -e "${GREEN}✓ Created detached signature: ${FILE_PATH}.asc${NC}"
            
            # Verify the detached signature
            echo -e "\n${GREEN}Verifying detached signature...${NC}"
            gpg --verify "${FILE_PATH}.asc" "$FILE_PATH"
        else
            # Create inline signature
            gpg --armor --sign --local-user "$GPG_KEY_ID" --output "${FILE_PATH}.signed" "$FILE_PATH"
            echo -e "${GREEN}✓ Created signed file: ${FILE_PATH}.signed${NC}"
            
            # Verify the signature
            echo -e "\n${GREEN}Verifying signature...${NC}"
            gpg --verify "${FILE_PATH}.signed"
        fi
        ;;
esac

echo -e "\n${GREEN}Creating SHA256 checksum...${NC}"
sha256sum "$FILE_PATH" > "${FILE_PATH}.sha256"
echo -e "${GREEN}✓ Created checksum file: ${FILE_PATH}.sha256${NC}"

# Sign the checksum file
gpg --armor --detach-sign --local-user "$GPG_KEY_ID" "${FILE_PATH}.sha256"
echo -e "${GREEN}✓ Signed checksum file: ${FILE_PATH}.sha256.asc${NC}"

echo -e "\n${GREEN}Signature Summary:${NC}"
echo "1. Original file: $FILE_PATH"
if [ "$DETACHED" = true ] || [[ "$FILE_TYPE" == *.AppImage ]]; then
    echo "2. Signature file: ${FILE_PATH}.asc"
else
    echo "2. Signed file: ${FILE_PATH}.signed (or embedded for .deb)"
fi
echo "3. Checksum file: ${FILE_PATH}.sha256"
echo "4. Signed checksum: ${FILE_PATH}.sha256.asc"

echo -e "\n${GREEN}To verify the signature, users can run:${NC}"
if [ "$DETACHED" = true ] || [[ "$FILE_TYPE" == *.AppImage ]]; then
    echo "  gpg --verify ${FILE_PATH}.asc $FILE_PATH"
else
    echo "  gpg --verify ${FILE_PATH}.signed"
fi
echo "  sha256sum -c ${FILE_PATH}.sha256"

echo -e "\n${GREEN}Code signing completed!${NC}"