package quickjswasi

import _ "embed"

// QuickJSWASM contains the binary contents of the QuickJS WASI reactor build.
//
// This is a reactor-model WASM that exports the QuickJS C API for re-entrant
// execution in host environments. Unlike the command model which blocks in
// _start(), the reactor model allows the host to control execution flow.
//
// The reactor exports the raw QuickJS C API functions from quickjs.h and
// quickjs-libc.h, allowing full control over runtime lifecycle.
//
// See: https://github.com/paralin/quickjs/tree/wasi-reactor-libc
//
//go:embed qjs-wasi.wasm
var QuickJSWASM []byte

// QuickJSWASMFilename is the filename for QuickJSWASM.
const QuickJSWASMFilename = "qjs-wasi.wasm"

// Memory management exports
const (
	// ExportMalloc allocates memory in WASM linear memory.
	// Signature: malloc(size: i32) -> i32 (pointer)
	ExportMalloc = "malloc"

	// ExportFree frees memory in WASM linear memory.
	// Signature: free(ptr: i32) -> void
	ExportFree = "free"

	// ExportRealloc reallocates memory in WASM linear memory.
	// Signature: realloc(ptr: i32, size: i32) -> i32 (pointer)
	ExportRealloc = "realloc"

	// ExportCalloc allocates zeroed memory in WASM linear memory.
	// Signature: calloc(nmemb: i32, size: i32) -> i32 (pointer)
	ExportCalloc = "calloc"
)

// Core runtime exports (quickjs.h)
const (
	// ExportJSNewRuntime creates a new JavaScript runtime.
	// Signature: JS_NewRuntime() -> i32 (JSRuntime*)
	ExportJSNewRuntime = "JS_NewRuntime"

	// ExportJSFreeRuntime frees a JavaScript runtime.
	// Signature: JS_FreeRuntime(rt: i32) -> void
	ExportJSFreeRuntime = "JS_FreeRuntime"

	// ExportJSNewContext creates a new JavaScript context.
	// Signature: JS_NewContext(rt: i32) -> i32 (JSContext*)
	ExportJSNewContext = "JS_NewContext"

	// ExportJSFreeContext frees a JavaScript context.
	// Signature: JS_FreeContext(ctx: i32) -> void
	ExportJSFreeContext = "JS_FreeContext"

	// ExportJSGetRuntime gets the runtime from a context.
	// Signature: JS_GetRuntime(ctx: i32) -> i32 (JSRuntime*)
	ExportJSGetRuntime = "JS_GetRuntime"

	// ExportJSSetMemoryLimit sets the memory limit for the runtime.
	// Signature: JS_SetMemoryLimit(rt: i32, limit: i64) -> void
	ExportJSSetMemoryLimit = "JS_SetMemoryLimit"

	// ExportJSSetMaxStackSize sets the maximum stack size.
	// Signature: JS_SetMaxStackSize(rt: i32, size: i64) -> void
	ExportJSSetMaxStackSize = "JS_SetMaxStackSize"

	// ExportJSSetGCThreshold sets the GC threshold.
	// Signature: JS_SetGCThreshold(rt: i32, threshold: i64) -> void
	ExportJSSetGCThreshold = "JS_SetGCThreshold"

	// ExportJSRunGC runs garbage collection.
	// Signature: JS_RunGC(rt: i32) -> void
	ExportJSRunGC = "JS_RunGC"
)

// Evaluation exports
const (
	// ExportJSEval evaluates JavaScript code.
	// Signature: JS_Eval(ctx: i32, input: i32, input_len: i64, filename: i32, eval_flags: i32) -> i64 (JSValue)
	ExportJSEval = "JS_Eval"

	// ExportJSEvalFunction evaluates a compiled function.
	// Signature: JS_EvalFunction(ctx: i32, fun_obj: i64) -> i64 (JSValue)
	ExportJSEvalFunction = "JS_EvalFunction"

	// ExportJSCall calls a JavaScript function.
	// Signature: JS_Call(ctx: i32, func_obj: i64, this_obj: i64, argc: i32, argv: i32) -> i64 (JSValue)
	ExportJSCall = "JS_Call"
)

