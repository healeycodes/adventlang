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
		} else {
			break
		}
	}
	return fmt.Errorf(s)
}

type Context struct {
	stackFrame StackFrame
}

func (context *Context) Init() {
	context.stackFrame = StackFrame{trace: "", entries: make(map[string]Value)}
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

// These errors are for escaping loop execution
// Error() is never surfaced to users
type BreakError struct{}

func (b *BreakError) Error() string {
	return "unreachable"
}

type ContinueError struct{}

func (c *ContinueError) Error() string {
	return "unreachable"
}

// Language value
// ranging from strings, numbers, to functions, lists, and dicts
type Value interface {
	String() string
	Equals(Value) (bool, error)
}

// Sometimes we want to bubble up a reference to a list or dict item
// so that it can be reassigned. Use `unref` to turn into a plain value
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

// Turn a variable into its resolution
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
	for _, statement := range functionValue.statements {
		if statement.Return != nil {
			return statement.Return.Expr.Eval(callFrame)
		}
		_, err := statement.Eval(callFrame)
		if err != nil {
			return nil, err
		}
	}
	return UndefinedValue{}, nil
}

type ListValue struct {
	val map[int]*Value
}

func (listValue *ListValue) Get(index int) (Value, error) {
	if index < 0 || index > len(listValue.val)-1 {
		return nil, fmt.Errorf("list index out of bounds: %v", index)
	}
	value, ok := listValue.val[index]
	if !ok {
		// All values between the bounds should be valid
		panic("unreachable")
	}
	return ReferenceValue{val: value}, nil
}

