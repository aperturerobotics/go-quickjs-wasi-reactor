// Package quickjs provides a high-level Go API for running JavaScript
// using the QuickJS WASI reactor module with wazero.
package quickjs

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"time"

	quickjswasi "github.com/aperturerobotics/go-quickjs-wasi-reactor"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// QuickJS wraps a QuickJS WASI reactor module providing a high-level API
// for JavaScript execution.
type QuickJS struct {
	runtime wazero.Runtime
	mod     api.Module

	// Memory management
	malloc  api.Function
	free    api.Function
	realloc api.Function

	// Core runtime
	jsNewRuntime  api.Function
	jsFreeRuntime api.Function
	jsNewContext  api.Function
	jsFreeContext api.Function
	jsGetRuntime  api.Function

	// Evaluation
	jsEval api.Function

	// Bytecode
	jsWriteObject api.Function
	jsReadObject  api.Function
	jsFree        api.Function

	// Value management
	jsFreeValue       api.Function
	jsToCStringLen2   api.Function
	jsFreeCString     api.Function
	jsGetPropertyStr  api.Function
	jsGetGlobalObject api.Function

	// Exception handling
	jsGetException api.Function
	jsHasException api.Function
	jsIsError      api.Function

	// Jobs
	jsExecutePendingJob api.Function
	jsIsJobPending      api.Function

	// Standard library
	jsInitModuleStd       api.Function
	jsInitModuleOS        api.Function
	jsInitModuleBJSON     api.Function
	jsStdInitHandlers     api.Function
	jsStdFreeHandlers     api.Function
	jsStdAddHelpers       api.Function
	jsStdLoopOnce         api.Function
	jsStdPollIO           api.Function
	jsStdDumpError        api.Function
	jsModuleLoader        api.Function
	jsSetModuleLoaderFunc api.Function
	jsModuleSetImportMeta api.Function

	// Reactor initialization exports
	qjsInitArgv   api.Function
	qjsGetContext api.Function
	qjsDestroy    api.Function

	// Runtime state (managed by QuickJS struct)
	rtPtr  uint32 // JSRuntime*
	ctxPtr uint32 // JSContext*
}

// CompileQuickJS compiles the embedded QuickJS WASM module.
// The compiled module can be reused across multiple QuickJS instances for better performance.
// The caller should also instantiate WASI on the runtime before using the compiled module.
func CompileQuickJS(ctx context.Context, r wazero.Runtime) (wazero.CompiledModule, error) {
	return r.CompileModule(ctx, quickjswasi.QuickJSWASM)
}

// NewQuickJS creates a new QuickJS instance using the embedded WASM reactor.
// The provided config is used for module instantiation (stdin, stdout, stderr, fs, etc.).
// Call Close() when done to release resources.
func NewQuickJS(ctx context.Context, r wazero.Runtime, config wazero.ModuleConfig) (*QuickJS, error) {
	// Instantiate WASI - required for the reactor module
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		return nil, err
	}

	// Compile the module
	compiled, err := CompileQuickJS(ctx, r)
	if err != nil {
		return nil, err
	}

	return NewQuickJSWithModule(ctx, r, compiled, config)
}

