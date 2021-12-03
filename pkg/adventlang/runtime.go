package adventlang

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// A note on function naming
// Use doFunction to avoid polluting the Go namespace with e.g.
// append, time, etc.

// Given a root content, add runtime functions to the module's scope
func InjectRuntime(context *Context) {
	setNativeFunc("import", NativeFunctionValue{name: "import", Exec: doImport}, &context.stackFrame)
	setNativeFunc("keys", NativeFunctionValue{name: "keys", Exec: doKeys}, &context.stackFrame)
	setNativeFunc("values", NativeFunctionValue{name: "keys", Exec: doValues}, &context.stackFrame)
	setNativeFunc("delete", NativeFunctionValue{name: "delete", Exec: doDelete}, &context.stackFrame)
	setNativeFunc("len", NativeFunctionValue{name: "len", Exec: doLen}, &context.stackFrame)
	setNativeFunc("append", NativeFunctionValue{name: "append", Exec: doAppend}, &context.stackFrame)
	setNativeFunc("prepend", NativeFunctionValue{name: "prepend", Exec: doPrepend}, &context.stackFrame)
	setNativeFunc("pop", NativeFunctionValue{name: "pop", Exec: doPop}, &context.stackFrame)
	setNativeFunc("popat", NativeFunctionValue{name: "popat", Exec: doPopat}, &context.stackFrame)
	setNativeFunc("prepop", NativeFunctionValue{name: "prepop", Exec: doPrepop}, &context.stackFrame)
	setNativeFunc("assert", NativeFunctionValue{name: "assert", Exec: doAssert}, &context.stackFrame)
	setNativeFunc("log", NativeFunctionValue{name: "log", Exec: doLog}, &context.stackFrame)
	setNativeFunc("time", NativeFunctionValue{name: "time", Exec: doTime}, &context.stackFrame)
	setNativeFunc("type", NativeFunctionValue{name: "type", Exec: doType}, &context.stackFrame)
	setNativeFunc("str", NativeFunctionValue{name: "str", Exec: doStr}, &context.stackFrame)
	setNativeFunc("num", NativeFunctionValue{name: "num", Exec: doNum}, &context.stackFrame)
	setNativeFunc("floor", NativeFunctionValue{name: "floor", Exec: doFloor}, &context.stackFrame)
	setNativeFunc("read_lines", NativeFunctionValue{name: "read_lines", Exec: doReadLines}, &context.stackFrame)
}

func setNativeFunc(key string, nativeFunc Value, frame *StackFrame) {
	frame.entries[key] = nativeFunc
}

type NativeFunctionValue struct {
	frame *StackFrame
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

// When importing, we run other modules as if they are standalone programs.
// When the program completes, we take its top level scope and return a dict
func doImport(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("import: incorrect number of arguments, wanted: 1, got: %v ", len(args)))
	}
	if strValue, okStr := args[0].(StringValue); okStr {
		source := ReadProgram(strValue.String())
		_, context, err := RunProgram(strValue.String(), source)
		if err != nil {
			return nil, err
		}
		dictValue := DictValue{val: map[string]*Value{}}
		for id, value := range context.stackFrame.entries {
			dictValue.Set(id, value)
		}
		return dictValue, nil
	}
	argType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"import: the single argument should be a string, got: "+argType.String())
}

func doKeys(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("keys: incorrect number of arguments, wanted: 1, got: %v ", len(args)))
	}
	if dictValue, okDict := args[0].(DictValue); okDict {
		listValue := ListValue{val: make(map[int]*Value)}
		for key := range dictValue.val {
			listValue.Append(StringValue{val: []byte(key)})
		}
		return listValue, nil
	}
	argType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"keys: the single argument should be a dictionary, got: "+argType.String())
}

func doValues(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("values: incorrect number of arguments, wanted: 1, got: %v ", len(args)))
	}
	if dictValue, okDict := args[0].(DictValue); okDict {
		listValue := ListValue{val: make(map[int]*Value)}
		for key := range dictValue.val {
			value, err := dictValue.Get(key)
			if err != nil {
				panic(err)
			}
			listValue.Append(*value)
		}
		return listValue, nil
	}
	argType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"values: the single argument should be a dictionary, got: "+argType.String())
}

