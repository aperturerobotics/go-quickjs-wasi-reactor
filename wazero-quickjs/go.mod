module github.com/aperturerobotics/go-quickjs-wasi-reactor/wazero-quickjs

go 1.24.4

require (
	github.com/aperturerobotics/go-quickjs-wasi-reactor v0.11.1-0.20260108062600-c6de205b900b // master
	github.com/tetratelabs/wazero v1.11.0
)

replace github.com/aperturerobotics/go-quickjs-wasi-reactor => ../

// Use aperture fork which exposes experimental/fsapi for pollable stdin
// https://github.com/tetratelabs/wazero/issues/1500#issuecomment-3041125375
replace github.com/tetratelabs/wazero => github.com/aperturerobotics/wazero v0.0.0-20250706223739-81a39a0d5d54
