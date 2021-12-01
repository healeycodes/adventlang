cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" website/wasm_exec.js
GOOS=js GOARCH=wasm go build -o website/adventlang.wasm web/run.go