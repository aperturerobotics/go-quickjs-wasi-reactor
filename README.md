# go-quickjs-wasi-reactor

[![GoDoc Widget]][GoDoc] [![Go Report Card Widget]][Go Report Card]

> A Go module that embeds the QuickJS-NG WASI WebAssembly runtime (reactor model).

[GoDoc]: https://godoc.org/github.com/aperturerobotics/go-quickjs-wasi-reactor
[GoDoc Widget]: https://godoc.org/github.com/aperturerobotics/go-quickjs-wasi-reactor?status.svg
[Go Report Card Widget]: https://goreportcard.com/badge/github.com/aperturerobotics/go-quickjs-wasi-reactor
[Go Report Card]: https://goreportcard.com/report/github.com/aperturerobotics/go-quickjs-wasi-reactor

## Related Projects

- [js-quickjs-wasi-reactor](https://github.com/aperturerobotics/js-quickjs-wasi-reactor) - JavaScript/TypeScript harness for browser environments
- [go-quickjs-wasi](https://github.com/paralin/go-quickjs-wasi) - Standard WASI command model with blocking `_start()` entry point
- [paralin/quickjs](https://github.com/paralin/quickjs) - Fork with `QJS_WASI_REACTOR` build target
- [QuickJS-NG reactor PR](https://github.com/quickjs-ng/quickjs/pull/1307) - Upstream PR for reactor support

## Variants

This repository provides the **reactor model** WASM binary for re-entrant execution. If you only need simple blocking execution where QuickJS runs to completion in `_start()`, see the **command model** variant above.

## About QuickJS-NG

QuickJS is a small and embeddable JavaScript engine. It aims to support the latest ECMAScript specification.

This project uses [QuickJS-NG] which is a fork of the original [QuickJS project]
by Fabrice Bellard and Charlie Gordon, after it went dormant, with the intent of
reigniting its development.

[QuickJS-NG]: https://github.com/quickjs-ng/quickjs
[QuickJS project]: https://bellard.org/quickjs/

## Purpose

This module provides easy access to the QuickJS-NG JavaScript engine compiled to
WebAssembly with WASI support using the **reactor model**. The WASM binary is
embedded directly in the Go module, making it easy to use QuickJS in Go
applications without external dependencies.

### Reactor Model

Unlike the standard WASI "command" model that blocks in `_start()`, the reactor
model exports the raw QuickJS C API functions, enabling full control over the
JavaScript runtime lifecycle from the host environment.

The reactor exports the complete QuickJS C API including:

**Core Runtime:**
- `JS_NewRuntime`, `JS_FreeRuntime` - Runtime lifecycle
- `JS_NewContext`, `JS_FreeContext` - Context lifecycle
- `JS_Eval` - Evaluate JavaScript code
- `JS_Call` - Call JavaScript functions

**Standard Library (quickjs-libc.h):**
- `js_init_module_std`, `js_init_module_os`, `js_init_module_bjson` - Module initialization
- `js_std_init_handlers`, `js_std_free_handlers` - I/O handler setup
- `js_std_add_helpers` - Add console.log, print, etc.
- `js_std_loop_once` - Run one iteration of the event loop (non-blocking)
- `js_std_poll_io` - Poll for I/O events

**Memory Management:**
- `malloc`, `free`, `realloc`, `calloc` - For host to allocate memory

## Features

- Embeds the QuickJS-NG WASI reactor WebAssembly binary
- Provides version information about the embedded QuickJS release
- High-level Go API via the `wazero-quickjs` subpackage
- Update script to build and copy from local QuickJS checkout

## Packages

### Root Package (`github.com/aperturerobotics/go-quickjs-wasi-reactor`)

Provides the embedded WASM binary and version information:

```go
package main

import (
    "fmt"
    quickjswasi "github.com/aperturerobotics/go-quickjs-wasi-reactor"
)

func main() {
    // Access the embedded WASM binary
    wasmBytes := quickjswasi.QuickJSWASM
    fmt.Printf("QuickJS WASM size: %d bytes\n", len(wasmBytes))

    // Get version information
    fmt.Printf("QuickJS version: %s\n", quickjswasi.Version)
    fmt.Printf("Download URL: %s\n", quickjswasi.DownloadURL)
}
```

### Wazero QuickJS Library (`github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs`)

High-level Go API for running JavaScript with wazero:

```go
package main

import (
    "context"
    "embed"
    "os"

    quickjs "github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs"
    "github.com/tetratelabs/wazero"
)

//go:embed scripts
var scriptsFS embed.FS

func main() {
    ctx := context.Background()
    r := wazero.NewRuntime(ctx)
    defer r.Close(ctx)

    config := wazero.NewModuleConfig().
        WithStdout(os.Stdout).
        WithStderr(os.Stderr).
        WithFS(scriptsFS)  // Mount embedded filesystem

    qjs, _ := quickjs.NewQuickJS(ctx, r, config)
    defer qjs.Close(ctx)

    // Option 1: Initialize with CLI args to load script via WASI filesystem
    qjs.InitArgv(ctx, []string{"qjs", "--std", "scripts/main.js"})

    // Option 2: Initialize with std module and eval code directly
    // qjs.InitStdModule(ctx)
    // qjs.Eval(ctx, `console.log("Hello from QuickJS!");`, false)

    // Run event loop until idle
    qjs.RunLoop(ctx)
}
```

See the [wazero-quickjs README](./wazero-quickjs/README.md) for more details.

## Command-Line REPL

A command-line JavaScript runner with interactive REPL mode is provided:

```bash
# Install
go install github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs/repl@master

# Interactive REPL
repl

# Run a JavaScript file
repl script.js

# Run as ES module
repl script.mjs --module
```

## Updating

To update to the latest QuickJS-NG reactor build from a local checkout:

```bash
# Set QUICKJS_DIR to your quickjs checkout (default: ../quickjs)
QUICKJS_DIR=/path/to/quickjs ./update-quickjs.bash
```

This script will:
1. Read the current branch and commit from your local quickjs checkout
2. Copy the `qjs-wasi-reactor.wasm` file
3. Generate version information constants

### Building the WASM file

To build the WASM file from source:

```bash
cd /path/to/quickjs

# Create build directory
mkdir -p build-wasi-reactor && cd build-wasi-reactor

# Configure with WASI SDK
cmake .. \
  -DCMAKE_TOOLCHAIN_FILE=/opt/wasi-sdk/share/cmake/wasi-sdk.cmake \
  -DQJS_WASI_REACTOR=ON \
  -DCMAKE_BUILD_TYPE=Release

# Build
make -j

# Copy output
cp qjs.wasm ../qjs-wasi-reactor.wasm
```

## Testing

```bash
go test ./...
cd wazero-quickjs && go test ./...
```

## License

This module is released under the same license as the embedded QuickJS-NG project.

MIT
