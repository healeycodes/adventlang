package main

import (
	"os"
	"syscall/js"

	"github.com/healeycodes/adventlang/pkg/adventlang"
)

func main() {
	c := make(chan struct{}, 0)
	js.Global().Set("adventlang", js.FuncOf(run))
	<-c
}

func run(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return "error: run(source) takes a single argument"
	}
	result, _, err := adventlang.RunProgram("web", args[0].String())
	if err != nil {
		println("uh oh.. while running: "+"web", err.Error(), "\n")
		os.Exit(1)
	}

	return js.ValueOf(result)
}
