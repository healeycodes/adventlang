package sauropod

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Program struct {
	Pos lexer.Position

	Statements []*Statement `@@*`
}

type Statement struct {
	Pos lexer.Position

	If     *IfStatement     `@@`
	For    *ForStatement    `| @@`
	While  *WhileStatement  `| @@`
	Return *ReturnStatement `| @@`
	Expr   *Expr            `| @@ ";"`
	Block  *Block           `| @@`
}

type IfStatement struct {
	Pos lexer.Position

	Condition *Expr  `"if" "(" @@ ")"`
	If        *Block `@@`
	Else      *Block `("else" @@)?`
}

type ForStatement struct {
	Pos lexer.Position

	Init      *Expr  `"for" "(" @@? ";"`
	Condition *Expr  `@@? ";"`
	Post      *Expr  `@@? ")"`
	Block     *Block `@@`
}

type WhileStatement struct {
	Pos lexer.Position

	Condition *Expr  ` "while" "(" @@? ")"`
	Block     *Block `@@`
}

type ReturnStatement struct {
	Pos lexer.Position

	Expr *Expr `"return" @@?`
}

type Block struct {
	Pos lexer.Position

	Statements []*Statement `"{" @@+ "}"`
}

type Expr struct {
	Pos lexer.Position

	Assignment *Assignment `@@`
}

type Assignment struct {
	Pos lexer.Position

	Let      *string   `@"let"?`
	LogicAnd *LogicAnd `@@`
	Op       *string   `( @"="`
	Next     *LogicAnd `  @@ )?`
}

type LogicAnd struct {
	Pos lexer.Position

	LogicOr *LogicOr  `@@`
	Op      *string   `( @( "and" )`
	Next    *LogicAnd `  @@ )?`
}

type LogicOr struct {
	Pos lexer.Position

	Equality *Equality `@@`
	Op       *string   `( @( "or" )`
	Next     *LogicOr  `  @@ )?`
}

type Equality struct {
	Pos lexer.Position

	Comparison *Comparison `@@`
	Op         *string     `[ @( "!" "=" | "=" "=" )`
	Next       *Equality   `  @@ ]`
}

type Comparison struct {
	Pos lexer.Position

	Addition *Addition   `@@`
	Op       *string     `[ @( ">" "=" | ">" | "<" "=" | "<" )`
	Next     *Comparison `  @@ ]`
}

type Addition struct {
	Pos lexer.Position

	Multiplication *Multiplication `@@`
	Op             *string         `[ @( "-" | "+" )`
	Next           *Addition       `  @@ ]`
}

type Multiplication struct {
	Pos lexer.Position

	Unary *Unary          `@@`
	Op    *string         `[ @( "/" | "*" )`
	Next  *Multiplication `  @@ ]`
}

type Unary struct {
	Pos lexer.Position

	Op      *string  `( @( "!" | "-" )`
	Unary   *Unary   `  @@ )`
	Primary *Primary `| @@`
}

type Primary struct {
	Pos lexer.Position

	FuncLiteral   *FuncLiteral   `@@`
	ListLiteral   *ListLiteral   `| @@`
	DictLiteral   *DictLiteral   `| @@`
	Call          *Call          `| @@`
	SubExpression *SubExpression `| @@`
	Number        *float64       `| ( @Float | @Int )`
	Str           *string        `| @String`
	True          *bool          `| @"true"`
	False         *bool          `| @"false"`
	Undefined     *string        `| @"undefined"`
	Ident         *string        `| @Ident`
}

type FuncLiteral struct {
	Pos lexer.Position

	Params []string `"func" "(" ( @Ident ( "," @Ident )* )? ")"`
	Block  *Block   `@@`
}

type ListLiteral struct {
	Pos lexer.Position

	Items []*Expr `"[" ( @@ ( "," @@ )* )? "]"`
}

type DictLiteral struct {
	Pos lexer.Position

	Items []*DictKV `"{" ( @@ ( "," @@ )* )? "}"`
}

type DictKV struct {
	Pos lexer.Position

	KeyExpr   *Expr   `( @@ |`
	KeyStr    *string `"'" @Ident "'")`
	ValueExpr *Expr   `":" @@`
}

type Call struct {
	Pos lexer.Position

	Ident     *string    `@Ident`
	CallChain *CallChain `@@`
}

type SubExpression struct {
	Pos lexer.Position

	SubExpression *Expr      `"(" @@ ")" `
	CallChain     *CallChain `@@?`
}

type CallChain struct {
	Pos lexer.Position

	Index    []*Expr    `( "[" @@ "]"`
	Property *string    ` | "." @Ident`
	Args     []*Expr    ` | "(" (@@ ("," @@)*)? ")" )`
	Next     *CallChain `@@?`
}

var (
	lex = lexer.MustSimple([]lexer.Rule{
		{"comment", `//.*|/\*.*?\*/`, nil},
		{"whitespace", `\s+`, nil},

		{"Float", `([0-9]*[.])?[0-9]+`, nil},
		{"Int", `[\d]+`, nil},
		{"String", `"([^"]*)"`, nil},
		{"Ident", `[\w]+`, nil},
		{"Punct", `[-[!*%()+_={}\|:;"<,>./]|]`, nil},
	})
	parser = participle.MustBuild(&Program{},
		participle.Lexer(lex),
		participle.UseLookahead(2))
)

func GetGrammer() string {
	return parser.String()
}

func GenerateAST(source string) (*Program, error) {
	ast := &Program{}
	err := parser.ParseString("", source, ast)
	if err != nil {
		return nil, err
	}
	return ast, nil
}
