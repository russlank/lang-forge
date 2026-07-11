//go:build langforge_generated

package datakeeper

import (
	"fmt"
	"strings"

	dksgenerated "github.com/russlank/lang-forge/examples/go/datakeeper/generated"
)

// Parse converts source text into a DataKeeper script AST.
//
// The `.lf` grammar declares named RHS labels and semantic types. The generated
// parser turns those declarations into typed reducer contexts, so this adapter
// can describe each reduction in domain terms instead of numeric stack slots.
func Parse(source string) (*Script, error) {
	value, err := dksgenerated.ParseWithReducerFromLexemeSource(dksgenerated.NewScanner(source), dataKeeperReducers)
	if err != nil {
		return nil, err
	}
	script, ok := value.(*Script)
	if !ok {
		return nil, fmt.Errorf("generated parser returned %T, want *Script", value)
	}
	return script, nil
}

// dataKeeperReducers is the join point between datakeeper.lf and application
// code. Each key is generated from a `{go: ...}` action label, while each
// value is a handwritten function that decides the meaning of that reduction.
var dataKeeperReducers = dksgenerated.ReducerMap{
	dksgenerated.SemanticActionProgramWithParameters: dksgenerated.TypedProgramWithParameters(reduceProgramWithParameters),
	dksgenerated.SemanticActionProgramNoParameters:   dksgenerated.TypedProgramNoParameters(reduceProgramNoParameters),
	dksgenerated.SemanticActionParametersList:        dksgenerated.TypedParametersList(reduceParametersList),
	dksgenerated.SemanticActionParametersDecl:        dksgenerated.TypedParametersDecl(reduceParametersDecl),
	dksgenerated.SemanticActionParametersTailMore:    dksgenerated.TypedParametersTailMore(reduceParametersTailMore),
	dksgenerated.SemanticActionParametersTailEmpty:   dksgenerated.TypedParametersTailEmpty(reduceEmptyStringSlice),
	dksgenerated.SemanticActionCommandBlock:          dksgenerated.TypedCommandBlock(reduceCommandBlock),
	dksgenerated.SemanticActionStatements:            dksgenerated.TypedStatements(reduceStatements),
	dksgenerated.SemanticActionStatementsTailMore:    dksgenerated.TypedStatementsTailMore(reduceStatementsTailMore),
	dksgenerated.SemanticActionStatementsTailEmpty:   dksgenerated.TypedStatementsTailEmpty(reduceEmptyStatementSlice),
	dksgenerated.SemanticActionStatementPass:         dksgenerated.TypedStatementPass(reducePass),
	dksgenerated.SemanticActionAssign:                dksgenerated.TypedAssign(reduceAssign),
	dksgenerated.SemanticActionReplace:               dksgenerated.TypedReplace(reduceReplace),
	dksgenerated.SemanticActionSqlrun:                dksgenerated.TypedSqlrun(reduceRunSQL),
	dksgenerated.SemanticActionAddObject:             dksgenerated.TypedAddObject(reduceAddObject),
	dksgenerated.SemanticActionRemoveObject:          dksgenerated.TypedRemoveObject(reduceRemoveObject),
	dksgenerated.SemanticActionRunObjectsJob:         dksgenerated.TypedRunObjectsJob(reduceRunObjectsJob),
	dksgenerated.SemanticActionValueString:           dksgenerated.TypedValueString(reduceStringValue),
	dksgenerated.SemanticActionValueNumber:           dksgenerated.TypedValueNumber(reduceNumberValue),
	dksgenerated.SemanticActionValueIdent:            dksgenerated.TypedValueIdent(reduceIdentifierValue),
}

func reduceProgramWithParameters(ctx dksgenerated.ProgramWithParametersReduction) (*Script, error) {
	return &Script{Parameters: ctx.Parameters, Statements: ctx.Block}, nil
}

func reduceProgramNoParameters(ctx dksgenerated.ProgramNoParametersReduction) (*Script, error) {
	return &Script{Statements: ctx.Block}, nil
}

