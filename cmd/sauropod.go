package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/healeycodes/sauropod/pkg/sauropod"
)

func main() {
	flag.Parse()
	file := flag.Arg(0)
	if file == "" {
		panic("missing file arg")
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		println("while parsing %v: %v\n", file, err)
		os.Exit(1)
	}
	source := string(b)

	result, err := sauropod.RunProgram(source)
	if err != nil {
		fmt.Printf("while running %v: %v\n", file, err)
		os.Exit(1)
	}

	fmt.Println(*result)
}
