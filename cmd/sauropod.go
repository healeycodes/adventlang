package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/healeycodes/sauropod/pkg/sauropod"
)

func main() {
	flag.Parse()
	filename := flag.Arg(0)
	if filename == "" {
		panic("missing file argument")
	}

	source := sauropod.ReadProgram(filename)
	result, _, err := sauropod.RunProgram(filename, source)
	if err != nil {
		println("uh oh.. while running: "+filename, err.Error(), "\n")
		os.Exit(1)
	}

	fmt.Println(result)
}
