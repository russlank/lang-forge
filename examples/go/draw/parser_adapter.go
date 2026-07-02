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
// This file is handwritten adapter code. Grammar labels and semantic type
// declarations let the generated package expose typed contexts, so this layer
// contains no positional reduction indexes or semantic value casts.
func Parse(source string) (*Program, error) {
	value, err := drawgenerated.ParseWithReducerFromSource(drawgenerated.NewScanner(source), drawReducers)
	if err != nil {
		return nil, err
	}
	program, ok := value.(*Program)
	if !ok {
		return nil, fmt.Errorf("generated parser returned %T, want *Program", value)
	}
	return program, nil
}

// drawReducers connects each `{go: ...}` action label in draw.lf to the
// handwritten AST-building function below. The generated typed adapters expose
// labeled RHS values as fields, so drawing semantics stays independent from
// parser stack positions.
var drawReducers = drawgenerated.ReducerMap{
	drawgenerated.SemanticActionProgram:            drawgenerated.TypedProgram(reduceProgram),
	drawgenerated.SemanticActionStatements:         drawgenerated.TypedStatements(reduceStatements),
	drawgenerated.SemanticActionStatementTailMore:  drawgenerated.TypedStatementTailMore(reduceStatementTailMore),
	drawgenerated.SemanticActionStatementTailEmpty: drawgenerated.TypedStatementTailEmpty(reduceStatementTailEmpty),
	drawgenerated.SemanticActionPass:               drawgenerated.TypedPass(reducePass),
	drawgenerated.SemanticActionCanvas:             drawgenerated.TypedCanvas(reduceCanvas),
	drawgenerated.SemanticActionBackground:         drawgenerated.TypedBackground(reduceBackground),
	drawgenerated.SemanticActionStroke:             drawgenerated.TypedStroke(reduceStroke),
	drawgenerated.SemanticActionFill:               drawgenerated.TypedFill(reduceFill),
	drawgenerated.SemanticActionFillNone:           drawgenerated.TypedFillNone(reduceFillNone),
	drawgenerated.SemanticActionWidth:              drawgenerated.TypedWidth(reduceWidth),
	drawgenerated.SemanticActionAssign:             drawgenerated.TypedAssign(reduceAssign),
	drawgenerated.SemanticActionDefineFigure:       drawgenerated.TypedDefineFigure(reduceDefineFigure),
	drawgenerated.SemanticActionDraw:               drawgenerated.TypedDraw(reduceDraw),
	drawgenerated.SemanticActionRepdraw:            drawgenerated.TypedRepdraw(reduceRepdraw),
	drawgenerated.SemanticActionFigureRefNamed:     drawgenerated.TypedFigureRefNamed(reduceNamedFigureRef),
	drawgenerated.SemanticActionFigureRefInline:    drawgenerated.TypedFigureRefInline(reduceInlineFigureRef),
	drawgenerated.SemanticActionFigureBlock:        drawgenerated.TypedFigureBlock(reduceFigureBlock),
	drawgenerated.SemanticActionFigures:            drawgenerated.TypedFigures(reduceFigures),
	drawgenerated.SemanticActionFigureTailMore:     drawgenerated.TypedFigureTailMore(reduceFigureTailMore),
	drawgenerated.SemanticActionFigureTailEmpty:    drawgenerated.TypedFigureTailEmpty(reduceFigureTailEmpty),
	drawgenerated.SemanticActionPrimitivePoint:     drawgenerated.TypedPrimitivePoint(reducePrimitivePoint),
	drawgenerated.SemanticActionPrimitiveLine:      drawgenerated.TypedPrimitiveLine(reducePrimitiveLine),
	drawgenerated.SemanticActionPrimitiveBox:       drawgenerated.TypedPrimitiveBox(reducePrimitiveBox),
	drawgenerated.SemanticActionPrimitiveCircle:    drawgenerated.TypedPrimitiveCircle(reducePrimitiveCircle),
	drawgenerated.SemanticActionColor:              drawgenerated.TypedColor(reduceColor),
	drawgenerated.SemanticActionExpr:               drawgenerated.TypedExpr(reduceExpr),
	drawgenerated.SemanticActionExprTailAdd:        drawgenerated.TypedExprTailAdd(reduceExprTailAdd),
	drawgenerated.SemanticActionExprTailSubtract:   drawgenerated.TypedExprTailSubtract(reduceExprTailSubtract),
	drawgenerated.SemanticActionExprTailEmpty:      drawgenerated.TypedExprTailEmpty(reduceExprTailEmpty),
	drawgenerated.SemanticActionTerm:               drawgenerated.TypedTerm(reduceTerm),
	drawgenerated.SemanticActionTermTailMultiply:   drawgenerated.TypedTermTailMultiply(reduceTermTailMultiply),
	drawgenerated.SemanticActionTermTailDivide:     drawgenerated.TypedTermTailDivide(reduceTermTailDivide),
	drawgenerated.SemanticActionTermTailEmpty:      drawgenerated.TypedTermTailEmpty(reduceTermTailEmpty),
	drawgenerated.SemanticActionUnaryNegate:        drawgenerated.TypedUnaryNegate(reduceUnaryNegate),
	drawgenerated.SemanticActionExprPass:           drawgenerated.TypedExprPass(reduceExprPass),
	drawgenerated.SemanticActionNumber:             drawgenerated.TypedNumber(reduceNumber),
	drawgenerated.SemanticActionVariable:           drawgenerated.TypedVariable(reduceVariable),
	drawgenerated.SemanticActionCall:               drawgenerated.TypedCall(reduceCall),
	drawgenerated.SemanticActionGroup:              drawgenerated.TypedGroup(reduceGroup),
}