// NewQuickJSWithModule creates a new QuickJS instance using a pre-compiled module.
// This is useful when you want to reuse a compiled module across multiple instances
// for better startup performance.
//
// Prerequisites:
//   - WASI must be instantiated on the runtime (wasi_snapshot_preview1.Instantiate)
//   - The compiled module must be from CompileQuickJS or compiled from quickjswasi.QuickJSWASM
//
// The provided config is used for module instantiation (stdin, stdout, stderr, fs, etc.).
// Call Close() when done to release resources.
func NewQuickJSWithModule(ctx context.Context, r wazero.Runtime, compiled wazero.CompiledModule, config wazero.ModuleConfig) (*QuickJS, error) {
	// Instantiate without running _start (reactor mode)
	mod, err := r.InstantiateModule(ctx, compiled, config.WithName(quickjswasi.QuickJSWASMFilename))
	if err != nil {
		return nil, err
	}

	// Call _initialize to set up WASI environment (env vars, args, etc.)
	initializeFn := mod.ExportedFunction("_initialize")
	if initializeFn != nil {
		if _, err := initializeFn.Call(ctx); err != nil {
			_ = mod.Close(ctx)
			return nil, errors.New("_initialize failed: " + err.Error())
		}
	}

	q := &QuickJS{
		runtime: r,
		mod:     mod,

		// Memory management
		malloc:  mod.ExportedFunction(quickjswasi.ExportMalloc),
		free:    mod.ExportedFunction(quickjswasi.ExportFree),
		realloc: mod.ExportedFunction(quickjswasi.ExportRealloc),

		// Core runtime
		jsNewRuntime:  mod.ExportedFunction(quickjswasi.ExportJSNewRuntime),
		jsFreeRuntime: mod.ExportedFunction(quickjswasi.ExportJSFreeRuntime),
		jsNewContext:  mod.ExportedFunction(quickjswasi.ExportJSNewContext),
		jsFreeContext: mod.ExportedFunction(quickjswasi.ExportJSFreeContext),
		jsGetRuntime:  mod.ExportedFunction(quickjswasi.ExportJSGetRuntime),

		// Evaluation
		jsEval: mod.ExportedFunction(quickjswasi.ExportJSEval),

		// Bytecode
		jsWriteObject: mod.ExportedFunction(quickjswasi.ExportJSWriteObject),
		jsReadObject:  mod.ExportedFunction(quickjswasi.ExportJSReadObject),
		jsFree:        mod.ExportedFunction(quickjswasi.ExportJSFree),

		// Value management
		jsFreeValue:       mod.ExportedFunction(quickjswasi.ExportJSFreeValue),
		jsToCStringLen2:   mod.ExportedFunction(quickjswasi.ExportJSToCStringLen2),
		jsFreeCString:     mod.ExportedFunction(quickjswasi.ExportJSFreeCString),
		jsGetPropertyStr:  mod.ExportedFunction(quickjswasi.ExportJSGetPropertyStr),
		jsGetGlobalObject: mod.ExportedFunction(quickjswasi.ExportJSGetGlobalObject),

		// Exception handling
		jsGetException: mod.ExportedFunction(quickjswasi.ExportJSGetException),
		jsHasException: mod.ExportedFunction(quickjswasi.ExportJSHasException),
		jsIsError:      mod.ExportedFunction(quickjswasi.ExportJSIsError),

		// Jobs
		jsExecutePendingJob: mod.ExportedFunction(quickjswasi.ExportJSExecutePendingJob),
		jsIsJobPending:      mod.ExportedFunction(quickjswasi.ExportJSIsJobPending),

		// Standard library
		jsInitModuleStd:       mod.ExportedFunction(quickjswasi.ExportJSInitModuleStd),
		jsInitModuleOS:        mod.ExportedFunction(quickjswasi.ExportJSInitModuleOS),
		jsInitModuleBJSON:     mod.ExportedFunction(quickjswasi.ExportJSInitModuleBJSON),
		jsStdInitHandlers:     mod.ExportedFunction(quickjswasi.ExportJSStdInitHandlers),
		jsStdFreeHandlers:     mod.ExportedFunction(quickjswasi.ExportJSStdFreeHandlers),
		jsStdAddHelpers:       mod.ExportedFunction(quickjswasi.ExportJSStdAddHelpers),
		jsStdLoopOnce:         mod.ExportedFunction(quickjswasi.ExportJSStdLoopOnce),
		jsStdPollIO:           mod.ExportedFunction(quickjswasi.ExportJSStdPollIO),
		jsStdDumpError:        mod.ExportedFunction(quickjswasi.ExportJSStdDumpError),
		jsModuleLoader:        mod.ExportedFunction(quickjswasi.ExportJSModuleLoader),
		jsSetModuleLoaderFunc: mod.ExportedFunction(quickjswasi.ExportJSSetModuleLoaderFunc),
		jsModuleSetImportMeta: mod.ExportedFunction(quickjswasi.ExportJSModuleSetImportMeta),

		// Reactor initialization
		qjsInitArgv:   mod.ExportedFunction(quickjswasi.ExportQJSInitArgv),
		qjsGetContext: mod.ExportedFunction(quickjswasi.ExportQJSGetContext),
		qjsDestroy:    mod.ExportedFunction(quickjswasi.ExportQJSDestroy),
	}

	// Validate required exports
	if q.malloc == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportMalloc)
	}
	if q.free == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportFree)
	}
	if q.jsNewRuntime == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportJSNewRuntime)
	}
	if q.jsFreeRuntime == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportJSFreeRuntime)
	}
	if q.jsNewContext == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportJSNewContext)
	}
	if q.jsFreeContext == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportJSFreeContext)
	}
	if q.jsEval == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportJSEval)
	}
	if q.jsFreeValue == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportJSFreeValue)
	}
	if q.jsStdLoopOnce == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportJSStdLoopOnce)
	}
	if q.qjsInitArgv == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportQJSInitArgv)
	}
	if q.qjsGetContext == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportQJSGetContext)
	}
	if q.qjsDestroy == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportQJSDestroy)
	}

	return q, nil
}

