package sauropod

import (
	"fmt"
	"strconv"
	"strings"
)

type StackFrame struct {
	trace   string
	entries map[string]Value
	parent  *StackFrame
}

func traceError(frame *StackFrame, position string, message string) error {
	s := frame.trace + "\n\t " + position + " " + message
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
	return nil, fmt.Errorf("missing key: %v", key)
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

func (_ ReferenceValue) String() string {
	return "reference"
}

func (_ ReferenceValue) Equals(other Value) (bool, error) {
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
	var result Value
	result = UndefinedValue{}
	var err error
	for _, node := range program.Statements {
		result, err = node.Eval(frame)
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
		return (*statement.If).Eval(frame)
	}

	// For    *ForStatement    `| @@`
	// While  *WhileStatement  `| @@`
	// Return *ReturnStatement `| @@`
	// Block  *Block           `| @@`

	if statement.Expr != nil {
		return (*statement.Expr).Eval(frame)
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
			return (*ifStatement.If).Eval(ifFrame)
		}
		return (*ifStatement.Else).Eval(ifFrame)
	}
	return nil, traceError(frame, ifStatement.Condition.Pos.String(),
		"conditional should evaluate to true or false")
}

func (block Block) String() string {
	return "block"
}

func (block Block) Equals(other Value) (bool, error) {
	return false, nil
}

func (block Block) Eval(frame *StackFrame) (Value, error) {
	var result Value
	result = UndefinedValue{}
	var err error
	for _, node := range block.Statements {
		result, err = node.Eval(frame)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
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
		frame.Set(leftId.val, right)
		return right, nil
	}
	return nil, traceError(frame, assignment.LogicAnd.Pos.String(), "can't assign to non-identifier: "+left.String())
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
	leftType, err := getType([]Value{left})
	if err != nil {
		return nil, err
	}
	rightType, err := getType([]Value{right})
	if err != nil {
		return nil, err
	}
	return nil, traceError(frame, comparison.Addition.Pos.String(), "only numbers can be compared with "+*comparison.Op+"  found: "+leftType.String()+" and "+rightType.String())
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
	if *addition.Op == "+" {
		if okLeft && !okRight || okRight && !okLeft {
			return nil, err
		}
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
	return nil, err
}