func doDelete(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, traceError(frame, position,
			fmt.Sprintf("delete: incorrect number of arguments, wanted: 2, got: %v ", len(args)))
	}

	if dictValue, okDict := args[0].(DictValue); okDict {
		if strValue, okStr := args[1].(StringValue); okStr {
			dictValue.Delete(strValue.String())
			return UndefinedValue{}, nil
		} else {
			secondType, err := doType(frame, position, []Value{args[0]})
			if err != nil {
				return nil, err
			}
			return nil, traceError(frame, position,
				"delete: the 2nd argument should be a string, got: "+secondType.String())
		}
	} else {
		firstType, err := doType(frame, position, []Value{args[0]})
		if err != nil {
			return nil, err
		}
		return nil, traceError(frame, position,
			"delete: 1st argument should be a dictionary, got: "+firstType.String())
	}
}

func doLen(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("len: incorrect number of arguments, wanted: 1, got: %v ", len(args)))
	}
	if idValue, idOk := args[0].(IdentifierValue); idOk {
		unwrapped, err := unwrap(idValue, frame)
		if err != nil {
			return nil, err
		}
		return doLen(frame, position, []Value{unwrapped})
	}
	if strValue, strOk := args[0].(StringValue); strOk {
		return NumberValue{val: float64(len(strValue.val))}, nil
	}
	if listValue, listOk := args[0].(ListValue); listOk {
		return NumberValue{val: float64(len(listValue.val))}, nil
	}
	argType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"len: the single argument should be a variable, string, or list, got: "+argType.String())
}

func doAppend(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, traceError(frame, position,
			fmt.Sprintf("append: incorrect number of arguments, wanted: 2, got: %v ", len(args)))
	}
	if listValue, listOk := args[0].(ListValue); listOk {
		// 2nd argument can be any type
		// anything the user has access to should fit in a list
		listValue.Append(args[1])
		return UndefinedValue{}, nil
	}
	firstType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"append: 1st argument should be a list, got: "+firstType.String())
}

func doPrepend(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, traceError(frame, position,
			fmt.Sprintf("prepend: incorrect number of arguments, wanted: 2, got: %v ", len(args)))
	}
	if listValue, listOk := args[0].(ListValue); listOk {
		// 2nd argument can be any type
		// anything the user has access to should fit in a list
		listValue.Prepend(args[1])
		return UndefinedValue{}, nil
	}
	firstType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"prepend: 1st argument should be a list, got: "+firstType.String())
}

func doPop(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("pop: incorrect number of arguments, wanted: 1, got: %v ", len(args)))
	}
	if listValue, listOk := args[0].(ListValue); listOk {
		if len(listValue.val) == 0 {
			return nil, traceError(frame, position, "pop: called on an empty list")
		}
		return listValue.Popat(len(listValue.val) - 1)
	}
	firstType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"pop: the single argument should be a list, got: "+firstType.String())
}

func doPopat(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, traceError(frame, position,
			fmt.Sprintf("popat: incorrect number of arguments, wanted: 2, got: %v ", len(args)))
	}
	if listValue, listOk := args[0].(ListValue); listOk {
		if len(listValue.val) == 0 {
			return nil, traceError(frame, position, "popat: called on an empty list")
		}
		if numValue, numOk := args[1].(NumberValue); numOk {
			// Floor the index
			return listValue.Popat(int(numValue.val))
		}
		secondType, err := doType(frame, position, []Value{args[0]})
		if err != nil {
			return nil, err
		}
		return nil, traceError(frame, position,
			"popat: the 2nd argument should be a number, got: "+secondType.String())
	}
	firstType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"popat: the 1st argument should be a list, got: "+firstType.String())
}

func doPrepop(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("prepop: incorrect number of arguments, wanted: 1, got: %v ", len(args)))
	}
	if listValue, listOk := args[0].(ListValue); listOk {
		if len(listValue.val) == 0 {
			return nil, traceError(frame, position, "prepop: called on an empty list")
		}
		return listValue.Popat(0)
	}
	firstType, err := doType(frame, position, []Value{args[0]})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		"prepop: the single argument should be a list, got: "+firstType.String())
}

func doAssert(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, traceError(frame, position,
			fmt.Sprintf("assert: incorrect number of arguments, wanted: 2, got: %v", len(args)))
	}
	equal, err := args[0].Equals(args[1])
	if err != nil {
		return nil, err
	}
	if !equal {
		return nil, traceError(frame, position,
			fmt.Sprintf("assert failed: %v == %v", args[0], args[1]))
	}
	return UndefinedValue{}, nil
}

