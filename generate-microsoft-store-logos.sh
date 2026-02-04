#!/bin/bash
# Generate Microsoft Store logo variants from SVG source
# Requires: ImageMagick (brew install imagemagick)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SVG_SOURCE="${SCRIPT_DIR}/docs/logo/logo.svg"
OUT_DIR="${SCRIPT_DIR}/docs/logo/microsoft-store"

mkdir -p "$OUT_DIR"

# All sizes to generate (WxH)
SIZES=(
  "50x50"
  "44x44"
  "150x150"
  "310x150"
  "620x300"
  "720x1080"
  "1440x2160"
  "1080x1080"
  "2160x2160"
  "300x300"
  "71x71"
  "1920x1080"
  "3840x2160"
  "584x800"
)

echo "Generating logo variants from: ${SVG_SOURCE}"
echo "Output directory: ${OUT_DIR}"
echo ""

for size in "${SIZES[@]}"; do
  echo "Generating ${size}..."

  # Extract width to determine if this is a small icon
  width="${size%x*}"

  if [ "$width" -lt 100 ]; then
    # Small icons: render to intermediate size first, then downscale with Mitchell filter
    magick -density 300 -background transparent "$SVG_SOURCE" \
      -resize 512x512 \
      -filter Mitchell \
      -resize "${size}" \
      -gravity center \
      -extent "${size}" \
      -quality 100 \
      "${OUT_DIR}/logo_${size}.png"
  else
    # Larger sizes: use high density for sharp SVG rendering
    magick -density 1200 -background transparent "$SVG_SOURCE" \
      -resize "${size}" \
      -gravity center \
      -extent "${size}" \
      -quality 95 \
      "${OUT_DIR}/logo_${size}.png"
  fi
done

echo ""
echo "Done! Generated files:"
ls -lh "${OUT_DIR}/"*.png
