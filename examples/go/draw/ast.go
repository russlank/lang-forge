package draw

import drawmodel "github.com/russlank/lang-forge/examples/go/draw/model"

// The public draw package preserves its original AST names as aliases while
// generated typed contexts and handwritten rendering share the cycle-free
// model package.
type (
	Color                 = drawmodel.Color
	Program               = drawmodel.Program
	Statement             = drawmodel.Statement
	CanvasStatement       = drawmodel.CanvasStatement
	BackgroundStatement   = drawmodel.BackgroundStatement
	StrokeStatement       = drawmodel.StrokeStatement
	FillStatement         = drawmodel.FillStatement
	WidthStatement        = drawmodel.WidthStatement
	AssignStatement       = drawmodel.AssignStatement
	DefineFigureStatement = drawmodel.DefineFigureStatement
	DrawStatement         = drawmodel.DrawStatement
	RepDrawStatement      = drawmodel.RepDrawStatement
	PrimitiveStatement    = drawmodel.PrimitiveStatement
	FigureBlock           = drawmodel.FigureBlock
	FigureRef             = drawmodel.FigureRef
	NamedFigureRef        = drawmodel.NamedFigureRef
	InlineFigureRef       = drawmodel.InlineFigureRef
	Expr                  = drawmodel.Expr
	NumberExpr            = drawmodel.NumberExpr
	VariableExpr          = drawmodel.VariableExpr
	UnaryExpr             = drawmodel.UnaryExpr
	BinaryExpr            = drawmodel.BinaryExpr
	CallExpr              = drawmodel.CallExpr
	BinaryTail            = drawmodel.BinaryTail
)
