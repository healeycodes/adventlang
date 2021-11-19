package sauropod

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type StackFrame struct {
	trace   string
	entries map[string]Value
	parent  *StackFrame
}

func traceError(frame *StackFrame, position string, message string) error {
	s := frame.trace + "\n" + position + " " + message
	for {
		if parent := frame.parent; parent != nil {
			frame = parent
			s = parent.trace + "\n" + s
		} else {
			break
		}
	}
	return fmt.Errorf(s)
}

type Context struct {
	stackFrame StackFrame
}

func (context *Context) Init(trace string) {
	context.stackFrame = StackFrame{trace: trace, entries: make(map[string]Value)}
}

func (frame *StackFrame) String() string {
	s := ""
	for {
		s += "{\n"
		for key, value := range frame.entries {
			s += fmt.Sprintf("\t %v: %v\n", key, value)
		}
		s += "}"
		if parent := frame.parent; parent != nil {
			frame = parent
		} else {
			break
		}
	}
	return s
}

func (frame *StackFrame) GetChild(trace string) *StackFrame {
	childFrame := StackFrame{trace: trace, parent: frame, entries: make(map[string]Value)}
	return &childFrame
}

// Get a variable's value by looking through every scope (bottom to top)
func (frame *StackFrame) Get(key string) (Value, error) {
	for {
		value, ok := frame.entries[key]
		if ok {
			return value, nil
		}
		if parent := frame.parent; parent != nil {
			frame = parent
		} else {
			break
		}
	}
	return nil, fmt.Errorf("variable not declared: %v", key)
}

// Set a variable by looking through every scope (bottom to top)
// until an existing variable is found. If there is no matching
// variable, declare a new variable in the current scope
func (frame *StackFrame) Set(key string, value Value) {
	currentFrame := frame
	for {
		_, ok := frame.entries[key]
		if ok {
			frame.entries[key] = value
			return
		}
		if parent := frame.parent; parent != nil {
			frame = parent
		} else {
			break
		}
	}
	currentFrame.entries[key] = value
}

type Value interface {
	String() string
	Equals(Value) (bool, error)
}

type ReferenceValue struct {
	val *Value
}

func (referenceValue ReferenceValue) String() string {
	return "reference"
}

func (referenceValue ReferenceValue) Equals(other Value) (bool, error) {
	return false, nil
}

func unref(value Value) Value {
	if refValue, okRef := value.(ReferenceValue); okRef {
		return *refValue.val
	}
	return value
}

func unwrap(value Value, frame *StackFrame) (Value, error) {
	if idValue, okId := value.(IdentifierValue); okId {
		return frame.Get(idValue.val)
	}
	value = unref(value)
	return value, nil
}

type UndefinedValue struct{}

func (undefinedValue UndefinedValue) String() string {
	return "undefined"
}

func (undefinedValue UndefinedValue) Equals(other Value) (bool, error) {
	if _, ok := other.(UndefinedValue); ok {
		return true, nil
	}
	return false, nil
}

type IdentifierValue struct {
	val string
}

func (identifierValue IdentifierValue) String() string {
	return identifierValue.val
}

func (identifierValue IdentifierValue) Equals(other Value) (bool, error) {
	return false, fmt.Errorf("tried to compare an uninitialized identifier: %v", identifierValue)
}

type NumberValue struct {
	val float64
}

func (numberValue NumberValue) String() string {
	return nToS(numberValue.val)
}

func (numberValue NumberValue) Equals(other Value) (bool, error) {
	if other, ok := unref(other).(NumberValue); ok {
		return numberValue.val == other.val, nil
	}
	return false, nil
}

func nvToS(numberValue NumberValue) string {
	return nToS(numberValue.val)
}

func nToS(n float64) string {
	return strconv.FormatFloat(n, 'f', -1, 64)
}

type StringValue struct {
	val []byte
}

func (stringValue StringValue) String() string {
	return string(stringValue.val)
}

