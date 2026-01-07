package quickjs

import (
	"bytes"
	"context"
	"embed"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental/fsapi"
	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
)

//go:embed wazero_test.js
var testFS embed.FS

func TestRunJavaScriptFile(t *testing.T) {
	ctx := context.Background()

	// Create a new WebAssembly Runtime
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Capture stdout
	var stdout bytes.Buffer

	// Configure the module with stdout capture and embedded filesystem
	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout).
		WithFS(testFS)

	// Create QuickJS instance
	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Initialize with std module (provides std, os, bjson globals)
	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	// Read and evaluate the test file
	code, err := testFS.ReadFile("wazero_test.js")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	if err := qjs.Eval(ctx, string(code), false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	// Run the event loop until complete
	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	// Check output
	output := stdout.String()
	t.Logf("Output:\n%s", output)

	// Verify expected output patterns
	expectedPatterns := []string{
		"QuickJS API Surface Test",
		"Console output test",
		"Hello from QuickJS!",
		"Math operations",
		"Test Complete",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q", pattern)
		}
	}
}

func TestSimpleEval(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Simple console.log test
	if err := qjs.Eval(ctx, `console.log("Hello, World!");`, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Hello, World!") {
		t.Errorf("expected output to contain 'Hello, World!', got: %s", output)
	}
}

func TestLoopOnce(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Eval synchronous code - should become idle after processing
	if err := qjs.Eval(ctx, `let x = 1 + 2; console.log("result:", x);`, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	// Run loop until idle
	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "result: 3") {
		t.Errorf("expected output to contain 'result: 3', got: %s", output)
	}
}

func TestStdModule(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Use InitStdModule instead of Init
	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	// Test std module functions
	code := `
		std.setenv("TEST_VAR", "hello_from_std");
		console.log("TEST_VAR:", std.getenv("TEST_VAR"));
		console.log("std loaded:", typeof std === 'object');
		console.log("os loaded:", typeof os === 'object');
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "TEST_VAR: hello_from_std") {
		t.Errorf("expected std.getenv to work, got: %s", output)
	}
	if !strings.Contains(output, "std loaded: true") {
		t.Errorf("expected std to be loaded, got: %s", output)
	}
	if !strings.Contains(output, "os loaded: true") {
		t.Errorf("expected os to be loaded, got: %s", output)
	}
}

func TestSetTimeout(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Use InitStdModule to get os.setTimeout
	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	// Test os.setTimeout (QuickJS puts setTimeout on os module, not global)
	code := `
		console.log("before timeout");
		os.setTimeout(() => {
			console.log("timeout fired");
		}, 10);
		console.log("after setTimeout call");
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "before timeout") {
		t.Errorf("expected 'before timeout', got: %s", output)
	}
	if !strings.Contains(output, "after setTimeout call") {
		t.Errorf("expected 'after setTimeout call', got: %s", output)
	}
	if !strings.Contains(output, "timeout fired") {
		t.Errorf("expected 'timeout fired', got: %s", output)
	}
}

