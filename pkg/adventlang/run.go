package adventlang

import (
	"fmt"
	"io/ioutil"
	"os"
)

const VERSION = 0.1

func ReadProgram(filename string) string {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		println("while trying to read: ", filename, err.Error())
		os.Exit(1)
	}
	return string(b)
}

func RunProgram(filename string, source string) (string, *Context, error) {
	program, err := GenerateAST(source)
	if err != nil {
		return "", nil, fmt.Errorf("\n%v:%v", filename, err.Error())
	}

	context := Context{}
	context.Init(filename)
	InjectRuntime(&context)

	result, err := program.Eval(&context.stackFrame)
	if err != nil {
		return "", nil, err
	}

	return result.String(), &context, nil
}
