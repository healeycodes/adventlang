package adventlang

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type StackFrame struct {
	filename string
	trace    string
	entries  map[string]Value
	parent   *StackFrame
}

func traceError(frame *StackFrame, position string, message string) error {
	s := "\n" + frame.trace + "\n" + frame.filename + ":" + position + ": " + message
	for {
		if parent := frame.parent; parent != nil {
			frame = parent
			// TODO: build a better stack trace here instead of just reporting the
			// last two spans
		} else {
			break
		}
	}
	return fmt.Errorf(s)
}

type Context struct {
	stackFrame StackFrame
}

func (context *Context) Init(filename string) {
	context.stackFrame = StackFrame{
		filename: filename,
		trace:    "",
		entries:  make(map[string]Value),
	}
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
	childFrame := StackFrame{
		filename: frame.filename,
		trace:    trace,
		parent:   frame,
		entries:  make(map[string]Value),
	}
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

// Get a reference's internal value
func unref(value Value) Value {
	if refValue, okRef := value.(ReferenceValue); okRef {
		return *refValue.val
	}
	return value
}

// Turn an identifier into its resolution
func unwrap(value Value, frame *StackFrame) (Value, error) {
	// TODO: I'm not sure if this function can ever error
	// perhaps we can just return Value
	if idValue, okId := value.(IdentifierValue); okId {
		return frame.Get(idValue.val)
	}
	value = unref(value)
	return value, nil
}

type ReturnError struct {
	val Value
}

func (r ReturnError) Error() string {
	return "return statement used outside of a function, tried to return: " + r.val.String()
}

type BreakError struct {
	// TODO: Use this in traces
	position string
}

func (b BreakError) Error() string {
	return "break statement used outside of a loop"
}

type ContinueError struct {
	// TODO: Use this in traces
	position string
}

func (c ContinueError) Error() string {
	return "continue statement used outside of a loop"
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

func (strValue StringValue) Get(index int) (Value, error) {
	if index < 0 || index > len(strValue.val)-1 {
		return nil, fmt.Errorf("string index out of bounds: %v", index)
	}
	var value Value = StringValue{val: []byte{strValue.val[index]}}
	return ReferenceValue{val: &value}, nil
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
	// TODO: stringify function body?
	return "function (" + strings.Join(functionValue.parameters, ",") + ") "
}

func (functionValue FunctionValue) Equals(other Value) (bool, error) {
	return false, nil
}

func (functionValue FunctionValue) Exec(position string, args []Value) (Value, error) {
	callFrame := functionValue.frame.GetChild(functionValue.frame.filename + ":" + position + ": function call")
	if len(args) != len(functionValue.parameters) {
		return nil, traceError(callFrame, position,
			fmt.Sprintf("incorrect number of arguments, wanted: %v, got: %v", len(functionValue.parameters), len(args)))
	}
	for i, parameter := range functionValue.parameters {
		callFrame.Set(parameter, args[i])
	}
	for _, statement := range functionValue.statements {
		_, err := statement.Eval(callFrame)
		if err != nil {
			// Catch the bubbling return here
			if retErr, okRet := err.(ReturnError); okRet {
				value, err := unwrap(retErr.val, callFrame)
				if err != nil {
					return nil, err
				}
				return value, nil
			}
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

func (listValue ListValue) Popat(index int) (Value, error) {
	// Remove and return an item at `index`.
	// Correcting the remaining indexes costs O(N)
	if index < 0 || index > len(listValue.val)-1 {
		return nil, fmt.Errorf("list index out of bounds: %v", index)
	}
	item := *listValue.val[index]
	for i := index + 1; i < len(listValue.val); i++ { // b.
		// Overwrite an item by shifting down
		listValue.val[i-1] = listValue.val[i]
	}
	// Delete the last duplicate item
	delete(listValue.val, len(listValue.val)-1) // _______ c.
	return item, nil
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

func (dictValue *DictValue) Delete(key string) {
	delete(dictValue.val, key)
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
	value, err := evalBlock(frame, program.Statements)
	if err != nil {
		return nil, err
	}
	value, err = unwrap(value, frame)
	if err != nil {
		return nil, err
	}
	return value, nil
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
		// In this if block, we can escape to the nearest func
		if statement.Return.Expr == nil {
			return nil, ReturnError{val: UndefinedValue{}}
		}
		value, err := statement.Return.Expr.Eval(frame)
		if err != nil {
			return nil, err
		}
		return nil, ReturnError{val: value}
	}
	if statement.Break != nil {
		// Escape up to a loop (or error out)
		return nil, BreakError{position: statement.Pos.String()}
	}
	if statement.Continue != nil {
		// Escape up to a loop (or error out)
		return nil, ContinueError{position: statement.Pos.String()}
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
	ifFrame := frame.GetChild(frame.filename + ":" + ifStatement.Pos.String() + ": if statement")
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
	forFrame := frame.GetChild(frame.filename + ":" + forStatement.Pos.String() + ": for loop")
	// Having no init is fine
	if forStatement.Init != nil {
		_, err := forStatement.Init.Eval(forFrame)
		if err != nil {
			return nil, err
		}
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
	whileFrame := frame.GetChild(frame.filename + ":" + whileStatement.Pos.String() + ": while loop")
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
	left, err := assignment.LogicOr.Eval(frame)
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
				return nil, traceError(frame, assignment.LogicOr.Pos.String(),
					"can't assign to unknown variable: "+left.String())
			}
		}
		frame.Set(leftId.val, right)
		return right, nil
	}
	return nil, traceError(frame, assignment.LogicOr.Pos.String(),
		"can't assign to non-variable: "+left.String())
}

func (logicAnd LogicAnd) String() string {
	return "logic and"
}

func (logicAnd LogicAnd) Equals(other Value) (bool, error) {
	return false, nil
}

func (logicAnd LogicAnd) Eval(frame *StackFrame) (Value, error) {
	left, err := logicAnd.Equality.Eval(frame)
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

	if leftBoolValue, okLeftBool := left.(BoolValue); okLeftBool {
		if rightBoolValue, okRightBool := right.(BoolValue); okRightBool {
			return BoolValue{val: leftBoolValue.val && rightBoolValue.val}, nil
		}
	}
	return nil, traceError(frame, logicAnd.Pos.String(),
		"only bools can be compared with 'and', found: "+left.String()+" and "+right.String())

}

func (logicOr LogicOr) String() string {
	return "logic or"
}

func (logicOr LogicOr) Equals(other Value) (bool, error) {
	return false, nil
}

func (logicOr LogicOr) Eval(frame *StackFrame) (Value, error) {
	left, err := logicOr.LogicAnd.Eval(frame)
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

	if leftBoolValue, okLeftBool := left.(BoolValue); okLeftBool {
		if rightBoolValue, okRightBool := right.(BoolValue); okRightBool {
			return BoolValue{val: leftBoolValue.val || rightBoolValue.val}, nil
		}
	}
	return nil, traceError(frame, logicOr.Pos.String(),
		"only bools can be compared with 'or', found: "+left.String()+" and "+right.String())

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

	// Dicts, lists, functions are never equal
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
	return nil, traceError(frame, comparison.Addition.Pos.String(),
		"only numbers can be compared with "+*comparison.Op+" found: "+left.String()+" and "+right.String())
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
		"'+' can only be used between [string, string], [number, number], not: ["+left.String()+", "+right.String()+"]")

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
		"'*', '/', and '%' can only be used between [string, string], [number, number], not: ["+left.String()+", "+right.String()+"]")

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
		return NumberValue{
			val: float64(int(math.Round(leftNum.val)) % int(math.Round(rightNum.val))),
		}, nil
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
		return nil, traceError(frame, unary.Unary.Pos.String(),
			"expected bool after '!', found"+value.String())
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
		return nil, traceError(frame, unary.Unary.Pos.String(),
			"expected bool after '-', found"+value.String())
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
	closureFrame := frame.GetChild(frame.filename + ":" + functionLiteral.Pos.String() + ": function declared")
	functionValue := FunctionValue{
		position:   functionLiteral.Pos.String(),
		parameters: functionLiteral.Params,
		frame:      closureFrame,
		statements: functionLiteral.Block,
	}
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
				value, err = unwrap(value, frame)
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
	var condition Value
	var err error
	for {
		// Having no condition is fine, assume truthy
		if conditionExpr != nil {
			condition, err = conditionExpr.Eval(loopFrame)
			if err != nil {
				return nil, err
			}
		} else {
			condition = BoolValue{val: true}
		}

		if boolValue, okBool := condition.(BoolValue); okBool {
			if !boolValue.val {
				return UndefinedValue{}, nil
			}
			for _, statement := range block {
				_, err = statement.Eval(loopFrame)
				if err != nil {
					if _, okCont := err.(ContinueError); okCont {
						break
					}
					if _, okCont := err.(BreakError); okCont {
						return UndefinedValue{}, nil
					}
					return nil, err
				}
			}
			if post != nil {
				_, err = post.Eval(loopFrame)
				if err != nil {
					return nil, err
				}
			}
		} else {
			valueType, err := doType(loopFrame, conditionExpr.Pos.String(), []Value{condition})
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
			index, err := callChain.Index.Expr.Eval(frame)
			if err != nil {
				return nil, err
			}
			index, err = unwrap(index, frame)
			if err != nil {
				return nil, err
			}
			if dictValue, okDict := value.(DictValue); okDict {
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
					valueType, err := doType(frame, callChain.Index.Expr.Pos.String(), []Value{index})
					if err != nil {
						return nil, err
					}
					return nil, traceError(frame, callChain.Pos.String(),
						fmt.Sprintf("dictionaries can only be accessed by string: got '%v' of type %v", index, valueType))
				}
			}
			if listValue, okList := value.(ListValue); okList {
				if numberValue, okNumber := index.(NumberValue); okNumber {
					// Note that floats are floored here
					value, err = listValue.Get(int(numberValue.val))
					if err != nil {
						return nil, traceError(frame, callChain.Index.Expr.Pos.String(), err.Error())
					}
				} else {
					valueType, err := doType(frame, callChain.Index.Expr.Pos.String(), []Value{index})
					if err != nil {
						return nil, err
					}
					return nil, traceError(frame, callChain.Pos.String(),
						fmt.Sprintf("lists can only be accessed by number: got '%v' of type %v", index, valueType))
				}
			}
			if strValue, okStr := value.(StringValue); okStr {
				if numberValue, okNumber := index.(NumberValue); okNumber {
					// Note that floats are floored here
					value, err = strValue.Get(int(numberValue.val))
					if err != nil {
						return nil, traceError(frame, callChain.Index.Expr.Pos.String(), err.Error())
					}
				} else {
					valueType, err := doType(frame, callChain.Index.Expr.Pos.String(), []Value{index})
					if err != nil {
						return nil, err
					}
					return nil, traceError(frame, callChain.Pos.String(),
						fmt.Sprintf("strings can only be accessed by number: got '%v' of type %v", index, valueType))
				}
			}
		} else if callChain.Property != nil {
			if dictValue, okDict := value.(DictValue); okDict {
				reference, err := dictValue.Get(*callChain.Property.Ident)
				if err != nil {
					value = ReferenceValue{val: dictValue.Set(*callChain.Property.Ident, UndefinedValue{})}
				} else {
					value = ReferenceValue{val: reference}
				}
			}
			if listValue, okList := value.(ListValue); okList {
				// Check that the function will be called
				if callChain.Next != nil && callChain.Next.Args != nil {
					// Evaluate the arguments into values
					args, err := evalExprs(frame, callChain.Next.Args.Exprs)
					if err != nil {
						return nil, err
					}
					// Prepend the list value to the args
					// Note: args might be empty
					args = append([]Value{listValue}, args...)

					// Check for list functions
					if *callChain.Property.Ident == "append" {
						value, err = doAppend(frame, callChain.Pos.String(), args)
					} else if *callChain.Property.Ident == "pop" {
						value, err = doPop(frame, callChain.Pos.String(), args)
					} else if *callChain.Property.Ident == "prepend" {
						value, err = doPrepend(frame, callChain.Pos.String(), args)
					} else if *callChain.Property.Ident == "prepop" {
						value, err = doPrepop(frame, callChain.Pos.String(), args)
					} else if *callChain.Property.Ident == "popat" {
						value, err = doPopat(frame, callChain.Pos.String(), args)
					} else {
						return nil, traceError(frame, callChain.Next.Pos.String(),
							"unknown list function: "+*callChain.Property.Ident)
					}

					if err != nil {
						return nil, err
					}
					// Fast forward the callChain as we just handled the next step
					callChain = callChain.Next
				} else {
					// TODO: Are there any list properties we want to implement?
					return nil, traceError(frame, callChain.Pos.String(),
						"unknown list property: "+*callChain.Property.Ident)
				}
			}
		} else if callChain.Args != nil {
			args, err := evalExprs(frame, callChain.Args.Exprs)
			if err != nil {
				return nil, err
			}
			// TODO: do we need to unwrap here?
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
