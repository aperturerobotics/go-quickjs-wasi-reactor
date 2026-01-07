# go-quickjs-wasi-reactor

[![GoDoc Widget]][GoDoc] [![Go Report Card Widget]][Go Report Card]

> A Go module that embeds the QuickJS-NG WASI WebAssembly runtime (reactor model).

[GoDoc]: https://godoc.org/github.com/aperturerobotics/go-quickjs-wasi-reactor
[GoDoc Widget]: https://godoc.org/github.com/aperturerobotics/go-quickjs-wasi-reactor?status.svg
[Go Report Card Widget]: https://goreportcard.com/badge/github.com/aperturerobotics/go-quickjs-wasi-reactor
[Go Report Card]: https://goreportcard.com/report/github.com/aperturerobotics/go-quickjs-wasi-reactor

## Variants

This repository provides the **reactor model** WASM binary for re-entrant execution. If you only need simple blocking execution where QuickJS runs to completion in `_start()`, see the **command model** variant:

- [go-quickjs-wasi](https://github.com/paralin/go-quickjs-wasi) - Standard WASI command model with blocking `_start()` entry point

The reactor WASM binary is built from a fork of QuickJS-NG with the reactor build target:

- [paralin/quickjs](https://github.com/paralin/quickjs) - Fork with `QJS_WASI_REACTOR` build target (see [QJS_WASI_REACTOR.md](https://github.com/paralin/quickjs/blob/master/QJS_WASI_REACTOR.md) for design docs)

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
model exports functions that the host calls, enabling re-entrant execution in
JavaScript host environments (browsers, Node.js, Deno, Go with wazero, etc.).

The reactor exports:
- `qjs_init()` - Initialize empty runtime
- `qjs_init_argv(argc, argv)` - Initialize with CLI args (e.g. `["qjs", "--std", "script.js"]`)
- `qjs_eval(code, len, filename, is_module)` - Evaluate JS code from WASM memory
- `qjs_loop_once()` - Run one iteration of the event loop (non-blocking)
- `qjs_destroy()` - Cleanup runtime
- `malloc/free` - For host to allocate memory for code strings

## Features

- Embeds the QuickJS-NG WASI reactor WebAssembly binary
- Provides version information about the embedded QuickJS release
- High-level Go API via the `wazero-quickjs` subpackage
- Automatic update script to fetch the latest QuickJS-NG reactor release

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

To update to the latest QuickJS-NG reactor release:

```bash
./update-quickjs.bash
```

This script will:
1. Fetch the latest release information from the paralin/quickjs GitHub repository
2. Download the `qjs-wasi-reactor.wasm` file
3. Generate version information constants

## Testing

```bash
go test ./...
cd wazero-quickjs && go test ./...
```

## License

This module is released under the same license as the embedded QuickJS-NG project.

MIT
