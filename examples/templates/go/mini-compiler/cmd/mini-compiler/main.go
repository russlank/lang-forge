//go:build langforge_generated

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	minigen "github.com/russlank/lang-forge/examples/templates/go/mini-compiler/generated"
)

type program struct {
	Statements []statement
}

type statement struct {
	Expr expr
}

type expr interface {
	compile(*compiler)
}

type numberExpr struct {
	Value int
}

func (e numberExpr) compile(c *compiler) {
	c.emit("push", e.Value)
}

type addExpr struct {
	Left  expr
	Right expr
}

func (e addExpr) compile(c *compiler) {
	e.Left.compile(c)
	e.Right.compile(c)
	c.emit("add", 0)
}

type instruction struct {
	Op  string
	Arg int
}

type compiler struct {
	Code []instruction
}

func (c *compiler) emit(op string, arg int) {
	c.Code = append(c.Code, instruction{Op: op, Arg: arg})
}

func compileProgram(p program) []instruction {
	var c compiler
	for _, stmt := range p.Statements {
		stmt.Expr.compile(&c)
		c.emit("print", 0)
	}
	return c.Code
}

func run(code []instruction) ([]int, error) {
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

func parse(source string) (program, error) {
	lexemes, err := minigen.Tokenize(source)
	if err != nil {
		return program{}, err
	}
	value, err := minigen.ParseWithReducer(lexemes, minigen.ReducerFunc(reduce))
	if err != nil {
		return program{}, err
	}
	out, ok := value.(program)
	if !ok {
		return program{}, fmt.Errorf("parser returned %T, want program", value)
	}
	return out, nil
}

func reduce(ctx minigen.Reduction) (minigen.Value, error) {
	switch ctx.ActionID {
	case minigen.SemanticActionProgram:
		return program{Statements: statementsArg(ctx, 0, "statements")}, nil
	case minigen.SemanticActionStatements:
		return prependStatement(statementArg(ctx, 0, "statement"), statementsArg(ctx, 1, "statement tail")), nil
	case minigen.SemanticActionStatementsTailMore:
		return prependStatement(statementArg(ctx, 0, "statement"), statementsArg(ctx, 1, "statement tail")), nil
	case minigen.SemanticActionStatementsTailEmpty:
		return []statement{}, nil
	case minigen.SemanticActionPrint:
		return statement{Expr: exprArg(ctx, 1, "print expression")}, nil
	case minigen.SemanticActionAdd:
		return addExpr{Left: exprArg(ctx, 0, "left operand"), Right: exprArg(ctx, 2, "right operand")}, nil
	case minigen.SemanticActionPass:
		return ctx.Values[0], nil
	case minigen.SemanticActionNumber:
		text := lexemeArg(ctx, 0, "number").Text
		value, err := strconv.Atoi(text)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", text, err)
		}
		return numberExpr{Value: value}, nil
	default:
		if len(ctx.Values) == 1 {
			return ctx.Values[0], nil
		}
		return nil, nil
	}
}

func arg[T any](ctx minigen.Reduction, index int, name string) T {
	if index < 0 || index >= len(ctx.Values) {
		panic(fmt.Sprintf("rule %d missing %s at argument %d", ctx.Rule, name, index+1))
	}
	value, ok := ctx.Values[index].(T)
	if !ok {
		panic(fmt.Sprintf("rule %d argument %d for %s has type %T", ctx.Rule, index+1, name, ctx.Values[index]))
	}
	return value
}

func lexemeArg(ctx minigen.Reduction, index int, name string) minigen.Lexeme {
	return arg[minigen.Lexeme](ctx, index, name)
}

func exprArg(ctx minigen.Reduction, index int, name string) expr {
	return arg[expr](ctx, index, name)
}

func statementArg(ctx minigen.Reduction, index int, name string) statement {
	return arg[statement](ctx, index, name)
}

func statementsArg(ctx minigen.Reduction, index int, name string) []statement {
	return arg[[]statement](ctx, index, name)
}

func prependStatement(head statement, tail []statement) []statement {
	out := []statement{head}
	return append(out, tail...)
}

func report(name string, source string, code []instruction, output []int) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Mini compiler Go template: %s\n", name)
	fmt.Fprintln(&b, "source:")
	for _, line := range strings.Split(strings.TrimSpace(source), "\n") {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	fmt.Fprintln(&b, "stack code:")
	for i, inst := range code {
		if inst.Op == "push" {
			fmt.Fprintf(&b, "  %02d push %d\n", i, inst.Arg)
		} else {
			fmt.Fprintf(&b, "  %02d %s\n", i, inst.Op)
		}
	}
	fmt.Fprintf(&b, "output: %v\n", output)
	return b.String()
}

func main() {
	inputPath := flag.String("input", "input.mini", "mini language source file")
	logPath := flag.String("log", "", "optional report file")
	flag.Parse()
	source, err := os.ReadFile(*inputPath)
	if err != nil {
		exitf("read input: %v", err)
	}
	p, err := parse(string(source))
	if err != nil {
		exitf("parse: %v", err)
	}
	code := compileProgram(p)
	output, err := run(code)
	if err != nil {
		exitf("run: %v", err)
	}
	text := report(*inputPath, string(source), code, output)
	fmt.Print(text)
	if *logPath != "" {
		if err := os.WriteFile(*logPath, []byte(text), 0o644); err != nil {
			exitf("write log: %v", err)
		}
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
