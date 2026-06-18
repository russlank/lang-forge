//go:build langforge_generated

package datakeeper

import (
	"fmt"
	"strings"

	dksgenerated "github.com/russlank/lang-forge/examples/go/datakeeper/generated"
)

// Parse converts source text into a DataKeeper script AST.
//
// This file is handwritten adapter code. The generated scanner/parser itself
// lives under examples/go/datakeeper/generated and is recreated by the Makefile.
func Parse(source string) (*Script, error) {
	lexemes, err := dksgenerated.Tokenize(source)
	if err != nil {
		return nil, err
	}
	value, err := dksgenerated.ParseWithReducer(lexemes, dksgenerated.ReducerFunc(dataKeeperReduce))
	if err != nil {
		return nil, err
	}
	script, ok := value.(*Script)
	if !ok {
		return nil, fmt.Errorf("generated parser returned %T, want *Script", value)
	}
	return script, nil
}

var dataKeeperReducers = dksgenerated.ReducerMap{
	dksgenerated.SemanticActionProgramWithParameters: reduceProgramWithParameters,
	dksgenerated.SemanticActionProgramNoParameters:   reduceProgramNoParameters,
	dksgenerated.SemanticActionParametersList:        reduceParametersList,
	dksgenerated.SemanticActionParametersDecl:        reduceParametersDecl,
	dksgenerated.SemanticActionParametersTailMore:    reduceParametersTailMore,
	dksgenerated.SemanticActionParametersTailEmpty:   reduceEmptyStringSlice,
	dksgenerated.SemanticActionCommandBlock:          reduceCommandBlock,
	dksgenerated.SemanticActionStatements:            reduceStatements,
	dksgenerated.SemanticActionStatementsTailMore:    reduceStatementsTailMore,
	dksgenerated.SemanticActionStatementsTailEmpty:   reduceEmptyStatementSlice,
	dksgenerated.SemanticActionStatementPass:         reducePass,
	dksgenerated.SemanticActionAssign:                reduceAssign,
	dksgenerated.SemanticActionReplace:               reduceReplace,
	dksgenerated.SemanticActionSqlrun:                reduceRunSQL,
	dksgenerated.SemanticActionAddObject:             reduceAddObject,
	dksgenerated.SemanticActionRemoveObject:          reduceRemoveObject,
	dksgenerated.SemanticActionRunObjectsJob:         reduceRunObjectsJob,
	dksgenerated.SemanticActionValueString:           reduceStringValue,
	dksgenerated.SemanticActionValueNumber:           reduceNumberValue,
	dksgenerated.SemanticActionValueIdent:            reduceIdentifierValue,
}

func dataKeeperReduce(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	return dataKeeperReducers.Reduce(ctx)
}

func reduceProgramWithParameters(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	parameters, err := stringSliceArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	statements, err := statementSliceArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return &Script{Parameters: parameters, Statements: statements}, nil
}

func reduceProgramNoParameters(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	statements, err := statementSliceArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	return &Script{Statements: statements}, nil
}

func reduceParametersList(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	return stringSliceArg(ctx, 1)
}

func reduceParametersDecl(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	tail, err := stringSliceArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	return append([]string{name}, tail...), nil
}

func reduceParametersTailMore(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 1)
	if err != nil {
		return nil, err
	}
	tail, err := stringSliceArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return append([]string{name}, tail...), nil
}

func reduceEmptyStringSlice(dksgenerated.Reduction) (dksgenerated.Value, error) {
	return []string{}, nil
}

func reduceCommandBlock(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	return statementSliceArg(ctx, 1)
}

