package main

import (
	"context"
	"fmt"
	"log"

	quickjs "github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs"
	"github.com/tetratelabs/wazero"
)

func main() {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	config := wazero.NewModuleConfig()

	qjs, err := quickjs.NewQuickJS(ctx, r, config)
	if err != nil {
		log.Fatal(err)
	}
	defer qjs.Close(ctx)

	if err := qjs.InitStdModule(ctx); err != nil {
		log.Fatal("InitStdModule:", err)
	}
	fmt.Println("InitStdModule OK")

	if err := qjs.Eval(ctx, `console.log("hello");`, false); err != nil {
		log.Fatal("Eval:", err)
	}
	fmt.Println("Eval OK")

	if err := qjs.RunLoop(ctx); err != nil {
		log.Fatal("RunLoop:", err)
	}
	fmt.Println("RunLoop OK")
}
