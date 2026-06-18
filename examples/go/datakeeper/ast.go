package datakeeper

// Script is the parsed DataKeeper script program.
type Script struct {
	Parameters []string
	Statements []Statement
}

// Statement is implemented by every executable script statement.
type Statement interface {
	statementNode()
}

// AssignStatement assigns a value to a variable.
type AssignStatement struct {
	Name  string
	Value ValueExpr
}

func (AssignStatement) statementNode() {}

// ReplaceStatement replaces text inside the target variable.
type ReplaceStatement struct {
	Target string
	Old    ValueExpr
	New    ValueExpr
}

func (ReplaceStatement) statementNode() {}

// RunSQLStatement mocks running a SQL script against one instance.
type RunSQLStatement struct {
	Instance ValueExpr
	Script   ValueExpr
}

func (RunSQLStatement) statementNode() {}

// AddObjectStatement mocks adding an object XML document below a parent.
type AddObjectStatement struct {
	Parent ValueExpr
	XML    ValueExpr
}

func (AddObjectStatement) statementNode() {}

// RemoveObjectStatement mocks removing a named object below a parent.
type RemoveObjectStatement struct {
	Parent ValueExpr
	Name   ValueExpr
}

func (RemoveObjectStatement) statementNode() {}

// RunObjectsJobStatement mocks running one object job tag.
type RunObjectsJobStatement struct {
	Parent  ValueExpr
	Name    ValueExpr
	JobsTag ValueExpr
}

func (RunObjectsJobStatement) statementNode() {}

// ValueExpr is implemented by values accepted in DataKeeper expressions.
type ValueExpr interface {
	valueNode()
}

// StringLiteral is a string constant.
type StringLiteral struct {
	Value string
}

func (StringLiteral) valueNode() {}

// NumberLiteral is compiled as a string value to match the old VM behavior.
type NumberLiteral struct {
	Value string
}

func (NumberLiteral) valueNode() {}

// ReferenceExpr reads a variable or parameter value.
type ReferenceExpr struct {
	Name string
}

func (ReferenceExpr) valueNode() {}