// Value exports
const (
	// ExportJSNewStringLen creates a new string from bytes.
	// Signature: JS_NewStringLen(ctx: i32, str: i32, len: i64) -> i64 (JSValue)
	ExportJSNewStringLen = "JS_NewStringLen"

	// ExportJSNewObject creates a new empty object.
	// Signature: JS_NewObject(ctx: i32) -> i64 (JSValue)
	ExportJSNewObject = "JS_NewObject"

	// ExportJSNewArray creates a new array.
	// Signature: JS_NewArray(ctx: i32) -> i64 (JSValue)
	ExportJSNewArray = "JS_NewArray"

	// ExportJSNewArrayBufferCopy creates a new ArrayBuffer with copied data.
	// Signature: JS_NewArrayBufferCopy(ctx: i32, buf: i32, len: i64) -> i64 (JSValue)
	ExportJSNewArrayBufferCopy = "JS_NewArrayBufferCopy"

	// ExportJSToBool converts a value to boolean.
	// Signature: JS_ToBool(ctx: i32, val: i64) -> i32
	ExportJSToBool = "JS_ToBool"

	// ExportJSToInt32 converts a value to int32.
	// Signature: JS_ToInt32(ctx: i32, pres: i32, val: i64) -> i32
	ExportJSToInt32 = "JS_ToInt32"

	// ExportJSToInt64 converts a value to int64.
	// Signature: JS_ToInt64(ctx: i32, pres: i32, val: i64) -> i32
	ExportJSToInt64 = "JS_ToInt64"

	// ExportJSToFloat64 converts a value to float64.
	// Signature: JS_ToFloat64(ctx: i32, pres: i32, val: i64) -> i32
	ExportJSToFloat64 = "JS_ToFloat64"

	// ExportJSToCStringLen2 converts a value to a C string.
	// Signature: JS_ToCStringLen2(ctx: i32, plen: i32, val: i64, cesu8: i32) -> i32 (char*)
	ExportJSToCStringLen2 = "JS_ToCStringLen2"

	// ExportJSFreeCString frees a C string returned by JS_ToCStringLen2.
	// Signature: JS_FreeCString(ctx: i32, ptr: i32) -> void
	ExportJSFreeCString = "JS_FreeCString"

	// ExportJSGetArrayBuffer gets the underlying buffer of an ArrayBuffer.
	// Signature: JS_GetArrayBuffer(ctx: i32, psize: i32, obj: i64) -> i32 (uint8_t*)
	ExportJSGetArrayBuffer = "JS_GetArrayBuffer"

	// ExportJSDupValue duplicates a value (increases refcount).
	// Signature: JS_DupValue(ctx: i32, val: i64) -> i64 (JSValue)
	ExportJSDupValue = "JS_DupValue"

	// ExportJSFreeValue frees a value (decreases refcount).
	// Signature: JS_FreeValue(ctx: i32, val: i64) -> void
	ExportJSFreeValue = "JS_FreeValue"
)

// Property exports
const (
	// ExportJSGetPropertyStr gets a property by string key.
	// Signature: JS_GetPropertyStr(ctx: i32, this_obj: i64, prop: i32) -> i64 (JSValue)
	ExportJSGetPropertyStr = "JS_GetPropertyStr"

	// ExportJSGetPropertyUint32 gets a property by index.
	// Signature: JS_GetPropertyUint32(ctx: i32, this_obj: i64, idx: i32) -> i64 (JSValue)
	ExportJSGetPropertyUint32 = "JS_GetPropertyUint32"

	// ExportJSSetPropertyStr sets a property by string key.
	// Signature: JS_SetPropertyStr(ctx: i32, this_obj: i64, prop: i32, val: i64) -> i32
	ExportJSSetPropertyStr = "JS_SetPropertyStr"

	// ExportJSSetPropertyUint32 sets a property by index.
	// Signature: JS_SetPropertyUint32(ctx: i32, this_obj: i64, idx: i32, val: i64) -> i32
	ExportJSSetPropertyUint32 = "JS_SetPropertyUint32"

	// ExportJSHasProperty checks if an object has a property.
	// Signature: JS_HasProperty(ctx: i32, this_obj: i64, prop: i32) -> i32
	ExportJSHasProperty = "JS_HasProperty"

	// ExportJSDeleteProperty deletes a property.
	// Signature: JS_DeleteProperty(ctx: i32, this_obj: i64, prop: i32, flags: i32) -> i32
	ExportJSDeleteProperty = "JS_DeleteProperty"

	// ExportJSGetGlobalObject gets the global object.
	// Signature: JS_GetGlobalObject(ctx: i32) -> i64 (JSValue)
	ExportJSGetGlobalObject = "JS_GetGlobalObject"
)

// Exception exports
const (
	// ExportJSThrow throws an exception.
	// Signature: JS_Throw(ctx: i32, obj: i64) -> i64 (JSValue)
	ExportJSThrow = "JS_Throw"

	// ExportJSGetException gets the current exception.
	// Signature: JS_GetException(ctx: i32) -> i64 (JSValue)
	ExportJSGetException = "JS_GetException"

	// ExportJSHasException checks if there is a pending exception.
	// Signature: JS_HasException(ctx: i32) -> i32
	ExportJSHasException = "JS_HasException"

	// ExportJSIsError checks if a value is an Error object.
	// Signature: JS_IsError(ctx: i32, val: i64) -> i32
	ExportJSIsError = "JS_IsError"
)

