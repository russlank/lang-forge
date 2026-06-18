//go:build langforge_generated

package draw

import (
	"fmt"
	"image/color"
	"strconv"

	drawgenerated "github.com/russlank/lang-forge/examples/go/draw/generated"
)

// Parse converts source text into a DRAW AST using the generated parser.
//
// This file is handwritten adapter code. The generated scanner/parser itself
// lives under examples/go/draw/generated and is recreated by the Makefile.
func Parse(source string) (*Program, error) {
	lexemes, err := drawgenerated.Tokenize(source)
	if err != nil {
		return nil, err
	}
	value, err := drawgenerated.ParseWithReducer(lexemes, drawgenerated.ReducerFunc(drawReduce))
	if err != nil {
		return nil, err
	}
	program, ok := value.(*Program)
	if !ok {
		return nil, fmt.Errorf("generated parser returned %T, want *Program", value)
	}
	return program, nil
}

type binaryTail struct {
	op    string
	right Expr
}

var drawReducers = drawgenerated.ReducerMap{
	drawgenerated.SemanticActionProgram:            reduceProgram,
	drawgenerated.SemanticActionStatements:         reduceStatementList,
	drawgenerated.SemanticActionFigures:            reduceStatementList,
	drawgenerated.SemanticActionStatementTailMore:  reduceStatementTailMore,
	drawgenerated.SemanticActionFigureTailMore:     reduceStatementTailMore,
	drawgenerated.SemanticActionStatementTailEmpty: reduceEmptyStatements,
	drawgenerated.SemanticActionFigureTailEmpty:    reduceEmptyStatements,
	drawgenerated.SemanticActionPass:               reducePass,
	drawgenerated.SemanticActionCanvas:             reduceCanvas,
	drawgenerated.SemanticActionBackground:         reduceBackground,
	drawgenerated.SemanticActionStroke:             reduceStroke,
	drawgenerated.SemanticActionFill:               reduceFill,
	drawgenerated.SemanticActionFillNone:           reduceFillNone,
	drawgenerated.SemanticActionWidth:              reduceWidth,
	drawgenerated.SemanticActionAssign:             reduceAssign,
	drawgenerated.SemanticActionDefineFigure:       reduceDefineFigure,
	drawgenerated.SemanticActionDraw:               reduceDraw,
	drawgenerated.SemanticActionRepdraw:            reduceRepdraw,
	drawgenerated.SemanticActionFigureRefNamed:     reduceNamedFigureRef,
	drawgenerated.SemanticActionFigureRefInline:    reduceInlineFigureRef,
	drawgenerated.SemanticActionFigureBlock:        reduceFigureBlock,
	drawgenerated.SemanticActionPrimitivePoint:     reducePrimitivePoint,
	drawgenerated.SemanticActionPrimitiveLine:      reducePrimitiveLine,
	drawgenerated.SemanticActionPrimitiveBox:       reducePrimitiveBox,
	drawgenerated.SemanticActionPrimitiveCircle:    reducePrimitiveCircle,
	drawgenerated.SemanticActionColor:              reduceColor,
	drawgenerated.SemanticActionExpr:               reduceBinaryExpression,
	drawgenerated.SemanticActionTerm:               reduceBinaryExpression,
	drawgenerated.SemanticActionExprTailAdd:        reduceExprTailAdd,
	drawgenerated.SemanticActionExprTailSubtract:   reduceExprTailSubtract,
	drawgenerated.SemanticActionExprTailEmpty:      reduceEmptyBinaryTail,
	drawgenerated.SemanticActionTermTailEmpty:      reduceEmptyBinaryTail,
	drawgenerated.SemanticActionTermTailMultiply:   reduceTermTailMultiply,
	drawgenerated.SemanticActionTermTailDivide:     reduceTermTailDivide,
	drawgenerated.SemanticActionUnaryNegate:        reduceUnaryNegate,
	drawgenerated.SemanticActionNumber:             reduceNumber,
	drawgenerated.SemanticActionVariable:           reduceVariable,
	drawgenerated.SemanticActionCall:               reduceCall,
	drawgenerated.SemanticActionGroup:              reduceGroup,
}

