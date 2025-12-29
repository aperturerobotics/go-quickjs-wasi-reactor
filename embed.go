package quickjswasi

import _ "embed"

// QuickJSWASM contains the binary contents of the QuickJS WASI reactor build.
//
// This is a reactor-model WASM that exports functions for re-entrant execution
// in host environments. Unlike the command model which blocks in _start(),
// the reactor model allows the host to control execution flow.
//
// Exported functions:
//   - qjs_init() - Initialize empty runtime
//   - qjs_init_argv(argc, argv) - Initialize with CLI args (e.g. ["qjs", "--std", "script.js"])
//   - qjs_eval(code, len, filename, is_module) - Evaluate JS code from WASM memory
//   - qjs_loop_once() - Run one iteration of the event loop (non-blocking)
//   - qjs_destroy() - Cleanup runtime
//   - malloc/free - For host to allocate memory for code strings
//
// See: https://github.com/paralin/quickjs/tree/wasi-reactor
//
//go:embed qjs-wasi.wasm
var QuickJSWASM []byte

// QuickJSWASMFilename is the filename for QuickJSWASM.
const QuickJSWASMFilename = "qjs-wasi.wasm"

// Reactor export function names
const (
	// ExportInit is the function to initialize an empty QuickJS runtime.
	// Signature: qjs_init() -> i32 (0 on success)
	ExportInit = "qjs_init"

	// ExportInitArgv is the function to initialize QuickJS with CLI arguments.
	// This can be used to pass flags like --std or load scripts via WASI filesystem.
	// Signature: qjs_init_argv(argc: i32, argv: i32) -> i32 (0 on success)
	ExportInitArgv = "qjs_init_argv"

	// ExportEval is the function to evaluate JavaScript code.
	// Signature: qjs_eval(code: i32, len: i32, filename: i32, is_module: i32) -> i32
	// The filename parameter is a pointer to a null-terminated string (or 0 for default).
	ExportEval = "qjs_eval"

	// ExportLoopOnce runs one iteration of the event loop.
	// Signature: qjs_loop_once() -> i32
	// Returns:
	//   >0: next timer fires in N ms
	//    0: more microtasks pending, call again immediately
	//   -1: idle, no pending work
	//   -2: error occurred
	ExportLoopOnce = "qjs_loop_once"

	// ExportDestroy cleans up the QuickJS runtime.
	// Signature: qjs_destroy() -> void
	ExportDestroy = "qjs_destroy"

	// ExportMalloc allocates memory in WASM linear memory.
	// Signature: malloc(size: i32) -> i32 (pointer)
	ExportMalloc = "malloc"

	// ExportFree frees memory in WASM linear memory.
	// Signature: free(ptr: i32) -> void
	ExportFree = "free"
)

// Loop result constants from qjs_loop_once()
const (
	// LoopResultIdle indicates no pending work (-1)
	LoopResultIdle = -1
	// LoopResultError indicates an error occurred (-2)
	LoopResultError = -2
)