// Promise exports
const (
	// ExportJSPromiseState gets the state of a promise.
	// Signature: JS_PromiseState(ctx: i32, promise: i64) -> i32
	ExportJSPromiseState = "JS_PromiseState"

	// ExportJSPromiseResult gets the result of a promise.
	// Signature: JS_PromiseResult(ctx: i32, promise: i64) -> i64 (JSValue)
	ExportJSPromiseResult = "JS_PromiseResult"
)

// Job exports
const (
	// ExportJSExecutePendingJob executes a pending job.
	// Signature: JS_ExecutePendingJob(rt: i32, pctx: i32) -> i32
	ExportJSExecutePendingJob = "JS_ExecutePendingJob"

	// ExportJSIsJobPending checks if there are pending jobs.
	// Signature: JS_IsJobPending(rt: i32) -> i32
	ExportJSIsJobPending = "JS_IsJobPending"
)

// Module exports
const (
	// ExportJSSetModuleLoaderFunc sets the module loader function.
	// Signature: JS_SetModuleLoaderFunc(rt: i32, normalize: i32, loader: i32, opaque: i32) -> void
	ExportJSSetModuleLoaderFunc = "JS_SetModuleLoaderFunc"

	// ExportJSNewCModule creates a new C module.
	// Signature: JS_NewCModule(ctx: i32, name_str: i32, func: i32) -> i32 (JSModuleDef*)
	ExportJSNewCModule = "JS_NewCModule"

	// ExportJSAddModuleExport adds an export to a module.
	// Signature: JS_AddModuleExport(ctx: i32, m: i32, name_str: i32) -> i32
	ExportJSAddModuleExport = "JS_AddModuleExport"

	// ExportJSSetModuleExport sets an export value.
	// Signature: JS_SetModuleExport(ctx: i32, m: i32, name_str: i32, val: i64) -> i32
	ExportJSSetModuleExport = "JS_SetModuleExport"
)

// Reactor mode exports (qjs.c with QJS_WASI_REACTOR)
// These functions initialize and manage a global QuickJS runtime/context.
const (
	// ExportQJSInitArgv initializes the reactor with CLI arguments.
	// Pass same arguments as CLI: e.g. ["qjs", "--std", "script.js"]
	// Supported flags:
	//   --std        Load std, os, bjson modules as globals
	//   -m, --module Treat script as ES module
	//   -e, --eval   Evaluate expression
	//   -I           Include file before script
	// Signature: qjs_init_argv(argc: i32, argv: i32) -> i32
	// Returns: 0 on success, -1 on error
	ExportQJSInitArgv = "qjs_init_argv"

	// ExportQJSGetContext gets the reactor's JSContext.
	// Use with js_std_loop_once, JS_Eval, etc.
	// Signature: qjs_get_context() -> i32 (JSContext*)
	// Returns: JSContext pointer, or NULL if not initialized
	ExportQJSGetContext = "qjs_get_context"

	// ExportQJSDestroy cleans up the reactor runtime.
	// Signature: qjs_destroy() -> void
	ExportQJSDestroy = "qjs_destroy"
)