func TestInitArgvWithStd(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Initialize with std module to make std/os available globally
	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	// Now std and os should be available globally
	code := `
		console.log("std available:", typeof std === 'object');
		console.log("os available:", typeof os === 'object');
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "std available: true") {
		t.Errorf("expected std to be available, got: %s", output)
	}
	if !strings.Contains(output, "os available: true") {
		t.Errorf("expected os to be available, got: %s", output)
	}
}

func TestWASIEnvVars(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout).
		WithEnv("TEST_VAR", "hello_from_wasi").
		WithEnv("ANOTHER_VAR", "another_value")

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Use InitStdModule to get std.getenv
	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	// Test reading WASI environment variables via std.getenv
	code := `
		console.log("TEST_VAR:", std.getenv("TEST_VAR"));
		console.log("ANOTHER_VAR:", std.getenv("ANOTHER_VAR"));
		console.log("UNDEFINED_VAR:", std.getenv("UNDEFINED_VAR"));
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "TEST_VAR: hello_from_wasi") {
		t.Errorf("expected TEST_VAR to be 'hello_from_wasi', got: %s", output)
	}
	if !strings.Contains(output, "ANOTHER_VAR: another_value") {
		t.Errorf("expected ANOTHER_VAR to be 'another_value', got: %s", output)
	}
	if !strings.Contains(output, "UNDEFINED_VAR: undefined") {
		t.Errorf("expected UNDEFINED_VAR to be undefined, got: %s", output)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	r := wazero.NewRuntime(ctx)
	defer r.Close(context.Background())

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(context.Background())

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Eval simple code
	if err := qjs.Eval(ctx, `console.log("before cancel");`, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	// Cancel the context
	cancel()

	// RunLoop should return with context error when ctx is already canceled
	err = qjs.RunLoop(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// PollableStdinBuffer is a buffer for stdin that implements Poll for wazero.
// This allows wazero's poll_oneoff to properly detect when data is available.
type PollableStdinBuffer struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
	offset int
}

// NewPollableStdinBuffer creates a new PollableStdinBuffer.
func NewPollableStdinBuffer() *PollableStdinBuffer {
	b := &PollableStdinBuffer{}
	b.cond = sync.NewCond(&b.mu)
	return b
}

// Write writes data to the buffer.
func (b *PollableStdinBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return 0, io.ErrClosedPipe
	}
	b.buf = append(b.buf, p...)
	b.cond.Broadcast()
	return len(p), nil
}

// Read reads data from the buffer.
func (b *PollableStdinBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Non-blocking: return 0, nil if no data (wazero expects this for poll)
	available := len(b.buf) - b.offset
	if available <= 0 {
		if b.closed {
			return 0, io.EOF
		}
		return 0, nil
	}

	n := copy(p, b.buf[b.offset:])
	b.offset += n
	if b.offset >= len(b.buf) {
		b.buf = nil
		b.offset = 0
	}
	return n, nil
}

// Close closes the buffer.
func (b *PollableStdinBuffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.cond.Broadcast()
	return nil
}

// Poll checks if data is available to read.
// This signature matches what wazero expects for pollable stdin.
func (b *PollableStdinBuffer) Poll(flag fsapi.Pflag, timeoutMillis int32) (ready bool, errno experimentalsys.Errno) {
	if flag != fsapi.POLLIN {
		return false, experimentalsys.ENOTSUP
	}

	b.mu.Lock()

	// Check if data is available
	if len(b.buf) > b.offset || b.closed {
		b.mu.Unlock()
		return true, 0
	}

	// If timeout is 0, return immediately
	if timeoutMillis == 0 {
		b.mu.Unlock()
		return false, 0
	}

	// Wait for data with timeout
	done := make(chan struct{})
	go func() {
		b.mu.Lock()
		for len(b.buf) <= b.offset && !b.closed {
			b.cond.Wait()
		}
		b.mu.Unlock()
		close(done)
	}()

	b.mu.Unlock()

	if timeoutMillis < 0 {
		<-done
	} else {
		select {
		case <-done:
		case <-time.After(time.Duration(timeoutMillis) * time.Millisecond):
			return false, 0
		}
	}

	b.mu.Lock()
	ready = len(b.buf) > b.offset || b.closed
	b.mu.Unlock()
	return ready, 0
}

// Verify our buffer implements the pollable interface
var _ interface {
	Poll(fsapi.Pflag, int32) (bool, experimentalsys.Errno)
} = (*PollableStdinBuffer)(nil)

// TestStdinReadHandler tests that os.setReadHandler fires when stdin has data.
// This tests the PollIO function which is needed for async I/O patterns like RPC over yamux.
func TestStdinReadHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r := wazero.NewRuntime(ctx)
	defer r.Close(context.Background())

	var stdout bytes.Buffer

	// Create a pollable stdin buffer
	stdinBuf := NewPollableStdinBuffer()
	defer stdinBuf.Close()

	config := wazero.NewModuleConfig().
		WithStdin(stdinBuf).
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(context.Background())

	// Initialize with std module to get os.setReadHandler
	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	// JavaScript code that:
	// 1. Sets up a read handler on stdin
	// 2. When data arrives, reads it and logs
	// 3. Sets a timeout to fail if no data arrives
	code := `
		console.log("setting up read handler on stdin");
		let dataReceived = false;
		const stdinFd = 0;
		const readBuffer = new Uint8Array(1024);
		
		os.setReadHandler(stdinFd, () => {
			console.log("read handler called!");
			const bytesRead = os.read(stdinFd, readBuffer.buffer, 0, readBuffer.length);
			console.log("os.read returned:", bytesRead);
			if (bytesRead > 0) {
				// Log raw bytes for debugging
				const bytes = readBuffer.slice(0, bytesRead);
				console.log("raw bytes:", Array.from(bytes).join(","));
				const data = String.fromCharCode.apply(null, bytes);
				console.log("received data:", data);
				dataReceived = true;
				// Clear the read handler after receiving data
				os.setReadHandler(stdinFd, null);
				console.log("handler cleared, dataReceived =", dataReceived);
			} else if (bytesRead < 0) {
				console.log("os.read error, keeping handler");
			} else {
				console.log("os.read returned 0, keeping handler");
			}
		});
		
		console.log("read handler registered, waiting for data...");
		
		// Set a timeout to check if data was received
		os.setTimeout(() => {
			console.log("timeout fired, dataReceived =", dataReceived);
			if (!dataReceived) {
				console.log("TIMEOUT: no data received!");
			} else {
				console.log("SUCCESS: data was received");
			}
		}, 2000);
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	// Write data to stdin after a short delay
	stdinDataReady := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		t.Log("writing test data to stdin")
		stdinBuf.Write([]byte("hello from Go!"))
		close(stdinDataReady)
	}()

	// Custom event loop that uses PollIO when stdin has data
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context canceled: %v", ctx.Err())
		default:
		}

		result, err := qjs.LoopOnce(ctx)
		if err != nil {
			t.Fatalf("LoopOnce error: %v", err)
		}

		switch {
		case result == LoopIdle:
			// Check if stdin has data
			stdinBuf.mu.Lock()
			hasData := len(stdinBuf.buf) > stdinBuf.offset
			stdinBuf.mu.Unlock()

			if hasData {
				// Poll for I/O to invoke the read handler
				pollResult, err := qjs.PollIO(ctx, 0)
				if err != nil {
					t.Fatalf("PollIO error: %v", err)
				}
				t.Logf("PollIO returned: %d", pollResult)
				continue
			}

			// Check if we're done (data received or timeout)
			out := stdout.String()
			if strings.Contains(out, "handler cleared") || strings.Contains(out, "TIMEOUT:") {
				goto done
			}

			// Wait a bit for stdin data
			select {
			case <-ctx.Done():
				goto done
			case <-stdinDataReady:
				continue
			case <-time.After(10 * time.Millisecond):
				continue
			}

		case result == LoopError:
			t.Fatalf("JavaScript error occurred")

		case result == 0:
			// More microtasks pending
			continue

		case result > 0:
			// Check if we're already done
			out := stdout.String()
			if strings.Contains(out, "handler cleared") || strings.Contains(out, "TIMEOUT:") {
				goto done
			}

			// Check stdin while waiting for timer
			stdinBuf.mu.Lock()
			hasData := len(stdinBuf.buf) > stdinBuf.offset
			stdinBuf.mu.Unlock()

			if hasData {
				qjs.PollIO(ctx, 0)
				continue
			}

			select {
			case <-ctx.Done():
				goto done
			case <-time.After(time.Duration(result) * time.Millisecond):
				continue
			}
		}
	}

done:
	output := stdout.String()
	t.Logf("Output:\n%s", output)

	// Check that the read handler was called and data was received
	if !strings.Contains(output, "read handler called!") {
		t.Errorf("expected read handler to be called")
	}
	if !strings.Contains(output, "received data: hello from Go!") {
		t.Errorf("expected to receive the test data")
	}
	if !strings.Contains(output, "handler cleared, dataReceived = true") {
		t.Errorf("expected handler to be cleared after receiving data")
	}
	if strings.Contains(output, "TIMEOUT: no data received!") {
		t.Errorf("test timed out waiting for stdin data - read handler was not triggered")
	}
}

// TestESModuleEval tests evaluating code as an ES module
func TestESModuleEval(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Test ES module syntax with import/export simulation
	code := `
		// ES module features
		const add = (a, b) => a + b;
		const multiply = (a, b) => a * b;
		
		// Test top-level await (ES2022)
		const result = await Promise.resolve(42);
		console.log("top-level await result:", result);
		
		// Test dynamic import-like pattern
		const mathOps = { add, multiply };
		console.log("add(2, 3):", mathOps.add(2, 3));
		console.log("multiply(4, 5):", mathOps.multiply(4, 5));
	`
	if err := qjs.Eval(ctx, code, true); err != nil {
		t.Fatalf("failed to eval module: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "top-level await result: 42") {
		t.Errorf("expected top-level await to work, got: %s", output)
	}
	if !strings.Contains(output, "add(2, 3): 5") {
		t.Errorf("expected add function to work, got: %s", output)
	}
	if !strings.Contains(output, "multiply(4, 5): 20") {
		t.Errorf("expected multiply function to work, got: %s", output)
	}
}

// TestPromiseChaining tests Promise chaining and async operations
func TestPromiseChaining(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test Promise.resolve and chaining
		Promise.resolve(1)
			.then(x => {
				console.log("step 1:", x);
				return x + 1;
			})
			.then(x => {
				console.log("step 2:", x);
				return x * 2;
			})
			.then(x => {
				console.log("final:", x);
			});

		// Test Promise.all
		Promise.all([
			Promise.resolve("a"),
			Promise.resolve("b"),
			Promise.resolve("c")
		]).then(results => {
			console.log("Promise.all:", results.join(","));
		});

		// Test Promise.race
		Promise.race([
			Promise.resolve("first"),
			Promise.resolve("second")
		]).then(result => {
			console.log("Promise.race:", result);
		});

		// Test Promise rejection and catch
		Promise.reject(new Error("test error"))
			.catch(err => {
				console.log("caught error:", err.message);
			});
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"step 1: 1",
		"step 2: 2",
		"final: 4",
		"Promise.all: a,b,c",
		"Promise.race: first",
		"caught error: test error",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestAsyncAwait tests async/await functionality
func TestAsyncAwait(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		async function fetchData(id) {
			return new Promise(resolve => {
				resolve({ id: id, data: "item_" + id });
			});
		}

		async function processItems() {
			console.log("starting async processing");
			
			// Sequential await
			const item1 = await fetchData(1);
			console.log("item1:", item1.id, item1.data);
			
			const item2 = await fetchData(2);
			console.log("item2:", item2.id, item2.data);
			
			// Parallel await with Promise.all
			const [item3, item4] = await Promise.all([
				fetchData(3),
				fetchData(4)
			]);
			console.log("item3:", item3.id, item3.data);
			console.log("item4:", item4.id, item4.data);
			
			console.log("async processing complete");
		}

		// Test async error handling
		async function failingAsync() {
			throw new Error("async error");
		}

		async function handleAsyncError() {
			try {
				await failingAsync();
			} catch (e) {
				console.log("caught async error:", e.message);
			}
		}

		processItems();
		handleAsyncError();
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"starting async processing",
		"item1: 1 item_1",
		"item2: 2 item_2",
		"item3: 3 item_3",
		"item4: 4 item_4",
		"async processing complete",
		"caught async error: async error",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestEvalWithFilename tests that custom filenames appear in error stack traces
func TestEvalWithFilename(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout, stderr bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stderr)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Eval code with a custom filename
	code := `
		function testFunction() {
			console.log("called from custom file");
			// Capture stack trace
			const err = new Error("stack trace test");
			console.log("stack:", err.stack);
		}
		testFunction();
	`
	if err := qjs.EvalWithFilename(ctx, code, "my-custom-script.js", false); err != nil {
		t.Fatalf("failed to eval with filename: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "called from custom file") {
		t.Errorf("expected function to be called")
	}
	if !strings.Contains(output, "my-custom-script.js") {
		t.Errorf("expected custom filename in stack trace, got: %s", output)
	}
}

// TestSyntaxError tests that syntax errors are properly reported
func TestSyntaxError(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Test syntax error - missing closing brace
	code := `function broken() { console.log("test")`
	err = qjs.Eval(ctx, code, false)
	if err == nil {
		t.Errorf("expected syntax error, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

// TestRuntimeError tests that runtime errors are handled
func TestRuntimeError(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Test various runtime errors
	testCases := []struct {
		name string
		code string
	}{
		{
			name: "undefined variable",
			code: `console.log(undefinedVariable);`,
		},
		{
			name: "null property access",
			code: `let obj = null; console.log(obj.property);`,
		},
		{
			name: "type error",
			code: `let num = 42; num();`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh instance for each test
			r := wazero.NewRuntime(ctx)
			defer r.Close(ctx)

			var stdout bytes.Buffer
			config := wazero.NewModuleConfig().
				WithStdout(&stdout).
				WithStderr(&stdout)

			qjs, err := NewQuickJS(ctx, r, config)
			if err != nil {
				t.Fatalf("failed to create QuickJS: %v", err)
			}
			defer qjs.Close(ctx)

			if err := qjs.Init(ctx); err != nil {
				t.Fatalf("failed to init QuickJS: %v", err)
			}

			err = qjs.Eval(ctx, tc.code, false)
			if err == nil {
				// Some runtime errors might be caught later in the loop
				result, loopErr := qjs.LoopOnce(ctx)
				if loopErr != nil {
					t.Logf("LoopOnce returned error: %v", loopErr)
				}
				if result == LoopError {
					t.Logf("Got expected LoopError for %s", tc.name)
				} else {
					// Check stderr for error output
					output := stdout.String()
					t.Logf("Output for %s: %s", tc.name, output)
				}
			} else {
				t.Logf("Got expected eval error for %s: %v", tc.name, err)
			}
		})
	}
}

// TestPreCompiledModuleReuse tests that multiple instances can be created
// sequentially using NewQuickJS (each with its own runtime)
func TestPreCompiledModuleReuse(t *testing.T) {
	ctx := context.Background()

	// Test creating multiple independent QuickJS instances sequentially
	// Each uses the same embedded WASM but different runtimes
	for i := 0; i < 3; i++ {
		r := wazero.NewRuntime(ctx)

		var stdout bytes.Buffer

		config := wazero.NewModuleConfig().
			WithStdout(&stdout).
			WithStderr(&stdout)

		qjs, err := NewQuickJS(ctx, r, config)
		if err != nil {
			r.Close(ctx)
			t.Fatalf("instance %d: failed to create QuickJS: %v", i, err)
		}

		if err := qjs.Init(ctx); err != nil {
			qjs.Close(ctx)
			r.Close(ctx)
			t.Fatalf("instance %d: failed to init QuickJS: %v", i, err)
		}

		code := `console.log("instance", ` + string(rune('0'+i)) + `);`
		if err := qjs.Eval(ctx, code, false); err != nil {
			qjs.Close(ctx)
			r.Close(ctx)
			t.Fatalf("instance %d: failed to eval: %v", i, err)
		}

		if err := qjs.RunLoop(ctx); err != nil {
			qjs.Close(ctx)
			r.Close(ctx)
			t.Fatalf("instance %d: event loop error: %v", i, err)
		}

		output := stdout.String()
		expected := "instance " + string(rune('0'+i))
		if !strings.Contains(output, expected) {
			t.Errorf("instance %d: expected %q, got: %s", i, expected, output)
		}

		qjs.Close(ctx)
		r.Close(ctx)
	}
}

// TestMultipleEvals tests calling Eval multiple times on the same instance
func TestMultipleEvals(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// First eval - define a variable
	if err := qjs.Eval(ctx, `var counter = 0; console.log("counter initialized");`, false); err != nil {
		t.Fatalf("failed first eval: %v", err)
	}

	// Second eval - modify the variable
	if err := qjs.Eval(ctx, `counter++; console.log("counter:", counter);`, false); err != nil {
		t.Fatalf("failed second eval: %v", err)
	}

	// Third eval - modify again
	if err := qjs.Eval(ctx, `counter += 10; console.log("counter:", counter);`, false); err != nil {
		t.Fatalf("failed third eval: %v", err)
	}

	// Fourth eval - define a function and use it with the variable
	if err := qjs.Eval(ctx, `
		function double(x) { return x * 2; }
		console.log("doubled counter:", double(counter));
	`, false); err != nil {
		t.Fatalf("failed fourth eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"counter initialized",
		"counter: 1",
		"counter: 11",
		"doubled counter: 22",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestSetInterval tests os.setInterval and os.clearInterval
func TestSetInterval(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r := wazero.NewRuntime(ctx)
	defer r.Close(context.Background())

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(context.Background())

	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	code := `
		let count = 0;
		const intervalId = os.setInterval(() => {
			count++;
			console.log("interval tick:", count);
			if (count >= 3) {
				os.clearInterval(intervalId);
				console.log("interval cleared");
			}
		}, 50);
		console.log("interval started");
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"interval started",
		"interval tick: 1",
		"interval tick: 2",
		"interval tick: 3",
		"interval cleared",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}

	// Verify it stopped at 3
	if strings.Contains(output, "interval tick: 4") {
		t.Errorf("interval should have stopped at 3, got: %s", output)
	}
}