func drawReduce(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return drawReducers.Reduce(ctx)
}

func reduceProgram(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	statements, err := statementSliceArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	return &Program{Statements: statements}, nil
}

func reduceStatementList(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	statement, err := statementArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	tail, err := statementSliceArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return append([]Statement{statement}, tail...), nil
}

func reduceStatementTailMore(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	statement, err := statementArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	tail, err := statementSliceArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return append([]Statement{statement}, tail...), nil
}

func reduceEmptyStatements(drawgenerated.Reduction) (drawgenerated.Value, error) {
	return []Statement{}, nil
}

func reducePass(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return valueArg(ctx, 0)
}

func reduceCanvas(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	width, err := exprArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	height, err := exprArg(ctx, 3)
	if err != nil {
		return nil, err
	}
	return CanvasStatement{Width: width, Height: height}, nil
}

func reduceBackground(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	c, err := colorArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return BackgroundStatement{Color: c}, nil
}

func reduceStroke(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	c, err := colorArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return StrokeStatement{Color: c}, nil
}

func reduceFill(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	c, err := colorArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return FillStatement{Color: c, Enabled: true}, nil
}

func reduceFillNone(drawgenerated.Reduction) (drawgenerated.Value, error) {
	return FillStatement{Enabled: false}, nil
}

func reduceWidth(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	value, err := exprArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return WidthStatement{Value: value}, nil
}

func reduceAssign(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	expr, err := exprArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return AssignStatement{Name: name, Expr: expr}, nil
}

func reduceDefineFigure(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	figure, err := figureBlockArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return DefineFigureStatement{Name: name, Figure: figure}, nil
}

func reduceDraw(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	target, err := figureRefArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return DrawStatement{Target: target}, nil
}

func reduceRepdraw(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	count, err := exprArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	target, err := figureRefArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return RepDrawStatement{Count: count, Target: target}, nil
}

func reduceNamedFigureRef(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	return NamedFigureRef{Name: name}, nil
}

func reduceInlineFigureRef(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	figure, err := figureBlockArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	return InlineFigureRef{Figure: figure}, nil
}

func reduceFigureBlock(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	statements, err := statementSliceArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return FigureBlock{Statements: statements}, nil
}

func reducePrimitivePoint(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return primitiveStatement(ctx, "point", 1, 3)
}

func reducePrimitiveLine(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return primitiveStatement(ctx, "line", 1, 3, 5, 7)
}

func reducePrimitiveBox(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return primitiveStatement(ctx, "box", 1, 3, 5, 7)
}

func reducePrimitiveCircle(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return primitiveStatement(ctx, "circle", 1, 3, 5)
}

func reduceColor(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	text, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	return parseHexColor(text)
}

func reduceBinaryExpression(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	left, err := exprArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	tails, err := binaryTailSliceArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return foldBinary(left, tails), nil
}

func reduceExprTailAdd(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return binaryTailList(ctx, "+", 1, 2)
}

func reduceExprTailSubtract(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return binaryTailList(ctx, "-", 1, 2)
}

func reduceEmptyBinaryTail(drawgenerated.Reduction) (drawgenerated.Value, error) {
	return []binaryTail{}, nil
}

func reduceTermTailMultiply(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return binaryTailList(ctx, "*", 1, 2)
}

func reduceTermTailDivide(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return binaryTailList(ctx, "/", 1, 2)
}

func reduceUnaryNegate(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	expr, err := exprArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return UnaryExpr{Op: "-", X: expr}, nil
}

func reduceNumber(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	text, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, fmt.Errorf("rule %d invalid number %q: %w", ctx.Rule, text, err)
	}
	return NumberExpr{Value: value}, nil
}

func reduceVariable(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	return VariableExpr{Name: name}, nil
}

func reduceCall(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	arg, err := exprArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return CallExpr{Name: name, Arg: arg}, nil
}

func reduceGroup(ctx drawgenerated.Reduction) (drawgenerated.Value, error) {
	return exprArg(ctx, 1)
}

