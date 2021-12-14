package main

import (
	"fmt"
	"syscall/js"

	"github.com/healeycodes/adventlang/pkg/adventlang"
)

func main() {
	js.Global().Set("adventlang", js.FuncOf(run))
	c := make(chan struct{}, 0)
	<-c
}

func run(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return js.ValueOf("error: run(source) takes a single argument")
	}
	result, _, err := adventlang.RunProgram("web", args[0].String())
	if err != nil {
		return js.ValueOf(fmt.Sprintf("uh oh..\n\n %v", err.Error()))
	}

	return js.ValueOf(result)
}