func (stringValue StringValue) Equals(other Value) (bool, error) {
	if otherStr, ok := unref(other).(StringValue); ok {
		a := stringValue.val
		b := otherStr.val
		if len(a) != len(b) {
			return false, nil
		}
		for i := range a {
			if a[i] != b[i] {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

type BoolValue struct {
	val bool
}

func (boolValue BoolValue) String() string {
	if boolValue.val {
		return "true"
	}
	return "false"
}

func (boolValue BoolValue) Equals(other Value) (bool, error) {
	if otherValue, okBool := unref(other).(BoolValue); okBool {
		return boolValue.val == otherValue.val, nil
	}
	return false, nil
}

type FunctionValue struct {
	position   string
	parameters []string
	frame      *StackFrame
	statements []*Statement
}

func (functionValue FunctionValue) String() string {
	// TODO: stringify function body
	return "function (" + strings.Join(functionValue.parameters, ",") + ") "
}

func (functionValue FunctionValue) Equals(other Value) (bool, error) {
	return false, nil
}

func (functionValue FunctionValue) Exec(position string, args []Value) (Value, error) {
	callFrame := functionValue.frame.GetChild("function called: " + position)
	if len(args) != len(functionValue.parameters) {
		return nil, traceError(functionValue.frame, position,
			fmt.Sprintf("incorrect number of arguments, wanted: %v, got: %v", len(functionValue.parameters), len(args)))
	}
	for i, parameter := range functionValue.parameters {
		callFrame.Set(parameter, args[i])
	}
	var result Value
	result = UndefinedValue{}
	var err error
	for _, statement := range functionValue.statements {
		result, err = statement.Eval(callFrame)
		if err != nil {
			return nil, err
		}
		if statement.Return != nil {
			break
		}
	}
	return result, nil
}

type DictValue struct {
	val map[string]*Value
}

func (dictValue *DictValue) Get(key string) (*Value, error) {
	value, ok := dictValue.val[key]
	if ok {
		return value, nil
	}
	return nil, fmt.Errorf("key missing from dictionary: %v", key)
}

func (dictValue *DictValue) Set(key string, value Value) {
	dictValue.val[key] = &value
}

func (dictValue DictValue) String() string {
	s := make([]string, 0)
	s = append(s, "{")
	for key, value := range dictValue.val {
		s = append(s, fmt.Sprintf("\"%v\": %v", key, *value))
	}
	s = append(s, "}")
	return strings.Join(s, "")
}

func (dictValue DictValue) Equals(other Value) (bool, error) {
	return false, nil
}

// ---

func (program Program) String() string {
	s := make([]string, 0)
	for _, statement := range program.Statements {
		s = append(s, statement.String())
	}
	return strings.Join(s, ", ")
}

func (program Program) Equals(other Value) (bool, error) {
	return false, nil
}

func (program Program) Eval(frame *StackFrame) (Value, error) {
	return block(frame, program.Statements)
}

func block(frame *StackFrame, statements []*Statement) (Value, error) {
	var result Value
	result = UndefinedValue{}
	var err error
	for _, statement := range statements {
		result, err = statement.Eval(frame)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (statement Statement) String() string {
	return "statement"
}

func (statement Statement) Equals(other Value) (bool, error) {
	return false, nil
}

func (statement Statement) Eval(frame *StackFrame) (Value, error) {
	var result Value
	result = UndefinedValue{}

	if statement.If != nil {
		return statement.If.Eval(frame)
	}

	// For    *ForStatement    `| @@`
	// While  *WhileStatement  `| @@`
	if statement.Return != nil {
		return statement.Return.Expr.Eval(frame)
	}
	// Block  *Block           `| @@`

	if statement.Expr != nil {
		return statement.Expr.Eval(frame)
	}

	return result, nil
}

func (ifStatement IfStatement) String() string {
	return "if statement"
}

func (ifStatement IfStatement) Equals(other Value) (bool, error) {
	return false, nil
}

func (ifStatement IfStatement) Eval(frame *StackFrame) (Value, error) {
	ifFrame := frame.GetChild("if: " + ifStatement.Pos.String())
	condition, err := ifStatement.Condition.Eval(ifFrame)
	if err != nil {
		return nil, err
	}

	if boolValue, okBool := condition.(BoolValue); okBool {
		if boolValue.val {
			return block(ifFrame, ifStatement.If)
		}
		return block(ifFrame, ifStatement.Else)
	}
	return nil, traceError(frame, ifStatement.Condition.Pos.String(),
		"conditional should evaluate to true or false")
}

func (expr Expr) String() string {
	return "expression"
}

func (expr Expr) Equals(other Value) (bool, error) {
	return false, nil
}

func (expr Expr) Eval(frame *StackFrame) (Value, error) {
	return expr.Assignment.Eval(frame)
}

func (assignment Assignment) String() string {
	return "assignment"
}

func (assignment Assignment) Equals(other Value) (bool, error) {
	return false, nil
}

func (assignment Assignment) Eval(frame *StackFrame) (Value, error) {
	left, err := assignment.LogicAnd.Eval(frame)
	if err != nil {
		return nil, err
	}
	leftRef, leftRefOk := left.(ReferenceValue)

	if assignment.Op == nil {
		// Are we a naked variable?
		if leftId, okId := left.(IdentifierValue); okId {
			return frame.Get(leftId.val)
		}
		return left, nil
	}

	right, err := assignment.Next.Eval(frame)
	if err != nil {
		return nil, err
	}

	// assignment.Op is "="
	if idValue, okId := right.(IdentifierValue); okId {
		right, err = frame.Get(idValue.val)
		if err != nil {
			return nil, err
		}
	}
	if leftRefOk {
		*leftRef.val = right
		return right, nil
	}
	if leftId, okId := left.(IdentifierValue); okId {
		if assignment.Let == nil {
			_, err = frame.Get(leftId.val)
			if err != nil {
				return nil, traceError(frame, assignment.LogicAnd.Pos.String(), "can't assign to unknown variable: "+left.String())
			}
		}
		frame.Set(leftId.val, right)
		return right, nil
	}
	return nil, traceError(frame, assignment.LogicAnd.Pos.String(), "can't assign to non-variable: "+left.String())
}

func (logicAnd LogicAnd) String() string {
	return "logic and"
}

func (logicAnd LogicAnd) Equals(other Value) (bool, error) {
	return false, nil
}

func (logicAnd LogicAnd) Eval(frame *StackFrame) (Value, error) {
	left, err := logicAnd.LogicOr.Eval(frame)
	if err != nil {
		return nil, err
	}
	if logicAnd.Op == nil {
		return left, nil
	}
	right, err := logicAnd.Next.Eval(frame)
	if err != nil {
		return nil, err
	}
	left, err = unwrap(left, frame)
	if err != nil {
		return nil, err
	}
	right, err = unwrap(right, frame)
	if err != nil {
		return nil, err
	}

	if boolValue, okBool := left.(BoolValue); okBool {
		if boolValue.val {
			if boolValue, okBool := right.(BoolValue); okBool {
				if boolValue.val {
					return boolValue, nil
				}
			} else {
				return nil, traceError(frame, logicAnd.Pos.String(), "only bools can be compared with 'and', found: "+right.String())
			}
		}
	} else {
		return nil, traceError(frame, logicAnd.Pos.String(), "only bools can be compared with 'and', found: "+left.String())
	}
	panic("unreachable")
}

func (logicOr LogicOr) String() string {
	return "logic or"
}

func (logicOr LogicOr) Equals(other Value) (bool, error) {
	return false, nil
}

func (logicOr LogicOr) Eval(frame *StackFrame) (Value, error) {
	left, err := logicOr.Equality.Eval(frame)
	if err != nil {
		return nil, err
	}
	if logicOr.Op == nil {
		return left, nil
	}
	right, err := logicOr.Next.Eval(frame)
	if err != nil {
		return nil, err
	}
	left, err = unwrap(left, frame)
	if err != nil {
		return nil, err
	}
	right, err = unwrap(right, frame)
	if err != nil {
		return nil, err
	}

	if boolValue, okBool := left.(BoolValue); okBool {
		if boolValue.val {
			return boolValue, nil
		}
	} else {
		return nil, traceError(frame, logicOr.Pos.String(), "only bools can be compared with 'and', found: "+left.String())
	}
	if boolValue, okBool := right.(BoolValue); okBool {
		if boolValue.val {
			return boolValue, nil
		}
	} else {
		return nil, traceError(frame, logicOr.Pos.String(), "only bools can be compared with 'and', found: "+right.String())
	}
	panic("unreachable")
}

func (equality Equality) String() string {
	return "equality"
}

func (equality Equality) Equals(other Value) (bool, error) {
	return false, nil
}

func (equality Equality) Eval(frame *StackFrame) (Value, error) {
	left, err := equality.Comparison.Eval(frame)
	if err != nil {
		return nil, err
	}
	if equality.Op == nil {
		return left, nil
	}
	right, err := equality.Next.Eval(frame)
	if err != nil {
		return nil, err
	}
	left = unref(left)
	right = unref(right)

	if idValue, okId := left.(IdentifierValue); okId {
		value, err := frame.Get(idValue.val)
		if err != nil {
			return nil, err
		}
		left = value
	}
	if idValue, okId := right.(IdentifierValue); okId {
		value, err := frame.Get(idValue.val)
		if err != nil {
			return nil, err
		}
		right = value
	}

	// TODO: Check for equal dicts, lists, funcs here

	result, err := left.Equals(right)
	if err != nil {
		return nil, err
	}
	if *equality.Op == "==" {
		return BoolValue{val: result}, nil
	} else if *equality.Op == "!=" {
		return BoolValue{val: !result}, nil
	}
	panic("unreachable")
}

func (comparison Comparison) String() string {
	return "comparison"
}

func (comparison Comparison) Equals(other Value) (bool, error) {
	return false, nil
}

func (comparison Comparison) Eval(frame *StackFrame) (Value, error) {
	left, err := comparison.Addition.Eval(frame)
	if err != nil {
		return nil, err
	}
	if comparison.Op == nil {
		return left, nil
	}
	right, err := comparison.Next.Eval(frame)
	if err != nil {
		return nil, err
	}

	left, err = unwrap(left, frame)
	if err != nil {
		return nil, err
	}
	right, err = unwrap(right, frame)
	if err != nil {
		return nil, err
	}

	if leftNum, okNum := left.(NumberValue); okNum {
		if rightNum, okNum := right.(NumberValue); okNum {
			return BoolValue{val: *comparison.Op == "<" && leftNum.val < rightNum.val ||
				*comparison.Op == "<=" && leftNum.val <= rightNum.val ||
				*comparison.Op == ">" && leftNum.val > rightNum.val ||
				*comparison.Op == ">=" && leftNum.val >= rightNum.val}, nil
		}
	}
	return nil, traceError(frame, comparison.Addition.Pos.String(), "only numbers can be compared with "+*comparison.Op+"  found: "+left.String()+" and "+right.String())
}

func (addition Addition) String() string {
	return "addition"
}

func (addition Addition) Equals(other Value) (bool, error) {
	return false, nil
}

func (addition Addition) Eval(frame *StackFrame) (Value, error) {
	left, err := addition.Multiplication.Eval(frame)
	if err != nil {
		return nil, err
	}

	if addition.Op == nil {
		return left, nil
	}

	right, err := addition.Next.Eval(frame)
	if err != nil {
		return nil, err
	}

	left, err = unwrap(left, frame)
	if err != nil {
		return nil, err
	}

	right, err = unwrap(right, frame)
	if err != nil {
		return nil, err
	}

	err = traceError(frame, addition.Multiplication.Pos.String(),
		"'+' can only be used between [string, string], [number, number], [list, list], not: ["+left.String()+", "+right.String()+"]")

	leftStr, okLeft := left.(StringValue)
	rightStr, okRight := right.(StringValue)
	if *addition.Op == "+" && (okLeft && !okRight || okRight && !okLeft) {
		return nil, err
	} else if *addition.Op == "+" && okLeft && okRight {
		return StringValue{val: append([]byte{}, append(leftStr.val, rightStr.val...)...)}, nil
	}

	leftNum, okLeft := left.(NumberValue)
	rightNum, okRight := right.(NumberValue)
	if okLeft && !okRight || okRight && !okLeft {
		return nil, err
	}
	if *addition.Op == "+" && okLeft && okRight {
		return NumberValue{val: leftNum.val + rightNum.val}, nil
	}
	if *addition.Op == "-" && okLeft && okRight {
		return NumberValue{val: leftNum.val - rightNum.val}, nil
	}
	if *addition.Op == "-" {
		return nil, traceError(frame, addition.Multiplication.Pos.String(),
			"'-' and '+' can only be used between [number, number], not: ["+left.String()+", "+right.String()+"]")
	}
	return nil, err
}

func (multiplication Multiplication) String() string {
	return "multiplication"
}

func (multiplication Multiplication) Equals(other Value) (bool, error) {
	return false, nil
}

func (multiplication Multiplication) Eval(frame *StackFrame) (Value, error) {
	left, err := multiplication.Unary.Eval(frame)
	if err != nil {
		return nil, err
	}
	if multiplication.Op == nil {
		return left, nil
	}
	right, err := multiplication.Next.Eval(frame)
	if err != nil {
		return nil, err
	}

	left, err = unwrap(left, frame)
	if err != nil {
		return nil, err
	}
	right, err = unwrap(right, frame)
	if err != nil {
		return nil, err
	}

	err = traceError(frame, multiplication.Unary.Pos.String(),
		"'*' and '/' can only be used between [string, string], [number, number], [list, list], not: ["+left.String()+", "+right.String()+"]")

	leftNum, okLeft := left.(NumberValue)
	if !okLeft {
		return nil, err
	}
	rightNum, okRight := right.(NumberValue)
	if !okRight {
		return nil, err
	}
	if *multiplication.Op == "*" {
		return NumberValue{val: leftNum.val * rightNum.val}, nil
	}
	if *multiplication.Op == "/" {
		return NumberValue{val: leftNum.val / rightNum.val}, nil
	}
	if *multiplication.Op == "%" {
		return NumberValue{val: float64(int(math.Round(leftNum.val)) % int(math.Round(rightNum.val)))}, nil
	}
	panic("unreachable")
}

func (unary Unary) Eval(frame *StackFrame) (Value, error) {
	if unary.Op == nil {
		return unary.Primary.Eval(frame)
	}
	if *unary.Op == "!" {
		value, err := unary.Unary.Eval(frame)
		if err != nil {
			return nil, err
		}
		value, err = unwrap(value, frame)
		if err != nil {
			return nil, err
		}
		if boolValue, ok := value.(BoolValue); ok {
			return BoolValue{val: !boolValue.val}, nil
		}
		return nil, traceError(frame, unary.Unary.Pos.String(), "expected bool after '!', found"+value.String())
	}
	if *unary.Op == "-" {
		value, err := unary.Unary.Eval(frame)
		if err != nil {
			return nil, err
		}
		value, err = unwrap(value, frame)
		if err != nil {
			return nil, err
		}
		if numberValue, ok := value.(NumberValue); ok {
			return NumberValue{val: -numberValue.val}, nil
		}
		return nil, traceError(frame, unary.Unary.Pos.String(), "expected bool after '-', found"+value.String())
	}
	panic("unreachable")
}

func (primary Primary) String() string {
	return "primary"
}

func (functionLiteral FuncLiteral) String() string {
	return "function literal"
}

func (functionLiteral FuncLiteral) Equals(other Value) (bool, error) {
	return false, nil
}

func (functionLiteral FuncLiteral) Eval(frame *StackFrame) (Value, error) {
	closureFrame := frame.GetChild("function declared: " + functionLiteral.Pos.String())
	functionValue := FunctionValue{position: functionLiteral.Pos.String(), parameters: functionLiteral.Params, frame: closureFrame, statements: functionLiteral.Block}
	return functionValue, nil
}

func (primary Primary) Eval(frame *StackFrame) (Value, error) {
	if primary.FuncLiteral != nil {
		return primary.FuncLiteral.Eval(frame)
	}
	// List          *ListLiteral   `| @@`
	if primary.DictLiteral != nil {
		return primary.DictLiteral.Eval(frame)
	}
	if primary.Call != nil {
		return primary.Call.Eval(frame)
	}
	// SubExpression *SubExpression `| @@`
	if primary.Number != nil {
		return NumberValue{val: *primary.Number}, nil
	}
	if primary.Str != nil {
		// TODO: Parse strings without including quote `"` marks
		return StringValue{val: []byte(*primary.Str)[1 : len((*primary.Str))-1]}, nil
	}
	if primary.True != nil {
		return BoolValue{val: true}, nil
	}
	if primary.False != nil {
		return BoolValue{val: false}, nil
	}
	if primary.Undefined != nil {
		return UndefinedValue{}, nil
	}
	if ident := primary.Ident; ident != nil {
		identifierValue := IdentifierValue{val: *ident}
		return identifierValue, nil
	}
	panic("unimplemented")
}

// type FuncLiteral struct {

// type ListLiteral struct {

func (dictLiteral DictLiteral) String() string {
	return "dictionary literal"
}

func (dictLiteral DictLiteral) Equals(other Value) (bool, error) {
	return false, nil
}

func (dictLiteral DictLiteral) Eval(frame *StackFrame) (Value, error) {
	dictValue := DictValue{val: make(map[string]*Value)}
	if dictLiteral.Items != nil {
		for _, dictKV := range dictLiteral.Items {
			var key string
			if dictKV.KeyExpr != nil {
				value, err := dictKV.KeyExpr.Eval(frame)
				if err != nil {
					return nil, err
				}
				if strValue, okStr := value.(StringValue); okStr {
					key = string(strValue.val)
				}
			} else if dictKV.KeyStr != nil {
				key = *dictKV.KeyStr
			}

			value, err := dictKV.ValueExpr.Eval(frame)
			if err != nil {
				return nil, err
			}
			if key == "" {
				return nil, traceError(frame, dictLiteral.Pos.String(), "can't set empty string as dictionary key")
			}
			dictValue.Set(key, value)
		}
	}
	return dictValue, nil
}

func (call Call) String() string {
	return "call"
}

func (call Call) Equals(other Value) (bool, error) {
	return false, nil
}

func (call Call) Eval(frame *StackFrame) (Value, error) {
	println("call eval")
	var result Value
	callable, err := frame.Get(*call.Ident)
	if err != nil {
		return nil, err
	}

	if function, okFunction := callable.(FunctionValue); okFunction {
		println("it was a func..")
		if call.CallChain.Index != nil || call.CallChain.Property != nil {
			return nil, traceError(frame, call.CallChain.Pos.String(),
				"can't access into function, perhaps you meant to call it?")
		}
		args, err := evalExprs(frame, call.CallChain.Args.Exprs)
		if err != nil {
			return nil, err
		}
		result, err = function.Exec(call.Pos.String(), args)
		if err != nil {
			return nil, err
		}
	}

	callChain := call.CallChain
	next := true
	for next {
		println("loop")

		if callChain.Index != nil {
			//
		}
		if callChain.Property != nil {
			//
		}
		if callChain.Args != nil {
			if function, okFunction := result.(FunctionValue); okFunction {
				println("func")
				args, err := evalExprs(frame, callChain.Args.Exprs)
				if err != nil {
					return nil, err
				}
				result, err = function.Exec(callChain.Pos.String(), args)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, traceError(frame, callChain.Pos.String(),
					"can't access into function, perhaps you meant to call it?")
			}
		}
		callChain = callChain.Next
		next = callChain != nil
	}

	return result, nil
}

// type SubExpression struct {

// type CallChain struct {

func evalExprs(frame *StackFrame, exprs []*Expr) ([]Value, error) {
	ret := make([]Value, 0)
	for _, expr := range exprs {
		result, err := expr.Eval(frame)
		if err != nil {
			return nil, err
		}
		ret = append(ret, result)
	}
	return ret, nil
}