// allocString allocates a null-terminated string in WASM memory and returns the pointer.
// The caller must free the returned pointer.
func (q *QuickJS) allocString(ctx context.Context, s string) (uint32, error) {
	bytes := []byte(s)
	results, err := q.malloc.Call(ctx, uint64(len(bytes)+1))
	if err != nil {
		return 0, err
	}
	ptr := uint32(results[0])
	if ptr == 0 {
		return 0, errors.New("malloc returned null")
	}
	if !q.mod.Memory().Write(ptr, append(bytes, 0)) {
		q.free.Call(ctx, uint64(ptr))
		return 0, errors.New("failed to write string to memory")
	}
	return ptr, nil
}

// freePtr frees a pointer in WASM memory.
func (q *QuickJS) freePtr(ctx context.Context, ptr uint32) {
	if ptr != 0 {
		q.free.Call(ctx, uint64(ptr))
	}
}

// Init initializes the QuickJS runtime and context with optional command-line arguments.
// This must be called before Eval. Uses qjs_init_argv which sets up the module loader.
//
// Supported flags in args:
//   - --std: Load std, os, bjson modules as globals
//   - -m, --module: Treat script as ES module
//   - -e, --eval: Evaluate expression
//   - -I, --include: Include file before script
//
// Example: Init(ctx, []string{"qjs", "--std", "/boot/script.js"})
// Pass nil for default initialization.
func (q *QuickJS) Init(ctx context.Context, args []string) error {
	if q.ctxPtr != 0 {
		return errors.New("QuickJS already initialized")
	}

	argc := len(args)
	if argc == 0 {
		args = []string{"qjs"}
		argc = 1
	}

	// Allocate argv strings
	argPtrs := make([]uint32, argc)
	for i, arg := range args {
		ptr, err := q.allocString(ctx, arg)
		if err != nil {
			for j := 0; j < i; j++ {
				q.freePtr(ctx, argPtrs[j])
			}
			return err
		}
		argPtrs[i] = ptr
	}

	// Allocate argv pointer array
	argvResults, err := q.malloc.Call(ctx, uint64(argc*4))
	if err != nil {
		for _, ptr := range argPtrs {
			q.freePtr(ctx, ptr)
		}
		return err
	}
	argvPtr := uint32(argvResults[0])
	if argvPtr == 0 {
		for _, ptr := range argPtrs {
			q.freePtr(ctx, ptr)
		}
		return errors.New("malloc returned null for argv")
	}

	// Write argv pointers
	for i, ptr := range argPtrs {
		ptrBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(ptrBytes, ptr)
		q.mod.Memory().Write(argvPtr+uint32(i*4), ptrBytes)
	}

	// Call qjs_init_argv
	initResults, err := q.qjsInitArgv.Call(ctx, uint64(argc), uint64(argvPtr))

	// Free argv memory
	q.freePtr(ctx, argvPtr)
	for _, ptr := range argPtrs {
		q.freePtr(ctx, ptr)
	}

	if err != nil {
		return errors.New("qjs_init_argv failed: " + err.Error())
	}
	if int32(initResults[0]) != 0 {
		return errors.New("qjs_init_argv returned error")
	}

	// Get the context pointer
	ctxResults, err := q.qjsGetContext.Call(ctx)
	if err != nil {
		return errors.New("qjs_get_context failed: " + err.Error())
	}
	q.ctxPtr = uint32(ctxResults[0])
	if q.ctxPtr == 0 {
		return errors.New("qjs_get_context returned null")
	}

	// Get the runtime pointer
	if q.jsGetRuntime != nil {
		rtResults, err := q.jsGetRuntime.Call(ctx, uint64(q.ctxPtr))
		if err == nil && len(rtResults) > 0 {
			q.rtPtr = uint32(rtResults[0])
		}
	}

	return nil
}