func reduceParametersList(ctx dksgenerated.ParametersListReduction) ([]string, error) {
	return ctx.Params, nil
}

func reduceParametersDecl(ctx dksgenerated.ParametersDeclReduction) ([]string, error) {
	return prependString(ctx.Name.Text, ctx.Tail), nil
}

func reduceParametersTailMore(ctx dksgenerated.ParametersTailMoreReduction) ([]string, error) {
	return prependString(ctx.Name.Text, ctx.Tail), nil
}

func reduceEmptyStringSlice(dksgenerated.ParametersTailEmptyReduction) ([]string, error) {
	return []string{}, nil
}

func reduceCommandBlock(ctx dksgenerated.CommandBlockReduction) ([]Statement, error) {
	return ctx.Statements, nil
}

func reduceStatements(ctx dksgenerated.StatementsReduction) ([]Statement, error) {
	return prependStatement(ctx.Head, ctx.Tail), nil
}

func reduceStatementsTailMore(ctx dksgenerated.StatementsTailMoreReduction) ([]Statement, error) {
	return prependStatement(ctx.Head, ctx.Tail), nil
}

func reduceEmptyStatementSlice(dksgenerated.StatementsTailEmptyReduction) ([]Statement, error) {
	return []Statement{}, nil
}

func reducePass(ctx dksgenerated.StatementPassReduction) (Statement, error) {
	return ctx.Value, nil
}

func reduceAssign(ctx dksgenerated.AssignReduction) (Statement, error) {
	return AssignStatement{Name: ctx.Name.Text, Value: ctx.Value}, nil
}

func reduceReplace(ctx dksgenerated.ReplaceReduction) (Statement, error) {
	return ReplaceStatement{Target: ctx.Target.Text, Old: ctx.Old, New: ctx.New}, nil
}

func reduceRunSQL(ctx dksgenerated.SqlrunReduction) (Statement, error) {
	return RunSQLStatement{Instance: ctx.Instance, Script: ctx.Script}, nil
}

func reduceAddObject(ctx dksgenerated.AddObjectReduction) (Statement, error) {
	return AddObjectStatement{Parent: ctx.Parent, XML: ctx.Xml}, nil
}

func reduceRemoveObject(ctx dksgenerated.RemoveObjectReduction) (Statement, error) {
	return RemoveObjectStatement{Parent: ctx.Parent, Name: ctx.Name}, nil
}

func reduceRunObjectsJob(ctx dksgenerated.RunObjectsJobReduction) (Statement, error) {
	return RunObjectsJobStatement{Parent: ctx.Parent, Name: ctx.Name, JobsTag: ctx.JobsTag}, nil
}

// Value reductions turn scanner lexemes into the expression nodes consumed by
// compiler.go. Terminals stay as generated Lexeme values so reducers can use
// the original text and report domain-specific literal errors.
func reduceStringValue(ctx dksgenerated.ValueStringReduction) (ValueExpr, error) {
	decoded, err := decodeStringLexeme(ctx.Token.Text)
	if err != nil {
		return nil, err
	}
	return StringLiteral{Value: decoded}, nil
}

func reduceNumberValue(ctx dksgenerated.ValueNumberReduction) (ValueExpr, error) {
	return NumberLiteral{Value: ctx.Token.Text}, nil
}

func reduceIdentifierValue(ctx dksgenerated.ValueIdentReduction) (ValueExpr, error) {
	return ReferenceExpr{Name: ctx.Token.Text}, nil
}

func prependString(head string, tail []string) []string {
	return append([]string{head}, tail...)
}

func prependStatement(head Statement, tail []Statement) []Statement {
	return append([]Statement{head}, tail...)
}

// decodeStringLexeme accepts both the legacy #{...#} form and normal quoted
// strings used by the sample inputs. Keeping this outside the grammar makes
// the scanner responsible for recognizing literals and the reducer responsible
// for interpreting their text.
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
