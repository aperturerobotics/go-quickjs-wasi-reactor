// Package quickjs provides a high-level Go API for running JavaScript
// using the QuickJS WASI reactor module with wazero.
package quickjs

import (
	"context"
	"encoding/binary"
	"errors"
	"time"

	quickjswasi "github.com/paralin/go-quickjs-wasi"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// QuickJS wraps a QuickJS WASI reactor module providing a high-level API
// for JavaScript execution.
type QuickJS struct {
	runtime  wazero.Runtime
	mod      api.Module
	init     api.Function
	initArgv api.Function
	eval     api.Function
	loopOnce api.Function
	destroy  api.Function
	malloc   api.Function
	free     api.Function
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
	// This is required for reactor modules to properly initialize WASI state.
	initializeFn := mod.ExportedFunction("_initialize")
	if initializeFn != nil {
		if _, err := initializeFn.Call(ctx); err != nil {
			_ = mod.Close(ctx)
			return nil, errors.New("_initialize failed: " + err.Error())
		}
	}

	q := &QuickJS{
		runtime:  r,
		mod:      mod,
		init:     mod.ExportedFunction(quickjswasi.ExportInit),
		initArgv: mod.ExportedFunction(quickjswasi.ExportInitArgv),
		eval:     mod.ExportedFunction(quickjswasi.ExportEval),
		loopOnce: mod.ExportedFunction(quickjswasi.ExportLoopOnce),
		destroy:  mod.ExportedFunction(quickjswasi.ExportDestroy),
		malloc:   mod.ExportedFunction(quickjswasi.ExportMalloc),
		free:     mod.ExportedFunction(quickjswasi.ExportFree),
	}

	// Validate required exports are present
	if q.init == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportInit)
	}
	if q.eval == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportEval)
	}
	if q.loopOnce == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportLoopOnce)
	}
	if q.destroy == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportDestroy)
	}
	if q.malloc == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportMalloc)
	}
	if q.free == nil {
		return nil, errors.New("missing export: " + quickjswasi.ExportFree)
	}

	return q, nil
}

// stdModuleInit is the JavaScript code to import and expose the std modules globally.
const stdModuleInit = `import * as bjson from 'qjs:bjson';
import * as std from 'qjs:std';
import * as os from 'qjs:os';
globalThis.bjson = bjson;
globalThis.std = std;
globalThis.os = os;
`

// Init initializes the QuickJS runtime with an empty context.
// This must be called before Eval.
func (q *QuickJS) Init(ctx context.Context) error {
	results, err := q.init.Call(ctx)
	if err != nil {
		return err
	}
	if len(results) > 0 && results[0] != 0 {
		return errors.New("qjs_init failed")
	}
	return nil
}

// InitArgv initializes the QuickJS runtime with command-line arguments.
// This can be used to pass flags like --std or specify scripts to load via WASI filesystem.
// Example: InitArgv(ctx, []string{"qjs", "--std"}) to enable std module via CLI args.
func (q *QuickJS) InitArgv(ctx context.Context, args []string) error {
	if q.initArgv == nil {
		return errors.New("qjs_init_argv not available")
	}

	argc := uint64(len(args))
	if argc == 0 {
		return q.Init(ctx)
	}

	// Allocate and write each argument string
	argPtrs := make([]uint32, len(args))
	for i, arg := range args {
		argBytes := []byte(arg)
		ptrResults, err := q.malloc.Call(ctx, uint64(len(argBytes)+1))
		if err != nil {
			// Free previously allocated args
			for j := 0; j < i; j++ {
				q.free.Call(ctx, uint64(argPtrs[j]))
			}
			return err
		}
		argPtrs[i] = uint32(ptrResults[0])
		if argPtrs[i] == 0 {
			for j := 0; j < i; j++ {
				q.free.Call(ctx, uint64(argPtrs[j]))
			}
			return errors.New("malloc returned null for arg")
		}
		if !q.mod.Memory().Write(argPtrs[i], append(argBytes, 0)) {
			for j := 0; j <= i; j++ {
				q.free.Call(ctx, uint64(argPtrs[j]))
			}
			return errors.New("failed to write arg to memory")
		}
	}

	// Allocate argv array (array of pointers)
	argvPtrResults, err := q.malloc.Call(ctx, uint64(len(args)*4))
	if err != nil {
		for _, ptr := range argPtrs {
			q.free.Call(ctx, uint64(ptr))
		}
		return err
	}
	argvPtr := uint32(argvPtrResults[0])
	if argvPtr == 0 {
		for _, ptr := range argPtrs {
			q.free.Call(ctx, uint64(ptr))
		}
		return errors.New("malloc returned null for argv")
	}

	// Write pointer array
	for i, ptr := range argPtrs {
		ptrBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(ptrBytes, ptr)
		if !q.mod.Memory().Write(argvPtr+uint32(i*4), ptrBytes) {
			q.free.Call(ctx, uint64(argvPtr))
			for _, p := range argPtrs {
				q.free.Call(ctx, uint64(p))
			}
			return errors.New("failed to write argv to memory")
		}
	}

	// Call qjs_init_argv(argc, argv)
	results, err := q.initArgv.Call(ctx, argc, uint64(argvPtr))

	// Free all allocated memory
	q.free.Call(ctx, uint64(argvPtr))
	for _, ptr := range argPtrs {
		q.free.Call(ctx, uint64(ptr))
	}

	if err != nil {
		return err
	}
	if len(results) > 0 && results[0] != 0 {
		return errors.New("qjs_init_argv failed")
	}
	return nil
}