// jsValueIsException checks if a JSValue is an exception
func jsValueIsException(val uint64) bool {
	tag := int32(val >> 32)
	return tag == quickjswasi.JSTagException
}

// Eval evaluates JavaScript code.
// If isModule is true, the code is treated as an ES module.
func (q *QuickJS) Eval(ctx context.Context, code string, isModule bool) error {
	return q.EvalWithFilename(ctx, code, "<eval>", isModule)
}

// EvalWithFilename evaluates JavaScript code with a custom filename for error messages.
// If isModule is true, the code is treated as an ES module.
func (q *QuickJS) EvalWithFilename(ctx context.Context, code string, filename string, isModule bool) error {
	if q.ctxPtr == 0 {
		return errors.New("QuickJS not initialized")
	}

	// Allocate code string (allocString adds null terminator)
	codePtr, err := q.allocString(ctx, code)
	if err != nil {
		return err
	}

	// Allocate filename string
	filenamePtr, err := q.allocString(ctx, filename)
	if err != nil {
		q.freePtr(ctx, codePtr)
		return err
	}

	// Determine eval flags
	evalFlags := quickjswasi.JSEvalTypeGlobal
	if isModule {
		evalFlags = quickjswasi.JSEvalTypeModule
	}

	// Call JS_Eval(ctx, input, input_len, filename, eval_flags) -> JSValue
	evalResults, err := q.jsEval.Call(ctx,
		uint64(q.ctxPtr),
		uint64(codePtr),
		uint64(len(code)),
		uint64(filenamePtr),
		uint64(evalFlags))

	// Free allocated memory
	q.freePtr(ctx, codePtr)
	q.freePtr(ctx, filenamePtr)

	if err != nil {
		return err
	}

	// Check for exception
	val := evalResults[0]
	if jsValueIsException(val) {
		return q.dumpException(ctx)
	}

	// Free the result value
	q.jsFreeValue.Call(ctx, uint64(q.ctxPtr), val)

	return nil
}

// dumpException writes the pending JavaScript exception to stderr and returns
// an error standing for it.
func (q *QuickJS) dumpException(ctx context.Context) error {
	if q.jsStdDumpError != nil {
		q.jsStdDumpError.Call(ctx, uint64(q.ctxPtr))
	}
	return errors.New("JavaScript exception")
}

