#!/bin/bash
set -e

# Configuration
VENDOR_DIR="frontend/vendor"
FONTS_DIR="$VENDOR_DIR/fonts"
USER_AGENT="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"

# Create directories
mkdir -p "$FONTS_DIR"

echo "Downloading HTMX libraries..."
curl -sL "https://unpkg.com/htmx.org@1.9.10" -o "$VENDOR_DIR/htmx.min.js"
curl -sL "https://unpkg.com/htmx.org/dist/ext/ws.js" -o "$VENDOR_DIR/ws.js"

echo "Downloading Firebase SDKs..."
curl -sL "https://www.gstatic.com/firebasejs/10.8.0/firebase-app.js" -o "$VENDOR_DIR/firebase-app.js"
curl -sL "https://www.gstatic.com/firebasejs/10.8.0/firebase-firestore.js" -o "$VENDOR_DIR/firebase-firestore.js"

echo "Downloading Google Fonts CSS and assets..."
# Fetch CSS with a modern User-Agent to ensure WOFF2
curl -sL "https://fonts.googleapis.com/css2?family=Press+Start+2P&family=Inconsolata:wght@200..900&family=Noto+Color+Emoji&display=swap" \
     -H "User-Agent: $USER_AGENT" \
     -o "$VENDOR_DIR/fonts.css"

# Extract font URLs, download them, and rewrite the CSS to point to local files
grep -o 'https://fonts.gstatic.com/[^)]*' "$VENDOR_DIR/fonts.css" | sort -u | while read -r url; do
    filename=$(basename "$url")
    echo "  Downloading font: $filename"
    curl -sL "$url" -o "$FONTS_DIR/$filename"
    # Replace the absolute URL with local relative path in the CSS
    # Escape dots in URL for sed
    escaped_url=$(echo "$url" | sed 's/\./\\./g')
    sed -i '' "s|$url|./fonts/$filename|g" "$VENDOR_DIR/fonts.css"
done

echo "Patching internal CDN references in JS files..."
# Replace Firebase CDN URLs with local relative paths
# We use a pattern that matches the specific versioned URLs used by Firebase
sed -i '' 's|https://www.gstatic.com/firebasejs/10.8.0/|./|g' "$VENDOR_DIR"/*.js

echo "Vendoring complete!"
