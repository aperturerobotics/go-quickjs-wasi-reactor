#!/bin/bash

set -euo pipefail

# QuickJS WASI Reactor Update Script
# Builds and copies the reactor variant from a local quickjs checkout

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
QUICKJS_DIR="${QUICKJS_DIR:-../quickjs}"

# Check if quickjs directory exists
if [ ! -d "$QUICKJS_DIR" ]; then
    echo "Error: QuickJS directory not found at $QUICKJS_DIR"
    echo "Set QUICKJS_DIR environment variable to point to your quickjs checkout"
    exit 1
fi

cd "$QUICKJS_DIR"

# Get current branch and commit info
BRANCH=$(git rev-parse --abbrev-ref HEAD)
COMMIT=$(git rev-parse HEAD)
SHORT_COMMIT=$(git rev-parse --short HEAD)

echo "Building QuickJS WASI reactor from:"
echo "  Directory: $QUICKJS_DIR"
echo "  Branch: $BRANCH"
echo "  Commit: $SHORT_COMMIT"

# Check for wasm file
if [ ! -f "qjs-wasi-reactor.wasm" ]; then
    echo ""
    echo "Error: qjs-wasi-reactor.wasm not found in $QUICKJS_DIR"
    echo ""
    echo "Build it with:"
    echo "  mkdir -p build-wasi-reactor && cd build-wasi-reactor"
    echo "  cmake .. -DCMAKE_TOOLCHAIN_FILE=/path/to/wasi-sdk/share/cmake/wasi-sdk.cmake \\"
    echo "           -DQJS_WASI_REACTOR=ON -DCMAKE_BUILD_TYPE=Release"
    echo "  make -j"
    echo "  cp qjs.wasm ../qjs-wasi-reactor.wasm"
    exit 1
fi

# Copy WASM file
echo ""
echo "Copying qjs-wasi-reactor.wasm to qjs-wasi.wasm..."
cp "qjs-wasi-reactor.wasm" "$SCRIPT_DIR/qjs-wasi.wasm"

echo "Copied successfully ($(wc -c < "$SCRIPT_DIR/qjs-wasi.wasm" | tr -d ' ') bytes)"

# Generate version info Go file
echo "Generating version.go..."
cat > "$SCRIPT_DIR/version.go" << EOF
package quickjswasi

// QuickJS-NG WASI Reactor version information
const (
	// Version is the QuickJS-NG reactor version
	// Built from paralin/quickjs $BRANCH branch
	Version = "v0.11.0-$BRANCH"
	// GitCommit is the git commit hash of the quickjs source
	GitCommit = "$COMMIT"
	// DownloadURL is the URL where this WASM file was downloaded from (or empty for local build)
	DownloadURL = ""
)
EOF

echo "Generated version.go with commit $SHORT_COMMIT"
echo ""
echo "Update complete!"
