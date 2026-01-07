module github.com/aperturerobotics/go-quickjs-wasi/wazero-quickjs

go 1.24.0

require (
	github.com/aperturerobotics/go-quickjs-wasi v0.11.1-0.20251229075347-4b963494666d
	github.com/tetratelabs/wazero v1.11.0
)

replace github.com/aperturerobotics/go-quickjs-wasi => ../

// Use aperture fork which exposes experimental/fsapi for pollable stdin
// https://github.com/tetratelabs/wazero/issues/1500#issuecomment-3041125375
replace github.com/tetratelabs/wazero => github.com/aperturerobotics/wazero v0.0.0-20250706223739-81a39a0d5d54
