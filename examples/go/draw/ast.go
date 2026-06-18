package draw

import "image/color"

// Program is a parsed DRAW image script.
type Program struct {
	Statements []Statement
}

// Statement is implemented by all top-level and figure-block commands.
type Statement interface {
	statementNode()
}

type CanvasStatement struct {
	Width  Expr
	Height Expr
}

func (CanvasStatement) statementNode() {}

type BackgroundStatement struct {
	Color color.RGBA
}

func (BackgroundStatement) statementNode() {}

type StrokeStatement struct {
	Color color.RGBA
}

func (StrokeStatement) statementNode() {}

type FillStatement struct {
	Color   color.RGBA
	Enabled bool
}

func (FillStatement) statementNode() {}

type WidthStatement struct {
	Value Expr
}

func (WidthStatement) statementNode() {}

type AssignStatement struct {
	Name string
	Expr Expr
}

func (AssignStatement) statementNode() {}

type DefineFigureStatement struct {
	Name   string
	Figure FigureBlock
}

func (DefineFigureStatement) statementNode() {}

type DrawStatement struct {
	Target FigureRef
}

func (DrawStatement) statementNode() {}

type RepDrawStatement struct {
	Count  Expr
	Target FigureRef
}

func (RepDrawStatement) statementNode() {}

type PrimitiveStatement struct {
	Kind string
	Args []Expr
}

func (PrimitiveStatement) statementNode() {}

type FigureBlock struct {
	Statements []Statement
}

type FigureRef interface {
	figureRefNode()
}

type NamedFigureRef struct {
	Name string
}

func (NamedFigureRef) figureRefNode() {}

type InlineFigureRef struct {
	Figure FigureBlock
}

func (InlineFigureRef) figureRefNode() {}

// Expr is implemented by all numeric expressions.
type Expr interface {
	exprNode()
}

type NumberExpr struct {
	Value float64
}

func (NumberExpr) exprNode() {}

type VariableExpr struct {
	Name string
}

func (VariableExpr) exprNode() {}

type UnaryExpr struct {
	Op string
	X  Expr
}

func (UnaryExpr) exprNode() {}

type BinaryExpr struct {
	Op          string
	Left, Right Expr
}

func (BinaryExpr) exprNode() {}

type CallExpr struct {
	Name string
	Arg  Expr
}

func (CallExpr) exprNode() {}
