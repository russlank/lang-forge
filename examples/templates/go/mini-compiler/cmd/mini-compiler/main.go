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
	minimodel "github.com/russlank/lang-forge/examples/templates/go/mini-compiler/model"
)

var reducers = minigen.ReducerMap{
	minigen.SemanticActionProgram: minigen.TypedProgram(func(ctx minigen.ProgramReduction) (minimodel.Program, error) {
		return minimodel.Program{Statements: ctx.Statements}, nil
	}),
	minigen.SemanticActionStatements: minigen.TypedStatements(func(ctx minigen.StatementsReduction) ([]minimodel.Statement, error) {
		return prependStatement(ctx.Head, ctx.Tail), nil
	}),
	minigen.SemanticActionStatementsTailMore: minigen.TypedStatementsTailMore(func(ctx minigen.StatementsTailMoreReduction) ([]minimodel.Statement, error) {
		return prependStatement(ctx.Head, ctx.Tail), nil
	}),
	minigen.SemanticActionStatementsTailEmpty: minigen.TypedStatementsTailEmpty(func(minigen.StatementsTailEmptyReduction) ([]minimodel.Statement, error) {
		return []minimodel.Statement{}, nil
	}),
	minigen.SemanticActionPrint: minigen.TypedPrint(func(ctx minigen.PrintReduction) (minimodel.Statement, error) {
		return minimodel.Statement{Expr: ctx.Expr}, nil
	}),
	minigen.SemanticActionAdd: minigen.TypedAdd(func(ctx minigen.AddReduction) (minimodel.Expr, error) {
		return minimodel.AddExpr{Left: ctx.Left, Right: ctx.Right}, nil
	}),
	minigen.SemanticActionPass: minigen.TypedPass(func(ctx minigen.PassReduction) (minimodel.Expr, error) {
		return ctx.Value, nil
	}),
	minigen.SemanticActionNumber: minigen.TypedNumber(reduceNumber),
}

func parse(source string) (minimodel.Program, error) {
	value, err := minigen.ParseWithReducerFromLexemeSource(minigen.NewScanner(source), reducers)
	if err != nil {
		return minimodel.Program{}, err
	}
	out, ok := value.(minimodel.Program)
	if !ok {
		return minimodel.Program{}, fmt.Errorf("parser returned %T, want model.Program", value)
	}
	return out, nil
}

func reduceNumber(ctx minigen.NumberReduction) (minimodel.Expr, error) {
	value, err := strconv.Atoi(ctx.Token.Text)
	if err != nil {
		return nil, fmt.Errorf("rule %d number %q: %w", ctx.Reduction.Rule, ctx.Token.Text, err)
	}
	return minimodel.NumberExpr{Value: value}, nil
}

func prependStatement(head minimodel.Statement, tail []minimodel.Statement) []minimodel.Statement {
	out := []minimodel.Statement{head}
	return append(out, tail...)
}

func report(name string, source string, code []minimodel.Instruction, output []int) string {
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
	code := minimodel.CompileProgram(p)
	output, err := minimodel.Run(code)
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
