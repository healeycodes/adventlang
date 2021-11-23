package sauropod

const VERSION = 0.1

func RunProgram(filename string, source string) (string, error) {
	program, err := GenerateAST(source)
	if err != nil {
		return "", err
	}

	context := Context{}
	context.Init()
	InjectRuntime(&context)

	result, err := program.Eval(&context.stackFrame)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}
