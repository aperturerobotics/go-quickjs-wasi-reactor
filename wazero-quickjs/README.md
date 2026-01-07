# wazero-quickjs

High-level Go API for running JavaScript using QuickJS-NG with [wazero](https://wazero.io/).

## Installation

```bash
go get github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs
```

## Usage

### Option 1: Eval code directly

```go
package main

import (
    "context"
    "os"

    quickjs "github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs"
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
    if err := qjs.Eval(ctx, `console.log("Hello from QuickJS!")`, false); err != nil {
        panic(err)
    }

    // Run the event loop until idle
    if err := qjs.RunLoop(ctx); err != nil {
        panic(err)
    }
}
```

### Option 2: Load scripts via WASI filesystem

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

    qjs, err := quickjs.NewQuickJS(ctx, r, config)
    if err != nil {
        panic(err)
    }
    defer qjs.Close(ctx)

    // Initialize and set up scriptArgs
    if err := qjs.InitArgv(ctx, []string{"qjs", "scripts/main.js"}); err != nil {
        panic(err)
    }

    // Load and run the script
    if err := qjs.Eval(ctx, `import 'scripts/main.js'`, true); err != nil {
        panic(err)
    }

    // Run the event loop until idle
    if err := qjs.RunLoop(ctx); err != nil {
        panic(err)
    }
}
```

### Option 3: Non-blocking event loop

For fine-grained control, use `LoopOnce()` instead of `RunLoop()`:

```go
package main

import (
    "context"
    "errors"
    "os"
    "time"

    quickjs "github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs"
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

    qjs.InitStdModule(ctx)
    qjs.Eval(ctx, `os.setTimeout(() => console.log("timer!"), 100)`, false)

    // Manual event loop - host has full control
    for {
        result, err := qjs.LoopOnce(ctx)
        if err != nil {
            panic(err)
        }

        switch {
        case result == quickjs.LoopIdle:
            return // No more work
        case result == quickjs.LoopError:
            panic(errors.New("JavaScript error"))
        case result == 0:
            continue // More microtasks pending
        case result > 0:
            // Timer pending - host can do other work here
            time.Sleep(time.Duration(result) * time.Millisecond)
        }
    }
}
```

## API

### `NewQuickJS(ctx, runtime, config) (*QuickJS, error)`

Creates a new QuickJS instance. The `config` parameter configures stdin, stdout, stderr, and filesystem access.

### `(*QuickJS) Init(ctx) error`

Initializes the QuickJS runtime and context with std modules available for import.
Modules `qjs:std`, `qjs:os`, and `qjs:bjson` can be imported in evaluated code.

### `(*QuickJS) InitStdModule(ctx) error`

Like `Init()` but also exposes `std`, `os`, and `bjson` as global objects.
Convenient for REPL-style usage where you want immediate access to the std library.

### `(*QuickJS) InitArgv(ctx, args) error`

Like `Init()` but also sets up `scriptArgs` global with the provided arguments.
Use this when your JavaScript code needs to access command-line arguments.

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
go install github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs/repl@master

# Interactive REPL (no arguments)
repl

# Run a JavaScript file
repl script.js

# Run as ES module
repl script.mjs --module
```

### `(*QuickJS) PollIO(ctx, timeoutMs) (int32, error)`

Polls for I/O events and invokes registered read/write handlers. Call this when
the host knows stdin or other fds have data available, otherwise `os.setReadHandler`
callbacks will never fire.

## Notes

- `setTimeout` and `setInterval` are on the `os` module, not global (use `os.setTimeout()`)
- The std module provides environment variable access via `std.getenv()`, `std.setenv()`, etc.
- Promises work out of the box; `RunLoop` will wait for all pending promises
- Use `InitStdModule()` for std/os/bjson globals, or `Init()` and import them manually
