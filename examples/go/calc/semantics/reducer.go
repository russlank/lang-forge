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
	calc.SemanticActionStart:    reducePass,
	calc.SemanticActionPass:     reducePass,
	calc.SemanticActionNumber:   reduceNumber,
	calc.SemanticActionGroup:    reduceGroup,
	calc.SemanticActionNegate:   reduceNegate,
	calc.SemanticActionAdd:      reduceBinary(func(left, right float64) float64 { return left + right }),
	calc.SemanticActionSubtract: reduceBinary(func(left, right float64) float64 { return left - right }),
	calc.SemanticActionMultiply: reduceBinary(func(left, right float64) float64 { return left * right }),
	calc.SemanticActionDivide:   reduceBinary(func(left, right float64) float64 { return left / right }),
}

// Reduce evaluates one calculator grammar reduction.
func Reduce(ctx calc.Reduction) (calc.Value, error) {
	return reducers.Reduce(ctx)
}

func reducePass(ctx calc.Reduction) (calc.Value, error) {
	return NumberArg(ctx, 0)
}

func reduceNumber(ctx calc.Reduction) (calc.Value, error) {
	lexeme, err := LexemeArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	value, err := strconv.ParseFloat(lexeme.Text, 64)
	if err != nil {
		return nil, fmt.Errorf("rule %d number %q: %w", ctx.Rule, lexeme.Text, err)
	}
	return value, nil
}

func reduceGroup(ctx calc.Reduction) (calc.Value, error) {
	return NumberArg(ctx, 1)
}

func reduceNegate(ctx calc.Reduction) (calc.Value, error) {
	value, err := NumberArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return -value, nil
}

func reduceBinary(op func(left float64, right float64) float64) calc.ReductionHandler {
	return func(ctx calc.Reduction) (calc.Value, error) {
		left, err := NumberArg(ctx, 0)
		if err != nil {
			return nil, err
		}
		right, err := NumberArg(ctx, 2)
		if err != nil {
			return nil, err
		}
		return op(left, right), nil
	}
}

// NumberArg returns a float64 reduction argument.
func NumberArg(ctx calc.Reduction, index int) (float64, error) {
	if index < 0 || index >= len(ctx.Values) {
		return 0, fmt.Errorf("rule %d action %q missing numeric argument %d", ctx.Rule, ctx.Action, index+1)
	}
	value, ok := ctx.Values[index].(float64)
	if !ok {
		return 0, fmt.Errorf("rule %d action %q argument %d has type %T, want float64", ctx.Rule, ctx.Action, index+1, ctx.Values[index])
	}
	return value, nil
}

// LexemeArg returns a generated scanner lexeme reduction argument.
func LexemeArg(ctx calc.Reduction, index int) (calc.Lexeme, error) {
	if index < 0 || index >= len(ctx.Values) {
		return calc.Lexeme{}, fmt.Errorf("rule %d action %q missing lexeme argument %d", ctx.Rule, ctx.Action, index+1)
	}
	lexeme, ok := ctx.Values[index].(calc.Lexeme)
	if !ok {
		return calc.Lexeme{}, fmt.Errorf("rule %d action %q argument %d has type %T, want Lexeme", ctx.Rule, ctx.Action, index+1, ctx.Values[index])
	}
	return lexeme, nil
}