func reduceProgram(ctx drawgenerated.ProgramReduction) (*Program, error) {
	return &Program{Statements: ctx.Statements}, nil
}

func reduceStatements(ctx drawgenerated.StatementsReduction) ([]Statement, error) {
	return prependStatement(ctx.Head, ctx.Tail), nil
}

func reduceStatementTailMore(ctx drawgenerated.StatementTailMoreReduction) ([]Statement, error) {
	return prependStatement(ctx.Head, ctx.Tail), nil
}

func reduceStatementTailEmpty(drawgenerated.StatementTailEmptyReduction) ([]Statement, error) {
	return []Statement{}, nil
}

func reducePass(ctx drawgenerated.PassReduction) (Statement, error) {
	return ctx.Value, nil
}

func reduceCanvas(ctx drawgenerated.CanvasReduction) (Statement, error) {
	return CanvasStatement{Width: ctx.Width, Height: ctx.Height}, nil
}

func reduceBackground(ctx drawgenerated.BackgroundReduction) (Statement, error) {
	return BackgroundStatement{Color: ctx.Color}, nil
}

func reduceStroke(ctx drawgenerated.StrokeReduction) (Statement, error) {
	return StrokeStatement{Color: ctx.Color}, nil
}

func reduceFill(ctx drawgenerated.FillReduction) (Statement, error) {
	return FillStatement{Color: ctx.Color, Enabled: true}, nil
}

func reduceFillNone(drawgenerated.FillNoneReduction) (Statement, error) {
	return FillStatement{Enabled: false}, nil
}

func reduceWidth(ctx drawgenerated.WidthReduction) (Statement, error) {
	return WidthStatement{Value: ctx.Value}, nil
}

func reduceAssign(ctx drawgenerated.AssignReduction) (Statement, error) {
	return AssignStatement{Name: ctx.Name.Text, Expr: ctx.Value}, nil
}

func reduceDefineFigure(ctx drawgenerated.DefineFigureReduction) (Statement, error) {
	return DefineFigureStatement{Name: ctx.Name.Text, Figure: ctx.Figure}, nil
}

func reduceDraw(ctx drawgenerated.DrawReduction) (Statement, error) {
	return DrawStatement{Target: ctx.Target}, nil
}

func reduceRepdraw(ctx drawgenerated.RepdrawReduction) (Statement, error) {
	return RepDrawStatement{Count: ctx.Count, Target: ctx.Target}, nil
}

func reduceNamedFigureRef(ctx drawgenerated.FigureRefNamedReduction) (FigureRef, error) {
	return NamedFigureRef{Name: ctx.Name.Text}, nil
}

func reduceInlineFigureRef(ctx drawgenerated.FigureRefInlineReduction) (FigureRef, error) {
	return InlineFigureRef{Figure: ctx.Figure}, nil
}

func reduceFigureBlock(ctx drawgenerated.FigureBlockReduction) (FigureBlock, error) {
	return FigureBlock{Statements: ctx.Statements}, nil
}

func reduceFigures(ctx drawgenerated.FiguresReduction) ([]Statement, error) {
	return prependStatement(ctx.Head, ctx.Tail), nil
}

func reduceFigureTailMore(ctx drawgenerated.FigureTailMoreReduction) ([]Statement, error) {
	return prependStatement(ctx.Head, ctx.Tail), nil
}

