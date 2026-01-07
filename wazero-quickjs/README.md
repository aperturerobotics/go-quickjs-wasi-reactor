# wazero-quickjs

High-level Go API for running JavaScript using QuickJS-NG with [wazero](https://wazero.io/).

## Installation

```bash
go get github.com/aperturerobotics/go-quickjs-wasi/wazero-quickjs
```

## Usage

### Option 1: Load scripts via WASI filesystem

```go
package main

import (
    "context"
    "embed"
    "os"

    quickjs "github.com/aperturerobotics/go-quickjs-wasi/wazero-quickjs"
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

    qjs, err := quickjs.NewQuickJS(ctx, r, config)
    if err != nil {
        panic(err)
    }
    defer qjs.Close(ctx)

    // Initialize with CLI args - loads script via WASI filesystem
    // --std enables the std and os modules
    if err := qjs.InitArgv(ctx, []string{"qjs", "--std", "scripts/main.js"}); err != nil {
        panic(err)
    }

    // Run the event loop until idle
    if err := qjs.RunLoop(ctx); err != nil {
        panic(err)
    }
}
```

### Option 2: Eval code directly

```go
package main

import (
    "context"
    "os"

    quickjs "github.com/aperturerobotics/go-quickjs-wasi/wazero-quickjs"
    "github.com/tetratelabs/wazero"
)

func main() {
    ctx := context.Background()
    r := wazero.NewRuntime(ctx)
    defer r.Close(ctx)

    config := wazero.NewModuleConfig().
        WithStdout(os.Stdout).
        WithStderr(os.Stderr)

    qjs, err := quickjs.NewQuickJS(ctx, r, config)
    if err != nil {
        panic(err)
    }
    defer qjs.Close(ctx)

    // Initialize with std module (provides std, os, bjson globals)
    if err := qjs.InitStdModule(ctx); err != nil {
        panic(err)
    }

    // Evaluate JavaScript code
    code := `
        console.log("Hello from QuickJS!");
        
        // With std module, you have access to:
        // - std.getenv(), std.setenv(), etc.
        // - os.setTimeout(), os.setInterval(), etc.
        // - bjson for binary JSON
        
        os.setTimeout(() => {
            console.log("Timer fired!");
        }, 100);
    `
    if err := qjs.Eval(ctx, code, false); err != nil {
        panic(err)
    }

    // Run the event loop until idle
    if err := qjs.RunLoop(ctx); err != nil {
        panic(err)
    }
}
```

## API

### `NewQuickJS(ctx, runtime, config) (*QuickJS, error)`

Creates a new QuickJS instance. The `config` parameter configures stdin, stdout, stderr, and filesystem access.

### `(*QuickJS) Init(ctx) error`

Initializes an empty QuickJS runtime. Must be called before `Eval`.

### `(*QuickJS) InitArgv(ctx, args) error`

Initializes the QuickJS runtime with command-line arguments. This can be used to:
- Pass flags like `--std` to enable std/os modules
- Load scripts via WASI filesystem (e.g., `["qjs", "--std", "/path/to/script.js"]`)

### `(*QuickJS) InitStdModule(ctx) error`

Initializes the QuickJS runtime and loads the std modules, making `std`, `os`, and `bjson` available as globals. Equivalent to calling `Init()` followed by evaluating the module imports.

### `(*QuickJS) Eval(ctx, code, isModule) error`

Evaluates JavaScript code. Set `isModule` to `true` for ES module syntax.

### `(*QuickJS) EvalWithFilename(ctx, code, filename, isModule) error`

Evaluates JavaScript code with a custom filename for error messages.

### `(*QuickJS) LoopOnce(ctx) (LoopResult, error)`

Runs one iteration of the event loop. Returns:
- `> 0`: next timer fires in N milliseconds
- `0`: more microtasks pending, call again immediately
- `LoopIdle (-1)`: no pending work
- `LoopError (-2)`: error occurred

### `(*QuickJS) RunLoop(ctx) error`

Runs the event loop until idle or context is canceled. This blocks until all JavaScript execution completes.

### `(*QuickJS) Close(ctx) error`

Destroys the QuickJS runtime and releases resources.

## Subdirectories

### `example/`

A minimal example demonstrating library usage.

```bash
cd example && go run .
```

### `repl/`

A command-line JavaScript runner with interactive REPL mode.

```bash
# Run directly
cd repl && go run .

# Install globally
go install github.com/aperturerobotics/go-quickjs-wasi/wazero-quickjs/repl@master

# Interactive REPL (no arguments)
repl

# Run a JavaScript file
repl script.js

# Run as ES module
repl script.mjs --module
```

## Notes

- `setTimeout` and `setInterval` are on the `os` module, not global (use `os.setTimeout()`)
- The std module provides environment variable access via `std.getenv()`, `std.setenv()`, etc.
- Promises work out of the box; `RunLoop` will wait for all pending promises
- Use `InitArgv` with `--std` flag or `InitStdModule` to enable std/os modules
