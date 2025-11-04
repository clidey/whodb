#!/bin/bash
#
# Copyright 2025 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -e

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <architecture> <version>"
    echo "Example: $0 amd64 0.61.0"
    exit 1
fi

TARGET_ARCH=$1
VERSION=$2
echo "Building AppImage for ${TARGET_ARCH}..."

case "$TARGET_ARCH" in
    amd64)
        BUILD_ARCH="x86_64"
        APPIMAGETOOL_ARCH="x86_64"
        APPIMAGE_ARCH_ENV="x86_64"
        ;;
    arm64)
        BUILD_ARCH="aarch64"
        APPIMAGETOOL_ARCH="aarch64"
        APPIMAGE_ARCH_ENV="aarch64"
        ;;
    *)
        BUILD_ARCH="$TARGET_ARCH"
        APPIMAGETOOL_ARCH="$TARGET_ARCH"
        APPIMAGE_ARCH_ENV="$TARGET_ARCH"
        ;;
esac

APPDIR="WhoDB-${APPIMAGE_ARCH_ENV}.AppDir"

# Create AppDir structure
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin"
mkdir -p "$APPDIR/usr/share/applications"
mkdir -p "$APPDIR/usr/share/icons/hicolor/256x256/apps"

# Copy binary
BINARY_PATH="desktop-ce/build/linux/${BUILD_ARCH}/whodb"
if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Compiled binary not found at $BINARY_PATH"
    exit 1
fi
cp "$BINARY_PATH" "$APPDIR/usr/bin/"
chmod +x "$APPDIR/usr/bin/whodb"

# Copy desktop file and icon
cp linux/whodb.desktop "$APPDIR/usr/share/applications/"
cp linux/icon.png "$APPDIR/usr/share/icons/hicolor/256x256/apps/whodb.png"

# Create AppRun symlink so appimagetool infers the same architecture as the main binary
ln -sf usr/bin/whodb "$APPDIR/AppRun"

# Create symlinks for AppImage structure
ln -sf usr/share/applications/whodb.desktop "$APPDIR/whodb.desktop"
ln -sf usr/share/icons/hicolor/256x256/apps/whodb.png "$APPDIR/whodb.png"
ln -sf usr/share/icons/hicolor/256x256/apps/whodb.png "$APPDIR/.DirIcon"

# Download appimagetool if not present
if [ ! -f "appimagetool-${TARGET_ARCH}.AppImage" ]; then
    echo "Downloading appimagetool for ${TARGET_ARCH}..."
    wget -q "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-${APPIMAGETOOL_ARCH}.AppImage" -O "appimagetool-${TARGET_ARCH}.AppImage"
    chmod +x "appimagetool-${TARGET_ARCH}.AppImage"
fi

# Build AppImage
APPIMAGE_ARCH="${APPIMAGE_ARCH_ENV:-$APPIMAGETOOL_ARCH}"
echo "Using AppImage ARCH override: ${APPIMAGE_ARCH}"
echo "Inspecting AppDir executables:"
find "$APPDIR" -maxdepth 4 -type f -exec file {} \;
echo "Binary details:"
ls -l "$APPDIR/usr/bin/"
file "$APPDIR/usr/bin/whodb"
OUTPUT_APPIMAGE="WhoDB-${VERSION}-${TARGET_ARCH}.AppImage"

echo "Running appimagetool with ARCH=${APPIMAGE_ARCH}"
env ARCH="${APPIMAGE_ARCH}" "./appimagetool-${TARGET_ARCH}.AppImage" --verbose "$APPDIR" "$OUTPUT_APPIMAGE"

# Ensure resulting AppImage is marked executable
chmod +x "$OUTPUT_APPIMAGE"

echo "✓ AppImage created: $OUTPUT_APPIMAGE"
