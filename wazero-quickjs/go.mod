module github.com/paralin/go-quickjs-wasi/wazero-quickjs

go 1.24.0

require (
	github.com/paralin/go-quickjs-wasi v0.11.1-0.20251229075347-4b963494666d
	github.com/tetratelabs/wazero v1.11.0
)

require golang.org/x/sys v0.38.0 // indirect

replace github.com/paralin/go-quickjs-wasi => ../