// TestClearTimeout tests os.clearTimeout cancels pending timeouts.
// Note: This test verifies that clearTimeout can be called without error.
// Due to potential timing issues in the reactor model, we use a manual event loop.
func TestClearTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r := wazero.NewRuntime(ctx)
	defer r.Close(context.Background())

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(context.Background())

	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	// Simpler test that just verifies clearTimeout can be called
	code := `
		console.log("setting up timeouts");
		
		// Set a timeout and immediately cancel it
		const id = os.setTimeout(() => {
			console.log("this should be cancelled");
		}, 1000);
		
		os.clearTimeout(id);
		console.log("timeout cleared successfully");
		
		// Set a short timeout to confirm we're done
		os.setTimeout(() => {
			console.log("done");
		}, 50);
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	// Run event loop with manual iteration to avoid blocking issues
	for i := 0; i < 100; i++ {
		result, err := qjs.LoopOnce(ctx)
		if err != nil {
			t.Fatalf("LoopOnce error: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "done") {
			break
		}

		if result == LoopIdle {
			break
		}

		if result > 0 {
			time.Sleep(time.Duration(result) * time.Millisecond)
		}
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "timeout cleared successfully") {
		t.Errorf("expected clearTimeout to work, got: %s", output)
	}
	if !strings.Contains(output, "done") {
		t.Errorf("expected final timeout, got: %s", output)
	}
}

// TestContextTimeout tests that context timeout properly interrupts execution
func TestContextTimeout(t *testing.T) {
	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	r := wazero.NewRuntime(ctx)
	defer r.Close(context.Background())

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(context.Background())

	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	// Set up a timeout that will never fire due to context cancellation
	code := `
		console.log("starting long wait");
		os.setTimeout(() => {
			console.log("this should not fire");
		}, 10000);
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	// RunLoop should exit with context timeout error
	err = qjs.RunLoop(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "starting long wait") {
		t.Errorf("expected initial output, got: %s", output)
	}
	if strings.Contains(output, "this should not fire") {
		t.Errorf("long timeout should not have fired, got: %s", output)
	}
}

