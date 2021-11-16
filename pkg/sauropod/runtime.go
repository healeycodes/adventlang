package sauropod

import "fmt"

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
	case ListValue:
		return StringValue{val: []byte("list")}, nil
	case DictValue:
		return StringValue{val: []byte("dict")}, nil
	case UndefinedValue:
		return StringValue{val: []byte("undefined")}, nil
	}
	return nil, fmt.Errorf("unknown type")
}
