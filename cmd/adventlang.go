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
		println("missing file argument")
		os.Exit(1)
	}

	source := adventlang.ReadProgram(filename)
	result, _, err := adventlang.RunProgram(filename, source)
	if err != nil {
		println("uh oh.. while running: "+filename, err.Error(), "\n")
		os.Exit(1)
	}

	fmt.Println(result)
}