// InitStdModule initializes the QuickJS runtime and loads the std modules.
// This makes std, os, and bjson available as globals.
// This must be called before Eval.
func (q *QuickJS) InitStdModule(ctx context.Context) error {
	if err := q.Init(ctx); err != nil {
		return err
	}
	return q.Eval(ctx, stdModuleInit, true)
}

// Eval evaluates JavaScript code.
// If isModule is true, the code is treated as an ES module.
func (q *QuickJS) Eval(ctx context.Context, code string, isModule bool) error {
	return q.EvalWithFilename(ctx, code, "<eval>", isModule)
}

// EvalWithFilename evaluates JavaScript code with a custom filename for error messages.
// If isModule is true, the code is treated as an ES module.
func (q *QuickJS) EvalWithFilename(ctx context.Context, code string, filename string, isModule bool) error {
	codeBytes := []byte(code)
	codeLen := uint64(len(codeBytes))
	filenameBytes := []byte(filename)

	// Allocate memory for the code (+ 1 for null terminator)
	codePtrResults, err := q.malloc.Call(ctx, codeLen+1)
	if err != nil {
		return err
	}
	codePtr := uint32(codePtrResults[0])
	if codePtr == 0 {
		return errors.New("malloc returned null for code")
	}

	// Allocate memory for the filename (+ 1 for null terminator)
	filenamePtrResults, err := q.malloc.Call(ctx, uint64(len(filenameBytes)+1))
	if err != nil {
		q.free.Call(ctx, uint64(codePtr))
		return err
	}
	filenamePtr := uint32(filenamePtrResults[0])
	if filenamePtr == 0 {
		q.free.Call(ctx, uint64(codePtr))
		return errors.New("malloc returned null for filename")
	}

	// Write code to WASM memory with null terminator
	if !q.mod.Memory().Write(codePtr, append(codeBytes, 0)) {
		q.free.Call(ctx, uint64(codePtr))
		q.free.Call(ctx, uint64(filenamePtr))
		return errors.New("failed to write code to memory")
	}

	// Write filename to WASM memory with null terminator
	if !q.mod.Memory().Write(filenamePtr, append(filenameBytes, 0)) {
		q.free.Call(ctx, uint64(codePtr))
		q.free.Call(ctx, uint64(filenamePtr))
		return errors.New("failed to write filename to memory")
	}

	// Call eval: qjs_eval(code, len, filename, is_module)
	isModuleInt := uint64(0)
	if isModule {
		isModuleInt = 1
	}
	evalResults, err := q.eval.Call(ctx, uint64(codePtr), codeLen, uint64(filenamePtr), isModuleInt)

	// Free the allocated memory
	q.free.Call(ctx, uint64(codePtr))
	q.free.Call(ctx, uint64(filenamePtr))

	if err != nil {
		return err
	}

	// Check result (0 = success for eval)
	if len(evalResults) > 0 && int32(evalResults[0]) != 0 {
		return errors.New("eval failed")
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
	results, err := q.loopOnce.Call(ctx)
	if err != nil {
		return LoopError, err
	}
	if len(results) == 0 {
		return LoopError, errors.New("qjs_loop_once returned no result")
	}
	return LoopResult(int32(results[0])), nil
}

// RunLoop runs the event loop until idle or context is canceled.
// This blocks until all JavaScript execution completes (no more pending
// timers or microtasks) or the context is canceled.
func (q *QuickJS) RunLoop(ctx context.Context) error {
	for {
		// Check for context cancellation
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
			// No more work to do
			return nil
		case result == LoopError:
			return errors.New("JavaScript error occurred")
		case result == 0:
			// More microtasks pending, continue immediately
			continue
		case result > 0:
			// Wait for next timer or context cancellation
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
	if q.destroy != nil {
		_, err := q.destroy.Call(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