func doLog(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) == 0 {
		return nil, traceError(frame, position,
			fmt.Sprintf("log: incorrect number of arguments, wanted: at least 1, got: %v", len(args)))
	}
	s := make([]string, len(args))
	for i := range args {
		s[i] = args[i].String()
	}
	println(strings.Join(s, ", "))
	return UndefinedValue{}, nil
}

func doTime(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 0 {
		return nil, traceError(frame, position,
			fmt.Sprintf("time: incorrect number of arguments, wanted: 0, got: %v", len(args)))
	}
	return NumberValue{val: float64(time.Now().UnixNano() / int64(time.Millisecond))}, nil
}

func doType(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("type: incorrect number of arguments, wanted: 1, got: %v", len(args)))
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
	case ReferenceValue:
		return StringValue{val: []byte("reference")}, nil
	}
	panic("unreachable")
}

func doStr(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("str: incorrect number of arguments, wanted: 1, got: %v", len(args)))
	}
	value := args[0]
	if strValue, okStr := value.(StringValue); okStr {
		return strValue, nil
	}
	if numValue, okNum := value.(NumberValue); okNum {
		return StringValue{val: []byte(nvToS(numValue))}, nil
	}
	if boolValue, okBool := value.(BoolValue); okBool {
		if boolValue.val {
			return StringValue{val: []byte("true")}, nil
		}
		return StringValue{val: []byte("false")}, nil
	}

	valueType, err := doType(frame, position, args)
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		fmt.Sprintf("str: expects a single argument of type string, number, or bool, got: %v", valueType))
}

func doFloor(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("floor: incorrect number of arguments, wanted: 1, got: %v", len(args)))
	}
	value := args[0]
	if numValue, okNum := value.(NumberValue); okNum {
		return NumberValue{val: float64(int(numValue.val))}, nil
	}
	valueType, err := doType(frame, position, args)
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		fmt.Sprintf("num: expects a single argument of type number, got: %v", valueType))
}

func doNum(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, traceError(frame, position,
			fmt.Sprintf("num: incorrect number of arguments, wanted: 1, got: %v", len(args)))
	}
	value := args[0]
	if strValue, okStr := value.(StringValue); okStr {
		f, err := strconv.ParseFloat(string(strValue.val), 64)
		if err != nil {
			traceError(frame, position,
				fmt.Sprintf("num: couldn't convert %v to number", strValue))
		}
		return NumberValue{val: f}, nil
	}
	valueType, err := doType(frame, position, args)
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, position,
		fmt.Sprintf("num: expects a single argument of type string, got: %v", valueType))
}

func doReadLines(frame *StackFrame, position string, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, traceError(frame, position,
			fmt.Sprintf("read_lines: incorrect number of arguments, wanted: 2, got: %v", len(args)))
	}
	var path string
	var callback FunctionValue
	if stringValue, stringOk := args[0].(StringValue); stringOk {
		path = stringValue.String()
	} else {
		valueType, err := doType(frame, position, args)
		if err != nil {
			return nil, err
		}
		return nil, traceError(frame, position,
			fmt.Sprintf("read_lines: expects the 1st argument to be a filepath, got: %v", valueType))
	}
	if functionValue, functionOk := args[1].(FunctionValue); functionOk {
		callback = functionValue
	} else {
		valueType, err := doType(frame, position, args)
		if err != nil {
			return nil, err
		}
		return nil, traceError(frame, position,
			fmt.Sprintf("read_lines: expects the 2nd argument to be a function, got: %v", valueType))
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, traceError(frame, position,
			fmt.Sprintf("read_lines: while reading %v: %v", path, err))
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		arg := StringValue{val: []byte(scanner.Text())}
		_, err = callback.Exec(callback.position, []Value{arg})
		if err != nil {
			return nil, traceError(frame, position,
				fmt.Sprintf("read_lines: while reading %v: %v", path, err))
		}
	}
	if err := scanner.Err(); err != nil {
		if err != nil {
			return nil, traceError(frame, position,
				fmt.Sprintf("read_lines: while reading %v: %v", path, err))
		}
	}
	return UndefinedValue{}, nil
}
