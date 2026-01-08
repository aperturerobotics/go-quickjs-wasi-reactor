#!/bin/bash
set -euo pipefail

# QuickJS WASI Reactor Update Script
# Downloads the reactor variant from paralin/quickjs releases

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="paralin/quickjs"
ASSET_NAME="qjs-wasi-reactor.wasm"
OUTPUT_NAME="qjs-wasi.wasm"

echo "Fetching latest reactor release from $REPO..."

# Get the latest wasi-reactor release (filter by tag pattern)
RELEASE_INFO=$(gh release view --repo "$REPO" --json tagName,assets \
    $(gh release list --repo "$REPO" --limit 20 | grep "wasi.*reactor" | head -1 | awk '{print $1}'))

TAG=$(echo "$RELEASE_INFO" | jq -r '.tagName')
DOWNLOAD_URL=$(echo "$RELEASE_INFO" | jq -r ".assets[] | select(.name == \"$ASSET_NAME\") | .url")

if [ -z "$TAG" ] || [ "$TAG" = "null" ]; then
    echo "Error: Could not find reactor release"
    exit 1
fi

if [ -z "$DOWNLOAD_URL" ] || [ "$DOWNLOAD_URL" = "null" ]; then
    echo "Error: Could not find $ASSET_NAME in release $TAG"
    exit 1
fi

echo "Found release: $TAG"
echo "Downloading $ASSET_NAME..."

# Download the WASM file
gh release download "$TAG" --repo "$REPO" --pattern "$ASSET_NAME" --output "$SCRIPT_DIR/$OUTPUT_NAME" --clobber

echo "Downloaded $OUTPUT_NAME ($(wc -c < "$SCRIPT_DIR/$OUTPUT_NAME" | tr -d ' ') bytes)"

# Generate version info Go file
echo "Generating version.go..."
cat > "$SCRIPT_DIR/version.go" << EOF
package quickjswasi

// QuickJS-NG WASI Reactor version information
const (
	// Version is the QuickJS-NG reactor version
	Version = "$TAG"
	// DownloadURL is the URL where this WASM file was downloaded from
	DownloadURL = "https://github.com/$REPO/releases/download/$TAG/$ASSET_NAME"
)
EOF

echo "Generated version.go with version $TAG"
echo ""
echo "Update complete!"