func reduceFigureTailEmpty(drawgenerated.FigureTailEmptyReduction) ([]Statement, error) {
	return []Statement{}, nil
}

func reducePrimitivePoint(ctx drawgenerated.PrimitivePointReduction) (Statement, error) {
	return PrimitiveStatement{Kind: "point", Args: []Expr{ctx.X, ctx.Y}}, nil
}

func reducePrimitiveLine(ctx drawgenerated.PrimitiveLineReduction) (Statement, error) {
	return PrimitiveStatement{Kind: "line", Args: []Expr{ctx.X1, ctx.Y1, ctx.X2, ctx.Y2}}, nil
}

func reducePrimitiveBox(ctx drawgenerated.PrimitiveBoxReduction) (Statement, error) {
	return PrimitiveStatement{Kind: "box", Args: []Expr{ctx.X1, ctx.Y1, ctx.X2, ctx.Y2}}, nil
}

func reducePrimitiveCircle(ctx drawgenerated.PrimitiveCircleReduction) (Statement, error) {
	return PrimitiveStatement{Kind: "circle", Args: []Expr{ctx.Cx, ctx.Cy, ctx.Radius}}, nil
}

func reduceColor(ctx drawgenerated.ColorReduction) (Color, error) {
	return parseHexColor(ctx.Literal.Text)
}

func reduceExpr(ctx drawgenerated.ExprReduction) (Expr, error) {
	return foldBinary(ctx.Left, ctx.Tail), nil
}

func reduceExprTailAdd(ctx drawgenerated.ExprTailAddReduction) ([]BinaryTail, error) {
	return prependBinaryTail("+", ctx.Right, ctx.Tail), nil
}

func reduceExprTailSubtract(ctx drawgenerated.ExprTailSubtractReduction) ([]BinaryTail, error) {
	return prependBinaryTail("-", ctx.Right, ctx.Tail), nil
}

func reduceExprTailEmpty(drawgenerated.ExprTailEmptyReduction) ([]BinaryTail, error) {
	return []BinaryTail{}, nil
}

func reduceTerm(ctx drawgenerated.TermReduction) (Expr, error) {
	return foldBinary(ctx.Left, ctx.Tail), nil
}

func reduceTermTailMultiply(ctx drawgenerated.TermTailMultiplyReduction) ([]BinaryTail, error) {
	return prependBinaryTail("*", ctx.Right, ctx.Tail), nil
}

func reduceTermTailDivide(ctx drawgenerated.TermTailDivideReduction) ([]BinaryTail, error) {
	return prependBinaryTail("/", ctx.Right, ctx.Tail), nil
}

func reduceTermTailEmpty(drawgenerated.TermTailEmptyReduction) ([]BinaryTail, error) {
	return []BinaryTail{}, nil
}

func reduceUnaryNegate(ctx drawgenerated.UnaryNegateReduction) (Expr, error) {
	return UnaryExpr{Op: "-", X: ctx.Operand}, nil
}

func reduceExprPass(ctx drawgenerated.ExprPassReduction) (Expr, error) {
	return ctx.Value, nil
}

func reduceNumber(ctx drawgenerated.NumberReduction) (Expr, error) {
	value, err := strconv.ParseFloat(ctx.Token.Text, 64)
	if err != nil {
		return nil, fmt.Errorf("rule %d invalid number %q: %w", ctx.Reduction.Rule, ctx.Token.Text, err)
	}
	return NumberExpr{Value: value}, nil
}

func reduceVariable(ctx drawgenerated.VariableReduction) (Expr, error) {
	return VariableExpr{Name: ctx.Name.Text}, nil
}

func reduceCall(ctx drawgenerated.CallReduction) (Expr, error) {
	return CallExpr{Name: ctx.Function.Text, Arg: ctx.Argument}, nil
}

func reduceGroup(ctx drawgenerated.GroupReduction) (Expr, error) {
	return ctx.Value, nil
}

func prependStatement(head Statement, tail []Statement) []Statement {
	return append([]Statement{head}, tail...)
}

func prependBinaryTail(op string, right Expr, tail []BinaryTail) []BinaryTail {
	return append([]BinaryTail{{Op: op, Right: right}}, tail...)
}

// foldBinary turns the grammar's right-recursive expression tail into the
// left-associative AST expected by the interpreter.
func foldBinary(left Expr, tails []BinaryTail) Expr {
	out := left
	for _, tail := range tails {
		out = BinaryExpr{Op: tail.Op, Left: out, Right: tail.Right}
	}
	return out
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