func primitiveStatement(ctx drawgenerated.Reduction, kind string, indexes ...int) (drawgenerated.Value, error) {
	args := make([]Expr, 0, len(indexes))
	for _, index := range indexes {
		expr, err := exprArg(ctx, index)
		if err != nil {
			return nil, err
		}
		args = append(args, expr)
	}
	return PrimitiveStatement{Kind: kind, Args: args}, nil
}

func binaryTailList(ctx drawgenerated.Reduction, op string, exprIndex int, tailIndex int) (drawgenerated.Value, error) {
	expr, err := exprArg(ctx, exprIndex)
	if err != nil {
		return nil, err
	}
	tail, err := binaryTailSliceArg(ctx, tailIndex)
	if err != nil {
		return nil, err
	}
	return append([]binaryTail{{op: op, right: expr}}, tail...), nil
}

func foldBinary(left Expr, tails []binaryTail) Expr {
	out := left
	for _, tail := range tails {
		out = BinaryExpr{Op: tail.op, Left: out, Right: tail.right}
	}
	return out
}

func valueArg(ctx drawgenerated.Reduction, index int) (drawgenerated.Value, error) {
	if index < 0 || index >= len(ctx.Values) {
		return nil, fmt.Errorf("rule %d action %q missing argument %d", ctx.Rule, ctx.Action, index+1)
	}
	return ctx.Values[index], nil
}

func lexemeTextArg(ctx drawgenerated.Reduction, index int) (string, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return "", err
	}
	lexeme, ok := value.(drawgenerated.Lexeme)
	if !ok {
		return "", fmt.Errorf("rule %d action %q argument %d has type %T, want Lexeme", ctx.Rule, ctx.Action, index+1, value)
	}
	return lexeme.Text, nil
}

func statementArg(ctx drawgenerated.Reduction, index int) (Statement, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	statement, ok := value.(Statement)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want Statement", ctx.Rule, ctx.Action, index+1, value)
	}
	return statement, nil
}

func statementSliceArg(ctx drawgenerated.Reduction, index int) ([]Statement, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	statements, ok := value.([]Statement)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want []Statement", ctx.Rule, ctx.Action, index+1, value)
	}
	return statements, nil
}

func exprArg(ctx drawgenerated.Reduction, index int) (Expr, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	expr, ok := value.(Expr)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want Expr", ctx.Rule, ctx.Action, index+1, value)
	}
	return expr, nil
}

func binaryTailSliceArg(ctx drawgenerated.Reduction, index int) ([]binaryTail, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	tails, ok := value.([]binaryTail)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want []binaryTail", ctx.Rule, ctx.Action, index+1, value)
	}
	return tails, nil
}

func figureBlockArg(ctx drawgenerated.Reduction, index int) (FigureBlock, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return FigureBlock{}, err
	}
	figure, ok := value.(FigureBlock)
	if !ok {
		return FigureBlock{}, fmt.Errorf("rule %d action %q argument %d has type %T, want FigureBlock", ctx.Rule, ctx.Action, index+1, value)
	}
	return figure, nil
}

func figureRefArg(ctx drawgenerated.Reduction, index int) (FigureRef, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	ref, ok := value.(FigureRef)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want FigureRef", ctx.Rule, ctx.Action, index+1, value)
	}
	return ref, nil
}

func colorArg(ctx drawgenerated.Reduction, index int) (color.RGBA, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return color.RGBA{}, err
	}
	c, ok := value.(color.RGBA)
	if !ok {
		return color.RGBA{}, fmt.Errorf("rule %d action %q argument %d has type %T, want color.RGBA", ctx.Rule, ctx.Action, index+1, value)
	}
	return c, nil
}

func parseHexColor(text string) (color.RGBA, error) {
	if len(text) != 7 || text[0] != '#' {
		return color.RGBA{}, fmt.Errorf("invalid color %q", text)
	}
	value, err := strconv.ParseUint(text[1:], 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid color %q", text)
	}
	return color.RGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 255}, nil
}
