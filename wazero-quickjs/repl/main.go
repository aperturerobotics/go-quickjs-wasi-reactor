package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	quickjs "github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs"
	"github.com/tetratelabs/wazero"
)

func main() {
	ctx := context.Background()

	// Create a new WebAssembly Runtime
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Configure the module with stdin, stdout, stderr, and filesystem
	fsRoot, err := os.OpenRoot(".")
	if err != nil {
		log.Panicln(err.Error())
	}

	config := wazero.NewModuleConfig().
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithFS(fsRoot.FS())

	// Create QuickJS instance
	qjs, err := quickjs.NewQuickJS(ctx, r, config)
	if err != nil {
		log.Panicf("failed to create QuickJS: %v", err)
	}
	defer qjs.Close(ctx)

	// Initialize runtime with --std (provides std, os, bjson globals)
	if err := qjs.Init(ctx, []string{"qjs", "--std"}); err != nil {
		log.Panicf("failed to init QuickJS: %v", err)
	}

	// If a file argument is provided, read and execute it
	if len(os.Args) > 1 {
		filename := os.Args[1]
		code, err := os.ReadFile(filename)
		if err != nil {
			log.Panicf("failed to read file %s: %v", filename, err)
		}

		// Check if it's a module (--module flag or .mjs extension)
		isModule := false
		for _, arg := range os.Args[2:] {
			if arg == "--module" || arg == "-m" {
				isModule = true
				break
			}
		}

		if err := qjs.Eval(ctx, string(code), isModule); err != nil {
			log.Panicf("failed to eval: %v", err)
		}

		// Run the event loop until complete
		if err := qjs.RunLoop(ctx); err != nil {
			log.Panicf("event loop error: %v", err)
		}
	} else {
		// Interactive REPL mode
		runREPL(ctx, qjs)
	}
}

func runREPL(ctx context.Context, qjs *quickjs.QuickJS) {
	fmt.Println("QuickJS REPL (type 'exit' or Ctrl+D to quit)")
	fmt.Println("std, os, and bjson modules are available as globals")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	var multiLine strings.Builder
	inMultiLine := false

	for {
		if inMultiLine {
			fmt.Print("... ")
		} else {
			fmt.Print("qjs> ")
		}

		if !scanner.Scan() {
			// EOF (Ctrl+D)
			fmt.Println()
			break
		}

		line := scanner.Text()

		// Check for exit command
		if !inMultiLine && (line == "exit" || line == "quit") {
			break
		}

		// Handle multi-line input
		if inMultiLine {
			multiLine.WriteString("\n")
			multiLine.WriteString(line)

			// Check if we should end multi-line mode
			// Simple heuristic: if line is empty or doesn't end with {, [, (, or \
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || (!strings.HasSuffix(trimmed, "{") &&
				!strings.HasSuffix(trimmed, "[") &&
				!strings.HasSuffix(trimmed, "(") &&
				!strings.HasSuffix(trimmed, ",") &&
				!strings.HasSuffix(trimmed, "\\")) {
				// Try to evaluate
				code := multiLine.String()
				multiLine.Reset()
				inMultiLine = false
				evalAndRun(ctx, qjs, code)
			}
			continue
		}

		// Check if we should start multi-line mode
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, "{") ||
			strings.HasSuffix(trimmed, "[") ||
			strings.HasSuffix(trimmed, "(") ||
			strings.HasSuffix(trimmed, ",") ||
			strings.HasSuffix(trimmed, "\\") {
			multiLine.WriteString(line)
			inMultiLine = true
			continue
		}

		// Single line - evaluate immediately
		if trimmed != "" {
			evalAndRun(ctx, qjs, line)
		}
	}
}

func evalAndRun(ctx context.Context, qjs *quickjs.QuickJS, code string) {
	if err := qjs.Eval(ctx, code, false); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	if err := qjs.RunLoop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}
