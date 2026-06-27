package datakeeper

import dksmodel "github.com/russlank/lang-forge/examples/go/datakeeper/model"

// The public package keeps its original AST names as aliases while generated
// typed reducer contexts and handwritten compiler/runtime code share the
// cycle-free model package.
type (
	Script                 = dksmodel.Script
	Statement              = dksmodel.Statement
	AssignStatement        = dksmodel.AssignStatement
	ReplaceStatement       = dksmodel.ReplaceStatement
	RunSQLStatement        = dksmodel.RunSQLStatement
	AddObjectStatement     = dksmodel.AddObjectStatement
	RemoveObjectStatement  = dksmodel.RemoveObjectStatement
	RunObjectsJobStatement = dksmodel.RunObjectsJobStatement
	ValueExpr              = dksmodel.ValueExpr
	StringLiteral          = dksmodel.StringLiteral
	NumberLiteral          = dksmodel.NumberLiteral
	ReferenceExpr          = dksmodel.ReferenceExpr
)