func reduceStatements(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
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

func reduceStatementsTailMore(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
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

func reduceEmptyStatementSlice(dksgenerated.Reduction) (dksgenerated.Value, error) {
	return []Statement{}, nil
}

func reducePass(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	return valueArg(ctx, 0)
}

func reduceAssign(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	value, err := valueExprArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	return AssignStatement{Name: name, Value: value}, nil
}

func reduceReplace(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	target, err := lexemeTextArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	oldValue, err := valueExprArg(ctx, 4)
	if err != nil {
		return nil, err
	}
	newValue, err := valueExprArg(ctx, 6)
	if err != nil {
		return nil, err
	}
	return ReplaceStatement{Target: target, Old: oldValue, New: newValue}, nil
}

func reduceRunSQL(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	instance, err := valueExprArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	script, err := valueExprArg(ctx, 4)
	if err != nil {
		return nil, err
	}
	return RunSQLStatement{Instance: instance, Script: script}, nil
}

func reduceAddObject(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	parent, err := valueExprArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	xml, err := valueExprArg(ctx, 4)
	if err != nil {
		return nil, err
	}
	return AddObjectStatement{Parent: parent, XML: xml}, nil
}

func reduceRemoveObject(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	parent, err := valueExprArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	name, err := valueExprArg(ctx, 4)
	if err != nil {
		return nil, err
	}
	return RemoveObjectStatement{Parent: parent, Name: name}, nil
}

func reduceRunObjectsJob(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	parent, err := valueExprArg(ctx, 2)
	if err != nil {
		return nil, err
	}
	name, err := valueExprArg(ctx, 4)
	if err != nil {
		return nil, err
	}
	jobsTag, err := valueExprArg(ctx, 6)
	if err != nil {
		return nil, err
	}
	return RunObjectsJobStatement{Parent: parent, Name: name, JobsTag: jobsTag}, nil
}

func reduceStringValue(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	text, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	decoded, err := decodeStringLexeme(text)
	if err != nil {
		return nil, err
	}
	return StringLiteral{Value: decoded}, nil
}

func reduceNumberValue(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	text, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	return NumberLiteral{Value: text}, nil
}

func reduceIdentifierValue(ctx dksgenerated.Reduction) (dksgenerated.Value, error) {
	name, err := lexemeTextArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	return ReferenceExpr{Name: name}, nil
}

func valueArg(ctx dksgenerated.Reduction, index int) (dksgenerated.Value, error) {
	if index < 0 || index >= len(ctx.Values) {
		return nil, fmt.Errorf("rule %d action %q missing argument %d", ctx.Rule, ctx.Action, index+1)
	}
	return ctx.Values[index], nil
}

func lexemeTextArg(ctx dksgenerated.Reduction, index int) (string, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return "", err
	}
	lexeme, ok := value.(dksgenerated.Lexeme)
	if !ok {
		return "", fmt.Errorf("rule %d action %q argument %d has type %T, want Lexeme", ctx.Rule, ctx.Action, index+1, value)
	}
	return lexeme.Text, nil
}

func stringSliceArg(ctx dksgenerated.Reduction, index int) ([]string, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	out, ok := value.([]string)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want []string", ctx.Rule, ctx.Action, index+1, value)
	}
	return out, nil
}

func statementArg(ctx dksgenerated.Reduction, index int) (Statement, error) {
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

func statementSliceArg(ctx dksgenerated.Reduction, index int) ([]Statement, error) {
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

func valueExprArg(ctx dksgenerated.Reduction, index int) (ValueExpr, error) {
	value, err := valueArg(ctx, index)
	if err != nil {
		return nil, err
	}
	expr, ok := value.(ValueExpr)
	if !ok {
		return nil, fmt.Errorf("rule %d action %q argument %d has type %T, want ValueExpr", ctx.Rule, ctx.Action, index+1, value)
	}
	return expr, nil
}

func decodeStringLexeme(text string) (string, error) {
	if strings.HasPrefix(text, "#{") && strings.HasSuffix(text, "#}") {
		return strings.TrimSuffix(strings.TrimPrefix(text, "#{"), "#}"), nil
	}
	if strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"") {
		return unescapeQuoted(text[1 : len(text)-1])
	}
	return "", fmt.Errorf("unknown string literal form")
}

func unescapeQuoted(text string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if ch != '\\' {
			b.WriteByte(ch)
			continue
		}
		i++
		if i >= len(text) {
			return "", fmt.Errorf("trailing string escape")
		}
		switch escaped := text[i]; escaped {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '"', '\\':
			b.WriteByte(escaped)
		default:
			b.WriteByte(escaped)
		}
	}
	return b.String(), nil
}
