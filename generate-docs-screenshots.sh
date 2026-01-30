#!/bin/bash
# Generate 1440x900 (16:10) screenshots from docs/images
# Used for Apple App Store submission images
# Uses blurred background letterboxing technique
# Requires: ImageMagick (brew install imagemagick)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_DIR="${SCRIPT_DIR}/docs/images"
OUT_DIR="${SCRIPT_DIR}/docs/images/1440x900"

TARGET_WIDTH=1440
TARGET_HEIGHT=900
TARGET_SIZE="${TARGET_WIDTH}x${TARGET_HEIGHT}"

mkdir -p "$OUT_DIR"

echo "Converting screenshots to ${TARGET_SIZE} (16:10)"
echo "Source: ${SOURCE_DIR}"
echo "Output: ${OUT_DIR}"
echo "Method: Blurred background letterbox"
echo ""

count=0
for img in "$SOURCE_DIR"/*.png; do
  [ -f "$img" ] || continue

  filename=$(basename "$img")

  # Get source dimensions
  dimensions=$(magick identify -format "%w %h" "$img")
  src_width=$(echo "$dimensions" | cut -d' ' -f1)
  src_height=$(echo "$dimensions" | cut -d' ' -f2)

  echo "Processing: ${filename} (${src_width}x${src_height} -> ${TARGET_SIZE})"

  # Blurred letterbox technique:
  # 1. Create blurred, stretched background (fills entire target)
  # 2. Create sharp resized foreground (fits within target, maintains aspect ratio)
  # 3. Composite foreground centered on background
  magick \
    \( "$img" -blur 0x30 -resize "${TARGET_SIZE}!" -modulate 70,60,100 \) \
    \( "$img" -filter Lanczos -resize "${TARGET_SIZE}" -unsharp 0.5x0.5+0.5+0.008 \) \
    -gravity center -composite \
    -quality 95 \
    "${OUT_DIR}/${filename}"

  ((count++))
done

echo ""
echo "Done! Converted ${count} images."
echo ""
echo "Output files:"
ls -lh "${OUT_DIR}/" | head -20
[ "$count" -gt 20 ] && echo "... and $((count - 20)) more"
