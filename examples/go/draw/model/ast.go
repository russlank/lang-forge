// Package model defines the dependency-only DRAW abstract syntax tree.
//
// Both the generated parser and the handwritten draw package depend on this
// package. Keeping the model independent prevents an import cycle when
// generated typed reduction contexts refer to application AST types.
package model

import "image/color"

// Color is the renderer's RGBA color value.
type Color = color.RGBA

// Program is a parsed DRAW image script.
type Program struct {
	Statements []Statement
}

// Statement is implemented by all top-level and figure-block commands.
type Statement interface {
	statementNode()
}

// CanvasStatement creates the target drawing surface.
type CanvasStatement struct {
	Width  Expr
	Height Expr
}

func (CanvasStatement) statementNode() {}

// BackgroundStatement fills the canvas background.
type BackgroundStatement struct {
	Color Color
}

func (BackgroundStatement) statementNode() {}

// StrokeStatement changes the active stroke color.
type StrokeStatement struct {
	Color Color
}

func (StrokeStatement) statementNode() {}

// FillStatement changes or disables the active fill color.
type FillStatement struct {
	Color   Color
	Enabled bool
}

func (FillStatement) statementNode() {}

// WidthStatement changes the active line width.
type WidthStatement struct {
	Value Expr
}

func (WidthStatement) statementNode() {}

// AssignStatement stores a numeric expression in a variable.
type AssignStatement struct {
	Name string
	Expr Expr
}

func (AssignStatement) statementNode() {}

// DefineFigureStatement stores a reusable figure block.
type DefineFigureStatement struct {
	Name   string
	Figure FigureBlock
}

func (DefineFigureStatement) statementNode() {}

// DrawStatement draws one named or inline figure.
type DrawStatement struct {
	Target FigureRef
}

func (DrawStatement) statementNode() {}

// RepDrawStatement draws a figure repeatedly.
type RepDrawStatement struct {
	Count  Expr
	Target FigureRef
}

func (RepDrawStatement) statementNode() {}

// PrimitiveStatement draws one point, line, box, or circle.
type PrimitiveStatement struct {
	Kind string
	Args []Expr
}

func (PrimitiveStatement) statementNode() {}

// FigureBlock is a reusable list of figure-local statements.
type FigureBlock struct {
	Statements []Statement
}

// FigureRef is implemented by named and inline figure references.
type FigureRef interface {
	figureRefNode()
}

// NamedFigureRef references a previously defined figure.
type NamedFigureRef struct {
	Name string
}

func (NamedFigureRef) figureRefNode() {}

// InlineFigureRef embeds a figure block at its use site.
type InlineFigureRef struct {
	Figure FigureBlock
}

func (InlineFigureRef) figureRefNode() {}

// Expr is implemented by all numeric expressions.
type Expr interface {
	exprNode()
}

// NumberExpr is a numeric literal.
type NumberExpr struct {
	Value float64
}

func (NumberExpr) exprNode() {}

// VariableExpr reads one variable.
type VariableExpr struct {
	Name string
}

func (VariableExpr) exprNode() {}

// UnaryExpr applies one unary operator.
type UnaryExpr struct {
	Op string
	X  Expr
}

func (UnaryExpr) exprNode() {}

// BinaryExpr applies one binary operator.
type BinaryExpr struct {
	Op          string
	Left, Right Expr
}

func (BinaryExpr) exprNode() {}

// CallExpr invokes one built-in numeric function.
type CallExpr struct {
	Name string
	Arg  Expr
}

func (CallExpr) exprNode() {}

// BinaryTail is one right-hand operation collected before left-folding an
// expression or term.
type BinaryTail struct {
	Op    string
	Right Expr
}