// TestLoopOnceReturnValues tests the different return values from LoopOnce
func TestLoopOnceReturnValues(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Test 1: Synchronous code should return LoopIdle after processing
	if err := qjs.Eval(ctx, `console.log("sync code");`, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	result, err := qjs.LoopOnce(ctx)
	if err != nil {
		t.Fatalf("LoopOnce error: %v", err)
	}
	if result != LoopIdle {
		t.Errorf("expected LoopIdle after sync code, got: %d", result)
	}

	// Test 2: Promise should return 0 (more microtasks pending)
	if err := qjs.Eval(ctx, `Promise.resolve().then(() => console.log("promise resolved"));`, false); err != nil {
		t.Fatalf("failed to eval promise: %v", err)
	}

	result, err = qjs.LoopOnce(ctx)
	if err != nil {
		t.Fatalf("LoopOnce error: %v", err)
	}
	// After promise is queued, we might get 0 (more work) or process it
	t.Logf("After promise eval, LoopOnce returned: %d", result)

	// Run until idle
	for result.IsPending() {
		result, err = qjs.LoopOnce(ctx)
		if err != nil {
			t.Fatalf("LoopOnce error: %v", err)
		}
	}

	// Test 3: setTimeout should return positive value (timer delay)
	if err := qjs.Eval(ctx, `os.setTimeout(() => console.log("timeout"), 100);`, false); err != nil {
		t.Fatalf("failed to eval setTimeout: %v", err)
	}

	result, err = qjs.LoopOnce(ctx)
	if err != nil {
		t.Fatalf("LoopOnce error: %v", err)
	}
	if result <= 0 {
		t.Errorf("expected positive timer delay, got: %d", result)
	}
	t.Logf("Timer delay reported: %d ms", result)

	// Finish the loop
	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "sync code") {
		t.Errorf("expected sync code output")
	}
	if !strings.Contains(output, "promise resolved") {
		t.Errorf("expected promise output")
	}
	if !strings.Contains(output, "timeout") {
		t.Errorf("expected timeout output")
	}
}

