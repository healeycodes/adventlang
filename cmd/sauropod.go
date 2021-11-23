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
	filename := flag.Arg(0)
	if filename == "" {
		panic("missing file arg")
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		println("error reading:", filename, err.Error())
		os.Exit(1)
	}

	result, err := sauropod.RunProgram(filename, string(b))
	if err != nil {
		println("uh oh..", err.Error())
		os.Exit(1)
	}

	fmt.Println(result)
}
