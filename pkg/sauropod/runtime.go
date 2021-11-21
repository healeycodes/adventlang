package sauropod

import (
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/participle/v2/lexer"
)

func getType(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("type() expects 1 argument")
	}
	value := args[0]
	switch value.(type) {
	case IdentifierValue:
		return StringValue{val: []byte("identifier")}, nil
	case StringValue:
		return StringValue{val: []byte("string")}, nil
	case NumberValue:
		return StringValue{val: []byte("number")}, nil
	case BoolValue:
		return StringValue{val: []byte("bool")}, nil
	case FunctionValue, NativeFunctionValue:
		return StringValue{val: []byte("function")}, nil
	// case ListValue:
	// 	return StringValue{val: []byte("list")}, nil
	case DictValue:
		return StringValue{val: []byte("dict")}, nil
	case UndefinedValue:
		return StringValue{val: []byte("undefined")}, nil
	}
	return nil, fmt.Errorf("unknown type")
}

func setNativeFunc(key string, nativeFunc Value, frame *StackFrame) {
	frame.entries[key] = nativeFunc
}

func InjectRuntime(context *Context) {
	setNativeFunc("assert", NativeFunctionValue{name: "assert", Exec: runAssert}, &context.stackFrame)
	setNativeFunc("log", NativeFunctionValue{name: "log", Exec: runLog}, &context.stackFrame)
	setNativeFunc("time", NativeFunctionValue{name: "time", Exec: runTime}, &context.stackFrame)
}

type NativeFunctionValue struct {
	frame *StackFrame
	Pos   lexer.Position
	name  string
	Exec  func(*StackFrame, string, []Value) (Value, error)
}

func (nativeFunctionValue NativeFunctionValue) String() string {
	return nativeFunctionValue.name + " function"
}

func (nativeFunctionValue NativeFunctionValue) Equals(other Value) (bool, error) {
	if otherNatVal, okNatVal := other.(NativeFunctionValue); okNatVal {
		return nativeFunctionValue.name == otherNatVal.name, nil
	}
	return false, nil
}

func runAssert(frame *StackFrame, position string, args []Value) (Value, error) {
	callFrame := frame.GetChild("assert() called: " + position)
	if len(args) != 2 {
		return nil, traceError(callFrame, position,
			fmt.Sprintf("incorrect number of arguments, wanted: %v, got: %v", 2, len(args)))
	}
	equal, err := args[0].Equals(args[1])
	if err != nil {
		return nil, err
	}
	if !equal {
		return nil, fmt.Errorf("assert failed: %v == %v", args[0], args[1])
	}
	return UndefinedValue{}, nil
}

func runLog(frame *StackFrame, position string, args []Value) (Value, error) {
	callFrame := frame.GetChild("log() called: " + position)
	if len(args) == 0 {
		return nil, traceError(callFrame, position,
			fmt.Sprintf("incorrect number of arguments, wanted: at least 1, got: %v", len(args)))
	}
	s := make([]string, len(args))
	for i := range args {
		s[i] = args[i].String()
	}
	println(strings.Join(s, ", "))
	return UndefinedValue{}, nil
}

func runTime(frame *StackFrame, position string, args []Value) (Value, error) {
	callFrame := frame.GetChild("time() called: " + position)
	if len(args) != 0 {
		return nil, traceError(callFrame, position,
			fmt.Sprintf("incorrect number of arguments, wanted: 0, got: %v", len(args)))
	}
	return NumberValue{val: float64(time.Now().UnixNano() / int64(time.Millisecond))}, nil
}