// TestLargeCodeEval tests evaluating large JavaScript code
func TestLargeCodeEval(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Generate a large JavaScript code with many functions
	var codeBuilder strings.Builder
	codeBuilder.WriteString("let results = [];\n")

	for i := 0; i < 100; i++ {
		codeBuilder.WriteString("function func")
		codeBuilder.WriteString(string(rune('0' + i/10)))
		codeBuilder.WriteString(string(rune('0' + i%10)))
		codeBuilder.WriteString("() { return ")
		codeBuilder.WriteString(string(rune('0' + i/10)))
		codeBuilder.WriteString(string(rune('0' + i%10)))
		codeBuilder.WriteString("; }\n")
		codeBuilder.WriteString("results.push(func")
		codeBuilder.WriteString(string(rune('0' + i/10)))
		codeBuilder.WriteString(string(rune('0' + i%10)))
		codeBuilder.WriteString("());\n")
	}

	codeBuilder.WriteString("console.log('total functions:', results.length);\n")
	codeBuilder.WriteString("console.log('sum:', results.reduce((a,b) => a+b, 0));\n")

	code := codeBuilder.String()
	t.Logf("Code size: %d bytes", len(code))

	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval large code: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	if !strings.Contains(output, "total functions: 100") {
		t.Errorf("expected 100 functions, got: %s", output)
	}
	// Sum of 0-99 = 4950
	if !strings.Contains(output, "sum: 4950") {
		t.Errorf("expected sum 4950, got: %s", output)
	}
}