// CompileModule compiles ES module source without evaluating it and returns
// the module as bytecode. The module stays registered under name, so a later
// import of name links it instead of loading the file. The bytecode omits the
// source text but keeps line numbers, and LoadModule registers it again in a
// later instance of the same reactor build without parsing the source.
func (q *QuickJS) CompileModule(ctx context.Context, code, name string) ([]byte, error) {
	if q.ctxPtr == 0 {
		return nil, errors.New("QuickJS not initialized")
	}
	if q.jsWriteObject == nil || q.jsFree == nil || q.jsModuleSetImportMeta == nil {
		return nil, errors.New("bytecode exports not available")
	}

	// Compile the module, which registers it under name.
	codePtr, err := q.allocString(ctx, code)
	if err != nil {
		return nil, err
	}
	defer q.freePtr(ctx, codePtr)
	namePtr, err := q.allocString(ctx, name)
	if err != nil {
		return nil, err
	}
	defer q.freePtr(ctx, namePtr)
	evalResults, err := q.jsEval.Call(ctx,
		uint64(q.ctxPtr),
		uint64(codePtr),
		uint64(len(code)),
		uint64(namePtr),
		quickjswasi.JSEvalTypeModule|quickjswasi.JSEvalFlagCompile)
	if err != nil {
		return nil, err
	}
	mod := evalResults[0]
	if jsValueIsException(mod) {
		return nil, q.dumpException(ctx)
	}

	// The context holds the module, so release the returned reference.
	defer q.jsFreeValue.Call(ctx, uint64(q.ctxPtr), mod)
	if err := q.setImportMeta(ctx, mod); err != nil {
		return nil, err
	}

	// Serialize the module into a buffer owned by the context allocator.
	sizePtr, err := q.malloc.Call(ctx, 4)
	if err != nil {
		return nil, err
	}
	defer q.freePtr(ctx, uint32(sizePtr[0]))
	writeResults, err := q.jsWriteObject.Call(ctx,
		uint64(q.ctxPtr),
		sizePtr[0],
		mod,
		quickjswasi.JSWriteObjBytecode|quickjswasi.JSWriteObjStripSource)
	if err != nil {
		return nil, err
	}
	bufPtr := uint32(writeResults[0])
	if bufPtr == 0 {
		return nil, q.dumpException(ctx)
	}
	defer q.jsFree.Call(ctx, uint64(q.ctxPtr), uint64(bufPtr))

	// Copy the bytecode out of guest memory.
	mem := q.mod.Memory()
	size, ok := mem.ReadUint32Le(uint32(sizePtr[0]))
	if !ok {
		return nil, errors.New("bytecode size out of range")
	}
	buf, ok := mem.Read(bufPtr, size)
	if !ok {
		return nil, errors.New("bytecode out of range")
	}
	return bytes.Clone(buf), nil
}

// LoadModule registers a module from bytecode returned by CompileModule,
// under the name it was compiled with, without evaluating it. A later import
// of that name links the module. QuickJS does not validate bytecode, so load
// only bytecode this host wrote with the same reactor build.
func (q *QuickJS) LoadModule(ctx context.Context, bytecode []byte) error {
	if q.ctxPtr == 0 {
		return errors.New("QuickJS not initialized")
	}
	if q.jsReadObject == nil || q.jsModuleSetImportMeta == nil {
		return errors.New("bytecode exports not available")
	}

	// Copy the bytecode into guest memory.
	bufResults, err := q.malloc.Call(ctx, uint64(len(bytecode)))
	if err != nil {
		return err
	}
	bufPtr := uint32(bufResults[0])
	if bufPtr == 0 {
		return errors.New("malloc returned null")
	}
	defer q.freePtr(ctx, bufPtr)
	if !q.mod.Memory().Write(bufPtr, bytecode) {
		return errors.New("failed to write bytecode to memory")
	}

	// Read the module, which registers it under its name.
	readResults, err := q.jsReadObject.Call(ctx,
		uint64(q.ctxPtr),
		uint64(bufPtr),
		uint64(len(bytecode)),
		quickjswasi.JSReadObjBytecode)
	if err != nil {
		return err
	}
	mod := readResults[0]
	if jsValueIsException(mod) {
		return q.dumpException(ctx)
	}
	defer q.jsFreeValue.Call(ctx, uint64(q.ctxPtr), mod)
	if int32(mod>>32) != quickjswasi.JSTagModule {
		return errors.New("bytecode is not a module")
	}
	return q.setImportMeta(ctx, mod)
}

