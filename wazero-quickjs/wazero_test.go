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

	// Initialize with argv to load the script via WASI filesystem
	if err := qjs.InitArgv(ctx, []string{"qjs", "wazero_test.js"}); err != nil {
		t.Fatalf("failed to init QuickJS: %v", err)
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

	// Initialize with --std flag via argv
	if err := qjs.InitArgv(ctx, []string{"qjs", "--std"}); err != nil {
		t.Fatalf("failed to init QuickJS with --std: %v", err)
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

	// Initialize with --std to get os.setReadHandler
	if err := qjs.InitArgv(ctx, []string{"qjs", "--std"}); err != nil {
		t.Fatalf("failed to init QuickJS with --std: %v", err)
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