func (listValue ListValue) String() string {
	items := make([]string, len(listValue.val))
	for i, item := range listValue.val {
		items[i] = (*item).String()
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func (listValue ListValue) Equals(other Value) (bool, error) {
	return false, nil
}

func (listValue ListValue) Append(other Value) {
	listValue.val[len(listValue.val)] = &other
}

func (listValue ListValue) Prepend(other Value) {
	// Add a new zeroth item.
	// Correcting the remaining indexes costs O(N)
	for i := len(listValue.val); i > 0; i-- {
		listValue.val[i] = listValue.val[i-1]
	}
	listValue.val[0] = &other
}

func (listValue ListValue) Pop() Value {
	last := *listValue.val[len(listValue.val)-1]
	delete(listValue.val, len(listValue.val)-1)
	return last
}

func (listValue ListValue) PopLeft() Value {
	// Remove and return the zeroth item.
	// Correcting the remaining indexes costs O(N)
	first := *listValue.val[0]
	delete(listValue.val, 0)
	for i := 0; i < len(listValue.val); i++ {
		listValue.val[i] = listValue.val[i+1]
	}
	delete(listValue.val, len(listValue.val)-1)
	return first
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

func (dictValue *DictValue) Set(key string, value Value) *Value {
	dictValue.val[key] = &value
	return &value
}

func (dictValue DictValue) String() string {
	s := make([]string, 0)
	for key, value := range dictValue.val {
		s = append(s, fmt.Sprintf("\"%v\": %v", key, *value))
	}
	return "{" + strings.Join(s, ", ") + "}"
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
	return evalBlock(frame, program.Statements)
}

func evalBlock(frame *StackFrame, statements []*Statement) (Value, error) {
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
	if statement.If != nil {
		return statement.If.Eval(frame)
	}
	if statement.For != nil {
		return statement.For.Eval(frame)
	}
	if statement.While != nil {
		return statement.While.Eval(frame)
	}
	if statement.Return != nil {
		return nil, traceError(frame, statement.Pos.String(),
			"return statement used outside of a function")
	}
	if statement.Break != nil {
		return nil, traceError(frame, statement.Pos.String(),
			"break statement used outside of a loop")
	}
	if statement.Continue != nil {
		return nil, traceError(frame, statement.Pos.String(),
			"continue statement used outside of a loop")
	}
	if statement.Expr != nil {
		return statement.Expr.Eval(frame)
	}
	panic("unreachable")
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
			return evalBlock(ifFrame, ifStatement.If)
		}
		return evalBlock(ifFrame, ifStatement.Else)
	}
	return nil, traceError(frame, ifStatement.Condition.Pos.String(),
		"conditional should evaluate to true or false")
}

func (forStatement ForStatement) String() string {
	return "for statement"
}

func (forStatement ForStatement) Equals(other Value) (bool, error) {
	return false, nil
}

func (forStatement ForStatement) Eval(frame *StackFrame) (Value, error) {
	forFrame := frame.GetChild("for: " + forStatement.Pos.String())
	_, err := forStatement.Init.Eval(forFrame)
	if err != nil {
		return nil, err
	}
	return evalLoop(forFrame, forStatement.Condition, forStatement.Block, forStatement.Post)
}

func (whileStatement WhileStatement) String() string {
	return "while statement"
}

func (whileStatement WhileStatement) Equals(other Value) (bool, error) {
	return false, nil
}

func (whileStatement WhileStatement) Eval(frame *StackFrame) (Value, error) {
	whileFrame := frame.GetChild("while: " + whileStatement.Pos.String())
	return evalLoop(whileFrame, whileStatement.Condition, whileStatement.Block, nil)
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
		if leftRefOk {
			return *leftRef.val, nil
		}
		return left, nil
	}

	// assignment.Op is "="
	right, err := assignment.Next.Eval(frame)
	if err != nil {
		return nil, err
	}
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
	return nil, traceError(frame, comparison.Addition.Pos.String(), "only numbers can be compared with "+*comparison.Op+" found: "+left.String()+" and "+right.String())
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

func (primary Primary) Eval(frame *StackFrame) (Value, error) {
	if primary.FuncLiteral != nil {
		return primary.FuncLiteral.Eval(frame)
	}
	if primary.ListLiteral != nil {
		return primary.ListLiteral.Eval(frame)
	}
	if primary.DictLiteral != nil {
		return primary.DictLiteral.Eval(frame)
	}
	if primary.Call != nil {
		return primary.Call.Eval(frame)
	}
	if primary.SubExpression != nil {
		return primary.SubExpression.Eval(frame)
	}
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
	panic("unreachable")
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

func (listLiteral ListLiteral) String() string {
	return "list literal"
}

func (listLiteral ListLiteral) Equals(other Value) (bool, error) {
	return false, nil
}

func (listLiteral ListLiteral) Eval(frame *StackFrame) (Value, error) {
	values := make(map[int]*Value, 0)
	for i, expr := range listLiteral.Items {
		value, err := expr.Eval(frame)
		if err != nil {
			return nil, err
		}
		values[i] = &value
	}
	return ListValue{val: values}, nil
}

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
	value, err := frame.Get(*call.Ident)
	if err != nil {
		return nil, err
	}
	return evalCallChain(frame, value, call.CallChain)
}

func (subExpression SubExpression) String() string {
	return "sub expression"
}

func (subExpression SubExpression) Equals(other Value) (bool, error) {
	return false, nil
}

func (subExpression SubExpression) Eval(frame *StackFrame) (Value, error) {
	value, err := subExpression.Expr.Eval(frame)
	if err != nil {
		return nil, err
	}
	if subExpression.CallChain != nil {
		return evalCallChain(frame, value, subExpression.CallChain)
	}
	return value, nil
}

func evalLoop(loopFrame *StackFrame, conditionExpr *Expr, block []*Statement, post *Expr) (Value, error) {
	for {
		condition, err := conditionExpr.Eval(loopFrame)
		if err != nil {
			return nil, err
		}
		if boolValue, okBool := condition.(BoolValue); okBool {
			if !boolValue.val {
				return UndefinedValue{}, nil
			}
			for _, statement := range block {
				if statement.Break != nil {
					return UndefinedValue{}, nil
				} else if statement.Continue != nil {
					break
				} else {
					_, err = statement.Eval(loopFrame)
					if err != nil {
						return nil, err
					}
				}
			}
			if post != nil {
				_, err = post.Eval(loopFrame)
				if err != nil {
					return nil, err
				}
			}
		} else {
			valueType, err := getType(loopFrame, conditionExpr.Pos.String(), []Value{condition})
			if err != nil {
				return nil, err
			}
			return nil, traceError(loopFrame, conditionExpr.Pos.String(),
				"loop condition expression should evaluate to a boolean, found: "+valueType.String())
		}
	}
}

func evalCallChain(frame *StackFrame, value Value, callChain *CallChain) (Value, error) {
	for {
		value = unref(value)
		if callChain.Index != nil {
			if dictValue, okDict := value.(DictValue); okDict {
				index, err := callChain.Index.Expr.Eval(frame)
				if err != nil {
					return nil, err
				}
				index, err = unwrap(index, frame)
				if err != nil {
					return nil, err
				}
				index = unref(index)
				// When indexing a dict by number, we stringify it
				if numberValue, okNumber := index.(NumberValue); okNumber {
					index = StringValue{val: []byte(nvToS(numberValue))}
				}
				if stringValue, okString := index.(StringValue); okString {
					reference, err := dictValue.Get(string(stringValue.val))
					if err != nil {
						value = ReferenceValue{val: dictValue.Set(string(stringValue.val), UndefinedValue{})}
					} else {
						value = ReferenceValue{val: reference}
					}
				} else {
					valueType, err := getType(frame, callChain.Index.Expr.Pos.String(), []Value{index})
					if err != nil {
						return nil, err
					}
					return nil, traceError(frame, callChain.Pos.String(), fmt.Sprintf("dictionaries can only be accessed by string: got '%v' of type %v", index, valueType))
				}
			}
			if listValue, okList := value.(ListValue); okList {
				index, err := callChain.Index.Expr.Eval(frame)
				if err != nil {
					return nil, err
				}
				index, err = unwrap(index, frame)
				if err != nil {
					return nil, err
				}
				index = unref(index)
				if numberValue, okNumber := index.(NumberValue); okNumber {
					// Note that floats are floored here
					value, err = listValue.Get(int(numberValue.val))
					if err != nil {
						return nil, traceError(frame, callChain.Index.Expr.Pos.String(), err.Error())
					}
				} else {
					valueType, err := getType(frame, callChain.Index.Expr.Pos.String(), []Value{index})
					if err != nil {
						return nil, err
					}
					return nil, traceError(frame, callChain.Pos.String(), fmt.Sprintf("lists can only be accessed by number: got '%v' of type %v", index, valueType))
				}
			}
		}
		if callChain.Property != nil {
			// TODO: Dict API (keys, values)
			// TODO: List API (append, etc.)
			if dictValue, okDict := value.(DictValue); okDict {
				reference, err := dictValue.Get(*callChain.Property.Ident)
				if err != nil {
					value = ReferenceValue{val: dictValue.Set(*callChain.Property.Ident, UndefinedValue{})}
				} else {
					value = ReferenceValue{val: reference}
				}
			}
		}
		if callChain.Args != nil {
			args, err := evalExprs(frame, callChain.Args.Exprs)
			if err != nil {
				return nil, err
			}
			// TODO: do we need unwrap/unref here?
			if function, okFunction := value.(FunctionValue); okFunction {
				value, err = function.Exec(callChain.Pos.String(), args)
				if err != nil {
					return nil, err
				}
			} else if nativeFunction, okNativeFunction := value.(NativeFunctionValue); okNativeFunction {
				nativeFunction.frame = frame
				value, err = nativeFunction.Exec(frame, callChain.Pos.String(), args)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, traceError(frame, callChain.Pos.String(), "only functions can be called")
			}
		}
		if callChain.Next == nil {
			break
		}
		callChain = callChain.Next
	}

	return value, nil
}

func evalExprs(frame *StackFrame, exprs []*Expr) ([]Value, error) {
	ret := make([]Value, 0)
	for _, expr := range exprs {
		result, err := expr.Eval(frame)
		if err != nil {
			return nil, err
		}
		unwrapped, err := unwrap(result, frame)
		if err != nil {
			return nil, err
		}
		ret = append(ret, unwrapped)
	}
	return ret, nil
}
