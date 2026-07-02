package csharp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/action"
	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/lex"
	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
	"github.com/russlank/lang-forge/internal/version"
)

// Input contains all validated artifacts required by the C# backend.
type Input struct {
	Spec       *spec.Spec
	DFA        *lex.DFA
	Grammar    *parse.Grammar
	ParseTable *parse.Table
}

// Summary is the machine-readable table dump written next to generated code.
type Summary struct {
	Spec       *spec.Spec     `json:"spec"`
	Lexer      *lex.DFA       `json:"lexer"`
	Grammar    *parse.Grammar `json:"grammar"`
	ParseTable *parse.Table   `json:"parseTable"`
}

// Manifest records high-level C# generation metadata.
type Manifest struct {
	Tool         string            `json:"tool"`
	Version      string            `json:"version"`
	Commit       string            `json:"commit"`
	BuildDate    string            `json:"buildDate,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	Source       string            `json:"source"`
	Target       string            `json:"target"`
	Namespace    string            `json:"namespace"`
	Scanner      lex.ScannerConfig `json:"scanner"`
	Semantics    spec.SemanticSpec `json:"semantics,omitempty"`
	Actions      []SemanticAction  `json:"semanticActions,omitempty"`
	Tokens       []string          `json:"tokens"`
	LexerStates  int               `json:"lexerStates"`
	ParserStates int               `json:"parserStates"`
	GrammarRules int               `json:"grammarRules"`
}

// SemanticAction records one target action label in generated metadata.
type SemanticAction struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Constant string `json:"csharpConstant,omitempty"`
}

// Generate writes the C# scanner, parser, manifest, and table dump.
func Generate(input Input, outDir string) error {
	if input.Spec == nil || input.DFA == nil || input.Grammar == nil || input.ParseTable == nil {
		return errors.New("csharp codegen input is incomplete")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	namespace, err := csharpNamespace(input.Spec.Package, filepath.Base(outDir))
	if err != nil {
		return err
	}
	tokens := tokenNames(input)
	tokenIDs := tokenIdentifiers(tokens)
	actionManifest := action.Build(input.Grammar, input.Spec.Semantics, "csharp")
	actions := semanticActionsFromNames(actionManifest.Names())
	manifest := Manifest{
		Tool:         version.Name,
		Version:      version.Version,
		Commit:       version.Commit,
		BuildDate:    version.BuildDate,
		Branch:       version.Branch,
		Source:       input.Spec.SourceFile,
		Target:       "csharp",
		Namespace:    namespace,
		Scanner:      input.DFA.Scanner,
		Semantics:    input.Spec.Semantics,
		Actions:      actions,
		Tokens:       tokens,
		LexerStates:  len(input.DFA.States),
		ParserStates: len(input.ParseTable.States),
		GrammarRules: len(input.Grammar.Rules),
	}
	if err := writeJSON(filepath.Join(outDir, "langforge.manifest.json"), manifest); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "langforge.actions.json"), actionManifest); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "langforge.tables.json"), Summary{Spec: input.Spec, Lexer: input.DFA, Grammar: input.Grammar, ParseTable: input.ParseTable}); err != nil {
		return err
	}
	if err := removeLegacyGeneratedFiles(outDir); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "Tokens.g.cs"), renderTokens(namespace, input.Spec.SourceFile, tokens, tokenIDs)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "Scanner.g.cs"), renderScanner(namespace, input.Spec.SourceFile, input.DFA, tokens, tokenIDs)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "Parser.g.cs"), renderParser(namespace, input.Spec.SourceFile, input.Spec, input.ParseTable, tokens, tokenIDs, actions, actionManifest)); err != nil {
		return err
	}
	return nil
}

func removeLegacyGeneratedFiles(outDir string) error {
	for _, name := range []string{"Tokens.cs", "Scanner.cs", "Parser.cs"} {
		err := os.Remove(filepath.Join(outDir, name))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func tokenNames(input Input) []string {
	seen := map[string]bool{}
	for _, tok := range input.Spec.TokenNames() {
		seen[tok] = true
	}
	for _, rule := range input.DFA.Rules {
		if rule.Token != "" && !rule.Skip && rule.Channel == "" {
			seen[rule.Token] = true
		}
	}
	out := make([]string, 0, len(seen))
	for tok := range seen {
		out = append(out, tok)
	}
	sort.Strings(out)
	return out
}

func tokenIdentifiers(tokens []string) map[string]string {
	used := map[string]int{"EOF": 1, "Error": 1}
	out := map[string]string{}
	for _, token := range tokens {
		name := exportedIdentifierSuffix(token)
		if name == "" {
			name = "Token"
		}
		out[token] = uniqueIdentifier(name, used)
	}
	return out
}

func renderTokens(namespace string, source string, tokens []string, tokenIDs map[string]string) string {
	var b strings.Builder
	b.WriteString(generatedHeader(namespace, source))
	b.WriteString("/// <summary>Identifies one terminal emitted by the scanner.</summary>\n")
	b.WriteString("public enum Token\n{\n")
	b.WriteString("    /// <summary>Parser end-of-input.</summary>\n")
	b.WriteString("    EOF = 0,\n")
	b.WriteString("    /// <summary>Unknown token value.</summary>\n")
	b.WriteString("    Error = 1,\n")
	value := 2
	for _, token := range tokens {
		b.WriteString(fmt.Sprintf("    %s = %d,\n", tokenIDs[token], value))
		value++
	}
	b.WriteString("}\n\n")
	b.WriteString("/// <summary>Token helper methods.</summary>\n")
	b.WriteString("public static class TokenExtensions\n{\n")
	b.WriteString("    /// <summary>Returns the grammar terminal name for a token.</summary>\n")
	b.WriteString("    public static string GrammarName(this Token token) => token switch\n    {\n")
	b.WriteString("        Token.EOF => \"EOF\",\n")
	b.WriteString("        Token.Error => \"ERROR\",\n")
	for _, token := range tokens {
		b.WriteString(fmt.Sprintf("        Token.%s => %s,\n", tokenIDs[token], csharpString(token)))
	}
	b.WriteString("        _ => \"UNKNOWN\",\n")
	b.WriteString("    };\n")
	b.WriteString("}\n")
	return b.String()
}

func renderScanner(namespace string, source string, dfa *lex.DFA, tokens []string, tokenIDs map[string]string) string {
	tokenSet := map[string]bool{}
	for _, token := range tokens {
		tokenSet[token] = true
	}
	var b strings.Builder
	b.WriteString(generatedHeader(namespace, source))
	b.WriteString("using System;\n")
	b.WriteString("using System.Buffers;\n")
	b.WriteString("using System.Collections.Generic;\n")
	b.WriteString("using System.Text;\n\n")
	b.WriteString("/// <summary>One scanner result with UTF-16 offsets and Unicode scalar positions.</summary>\n")
	b.WriteString("public readonly record struct Lexeme(Token Token, string Text, string Channel, int Start, int End, int StartLine, int StartColumn, int EndLine, int EndColumn);\n\n")
	b.WriteString("/// <summary>Incrementally tokenizes an input string.</summary>\n")
	b.WriteString("/// <remarks>Scanner methods are thread-safe. Concurrent calls to Next share one serialized input cursor.</remarks>\n")
	b.WriteString("public sealed class Scanner : ILexemeSource\n{\n")
	b.WriteString("    private readonly object _gate = new();\n")
	b.WriteString("    private readonly string _input;\n")
	b.WriteString("    private int _pos;\n")
	b.WriteString("    private int _line = 1;\n")
	b.WriteString("    private int _column = 1;\n")
	b.WriteString("    private bool _includeHidden;\n\n")
	b.WriteString("    public Scanner(string input)\n    {\n        _input = input ?? throw new ArgumentNullException(nameof(input));\n    }\n\n")
	b.WriteString("    /// <summary>Controls whether channel tokens are returned.</summary>\n")
	b.WriteString("    public void IncludeHidden(bool include)\n    {\n        lock (_gate) { _includeHidden = include; }\n    }\n\n")
	b.WriteString("    /// <summary>Returns every visible token in input.</summary>\n")
	b.WriteString("    public static IReadOnlyList<Lexeme> Tokenize(string input) => new Scanner(input).All();\n\n")
	b.WriteString("    /// <summary>Returns all tokens until end-of-input.</summary>\n")
	b.WriteString("    public IReadOnlyList<Lexeme> All()\n    {\n        var output = new List<Lexeme>();\n        while (Next(out var lexeme))\n        {\n            output.Add(lexeme);\n        }\n        return output;\n    }\n\n")
	b.WriteString("    /// <summary>Returns the next visible token, or false at end-of-input.</summary>\n")
	b.WriteString("    public bool Next(out Lexeme lexeme)\n    {\n        lock (_gate)\n        {\n            while (_pos < _input.Length)\n            {\n                int start = _pos;\n                int startLine = _line;\n                int startColumn = _column;\n                var match = MatchAt(_input, _pos);\n                if (match.Rule <= 0)\n                {\n                    throw new InvalidOperationException($\"no lexical rule matched offset {_pos} near '{Preview(_input, _pos)}'\");\n                }\n                if (match.End == _pos)\n                {\n                    throw new InvalidOperationException($\"lexer rule {match.Rule} matched empty input at offset {_pos}\");\n                }\n                var action = RuleActions[match.Rule];\n                var endPosition = AdvancePosition(_input, _pos, match.End, _line, _column);\n                lexeme = new Lexeme(action.Token, _input.Substring(start, match.End - start), action.Channel, start, match.End, startLine, startColumn, endPosition.Line, endPosition.Column);\n                _pos = match.End;\n                _line = endPosition.Line;\n                _column = endPosition.Column;\n                if (action.Skip) { continue; }\n                if (action.Channel.Length != 0 && !_includeHidden) { continue; }\n                return true;\n            }\n            lexeme = new Lexeme(Token.EOF, string.Empty, string.Empty, _pos, _pos, _line, _column, _line, _column);\n            return false;\n        }\n    }\n\n")
	b.WriteString("    private readonly record struct ScannerTransition(int Lo, int Hi, int Target);\n")
	b.WriteString("    private readonly record struct ScannerState(int Accept, ScannerTransition[] Transitions);\n")
	b.WriteString("    private readonly record struct RuleAction(Token Token, bool Skip, string Channel);\n")
	b.WriteString("    private readonly record struct MatchResult(int Rule, int End);\n")
	b.WriteString("    private readonly record struct DecodedRune(int Value, int Length);\n")
	b.WriteString("    private readonly record struct Position(int Line, int Column);\n\n")
	b.WriteString("    private static readonly ScannerState[] ScannerStates = new ScannerState[]\n    {\n")
	for _, st := range dfa.States {
		b.WriteString(fmt.Sprintf("        new ScannerState(%d, new ScannerTransition[] {", st.AcceptRule))
		for _, tr := range st.Transitions {
			for _, rr := range tr.Set.Normalize() {
				b.WriteString(fmt.Sprintf(" new ScannerTransition(%d, %d, %d),", rr.Lo, rr.Hi, tr.Target))
			}
		}
		b.WriteString(" }),\n")
	}
	b.WriteString("    };\n\n")
	b.WriteString("    private static readonly Dictionary<int, RuleAction> RuleActions = new Dictionary<int, RuleAction>\n    {\n")
	for _, rule := range dfa.Rules {
		token := "Token.Error"
		if tokenSet[rule.Token] {
			token = "Token." + tokenIDs[rule.Token]
		}
		if comment := sourceComment(rule.Span); comment != "" {
			b.WriteString("        " + comment + "\n")
		}
		b.WriteString(fmt.Sprintf("        [%d] = new RuleAction(%s, %t, %s),\n", rule.Index, token, rule.Skip, csharpString(rule.Channel)))
	}
	b.WriteString("    };\n\n")
	b.WriteString("    private static MatchResult MatchAt(string input, int start)\n    {\n        int stateID = 0;\n        int bestRule = ScannerStates[stateID].Accept;\n        int bestEnd = start;\n        for (int pos = start; pos < input.Length;)\n        {\n            var decoded = DecodeScannerRune(input, pos);\n            int next = -1;\n            foreach (var transition in ScannerStates[stateID].Transitions)\n            {\n                if (decoded.Value >= transition.Lo && decoded.Value <= transition.Hi)\n                {\n                    next = transition.Target;\n                    break;\n                }\n            }\n            if (next < 0) { break; }\n            pos += decoded.Length;\n            stateID = next;\n            if (ScannerStates[stateID].Accept > 0)\n            {\n                bestRule = ScannerStates[stateID].Accept;\n                bestEnd = pos;\n            }\n        }\n        return new MatchResult(bestRule, bestEnd);\n    }\n\n")
	b.WriteString("    private static DecodedRune DecodeScannerRune(string input, int pos)\n    {\n        var status = Rune.DecodeFromUtf16(input.AsSpan(pos), out var rune, out int consumed);\n        if (status != OperationStatus.Done)\n        {\n            throw new InvalidOperationException($\"invalid UTF-16 scalar at offset {pos}\");\n        }\n        return new DecodedRune(rune.Value, consumed);\n    }\n\n")
	b.WriteString("    private static Position AdvancePosition(string input, int start, int end, int line, int column)\n    {\n        for (int pos = start; pos < end;)\n        {\n            var rune = DecodeScannerRune(input, pos);\n            pos += rune.Length;\n            if (rune.Value == '\\n')\n            {\n                line++;\n                column = 1;\n            }\n            else\n            {\n                column++;\n            }\n        }\n        return new Position(line, column);\n    }\n\n")
	b.WriteString("    private static string Preview(string input, int pos)\n    {\n        int end = Math.Min(input.Length, pos + 16);\n        return input.Substring(pos, end - pos);\n    }\n")
	b.WriteString("}\n")
	return b.String()
}

func renderParser(namespace string, source string, project *spec.Spec, table *parse.Table, tokens []string, tokenIDs map[string]string, actions []SemanticAction, actionManifest action.Manifest) string {
	var b strings.Builder
	b.WriteString(generatedHeader(namespace, source))
	b.WriteString("using System;\n")
	b.WriteString("using System.Collections.Generic;\n")
	b.WriteString("using System.Linq;\n\n")
	semantics := spec.SemanticSpec{}
	if project != nil {
		semantics = project.Semantics
	}
	actionIDs := semanticActionIDs(actions)
	b.WriteString("/// <summary>Identifies one generated semantic reduction hook.</summary>\n")
	b.WriteString("public enum SemanticAction\n{\n")
	b.WriteString("    None = 0,\n")
	for _, action := range actions {
		b.WriteString(fmt.Sprintf("    %s = %d,\n", action.Constant, action.ID))
	}
	b.WriteString("}\n\n")
	b.WriteString("/// <summary>One expected terminal or reporting group.</summary>\n")
	b.WriteString("public sealed record ExpectedToken(string Symbol, string Display, IReadOnlyList<string> Members);\n\n")
	b.WriteString("/// <summary>Describes how one syntax error was recovered.</summary>\n")
	b.WriteString("public sealed record RecoveryAction(string Kind, int Discarded);\n\n")
	b.WriteString("/// <summary>One source-rich syntax diagnostic.</summary>\n")
	b.WriteString("public sealed record ParseDiagnostic(int State, string Unexpected, string UnexpectedDisplay, IReadOnlyList<ExpectedToken> Expected, int Start, int End, int StartLine, int StartColumn, int EndLine, int EndColumn, RecoveryAction Recovery);\n\n")
	b.WriteString("/// <summary>A possibly partial parser value plus every syntax diagnostic.</summary>\n")
	b.WriteString("public sealed record ParseResult(object? Value, IReadOnlyList<ParseDiagnostic> Diagnostics, bool Accepted);\n\n")
	b.WriteString("/// <summary>Thrown by compatibility parse APIs when syntax diagnostics were produced.</summary>\n")
	b.WriteString("public sealed class ParseException : InvalidOperationException\n{\n    public ParseException(IReadOnlyList<ParseDiagnostic> diagnostics) : base(FormatMessage(diagnostics)) => Diagnostics = diagnostics;\n    public IReadOnlyList<ParseDiagnostic> Diagnostics { get; }\n    private static string FormatMessage(IReadOnlyList<ParseDiagnostic> diagnostics)\n    {\n        if (diagnostics.Count == 0) { return \"parse error\"; }\n        var first = diagnostics[0];\n        string expected = first.Expected.Count == 0 ? string.Empty : $\"; expected {string.Join(\", \", first.Expected.Select(item => item.Display))}\";\n        string count = diagnostics.Count == 1 ? string.Empty : $\" ({diagnostics.Count} diagnostics)\";\n        return $\"parse error at {first.StartLine}:{first.StartColumn}: unexpected {first.UnexpectedDisplay}{expected}{count}\";\n    }\n}\n\n")
	b.WriteString("/// <summary>Synchronous pull source consumed by generated parsers.</summary>\n")
	b.WriteString("/// <remarks>Returning false means natural end-of-input. Returning one explicit EOF lexeme is also supported; later tokens are rejected.</remarks>\n")
	b.WriteString("public interface ILexemeSource\n{\n    bool Next(out Lexeme lexeme);\n}\n\n")
	b.WriteString("/// <summary>Generated parser runtime.</summary>\n")
	b.WriteString("/// <remarks>Parser instances are safe for concurrent Parse and ParseValue calls when the installed reducer is also safe.</remarks>\n")
	b.WriteString("public sealed class Parser\n{\n")
	b.WriteString("    private readonly IReducer? _reducer;\n\n")
	b.WriteString("    public Parser(IReducer? reducer = null)\n    {\n        if (reducer is IReducerCoverage coverage) { coverage.ValidateCoverage(); }\n        _reducer = reducer;\n    }\n\n")
	b.WriteString("    public static void Parse(IReadOnlyList<Lexeme> input) => new Parser().ParseInput(input);\n")
	b.WriteString("    public static void ParseFromSource(ILexemeSource source) => new Parser().ParseSource(source);\n")
	b.WriteString("    public static object? ParseValue(IReadOnlyList<Lexeme> input) => new Parser().ParseValueInput(input);\n")
	b.WriteString("    public static object? ParseValueFromSource(ILexemeSource source) => new Parser().ParseValueSource(source);\n")
	b.WriteString("    public static ParseResult ParseRecovering(IReadOnlyList<Lexeme> input) => new Parser().ParseRecoveringInput(input);\n")
	b.WriteString("    public static ParseResult ParseRecoveringFromSource(ILexemeSource source) => new Parser().ParseRecoveringSource(source);\n")
	b.WriteString("    public static object? ParseWithReducer(IReadOnlyList<Lexeme> input, IReducer reducer) => new Parser(reducer).ParseValueInput(input);\n")
	b.WriteString("    public static object? ParseWithReducerFromSource(ILexemeSource source, IReducer reducer) => new Parser(reducer).ParseValueSource(source);\n\n")
	b.WriteString("    public void ParseInput(IReadOnlyList<Lexeme> input) => ParseValueInput(input);\n")
	b.WriteString("    public void ParseSource(ILexemeSource source) => ParseValueSource(source);\n\n")
	b.WriteString("    public object? ParseValueInput(IReadOnlyList<Lexeme> input) => ParseValueSource(new LexemeListSource(input));\n\n")
	b.WriteString("    public object? ParseValueSource(ILexemeSource source)\n    {\n        var result = ParseRecoveringSource(source);\n        if (result.Diagnostics.Count != 0) { throw new ParseException(result.Diagnostics); }\n        return result.Value;\n    }\n\n")
	b.WriteString("    /// <summary>Parses with grammar-directed recovery and returns every syntax diagnostic.</summary>\n")
	b.WriteString("    public ParseResult ParseRecoveringInput(IReadOnlyList<Lexeme> input) => ParseRecoveringSource(new LexemeListSource(input));\n\n")
	b.WriteString("    /// <summary>Parses a synchronous pull source with grammar-directed recovery.</summary>\n")
	b.WriteString("    public ParseResult ParseRecoveringSource(ILexemeSource source)\n    {\n        var cursor = new LexemeSourceCursor(source);\n        var states = new List<int> { 0 };\n        var values = new List<object?>();\n        var diagnostics = new List<ParseDiagnostic>();\n        int recovering = 0;\n        int activeDiagnostic = -1;\n        while (true)\n        {\n            string lookahead = cursor.PeekSymbol();\n            if (!TryFindAction(states[^1], lookahead, out var action))\n            {\n                if (recovering == 0)\n                {\n                    diagnostics.Add(NewParseDiagnostic(states[^1], lookahead, cursor));\n                    activeDiagnostic = diagnostics.Count - 1;\n                    bool recovered = false;\n                    while (states.Count != 0)\n                    {\n                        if (TryFindAction(states[^1], \"error\", out var errorAction) && errorAction.Kind == \"shift\")\n                        {\n                            states.Add(errorAction.State);\n                            values.Add(ParserErrorLexeme(cursor));\n                            recovering = 3;\n                            diagnostics[activeDiagnostic] = diagnostics[activeDiagnostic] with { Recovery = new RecoveryAction(\"shift-error\", 0) };\n                            recovered = true;\n                            break;\n                        }\n                        if (states.Count == 1) { break; }\n                        states.RemoveAt(states.Count - 1);\n                        if (values.Count != 0) { values.RemoveAt(values.Count - 1); }\n                    }\n                    if (recovered) { continue; }\n                    diagnostics[activeDiagnostic] = diagnostics[activeDiagnostic] with { Recovery = new RecoveryAction(\"abort\", 0) };\n                    return new ParseResult(CurrentValue(values), diagnostics, false);\n                }\n                if (lookahead == \"$\")\n                {\n                    if (activeDiagnostic >= 0) { diagnostics[activeDiagnostic] = diagnostics[activeDiagnostic] with { Recovery = diagnostics[activeDiagnostic].Recovery with { Kind = \"abort\" } }; }\n                    return new ParseResult(CurrentValue(values), diagnostics, false);\n                }\n                cursor.Advance();\n                if (activeDiagnostic >= 0) { diagnostics[activeDiagnostic] = diagnostics[activeDiagnostic] with { Recovery = diagnostics[activeDiagnostic].Recovery with { Discarded = diagnostics[activeDiagnostic].Recovery.Discarded + 1 } }; }\n                continue;\n            }\n            switch (action.Kind)\n            {\n                case \"shift\":\n                    states.Add(action.State);\n                    values.Add(cursor.Advance());\n                    if (recovering > 0)\n                    {\n                        recovering--;\n                        if (recovering == 0 && activeDiagnostic >= 0)\n                        {\n                            diagnostics[activeDiagnostic] = diagnostics[activeDiagnostic] with { Recovery = diagnostics[activeDiagnostic].Recovery with { Kind = \"recovered\" } };\n                            activeDiagnostic = -1;\n                        }\n                    }\n                    break;\n                case \"reduce\":\n                    var rule = ParserRules[action.Rule];\n                    if (states.Count < rule.Size + 1) { throw new InvalidOperationException($\"parser stack underflow reducing rule {action.Rule}\"); }\n                    if (values.Count < rule.Size) { throw new InvalidOperationException($\"semantic value stack underflow reducing rule {action.Rule}\"); }\n                    var rhs = values.Skip(values.Count - rule.Size).Take(rule.Size).ToArray();\n                    values.RemoveRange(values.Count - rule.Size, rule.Size);\n                    object? value = Reduce(action.Rule, rule, rhs);\n                    states.RemoveRange(states.Count - rule.Size, rule.Size);\n                    if (!ParserGotos.TryGetValue(states[^1], out var gotoBySymbol) || !gotoBySymbol.TryGetValue(rule.LHS, out int gotoState))\n                    {\n                        throw new InvalidOperationException($\"missing goto from state {states[^1]} on {rule.LHS}\");\n                    }\n                    states.Add(gotoState);\n                    values.Add(value);\n                    break;\n                case \"accept\":\n                    if (activeDiagnostic >= 0) { diagnostics[activeDiagnostic] = diagnostics[activeDiagnostic] with { Recovery = diagnostics[activeDiagnostic].Recovery with { Kind = \"recovered\" } }; }\n                    return new ParseResult(CurrentValue(values), diagnostics, true);\n                default:\n                    throw new InvalidOperationException($\"invalid parser action '{action.Kind}'\");\n            }\n        }\n    }\n\n")
	b.WriteString("    private object? Reduce(int ruleID, ParserRule rule, object?[] values)\n    {\n        var ctx = new Reduction(ruleID, rule.LHS, rule.RHS, rule.Labels, rule.Action, SemanticActions.SemanticActionName(rule.Action), values);\n        if (_reducer is not null && rule.Action != SemanticAction.None)\n        {\n            return _reducer.Reduce(ctx);\n        }\n        return DefaultReduce(values);\n    }\n\n")
	b.WriteString("    private static bool TryFindAction(int state, string symbol, out ParserAction action)\n    {\n        if (ParserActions.TryGetValue(state, out var bySymbol) && bySymbol.TryGetValue(symbol, out action)) { return true; }\n        action = default;\n        return false;\n    }\n\n")
	b.WriteString("    private static object? CurrentValue(IReadOnlyList<object?> values) => values.Count == 0 ? null : values[^1];\n\n")
	b.WriteString("    private static object? DefaultReduce(IReadOnlyList<object?> values)\n    {\n        return values.Count switch\n        {\n            0 => null,\n            1 => values[0],\n            _ => values.ToArray(),\n        };\n    }\n\n")
	b.WriteString("    private sealed class LexemeListSource : ILexemeSource\n    {\n        private readonly IReadOnlyList<Lexeme> _input;\n        private int _pos;\n\n        public LexemeListSource(IReadOnlyList<Lexeme> input)\n        {\n            _input = input ?? throw new ArgumentNullException(nameof(input));\n        }\n\n        public bool Next(out Lexeme lexeme)\n        {\n            if (_pos >= _input.Count)\n            {\n                lexeme = default;\n                return false;\n            }\n            lexeme = _input[_pos++];\n            return true;\n        }\n    }\n\n")
	b.WriteString("    private sealed class LexemeSourceCursor\n    {\n        private readonly ILexemeSource _source;\n        private Lexeme _lookahead;\n        private string _symbol = \"$\";\n        private bool _ready;\n        private bool _sawEOF;\n        private Lexeme _last;\n        private bool _haveLast;\n\n        public LexemeSourceCursor(ILexemeSource source)\n        {\n            _source = source ?? throw new ArgumentNullException(nameof(source));\n        }\n\n        public string PeekSymbol()\n        {\n            if (_ready) { return _symbol; }\n            if (_sawEOF)\n            {\n                _lookahead = EOFLexeme();\n                _symbol = \"$\";\n                _ready = true;\n                return _symbol;\n            }\n            if (!_source.Next(out var lexeme))\n            {\n                _lookahead = EOFLexeme();\n                _symbol = \"$\";\n                _ready = true;\n                _sawEOF = true;\n                return _symbol;\n            }\n            if (lexeme.Token == Token.EOF)\n            {\n                lexeme = NormalizeEOFLexeme(lexeme);\n                if (_source.Next(out var extra))\n                {\n                    throw new InvalidOperationException($\"token after EOF in lexeme source: {TerminalName(extra.Token)} at {extra.StartLine}:{extra.StartColumn}\");\n                }\n                _lookahead = lexeme;\n                _symbol = \"$\";\n                _ready = true;\n                _sawEOF = true;\n                return _symbol;\n            }\n            _last = lexeme;\n            _haveLast = true;\n            _lookahead = lexeme;\n            _symbol = TerminalName(lexeme.Token);\n            _ready = true;\n            return _symbol;\n        }\n\n        public Lexeme Advance()\n        {\n            var symbol = PeekSymbol();\n            if (symbol == \"$\") { throw new InvalidOperationException(\"shift past end of input\"); }\n            var lexeme = _lookahead;\n            _ready = false;\n            return lexeme;\n        }\n\n        public Lexeme DiagnosticLexeme() => _ready ? _lookahead : EOFLexeme();\n\n        private Lexeme EOFLexeme()\n        {\n            if (_haveLast)\n            {\n                return new Lexeme(Token.EOF, string.Empty, string.Empty, _last.End, _last.End, _last.EndLine, _last.EndColumn, _last.EndLine, _last.EndColumn);\n            }\n            return new Lexeme(Token.EOF, string.Empty, string.Empty, 0, 0, 1, 1, 1, 1);\n        }\n\n        private Lexeme NormalizeEOFLexeme(Lexeme lexeme)\n        {\n            var fallback = EOFLexeme();\n            return new Lexeme(\n                Token.EOF,\n                lexeme.Text,\n                lexeme.Channel,\n                lexeme.Start == 0 && lexeme.End == 0 && _haveLast ? fallback.Start : lexeme.Start,\n                lexeme.Start == 0 && lexeme.End == 0 && _haveLast ? fallback.End : lexeme.End,\n                lexeme.StartLine <= 0 ? fallback.StartLine : lexeme.StartLine,\n                lexeme.StartColumn <= 0 ? fallback.StartColumn : lexeme.StartColumn,\n                lexeme.EndLine <= 0 ? fallback.EndLine : lexeme.EndLine,\n                lexeme.EndColumn <= 0 ? fallback.EndColumn : lexeme.EndColumn);\n        }\n    }\n\n")
	b.WriteString("    private static ParseDiagnostic NewParseDiagnostic(int state, string unexpected, LexemeSourceCursor cursor)\n    {\n        var lexeme = cursor.DiagnosticLexeme();\n        var expected = ParserExpected.TryGetValue(state, out var entries) ? entries : Array.Empty<ExpectedToken>();\n        return new ParseDiagnostic(state, unexpected, UnexpectedDisplay(unexpected), expected, lexeme.Start, lexeme.End, lexeme.StartLine, lexeme.StartColumn, lexeme.EndLine, lexeme.EndColumn, new RecoveryAction(\"none\", 0));\n    }\n\n")
	b.WriteString("    private static Lexeme ParserErrorLexeme(LexemeSourceCursor cursor)\n    {\n        var lexeme = cursor.DiagnosticLexeme();\n        return new Lexeme(Token.Error, string.Empty, string.Empty, lexeme.Start, lexeme.Start, lexeme.StartLine, lexeme.StartColumn, lexeme.StartLine, lexeme.StartColumn);\n    }\n\n")
	b.WriteString("    private static string UnexpectedDisplay(string symbol)\n    {\n        if (symbol == \"$\") { return \"end of input\"; }\n        return ParserTokenAliases.TryGetValue(symbol, out var display) ? display : symbol;\n    }\n\n")
	b.WriteString("    private static string TerminalName(Token token) => token switch\n    {\n        Token.EOF => \"$\",\n")
	for _, token := range tokens {
		b.WriteString(fmt.Sprintf("        Token.%s => %s,\n", tokenIDs[token], csharpString(token)))
	}
	b.WriteString("        _ => \"ERROR\",\n    };\n\n")
	b.WriteString("    private readonly record struct ParserAction(string Kind, int State, int Rule);\n")
	b.WriteString("    private readonly record struct ParserRule(string LHS, string[] RHS, string[] Labels, int Size, SemanticAction Action);\n\n")
	b.WriteString("    private static readonly Dictionary<string, string> ParserTokenAliases = new Dictionary<string, string>\n    {\n")
	for _, alias := range project.Grammar.ExpectedTokens.Aliases {
		b.WriteString(fmt.Sprintf("        [%s] = %s,\n", csharpString(alias.Token), csharpString(alias.Label)))
	}
	b.WriteString("    };\n\n")
	renderParserTables(&b, table, actionIDs)
	b.WriteString("}\n\n")
	renderSemanticSupport(&b, actions, semantics, actionManifest)
	return b.String()
}

func renderParserTables(b *strings.Builder, table *parse.Table, actionIDs map[string]string) {
	b.WriteString("    private static readonly Dictionary<int, Dictionary<string, ParserAction>> ParserActions = new Dictionary<int, Dictionary<string, ParserAction>>\n    {\n")
	for _, state := range sortedActionStates(table.Actions) {
		b.WriteString(fmt.Sprintf("        [%d] = new Dictionary<string, ParserAction>\n        {\n", state))
		for _, sym := range sortedActionSymbols(table.Actions[state]) {
			action := table.Actions[state][sym]
			if action.Kind == parse.ActionReduce {
				if rule, ok := ruleByID(table.Rules, action.Rule); ok {
					b.WriteString(indentComment(ruleSourceComment(rule, "csharp", "// "), "            "))
				}
			}
			b.WriteString(fmt.Sprintf("            [%s] = new ParserAction(%s, %d, %d),\n", csharpString(sym), csharpString(string(action.Kind)), action.State, action.Rule))
		}
		b.WriteString("        },\n")
	}
	b.WriteString("    };\n\n")
	b.WriteString("    private static readonly Dictionary<int, ExpectedToken[]> ParserExpected = new Dictionary<int, ExpectedToken[]>\n    {\n")
	for _, state := range sortedExpectedStates(table.Expected) {
		b.WriteString(fmt.Sprintf("        [%d] = new ExpectedToken[]\n        {\n", state))
		for _, expected := range table.Expected[state] {
			b.WriteString(fmt.Sprintf("            new ExpectedToken(%s, %s, %s),\n", csharpString(expected.Symbol), csharpString(expected.Display), renderStringArray(expected.Members)))
		}
		b.WriteString("        },\n")
	}
	b.WriteString("    };\n\n")
	b.WriteString("    private static readonly Dictionary<int, Dictionary<string, int>> ParserGotos = new Dictionary<int, Dictionary<string, int>>\n    {\n")
	for _, state := range sortedGotoStates(table.Gotos) {
		b.WriteString(fmt.Sprintf("        [%d] = new Dictionary<string, int>\n        {\n", state))
		for _, sym := range sortedGotoSymbols(table.Gotos[state]) {
			b.WriteString(fmt.Sprintf("            [%s] = %d,\n", csharpString(sym), table.Gotos[state][sym]))
		}
		b.WriteString("        },\n")
	}
	b.WriteString("    };\n\n")
	b.WriteString("    private static readonly Dictionary<int, ParserRule> ParserRules = new Dictionary<int, ParserRule>\n    {\n")
	for _, rule := range table.Rules {
		b.WriteString(indentComment(ruleSourceComment(rule, "csharp", "// "), "        "))
		action := semanticActionExpr(rule.Actions["csharp"], actionIDs)
		b.WriteString(fmt.Sprintf("        [%d] = new ParserRule(%s, %s, %s, %d, %s),\n", rule.ID, csharpString(rule.LHS), renderStringArray(rule.RHS), renderStringArray(rule.Labels), len(rule.RHS), action))
	}
	b.WriteString("    };\n")
}

func renderSemanticSupport(b *strings.Builder, actions []SemanticAction, semantics spec.SemanticSpec, actionManifest action.Manifest) {
	b.WriteString("/// <summary>Describes one grammar rule reduction.</summary>\n")
	b.WriteString("public sealed record Reduction(int Rule, string LHS, IReadOnlyList<string> RHS, IReadOnlyList<string> Labels, SemanticAction ActionID, string Action, IReadOnlyList<object?> Values)\n{\n")
	b.WriteString("    /// <summary>Returns the semantic value associated with a named RHS label.</summary>\n")
	b.WriteString("    public object? ValueFor(string label)\n    {\n        for (var index = 0; index < Labels.Count; index++)\n        {\n            if (Labels[index] == label)\n            {\n                if (index >= Values.Count)\n                {\n                    throw new InvalidOperationException($\"rule {Rule} action {Action} label {label} has no semantic value\");\n                }\n                return Values[index];\n            }\n        }\n        throw new InvalidOperationException($\"rule {Rule} action {Action} has no RHS label {label}\");\n    }\n}\n\n")
	b.WriteString("/// <summary>Receives target-tagged action hooks during parser reductions.</summary>\n")
	b.WriteString("public interface IReducer\n{\n    object? Reduce(Reduction ctx);\n}\n\n")
	b.WriteString("/// <summary>Validates that a reducer covers the generated semantic action set.</summary>\n")
	b.WriteString("public interface IReducerCoverage\n{\n    void ValidateCoverage();\n}\n\n")
	b.WriteString("/// <summary>Adapts a function to the generated reducer interface.</summary>\n")
	b.WriteString("public sealed class ReducerFunc : IReducer\n{\n    private readonly Func<Reduction, object?> _handler;\n    public ReducerFunc(Func<Reduction, object?> handler) => _handler = handler ?? throw new ArgumentNullException(nameof(handler));\n    public object? Reduce(Reduction ctx) => _handler(ctx);\n}\n\n")
	b.WriteString("/// <summary>Dispatches reductions by generated semantic action ID.</summary>\n")
	b.WriteString("public sealed class ReducerMap : Dictionary<SemanticAction, Func<Reduction, object?>>, IReducer, IReducerCoverage\n{\n")
	b.WriteString("    /// <summary>Reports missing or unknown semantic action handlers before parsing starts.</summary>\n")
	b.WriteString("    public void ValidateCoverage()\n    {\n        var missing = new List<string>();\n        for (var index = 1; index < SemanticActions.Count; index++)\n        {\n            var action = (SemanticAction)index;\n            if (!ContainsKey(action)) { missing.Add(action.ToString()); }\n        }\n        int firstUnknown = 0;\n        var hasUnknown = false;\n        foreach (var action in Keys)\n        {\n            if (action <= SemanticAction.None || (int)action >= SemanticActions.Count)\n            {\n                if (!hasUnknown || (int)action < firstUnknown)\n                {\n                    firstUnknown = (int)action;\n                    hasUnknown = true;\n                }\n            }\n        }\n        if (missing.Count == 0 && !hasUnknown) { return; }\n        var suffix = hasUnknown ? $\" firstUnknown={firstUnknown}\" : string.Empty;\n        throw new InvalidOperationException($\"semantic reducer coverage mismatch: missing=[{string.Join(\", \", missing)}]{suffix}\");\n    }\n\n")
	b.WriteString("    public object? Reduce(Reduction ctx)\n    {\n        if (!TryGetValue(ctx.ActionID, out var handler))\n        {\n            throw new InvalidOperationException($\"no reducer registered for action {ctx.ActionID}\");\n        }\n        return handler(ctx);\n    }\n}\n\n")
	renderTypedReductionContexts(b, actionManifest, actions)
	b.WriteString("/// <summary>Semantic metadata helpers for generated action IDs.</summary>\n")
	b.WriteString("public static class SemanticActions\n{\n")
	b.WriteString("    private static readonly string[] Names = new string[]\n    {\n        \"\",\n")
	for _, action := range actions {
		b.WriteString("        " + csharpString(action.Name) + ",\n")
	}
	b.WriteString("    };\n\n")
	b.WriteString("    private static readonly Dictionary<string, SemanticAction> ByName = new Dictionary<string, SemanticAction>\n    {\n")
	for _, action := range actions {
		b.WriteString(fmt.Sprintf("        [%s] = SemanticAction.%s,\n", csharpString(action.Name), action.Constant))
	}
	b.WriteString("    };\n\n")
	b.WriteString("    public static int Count => Names.Length;\n")
	b.WriteString("    public static string SemanticActionName(SemanticAction action) => (int)action >= 0 && (int)action < Names.Length ? Names[(int)action] : \"UNKNOWN\";\n")
	b.WriteString("    public static bool TryLookupSemanticAction(string name, out SemanticAction action) => ByName.TryGetValue(name, out action);\n")
	b.WriteString("}\n\n")
	b.WriteString("/// <summary>Records how generated parser action text is handled.</summary>\n")
	b.WriteString("public static class SemanticMetadata\n{\n")
	b.WriteString("    public const string Mode = " + csharpString(string(semantics.ModeFor("csharp"))) + ";\n")
	b.WriteString("}\n")
}

func renderTypedReductionContexts(b *strings.Builder, manifest action.Manifest, actions []SemanticAction) {
	if len(manifest.Actions) == 0 {
		return
	}
	constants := semanticActionIDs(actions)
	typedActions := make([]action.Action, 0, len(manifest.Actions))
	for _, semanticAction := range manifest.Actions {
		if !semanticAction.Typed || len(semanticAction.Rules) == 0 || constants[semanticAction.Name] == "" {
			continue
		}
		typedActions = append(typedActions, semanticAction)
	}
	if len(typedActions) == 0 {
		return
	}
	for _, semanticAction := range typedActions {
		constant := constants[semanticAction.Name]
		suffix := strings.TrimPrefix(constant, "SemanticAction.")
		contextType := suffix + "Reduction"
		handlerType := suffix + "Handler"
		fields := typedFields(semanticAction.Rules[0])

		b.WriteString(fmt.Sprintf("/// <summary>Generated typed context for the %s semantic action.</summary>\n", csharpString(semanticAction.Name)))
		b.WriteString(fmt.Sprintf("internal sealed record %s(Reduction Reduction", contextType))
		for _, field := range fields {
			b.WriteString(fmt.Sprintf(", %s %s", field.Type, field.Name))
		}
		b.WriteString(");\n\n")

		b.WriteString(fmt.Sprintf("/// <summary>Handles one typed %s reduction.</summary>\n", csharpString(semanticAction.Name)))
		b.WriteString(fmt.Sprintf("internal delegate %s %s(%s ctx);\n\n", semanticAction.ReturnType, handlerType, contextType))
	}

	b.WriteString("/// <summary>Factory methods that adapt generated typed reducer handlers.</summary>\n")
	b.WriteString("internal static class SemanticReducerContexts\n{\n")
	b.WriteString("    private static T SemanticValueAs<T>(Reduction ctx, string label)\n    {\n        var value = ctx.ValueFor(label);\n        if (value is T typed) { return typed; }\n        throw new InvalidOperationException($\"rule {ctx.Rule} action {ctx.Action} label {label} has type {value?.GetType().Name ?? \"<null>\"}, want {typeof(T).Name}\");\n    }\n\n")
	for _, semanticAction := range typedActions {
		constant := constants[semanticAction.Name]
		if constant == "" {
			continue
		}
		suffix := strings.TrimPrefix(constant, "SemanticAction.")
		contextType := suffix + "Reduction"
		handlerType := suffix + "Handler"
		constructor := "New" + contextType
		adapter := "Typed" + suffix
		fields := typedFields(semanticAction.Rules[0])

		b.WriteString(fmt.Sprintf("    /// <summary>Validates and converts an untyped reduction context for %s.</summary>\n", csharpString(semanticAction.Name)))
		b.WriteString(fmt.Sprintf("    internal static %s %s(Reduction ctx)\n    {\n", contextType, constructor))
		b.WriteString(fmt.Sprintf("        if (ctx.ActionID != %s) { throw new InvalidOperationException($\"typed context %s requires action %s, got {ctx.ActionID}\"); }\n", constant, contextType, suffix))
		b.WriteString(fmt.Sprintf("        return new %s(ctx", contextType))
		for _, field := range fields {
			b.WriteString(fmt.Sprintf(", SemanticValueAs<%s>(ctx, %s)", field.Type, csharpString(field.Label)))
		}
		b.WriteString(");\n    }\n\n")

		b.WriteString("    /// <summary>Adapts a typed handler to the generated reducer map shape.</summary>\n")
		b.WriteString(fmt.Sprintf("    internal static Func<Reduction, object?> %s(%s handler)\n    {\n", adapter, handlerType))
		b.WriteString("        if (handler is null) { throw new ArgumentNullException(nameof(handler)); }\n")
		b.WriteString(fmt.Sprintf("        return ctx => handler(%s(ctx));\n", constructor))
		b.WriteString("    }\n\n")
	}
	b.WriteString("}\n\n")
}

type typedField struct {
	Label string
	Name  string
	Type  string
}

func typedFields(rule action.Rule) []typedField {
	used := map[string]int{}
	var fields []typedField
	for _, operand := range rule.RHS {
		if operand.Label == "" {
			continue
		}
		base := exportedIdentifierSuffix(operand.Label)
		if base == "" {
			base = "Value"
		}
		used[base]++
		name := base
		if used[base] > 1 {
			name = fmt.Sprintf("%s%d", base, used[base])
		}
		fields = append(fields, typedField{
			Label: operand.Label,
			Name:  name,
			Type:  operand.Type,
		})
	}
	return fields
}

func semanticActions(rules []parse.Rule, target string) []SemanticAction {
	seen := map[string]bool{}
	var names []string
	for _, rule := range rules {
		name := strings.TrimSpace(rule.Actions[target])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return semanticActionsFromNames(names)
}

func semanticActionsFromNames(names []string) []SemanticAction {
	usedConstants := map[string]int{"None": 1}
	out := make([]SemanticAction, 0, len(names))
	for _, name := range names {
		id := len(out) + 1
		out = append(out, SemanticAction{
			ID:       id,
			Name:     name,
			Constant: uniqueIdentifier(exportedIdentifierSuffix(name), usedConstants),
		})
	}
	return out
}

func semanticActionIDs(actions []SemanticAction) map[string]string {
	out := map[string]string{}
	for _, action := range actions {
		out[action.Name] = "SemanticAction." + action.Constant
	}
	return out
}

func semanticActionExpr(name string, ids map[string]string) string {
	constant, ok := ids[strings.TrimSpace(name)]
	if !ok {
		return "SemanticAction.None"
	}
	return constant
}

func exportedIdentifierSuffix(name string) string {
	var b strings.Builder
	upperNext := true
	for _, r := range name {
		if isASCIIAlpha(r) || isASCIIDigit(r) {
			if upperNext && isASCIIAlpha(r) {
				r = toASCIIUpper(r)
			}
			b.WriteRune(r)
			upperNext = false
			continue
		}
		upperNext = true
	}
	return b.String()
}

func uniqueIdentifier(base string, used map[string]int) string {
	if base == "" {
		base = "Action"
	}
	if isASCIIDigit(rune(base[0])) {
		base = "N" + base
	}
	if used[base] == 0 {
		used[base] = 1
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", base, i)
		if used[candidate] == 0 {
			used[candidate] = 1
			return candidate
		}
	}
}

func csharpNamespace(specPackage string, outDirBase string) (string, error) {
	if specPackage != "" {
		if !isValidCSharpNamespace(specPackage) {
			return "", fmt.Errorf("invalid C# namespace %q", specPackage)
		}
		return specPackage, nil
	}
	part := sanitizeIdentifier(outDirBase)
	if part == "" {
		part = "Generated"
	}
	return "LangForge.Generated." + part, nil
}

func sanitizeIdentifier(value string) string {
	var b strings.Builder
	upperNext := true
	for _, r := range value {
		if isASCIIAlpha(r) || isASCIIDigit(r) {
			if b.Len() == 0 && isASCIIDigit(r) {
				continue
			}
			if upperNext && isASCIIAlpha(r) {
				r = toASCIIUpper(r)
			}
			b.WriteRune(r)
			upperNext = false
			continue
		}
		upperNext = true
	}
	return b.String()
}

func isValidCSharpNamespace(namespace string) bool {
	if namespace == "" {
		return false
	}
	for _, part := range strings.Split(namespace, ".") {
		if !isValidCSharpIdentifier(part) {
			return false
		}
	}
	return true
}

func isValidCSharpIdentifier(identifier string) bool {
	if identifier == "" {
		return false
	}
	for i, r := range identifier {
		if i == 0 {
			if !(r == '_' || isASCIIAlpha(r)) {
				return false
			}
			continue
		}
		if !(r == '_' || isASCIIAlpha(r) || isASCIIDigit(r)) {
			return false
		}
	}
	return true
}

func isASCIIAlpha(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func toASCIIUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}

func generatedHeader(namespace string, source string) string {
	var b strings.Builder
	b.WriteString("// <auto-generated />\n")
	b.WriteString("// Code generated by lang-forge; DO NOT EDIT.\n")
	if source != "" {
		b.WriteString("// Source: " + source + "\n")
	}
	b.WriteString("#nullable enable\n\n")
	b.WriteString("namespace " + namespace + ";\n\n")
	return b.String()
}

func sourceComment(span diagnostics.Span) string {
	ref := sourceRef(span)
	if ref == "" {
		return ""
	}
	return "// Source: " + ref
}

func ruleSourceComment(rule parse.Rule, target string, prefix string) string {
	var lines []string
	lines = append(lines, prefix+grammarRuleDisplay(rule, target))
	if ref := sourceRef(rule.Span); ref != "" {
		lines = append(lines, prefix+"Source: "+ref)
	}
	return strings.Join(lines, "\n") + "\n"
}

func grammarRuleDisplay(rule parse.Rule, target string) string {
	rhs := "%empty"
	if len(rule.RHS) > 0 {
		parts := make([]string, 0, len(rule.RHS))
		for index, symbol := range rule.RHS {
			label := ""
			if index < len(rule.Labels) {
				label = rule.Labels[index]
			}
			if label != "" {
				parts = append(parts, label+"="+symbol)
			} else {
				parts = append(parts, symbol)
			}
		}
		rhs = strings.Join(parts, " ")
	}
	display := fmt.Sprintf("Grammar rule %d: %s -> %s", rule.ID, rule.LHS, rhs)
	if action := commentSafe(strings.TrimSpace(rule.Actions[target])); action != "" {
		display += fmt.Sprintf(" {%s: %s}", target, action)
	}
	return display
}

func commentSafe(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func indentComment(comment string, indent string) string {
	if comment == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(comment, "\n"), "\n") {
		b.WriteString(indent + line + "\n")
	}
	return b.String()
}

func ruleByID(rules []parse.Rule, id int) (parse.Rule, bool) {
	for _, rule := range rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return parse.Rule{}, false
}

func sourceRef(span diagnostics.Span) string {
	if span.File == "" || span.Start.Line <= 0 {
		return ""
	}
	column := span.Start.Column
	if column <= 0 {
		column = 1
	}
	return fmt.Sprintf("%s:%d:%d", sanitizeSourceFile(span.File), span.Start.Line, column)
}

func sanitizeSourceFile(filename string) string {
	filename = strings.ReplaceAll(filename, "\r", "_")
	filename = strings.ReplaceAll(filename, "\n", "_")
	return filename
}

func renderStringArray(values []string) string {
	if len(values) == 0 {
		return "Array.Empty<string>()"
	}
	var b strings.Builder
	b.WriteString("new string[] {")
	for _, value := range values {
		b.WriteString(" " + csharpString(value) + ",")
	}
	b.WriteString(" }")
	return b.String()
}

func csharpString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "\"\""
	}
	return string(data)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeFile(path string, source string) error {
	return os.WriteFile(path, []byte(source), 0o644)
}

func sortedActionStates(in map[int]map[string]parse.Action) []int {
	out := make([]int, 0, len(in))
	for state := range in {
		out = append(out, state)
	}
	sort.Ints(out)
	return out
}

func sortedActionSymbols(in map[string]parse.Action) []string {
	out := make([]string, 0, len(in))
	for sym := range in {
		out = append(out, sym)
	}
	sort.Strings(out)
	return out
}

func sortedGotoStates(in map[int]map[string]int) []int {
	out := make([]int, 0, len(in))
	for state := range in {
		out = append(out, state)
	}
	sort.Ints(out)
	return out
}

func sortedGotoSymbols(in map[string]int) []string {
	out := make([]string, 0, len(in))
	for sym := range in {
		out = append(out, sym)
	}
	sort.Strings(out)
	return out
}

func sortedExpectedStates(in map[int][]parse.ExpectedToken) []int {
	out := make([]int, 0, len(in))
	for state := range in {
		out = append(out, state)
	}
	sort.Ints(out)
	return out
}
