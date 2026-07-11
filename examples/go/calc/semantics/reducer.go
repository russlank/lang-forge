//go:build langforge_generated

// Package semantics contains handwritten calculator reductions for the
// LangForge-generated calc parser.
//
// The calc grammar uses reducer-mode action labels such as {go: add}. LangForge
// copies those labels into generated SemanticAction constants and
// Reduction.ActionID; this package defines what each label means for the
// calculator.
package semantics

import (
	"fmt"
	"strconv"

	calc "github.com/russlank/lang-forge/examples/go/calc/generated"
)

// reducers maps generated action IDs back to the grammar rules in calc.lf.
//
// Reading tip for newcomers: each comment below repeats the associated grammar
// alternative. The generated typed context field names come from RHS labels
// such as value=Expr, left=Expr, right=Term, and token=Number.
var reducers = calc.ReducerMap{
	// S : value=Expr {go: start}
	calc.SemanticActionStart: calc.TypedStart(func(ctx calc.StartReduction) (float64, error) {
		return ctx.Value, nil
	}),
	// Expr : value=Term {go: pass}
	// Term : value=Factor {go: pass}
	calc.SemanticActionPass: calc.TypedPass(func(ctx calc.PassReduction) (float64, error) {
		return ctx.Value, nil
	}),
	// Factor : token=Number {go: number}
	calc.SemanticActionNumber: calc.TypedNumber(reduceNumber),
	// Factor : LParen value=Expr RParen {go: group}
	calc.SemanticActionGroup: calc.TypedGroup(func(ctx calc.GroupReduction) (float64, error) {
		return ctx.Value, nil
	}),
	// Factor : Minus value=Factor {go: negate}
	calc.SemanticActionNegate: calc.TypedNegate(func(ctx calc.NegateReduction) (float64, error) {
		return -ctx.Value, nil
	}),
	// Expr : left=Expr Plus right=Term {go: add}
	calc.SemanticActionAdd: calc.TypedAdd(func(ctx calc.AddReduction) (float64, error) {
		return ctx.Left + ctx.Right, nil
	}),
	// Expr : left=Expr Minus right=Term {go: subtract}
	calc.SemanticActionSubtract: calc.TypedSubtract(func(ctx calc.SubtractReduction) (float64, error) {
		return ctx.Left - ctx.Right, nil
	}),
	// Term : left=Term Mul right=Factor {go: multiply}
	calc.SemanticActionMultiply: calc.TypedMultiply(func(ctx calc.MultiplyReduction) (float64, error) {
		return ctx.Left * ctx.Right, nil
	}),
	// Term : left=Term Div right=Factor {go: divide}
	calc.SemanticActionDivide: calc.TypedDivide(reduceDivide),
}

// Reduce evaluates one calculator grammar reduction.
func Reduce(ctx calc.Reduction) (calc.Value, error) {
	return reducers.Reduce(ctx)
}

func reduceNumber(ctx calc.NumberReduction) (float64, error) {
	value, err := strconv.ParseFloat(ctx.Token.Text, 64)
	if err != nil {
		return 0, fmt.Errorf("rule %d number %q: %w", ctx.Reduction.Rule, ctx.Token.Text, err)
	}
	return value, nil
}

func reduceDivide(ctx calc.DivideReduction) (float64, error) {
	if ctx.Right == 0 {
		return 0, fmt.Errorf("rule %d action %q: division by zero", ctx.Reduction.Rule, ctx.Reduction.Action)
	}
	return ctx.Left / ctx.Right, nil
}
