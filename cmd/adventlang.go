package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/healeycodes/adventlang/pkg/adventlang"
)

func main() {
	flag.Parse()
	filename := flag.Arg(0)
	if filename == "" {
		panic("missing file argument")
	}

	source := adventlang.ReadProgram(filename)
	result, _, err := adventlang.RunProgram(filename, source)
	if err != nil {
		println("uh oh.. while running: "+filename, err.Error(), "\n")
		os.Exit(1)
	}

	fmt.Println(result)
}
