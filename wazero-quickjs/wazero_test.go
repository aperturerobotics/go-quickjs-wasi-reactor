package quickjs

import (
	"bytes"
	"context"
	"embed"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero"
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