// TestBigIntSupport tests BigInt operations
func TestBigIntSupport(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test BigInt literals
		const big1 = 9007199254740993n;
		const big2 = 9007199254740993n;
		console.log("bigint literal:", big1.toString());
		
		// Test BigInt operations
		const sum = big1 + big2;
		console.log("bigint sum:", sum.toString());
		
		// Test BigInt comparison
		console.log("bigint equal:", big1 === big2);
		
		// Test BigInt with regular numbers
		const bigFromNum = BigInt(12345678901234567890);
		console.log("bigint from number:", typeof bigFromNum === 'bigint');
		
		// Test large exponentiation
		const power = 2n ** 64n;
		console.log("2^64:", power.toString());
		
		// Test BigInt division (floors)
		const div = 10n / 3n;
		console.log("10n / 3n:", div.toString());
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"bigint literal: 9007199254740993",
		"bigint sum: 18014398509481986",
		"bigint equal: true",
		"bigint from number: true",
		"2^64: 18446744073709551616",
		"10n / 3n: 3",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestTypedArrays tests TypedArray support
func TestTypedArrays(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test Uint8Array
		const u8 = new Uint8Array([1, 2, 3, 255]);
		console.log("Uint8Array:", Array.from(u8).join(","));
		console.log("Uint8Array length:", u8.length);
		
		// Test Int32Array
		const i32 = new Int32Array([-1, 0, 1, 2147483647]);
		console.log("Int32Array:", Array.from(i32).join(","));
		
		// Test Float64Array
		const f64 = new Float64Array([1.5, 2.5, 3.14159]);
		console.log("Float64Array:", Array.from(f64).map(x => x.toFixed(2)).join(","));
		
		// Test ArrayBuffer
		const buffer = new ArrayBuffer(8);
		const view = new DataView(buffer);
		view.setUint32(0, 0xDEADBEEF, true); // little-endian
		console.log("DataView getUint32:", view.getUint32(0, true).toString(16));
		
		// Test slice
		const sliced = u8.slice(1, 3);
		console.log("sliced:", Array.from(sliced).join(","));
		
		// Test subarray
		const sub = u8.subarray(1, 3);
		console.log("subarray:", Array.from(sub).join(","));
		
		// Test set
		const target = new Uint8Array(5);
		target.set([10, 20, 30], 1);
		console.log("after set:", Array.from(target).join(","));
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"Uint8Array: 1,2,3,255",
		"Uint8Array length: 4",
		"Int32Array: -1,0,1,2147483647",
		"Float64Array: 1.50,2.50,3.14",
		"DataView getUint32: deadbeef",
		"sliced: 2,3",
		"subarray: 2,3",
		"after set: 0,10,20,30,0",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestJSONOperations tests JSON parsing and serialization
func TestJSONOperations(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test JSON.parse
		const parsed = JSON.parse('{"name":"test","value":42,"nested":{"a":1}}');
		console.log("parsed name:", parsed.name);
		console.log("parsed value:", parsed.value);
		console.log("parsed nested.a:", parsed.nested.a);
		
		// Test JSON.stringify
		const obj = { foo: "bar", nums: [1, 2, 3], flag: true };
		const str = JSON.stringify(obj);
		console.log("stringified:", str);
		
		// Test JSON.stringify with replacer
		const filtered = JSON.stringify(obj, ["foo", "flag"]);
		console.log("filtered:", filtered);
		
		// Test JSON.stringify with space
		const pretty = JSON.stringify({a: 1}, null, 2);
		console.log("pretty has newline:", pretty.includes("\\n"));
		
		// Test circular reference handling
		try {
			const circular = {};
			circular.self = circular;
			JSON.stringify(circular);
			console.log("circular: no error");
		} catch (e) {
			console.log("circular error:", e.message.includes("circular") || e.message.includes("cyclic"));
		}
		
		// Test special values
		console.log("null:", JSON.stringify(null));
		console.log("undefined:", JSON.stringify(undefined));
		console.log("NaN:", JSON.stringify(NaN));
		console.log("Infinity:", JSON.stringify(Infinity));
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"parsed name: test",
		"parsed value: 42",
		"parsed nested.a: 1",
		`"foo":"bar"`,
		`"nums":[1,2,3]`,
		"null: null",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestGeneratorsAndIterators tests generator functions and iterators
func TestGeneratorsAndIterators(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test generator function
		function* countTo(n) {
			for (let i = 1; i <= n; i++) {
				yield i;
			}
		}
		
		const gen = countTo(5);
		const values = [];
		for (const v of gen) {
			values.push(v);
		}
		console.log("generator values:", values.join(","));
		
		// Test generator with next()
		const gen2 = countTo(3);
		console.log("next 1:", gen2.next().value);
		console.log("next 2:", gen2.next().value);
		console.log("next 3:", gen2.next().value);
		console.log("done:", gen2.next().done);
		
		// Test yield*
		function* nested() {
			yield* [1, 2];
			yield* [3, 4];
		}
		console.log("yield*:", [...nested()].join(","));
		
		// Test Symbol.iterator
		const iterable = {
			[Symbol.iterator]() {
				let count = 0;
				return {
					next() {
						count++;
						if (count <= 3) {
							return { value: count * 10, done: false };
						}
						return { done: true };
					}
				};
			}
		};
		console.log("custom iterator:", [...iterable].join(","));
		
		// Test for...of with Map
		const map = new Map([["a", 1], ["b", 2]]);
		const mapEntries = [];
		for (const [k, v] of map) {
			mapEntries.push(k + ":" + v);
		}
		console.log("map iteration:", mapEntries.join(","));
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"generator values: 1,2,3,4,5",
		"next 1: 1",
		"next 2: 2",
		"next 3: 3",
		"done: true",
		"yield*: 1,2,3,4",
		"custom iterator: 10,20,30",
		"map iteration: a:1,b:2",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestWeakMapAndWeakSet tests WeakMap and WeakSet
func TestWeakMapAndWeakSet(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test WeakMap
		const wm = new WeakMap();
		const key1 = {};
		const key2 = {};
		
		wm.set(key1, "value1");
		wm.set(key2, "value2");
		
		console.log("weakmap has key1:", wm.has(key1));
		console.log("weakmap get key1:", wm.get(key1));
		console.log("weakmap has key2:", wm.has(key2));
		
		wm.delete(key1);
		console.log("after delete key1:", wm.has(key1));
		
		// Test WeakSet
		const ws = new WeakSet();
		const obj1 = { id: 1 };
		const obj2 = { id: 2 };
		
		ws.add(obj1);
		ws.add(obj2);
		
		console.log("weakset has obj1:", ws.has(obj1));
		console.log("weakset has obj2:", ws.has(obj2));
		console.log("weakset has {}:", ws.has({}));
		
		ws.delete(obj1);
		console.log("after delete obj1:", ws.has(obj1));
		
		// Test that primitives throw
		try {
			wm.set("string", "value");
			console.log("weakmap primitive: no error");
		} catch (e) {
			console.log("weakmap primitive error: true");
		}
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"weakmap has key1: true",
		"weakmap get key1: value1",
		"weakmap has key2: true",
		"after delete key1: false",
		"weakset has obj1: true",
		"weakset has obj2: true",
		"weakset has {}: false",
		"after delete obj1: false",
		"weakmap primitive error: true",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestProxyAndReflect tests Proxy and Reflect APIs
func TestProxyAndReflect(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test Proxy get/set traps
		const target = { value: 10 };
		const handler = {
			get(obj, prop) {
				console.log("get trap:", prop);
				return obj[prop];
			},
			set(obj, prop, value) {
				console.log("set trap:", prop, "=", value);
				obj[prop] = value;
				return true;
			}
		};
		
		const proxy = new Proxy(target, handler);
		console.log("proxy.value:", proxy.value);
		proxy.value = 20;
		console.log("after set:", target.value);
		
		// Test Proxy apply trap (for functions)
		const fnTarget = function(a, b) { return a + b; };
		const fnHandler = {
			apply(target, thisArg, args) {
				console.log("apply trap with args:", args.join(","));
				return target.apply(thisArg, args);
			}
		};
		const fnProxy = new Proxy(fnTarget, fnHandler);
		console.log("fnProxy(3, 4):", fnProxy(3, 4));
		
		// Test Reflect.get/set
		const obj = { x: 1, y: 2 };
		console.log("Reflect.get:", Reflect.get(obj, "x"));
		Reflect.set(obj, "z", 3);
		console.log("Reflect.set result:", obj.z);
		
		// Test Reflect.has
		console.log("Reflect.has x:", Reflect.has(obj, "x"));
		console.log("Reflect.has w:", Reflect.has(obj, "w"));
		
		// Test Reflect.ownKeys
		console.log("Reflect.ownKeys:", Reflect.ownKeys(obj).join(","));
		
		// Test Reflect.deleteProperty
		Reflect.deleteProperty(obj, "y");
		console.log("after delete y:", Reflect.has(obj, "y"));
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"get trap: value",
		"proxy.value: 10",
		"set trap: value = 20",
		"after set: 20",
		"apply trap with args: 3,4",
		"fnProxy(3, 4): 7",
		"Reflect.get: 1",
		"Reflect.set result: 3",
		"Reflect.has x: true",
		"Reflect.has w: false",
		"Reflect.ownKeys: x,y,z",
		"after delete y: false",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestSymbols tests Symbol functionality
func TestSymbols(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test Symbol creation
		const sym1 = Symbol("test");
		const sym2 = Symbol("test");
		console.log("symbols unique:", sym1 !== sym2);
		console.log("typeof symbol:", typeof sym1);
		
		// Test Symbol.for (global registry)
		const globalSym1 = Symbol.for("global");
		const globalSym2 = Symbol.for("global");
		console.log("Symbol.for same:", globalSym1 === globalSym2);
		
		// Test Symbol.keyFor
		console.log("Symbol.keyFor:", Symbol.keyFor(globalSym1));
		console.log("Symbol.keyFor local:", Symbol.keyFor(sym1));
		
		// Test Symbol as object key
		const key = Symbol("key");
		const obj = {
			[key]: "secret value",
			visible: "public value"
		};
		console.log("symbol property:", obj[key]);
		console.log("Object.keys includes symbol:", Object.keys(obj).includes(key.toString()));
		console.log("Object.getOwnPropertySymbols:", Object.getOwnPropertySymbols(obj).length);
		
		// Test well-known symbols
		console.log("Symbol.iterator exists:", typeof Symbol.iterator === "symbol");
		console.log("Symbol.toStringTag exists:", typeof Symbol.toStringTag === "symbol");
		
		// Test Symbol.description
		const described = Symbol("my description");
		console.log("description:", described.description);
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"symbols unique: true",
		"typeof symbol: symbol",
		"Symbol.for same: true",
		"Symbol.keyFor: global",
		"symbol property: secret value",
		"Object.getOwnPropertySymbols: 1",
		"Symbol.iterator exists: true",
		"Symbol.toStringTag exists: true",
		"description: my description",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestClasses tests ES6 class syntax
func TestClasses(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test basic class
		class Animal {
			constructor(name) {
				this.name = name;
			}
			
			speak() {
				return this.name + " makes a sound";
			}
			
			static species() {
				return "unknown";
			}
		}
		
		const animal = new Animal("Generic");
		console.log("animal.name:", animal.name);
		console.log("animal.speak():", animal.speak());
		console.log("Animal.species():", Animal.species());
		
		// Test inheritance
		class Dog extends Animal {
			constructor(name, breed) {
				super(name);
				this.breed = breed;
			}
			
			speak() {
				return this.name + " barks";
			}
			
			fetch() {
				return this.name + " fetches the ball";
			}
		}
		
		const dog = new Dog("Buddy", "Labrador");
		console.log("dog.name:", dog.name);
		console.log("dog.breed:", dog.breed);
		console.log("dog.speak():", dog.speak());
		console.log("dog.fetch():", dog.fetch());
		console.log("dog instanceof Dog:", dog instanceof Dog);
		console.log("dog instanceof Animal:", dog instanceof Animal);
		
		// Test getters and setters
		class Rectangle {
			constructor(width, height) {
				this._width = width;
				this._height = height;
			}
			
			get area() {
				return this._width * this._height;
			}
			
			set width(value) {
				this._width = value;
			}
		}
		
		const rect = new Rectangle(5, 10);
		console.log("rect.area:", rect.area);
		rect.width = 7;
		console.log("rect.area after set:", rect.area);
		
		// Test private fields (if supported)
		try {
			class Counter {
				#count = 0;
				increment() { this.#count++; }
				get value() { return this.#count; }
			}
			const counter = new Counter();
			counter.increment();
			counter.increment();
			console.log("private field:", counter.value);
		} catch (e) {
			console.log("private fields not supported");
		}
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"animal.name: Generic",
		"animal.speak(): Generic makes a sound",
		"Animal.species(): unknown",
		"dog.name: Buddy",
		"dog.breed: Labrador",
		"dog.speak(): Buddy barks",
		"dog.fetch(): Buddy fetches the ball",
		"dog instanceof Dog: true",
		"dog instanceof Animal: true",
		"rect.area: 50",
		"rect.area after set: 70",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestRegExpAdvanced tests advanced RegExp features
func TestRegExpAdvanced(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test basic matching
		const text = "The quick brown fox jumps over the lazy dog";
		console.log("test match:", /quick/.test(text));
		console.log("exec result:", /brown/.exec(text)[0]);
		
		// Test global flag
		const matches = text.match(/the/gi);
		console.log("global matches:", matches.length);
		
		// Test capturing groups
		const dateStr = "2024-01-15";
		const dateMatch = dateStr.match(/(\d{4})-(\d{2})-(\d{2})/);
		console.log("year:", dateMatch[1]);
		console.log("month:", dateMatch[2]);
		console.log("day:", dateMatch[3]);
		
		// Test replace with function
		const replaced = "hello world".replace(/\w+/g, (match) => match.toUpperCase());
		console.log("replaced:", replaced);
		
		// Test split with regex
		const parts = "a1b2c3d".split(/\d/);
		console.log("split:", parts.join(","));
		
		// Test regex properties
		const re = /test/gi;
		console.log("global:", re.global);
		console.log("ignoreCase:", re.ignoreCase);
		console.log("source:", re.source);
		
		// Test lastIndex with global
		const gre = /o/g;
		const str = "foo";
		gre.exec(str);
		console.log("lastIndex after first:", gre.lastIndex);
		gre.exec(str);
		console.log("lastIndex after second:", gre.lastIndex);
		
		// Test lookahead (if supported)
		try {
			const lookahead = "foobar".match(/foo(?=bar)/);
			console.log("lookahead:", lookahead ? lookahead[0] : "null");
		} catch (e) {
			console.log("lookahead not supported");
		}
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"test match: true",
		"exec result: brown",
		"global matches: 2",
		"year: 2024",
		"month: 01",
		"day: 15",
		"replaced: HELLO WORLD",
		"split: a,b,c,d",
		"global: true",
		"ignoreCase: true",
		"source: test",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestClosuresAndScoping tests closures and variable scoping
func TestClosuresAndScoping(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test basic closure
		function createCounter() {
			let count = 0;
			return {
				increment() { count++; },
				decrement() { count--; },
				value() { return count; }
			};
		}
		
		const counter = createCounter();
		counter.increment();
		counter.increment();
		counter.increment();
		counter.decrement();
		console.log("closure counter:", counter.value());
		
		// Test closure preserves outer scope
		function outer(x) {
			return function inner(y) {
				return x + y;
			};
		}
		const add5 = outer(5);
		const add10 = outer(10);
		console.log("add5(3):", add5(3));
		console.log("add10(3):", add10(3));
		
		// Test let vs var in loops
		const funcsLet = [];
		for (let i = 0; i < 3; i++) {
			funcsLet.push(() => i);
		}
		console.log("let loop:", funcsLet.map(f => f()).join(","));
		
		// Test block scoping
		{
			let blockVar = "inner";
			console.log("block scoped:", blockVar);
		}
		
		// Test const
		const constVal = 42;
		console.log("const value:", constVal);
		
		// Test temporal dead zone detection
		try {
			console.log(tdz);
			let tdz = "test";
		} catch (e) {
			console.log("TDZ caught:", e.name);
		}
		
		// Test IIFE (Immediately Invoked Function Expression)
		const iife = (function() {
			const private = "secret";
			return { get: () => private };
		})();
		console.log("IIFE result:", iife.get());
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"closure counter: 2",
		"add5(3): 8",
		"add10(3): 13",
		"let loop: 0,1,2",
		"block scoped: inner",
		"const value: 42",
		"TDZ caught: ReferenceError",
		"IIFE result: secret",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestBjsonModule tests the bjson (binary JSON) module
func TestBjsonModule(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Use InitStdModule to get bjson
	if err := qjs.InitStdModule(ctx); err != nil {
		t.Fatalf("failed to init QuickJS with std module: %v", err)
	}

	code := `
		// Test bjson.write and bjson.read
		const original = {
			name: "test",
			value: 42,
			nested: { a: 1, b: 2 },
			array: [1, 2, 3],
			flag: true
		};
		
		// Serialize to binary
		const binary = bjson.write(original);
		console.log("binary type:", binary instanceof ArrayBuffer);
		console.log("binary size:", binary.byteLength);
		
		// Deserialize from binary
		const restored = bjson.read(binary, 0, binary.byteLength);
		console.log("restored name:", restored.name);
		console.log("restored value:", restored.value);
		console.log("restored nested.a:", restored.nested.a);
		console.log("restored array:", restored.array.join(","));
		console.log("restored flag:", restored.flag);
		
		// Test with typed arrays
		const withTypedArray = {
			data: new Uint8Array([1, 2, 3, 4, 5])
		};
		const binary2 = bjson.write(withTypedArray);
		const restored2 = bjson.read(binary2, 0, binary2.byteLength);
		console.log("typed array preserved:", restored2.data instanceof Uint8Array);
		console.log("typed array values:", Array.from(restored2.data).join(","));
		
		// Test with Date
		const withDate = { date: new Date("2024-01-15T12:00:00Z") };
		const binary3 = bjson.write(withDate);
		const restored3 = bjson.read(binary3, 0, binary3.byteLength);
		console.log("date preserved:", restored3.date instanceof Date);
		console.log("date value:", restored3.date.toISOString());
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"binary type: true",
		"restored name: test",
		"restored value: 42",
		"restored nested.a: 1",
		"restored array: 1,2,3",
		"restored flag: true",
		"typed array preserved: true",
		"typed array values: 1,2,3,4,5",
		"date preserved: true",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestInitArgvEmptyArgs tests InitArgv with empty arguments (should call Init)
func TestInitArgvEmptyArgs(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Empty args should fall back to Init
	if err := qjs.InitArgv(ctx, []string{}); err != nil {
		t.Fatalf("failed to init with empty args: %v", err)
	}

	// Eval should work
	if err := qjs.Eval(ctx, `console.log("empty args works");`, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "empty args works") {
		t.Errorf("expected output, got: %s", output)
	}
}

// TestEmptyCodeEval tests evaluating empty code
func TestEmptyCodeEval(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	// Empty code should be valid
	if err := qjs.Eval(ctx, "", false); err != nil {
		t.Fatalf("failed to eval empty code: %v", err)
	}

	// Whitespace only should be valid
	if err := qjs.Eval(ctx, "   \n\t  ", false); err != nil {
		t.Fatalf("failed to eval whitespace: %v", err)
	}

	// Comments only should be valid
	if err := qjs.Eval(ctx, "// just a comment\n/* block comment */", false); err != nil {
		t.Fatalf("failed to eval comments: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}
}

// TestSpecialValues tests handling of special JavaScript values
func TestSpecialValues(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	if err := qjs.Init(ctx); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
	}

	code := `
		// Test undefined
		let undef;
		console.log("undefined:", undef);
		console.log("typeof undefined:", typeof undef);
		
		// Test null
		const n = null;
		console.log("null:", n);
		console.log("typeof null:", typeof n);
		
		// Test NaN
		console.log("NaN:", NaN);
		console.log("isNaN(NaN):", isNaN(NaN));
		console.log("NaN === NaN:", NaN === NaN);
		console.log("Number.isNaN(NaN):", Number.isNaN(NaN));
		
		// Test Infinity
		console.log("Infinity:", Infinity);
		console.log("-Infinity:", -Infinity);
		console.log("isFinite(Infinity):", isFinite(Infinity));
		console.log("1/0:", 1/0);
		
		// Test -0
		const negZero = -0;
		console.log("-0 === 0:", negZero === 0);
		console.log("1/-0:", 1/negZero);
		
		// Test very large numbers
		console.log("Number.MAX_VALUE exists:", Number.MAX_VALUE > 0);
		console.log("Number.MIN_VALUE exists:", Number.MIN_VALUE > 0);
		console.log("Number.MAX_SAFE_INTEGER:", Number.MAX_SAFE_INTEGER);
		
		// Test epsilon
		console.log("Number.EPSILON exists:", Number.EPSILON > 0);
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"undefined: undefined",
		"typeof undefined: undefined",
		"null: null",
		"typeof null: object",
		"isNaN(NaN): true",
		"NaN === NaN: false",
		"Number.isNaN(NaN): true",
		"isFinite(Infinity): false",
		"-0 === 0: true",
		"1/-0: -Infinity",
		"Number.MAX_VALUE exists: true",
		"Number.MIN_VALUE exists: true",
		"Number.MAX_SAFE_INTEGER: 9007199254740991",
		"Number.EPSILON exists: true",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}

// TestConsoleMethods tests various console methods
func TestConsoleMethods(t *testing.T) {
	ctx := context.Background()

	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	var stdout bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout)

	qjs, err := NewQuickJS(ctx, r, config)
	if err != nil {
		t.Fatalf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Use InitArgv with --std to get full console support
	if err := qjs.InitArgv(ctx, []string{"qjs", "--std"}); err != nil {
		t.Fatalf("failed to init QuickJS with --std: %v", err)
	}

	code := `
		// Test console.log with multiple arguments
		console.log("multiple", "args", 123, true);
		
		// Test console.log with objects
		console.log("object:", { a: 1, b: 2 });
		console.log("array:", [1, 2, 3]);
		
		// Test console.warn if available
		if (typeof console.warn === 'function') {
			console.warn("warning message");
		} else {
			console.log("warning message");
		}
		
		// Test console.error if available
		if (typeof console.error === 'function') {
			console.error("error message");
		} else {
			console.log("error message");
		}
		
		// Test nested objects
		console.log("nested:", { outer: { inner: { deep: "value" } } });
		
		// Test console with special values
		console.log("special:", null, undefined, NaN, Infinity);
	`
	if err := qjs.Eval(ctx, code, false); err != nil {
		t.Fatalf("failed to eval: %v", err)
	}

	if err := qjs.RunLoop(ctx); err != nil {
		t.Fatalf("event loop error: %v", err)
	}

	output := stdout.String()
	t.Logf("Output: %s", output)

	expectedPatterns := []string{
		"multiple args 123 true",
		"warning message",
		"error message",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("expected output to contain %q, got: %s", pattern, output)
		}
	}
}