// setImportMeta sets import.meta.url of a compiled module from its name, as the
// QuickJS module loader does.
func (q *QuickJS) setImportMeta(ctx context.Context, mod uint64) error {
	results, err := q.jsModuleSetImportMeta.Call(ctx, uint64(q.ctxPtr), mod, 0, 0)
	if err != nil {
		return err
	}
	if int32(results[0]) < 0 {
		return q.dumpException(ctx)
	}
	return nil
}

// LoopResult represents the result of a single event loop iteration.
type LoopResult int32

const (
	// LoopIdle indicates no pending work.
	LoopIdle LoopResult = -1
	// LoopError indicates an error occurred.
	LoopError LoopResult = -2
)

// IsPending returns true if there is more work to do (timers or microtasks).
func (r LoopResult) IsPending() bool {
	return r >= 0
}

// NextTimerMs returns the milliseconds until the next timer fires.
// Only valid when IsPending() is true and result > 0.
func (r LoopResult) NextTimerMs() int {
	if r > 0 {
		return int(r)
	}
	return 0
}

// LoopOnce runs one iteration of the QuickJS event loop.
// Returns:
//   - LoopResult > 0: next timer fires in N ms
//   - LoopResult == 0: more microtasks pending, call again immediately
//   - LoopIdle (-1): no pending work
//   - LoopError (-2): error occurred
func (q *QuickJS) LoopOnce(ctx context.Context) (LoopResult, error) {
	if q.ctxPtr == 0 {
		return LoopError, errors.New("QuickJS not initialized")
	}

	results, err := q.jsStdLoopOnce.Call(ctx, uint64(q.ctxPtr))
	if err != nil {
		return LoopError, err
	}
	if len(results) == 0 {
		return LoopError, errors.New("js_std_loop_once returned no result")
	}
	return LoopResult(int32(results[0])), nil
}

// PollIO polls for I/O events and invokes registered read/write handlers.
// This must be called when the host knows that stdin (or other fds) have data available,
// otherwise os.setReadHandler callbacks will never fire.
//
// Parameters:
//   - timeoutMs: Poll timeout in milliseconds
//     0 = non-blocking (check and return immediately)
//     >0 = wait up to timeoutMs for I/O events
//     -1 = block indefinitely (not recommended for reactor model)
//
// Returns:
//   - 0: success (handler invoked or no handlers registered)
//   - -1: error or no I/O handlers
//   - -2: not initialized or exception in handler
func (q *QuickJS) PollIO(ctx context.Context, timeoutMs int32) (int32, error) {
	if q.ctxPtr == 0 {
		return -2, errors.New("QuickJS not initialized")
	}
	if q.jsStdPollIO == nil {
		return -1, errors.New("js_std_poll_io not available")
	}
	results, err := q.jsStdPollIO.Call(ctx, uint64(q.ctxPtr), uint64(timeoutMs))
	if err != nil {
		return -2, err
	}
	if len(results) == 0 {
		return -2, errors.New("js_std_poll_io returned no result")
	}
	return int32(results[0]), nil
}

// RunLoop runs the event loop until idle or context is canceled.
// This blocks until all JavaScript execution completes (no more pending
// timers or microtasks) or the context is canceled.
func (q *QuickJS) RunLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := q.LoopOnce(ctx)
		if err != nil {
			return err
		}

		switch {
		case result == LoopIdle:
			return nil
		case result == LoopError:
			return errors.New("JavaScript error occurred")
		case result == 0:
			continue
		case result > 0:
			timer := time.NewTimer(time.Duration(result) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				continue
			}
		}
	}
}

// Close destroys the QuickJS runtime and releases resources.
func (q *QuickJS) Close(ctx context.Context) error {
	if q.ctxPtr != 0 {
		q.qjsDestroy.Call(ctx)
		q.ctxPtr = 0
		q.rtPtr = 0
	}
	return nil
}