// Standard library exports (quickjs-libc.h)
const (
	// ExportJSInitModuleStd initializes the 'std' module.
	// Signature: js_init_module_std(ctx: i32, module_name: i32) -> i32 (JSModuleDef*)
	ExportJSInitModuleStd = "js_init_module_std"

	// ExportJSInitModuleOS initializes the 'os' module.
	// Signature: js_init_module_os(ctx: i32, module_name: i32) -> i32 (JSModuleDef*)
	ExportJSInitModuleOS = "js_init_module_os"

	// ExportJSInitModuleBJSON initializes the 'bjson' module.
	// Signature: js_init_module_bjson(ctx: i32, module_name: i32) -> i32 (JSModuleDef*)
	ExportJSInitModuleBJSON = "js_init_module_bjson"

	// ExportJSStdInitHandlers initializes std handlers.
	// Signature: js_std_init_handlers(rt: i32) -> void
	ExportJSStdInitHandlers = "js_std_init_handlers"

	// ExportJSStdFreeHandlers frees std handlers.
	// Signature: js_std_free_handlers(rt: i32) -> void
	ExportJSStdFreeHandlers = "js_std_free_handlers"

	// ExportJSStdAddHelpers adds std helper functions (console, print, etc).
	// Signature: js_std_add_helpers(ctx: i32, argc: i32, argv: i32) -> void
	ExportJSStdAddHelpers = "js_std_add_helpers"

	// ExportJSStdLoop runs the event loop until no more work.
	// Signature: js_std_loop(ctx: i32) -> i32
	ExportJSStdLoop = "js_std_loop"

	// ExportJSStdLoopOnce runs one iteration of the event loop.
	// Signature: js_std_loop_once(ctx: i32) -> i32
	// Returns:
	//   > 0: Next timer fires in this many milliseconds
	//     0: More work pending; call again immediately
	//    -1: No pending work; event loop is idle
	//    -2: An exception occurred
	ExportJSStdLoopOnce = "js_std_loop_once"

	// ExportJSStdPollIO polls for I/O events.
	// Signature: js_std_poll_io(ctx: i32, timeout_ms: i32) -> i32
	// Parameters:
	//   timeout_ms: 0 = non-blocking, >0 = wait up to N ms, -1 = block
	// Returns:
	//   0: Success (handler invoked or no handlers)
	//   -1: Error
	//   -2: Exception in handler
	ExportJSStdPollIO = "js_std_poll_io"

	// ExportJSStdAwait awaits a promise.
	// Signature: js_std_await(ctx: i32, obj: i64) -> i64 (JSValue)
	ExportJSStdAwait = "js_std_await"

	// ExportJSStdDumpError dumps an error to stderr.
	// Signature: js_std_dump_error(ctx: i32) -> void
	ExportJSStdDumpError = "js_std_dump_error"

	// ExportJSStdEvalBinary evaluates bytecode.
	// Signature: js_std_eval_binary(ctx: i32, buf: i32, buf_len: i64, flags: i32) -> void
	ExportJSStdEvalBinary = "js_std_eval_binary"

	// ExportJSLoadFile loads a file into memory.
	// Signature: js_load_file(ctx: i32, pbuf_len: i32, filename: i32) -> i32 (uint8_t*)
	ExportJSLoadFile = "js_load_file"

	// ExportJSModuleSetImportMeta sets import.meta for a module.
	// Signature: js_module_set_import_meta(ctx: i32, func_val: i64, use_realpath: i32, is_main: i32) -> i32
	ExportJSModuleSetImportMeta = "js_module_set_import_meta"

	// ExportJSModuleLoader is the default module loader function.
	// Signature: js_module_loader(ctx: i32, module_name: i32, opaque: i32) -> i32 (JSModuleDef*)
	ExportJSModuleLoader = "js_module_loader"

	// ExportJSStdPromiseRejectionTracker tracks unhandled promise rejections.
	// Signature: js_std_promise_rejection_tracker(ctx: i32, promise: i64, reason: i64, is_handled: i32, opaque: i32) -> void
	ExportJSStdPromiseRejectionTracker = "js_std_promise_rejection_tracker"
)

// JS_EVAL_* flags for JS_Eval
const (
	JSEvalTypeGlobal    = 0 << 0 // global code (default)
	JSEvalTypeModule    = 1 << 0 // module code
	JSEvalFlagStrict    = 1 << 3 // force 'use strict'
	JSEvalFlagCompile   = 1 << 5 // compile only, do not run
	JSEvalFlagBacktrace = 1 << 6 // enable backtrace
)

// Loop result constants from js_std_loop_once()
const (
	// LoopResultIdle indicates no pending work (-1)
	LoopResultIdle = -1
	// LoopResultError indicates an error occurred (-2)
	LoopResultError = -2
)

// JSValue tag constants (matching quickjs.h)
const (
	JSTagFirst         = -11
	JSTagBigDecimal    = -11
	JSTagBigInt        = -10
	JSTagBigFloat      = -9
	JSTagSymbol        = -8
	JSTagString        = -7
	JSTagModule        = -3
	JSTagFunctionByte  = -2
	JSTagObject        = -1
	JSTagInt           = 0
	JSTagBool          = 1
	JSTagNull          = 2
	JSTagUndefined     = 3
	JSTagUninitialized = 4
	JSTagCatchOffset   = 5
	JSTagException     = 6
	JSTagFloat64       = 7
)

// JSValue helpers
// JSValue is a 64-bit value where:
// - For tagged values (tag < 0): lower 32 bits = pointer, upper 32 bits = tag
// - For immediate values (tag >= 0): depends on tag type
const (
	// JSValueUndefined is the undefined value
	JSValueUndefined = uint64(JSTagUndefined) << 32
	// JSValueNull is the null value
	JSValueNull = uint64(JSTagNull) << 32
	// JSValueException is the exception value (indicates an exception was thrown)
	JSValueException = uint64(JSTagException) << 32
)
