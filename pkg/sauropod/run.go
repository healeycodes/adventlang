package sauropod

import "fmt"

const VERSION = 0.1

func RunProgram(filename string, source string) (string, error) {
	program, err := GenerateAST(source)
	if err != nil {
		return "", err
	}

	context := Context{}
	context.Init(filename)
	InjectRuntime(&context)

	result, err := program.Eval((&context.stackFrame))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", result), nil
}
