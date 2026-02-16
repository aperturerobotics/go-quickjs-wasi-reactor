module github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs

go 1.24.4

require (
	github.com/aperturerobotics/go-quickjs-wasi-reactor v0.12.1 // master
	github.com/tetratelabs/wazero v1.11.0
)

require golang.org/x/sys v0.38.0 // indirect

replace github.com/aperturerobotics/go-quickjs-wasi-reactor => ../

// exports Pollable in experimental/sys for pollable stdin
// https://github.com/tetratelabs/wazero/issues/1500#issuecomment-3041125375
// https://github.com/wazero/wazero/pull/2476
replace github.com/tetratelabs/wazero => github.com/aperturerobotics/wazero v0.0.0-20260216034438-ad84e6308a28 // main
