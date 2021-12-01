cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" docs/wasm_exec.js
GOOS=js GOARCH=wasm go build -o docs/adventlang.wasm web/run.go