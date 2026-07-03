package model

import "fmt"

// Program is the root AST node produced by the LangForge parser.
type Program struct {
	Statements []Statement
}

// Statement represents one `print expr;` source statement.
type Statement struct {
	Expr Expr
}

// Expr is implemented by every mini-language expression node.
type Expr interface {
	compile(*compiler)
}

// NumberExpr stores an integer literal parsed from a Number token.
type NumberExpr struct {
	Value int
}

func (e NumberExpr) compile(c *compiler) {
	c.emit("push", e.Value)
}

// AddExpr stores the operands for `left + right`.
type AddExpr struct {
	Left  Expr
	Right Expr
}

func (e AddExpr) compile(c *compiler) {
	e.Left.compile(c)
	e.Right.compile(c)
	c.emit("add", 0)
}

// Instruction is one operation in the tiny stack-machine code emitted by the
// compiler stage.
type Instruction struct {
	Op  string
	Arg int
}

type compiler struct {
	Code []Instruction
}

func (c *compiler) emit(op string, arg int) {
	c.Code = append(c.Code, Instruction{Op: op, Arg: arg})
}

// CompileProgram lowers the AST into stack-machine instructions.
func CompileProgram(p Program) []Instruction {
	var c compiler
	for _, stmt := range p.Statements {
		stmt.Expr.compile(&c)
		c.emit("print", 0)
	}
	return c.Code
}

// Run executes stack-machine instructions and returns every printed value.
func Run(code []Instruction) ([]int, error) {
	var stack []int
	var output []int
	for pc, inst := range code {
		switch inst.Op {
		case "push":
			stack = append(stack, inst.Arg)
		case "add":
			if len(stack) < 2 {
				return nil, fmt.Errorf("pc %d: add needs two stack values", pc)
			}
			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]
			stack = append(stack, left+right)
		case "print":
			if len(stack) < 1 {
				return nil, fmt.Errorf("pc %d: print needs one stack value", pc)
			}
			output = append(output, stack[len(stack)-1])
			stack = stack[:len(stack)-1]
		default:
			return nil, fmt.Errorf("pc %d: unknown instruction %q", pc, inst.Op)
		}
	}
	return output, nil
}
