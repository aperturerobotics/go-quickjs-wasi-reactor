#!/bin/bash
set -euo pipefail

# QuickJS WASI Reactor Update Script
#
# Builds the reactor variant of quickjs-ng/quickjs from source with wasi-sdk.
# The upstream release asset links with the default 64 KiB shadow stack, which
# QuickJS overflows without a check when it links a large module, and newer
# releases export only the wasi entry points instead of the QuickJS C API.
# This build reserves an 8 MiB stack below the data segment so an overflow
# traps instead of corrupting memory.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="quickjs-ng/quickjs"
OUTPUT_NAME="qjs-wasi.wasm"
WASI_SDK_VERSION="29"
STACK_SIZE=$((8 * 1024 * 1024))

# Use tag from first argument, or find the latest release.
TAG="${1:-}"
if [ -z "$TAG" ]; then
    echo "Fetching latest release from $REPO..."
    TAG=$(gh release view --repo "$REPO" --json tagName --jq '.tagName')
fi
if [ -z "$TAG" ] || [ "$TAG" = "null" ]; then
    echo "Error: Could not determine release tag"
    exit 1
fi
echo "Release: $TAG"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

# Fetch wasi-sdk for the host.
case "$(uname -s)-$(uname -m)" in
    Linux-x86_64) SDK_ARCH="x86_64-linux" ;;
    Linux-aarch64) SDK_ARCH="arm64-linux" ;;
    Darwin-x86_64) SDK_ARCH="x86_64-macos" ;;
    Darwin-arm64) SDK_ARCH="arm64-macos" ;;
    *) echo "Error: unsupported host $(uname -s)-$(uname -m)"; exit 1 ;;
esac
SDK_NAME="wasi-sdk-${WASI_SDK_VERSION}.0-${SDK_ARCH}"
echo "Downloading $SDK_NAME..."
curl -fsSL "https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-${WASI_SDK_VERSION}/${SDK_NAME}.tar.gz" |
    tar -xz -C "$WORK_DIR"

# Build the reactor from the release tag.
echo "Building $TAG..."
git clone -q --depth 1 --branch "$TAG" "https://github.com/$REPO.git" "$WORK_DIR/quickjs"
cmake -S "$WORK_DIR/quickjs" -B "$WORK_DIR/build" \
    -DCMAKE_TOOLCHAIN_FILE="$WORK_DIR/$SDK_NAME/share/cmake/wasi-sdk.cmake" \
    -DQJS_BUILD_WERROR=ON \
    -DQJS_WASI_REACTOR=ON \
    -DCMAKE_EXE_LINKER_FLAGS="-Wl,-z,stack-size=$STACK_SIZE -Wl,--stack-first" >/dev/null
cmake --build "$WORK_DIR/build" --target qjs_wasi -j"$(( $(getconf _NPROCESSORS_ONLN) / 2 > 1 ? $(getconf _NPROCESSORS_ONLN) / 2 : 1 ))" >/dev/null
cp "$WORK_DIR/build/qjs.wasm" "$SCRIPT_DIR/$OUTPUT_NAME"
echo "Built $OUTPUT_NAME ($(wc -c < "$SCRIPT_DIR/$OUTPUT_NAME" | tr -d ' ') bytes)"

# Generate version info Go file.
echo "Generating version.go..."
cat > "$SCRIPT_DIR/version.go" << GOEOF
package quickjswasi

// QuickJS-NG WASI Reactor version information
const (
	// Version is the QuickJS-NG reactor version
	Version = "$TAG"
	// SourceURL is the source tree this WASM file was built from
	SourceURL = "https://github.com/$REPO/tree/$TAG"
	// WasiSDKVersion is the wasi-sdk release that built this WASM file
	WasiSDKVersion = "$WASI_SDK_VERSION"
	// StackSize is the WASM shadow stack size in bytes
	StackSize = $STACK_SIZE
)
GOEOF

echo "Generated version.go with version $TAG"
echo ""
echo "Update complete!"
