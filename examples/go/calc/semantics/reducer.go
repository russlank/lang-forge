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

var reducers = calc.ReducerMap{
	calc.SemanticActionStart: calc.TypedStart(func(ctx calc.StartReduction) (float64, error) {
		return ctx.Value, nil
	}),
	calc.SemanticActionPass: calc.TypedPass(func(ctx calc.PassReduction) (float64, error) {
		return ctx.Value, nil
	}),
	calc.SemanticActionNumber: calc.TypedNumber(reduceNumber),
	calc.SemanticActionGroup: calc.TypedGroup(func(ctx calc.GroupReduction) (float64, error) {
		return ctx.Value, nil
	}),
	calc.SemanticActionNegate: calc.TypedNegate(func(ctx calc.NegateReduction) (float64, error) {
		return -ctx.Value, nil
	}),
	calc.SemanticActionAdd: calc.TypedAdd(func(ctx calc.AddReduction) (float64, error) {
		return ctx.Left + ctx.Right, nil
	}),
	calc.SemanticActionSubtract: calc.TypedSubtract(func(ctx calc.SubtractReduction) (float64, error) {
		return ctx.Left - ctx.Right, nil
	}),
	calc.SemanticActionMultiply: calc.TypedMultiply(func(ctx calc.MultiplyReduction) (float64, error) {
		return ctx.Left * ctx.Right, nil
	}),
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
