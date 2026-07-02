package cpp

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

// Input contains all validated artifacts required by the C++ backend.
type Input struct {
	Spec       *spec.Spec
	DFA        *lex.DFA
	Grammar    *parse.Grammar
	ParseTable *parse.Table
}

// Summary is the machine-readable table dump written next to generated C++ code.
type Summary struct {
	Spec       *spec.Spec     `json:"spec"`
	Lexer      *lex.DFA       `json:"lexer"`
	Grammar    *parse.Grammar `json:"grammar"`
	ParseTable *parse.Table   `json:"parseTable"`
}

// Manifest records high-level C++ generation metadata.
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
	Constant string `json:"cppConstant,omitempty"`
}

// Generate writes the C++ scanner, parser, headers, manifest, and table dump.
func Generate(input Input, outDir string) error {
	if input.Spec == nil || input.DFA == nil || input.Grammar == nil || input.ParseTable == nil {
		return errors.New("cpp codegen input is incomplete")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	namespace, err := cppNamespace(input.Spec.Package, filepath.Base(outDir))
	if err != nil {
		return err
	}
	tokens := tokenNames(input)
	tokenIDs := tokenIdentifiers(tokens)
	actionManifest := action.Build(input.Grammar, input.Spec.Semantics, "cpp")
	actions := semanticActionsFromNames(actionManifest.Names())
	manifest := Manifest{
		Tool:         version.Name,
		Version:      version.Version,
		Commit:       version.Commit,
		BuildDate:    version.BuildDate,
		Branch:       version.Branch,
		Source:       input.Spec.SourceFile,
		Target:       "cpp",
		Namespace:    strings.Join(namespace, "::"),
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
	files := map[string]string{
		"tokens.hpp":       renderTokensHeader(namespace, input.Spec.SourceFile, tokens, tokenIDs),
		"scanner.hpp":      renderScannerHeader(namespace, input.Spec.SourceFile),
		"scanner.cpp":      renderScannerSource(namespace, input.Spec.SourceFile, input.DFA, tokens, tokenIDs),
		"parser.hpp":       renderParserHeader(namespace, input.Spec.SourceFile, actions),
		"parser_typed.hpp": renderTypedParserHeader(namespace, input.Spec.SourceFile, actionManifest, actions),
		"parser.cpp":       renderParserSource(namespace, input.Spec.SourceFile, input.Spec, input.ParseTable, tokens, tokenIDs, actions),
	}
	for name, content := range files {
		if err := writeFile(filepath.Join(outDir, name), content); err != nil {
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
	for token := range seen {
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func renderTokensHeader(namespace []string, source string, tokens []string, tokenIDs map[string]string) string {
	var b strings.Builder
	b.WriteString(generatedHeader(source, "tokens.hpp"))
	b.WriteString("#pragma once\n\n")
	b.WriteString("#include <array>\n")
	b.WriteString("#include <cstddef>\n")
	b.WriteString("#include <string_view>\n\n")
	b.WriteString(openNamespace(namespace))
	b.WriteString("/// Identifies one terminal emitted by the generated scanner.\n")
	b.WriteString("enum class Token : int {\n")
	b.WriteString("    End = 0,\n")
	b.WriteString("    Error = 1,\n")
	for i, token := range tokens {
		b.WriteString(fmt.Sprintf("    %s = %d,\n", tokenIDs[token], i+2))
	}
	b.WriteString("};\n\n")
	b.WriteString("/// Returns the grammar spelling for a scanner token.\n")
	b.WriteString("inline std::string_view token_name(Token token) noexcept {\n")
	b.WriteString(fmt.Sprintf("    static constexpr std::array<std::string_view, %d> names = {{\n", len(tokens)+2))
	b.WriteString("        \"EOF\",\n")
	b.WriteString("        \"ERROR\",\n")
	for _, token := range tokens {
		b.WriteString("        " + cppString(token) + ",\n")
	}
	b.WriteString("    }};\n")
	b.WriteString("    const auto index = static_cast<std::size_t>(token);\n")
	b.WriteString("    return index < names.size() ? names[index] : std::string_view{\"UNKNOWN\"};\n")
	b.WriteString("}\n\n")
	b.WriteString(closeNamespace(namespace))
	return b.String()
}

func renderScannerHeader(namespace []string, source string) string {
	var b strings.Builder
	b.WriteString(generatedHeader(source, "scanner.hpp"))
	b.WriteString("#pragma once\n\n")
	b.WriteString("#include \"tokens.hpp\"\n\n")
	b.WriteString("#include <cstddef>\n")
	b.WriteString("#include <mutex>\n")
	b.WriteString("#include <string_view>\n")
	b.WriteString("#include <vector>\n\n")
	b.WriteString(openNamespace(namespace))
	b.WriteString("/// One scanner result with byte offsets and Unicode scalar line/column positions.\n")
	b.WriteString("struct Lexeme {\n")
	b.WriteString("    Token token = Token::Error;\n")
	b.WriteString("    std::string_view text;\n")
	b.WriteString("    std::string_view channel;\n")
	b.WriteString("    std::size_t start = 0;\n")
	b.WriteString("    std::size_t end = 0;\n")
	b.WriteString("    int start_line = 1;\n")
	b.WriteString("    int start_column = 1;\n")
	b.WriteString("    int end_line = 1;\n")
	b.WriteString("    int end_column = 1;\n")
	b.WriteString("};\n\n")
	b.WriteString("/// Synchronous pull source consumed by generated parsers.\n")
	b.WriteString("///\n")
	b.WriteString("/// Returning false means natural end-of-input. A source may also return one\n")
	b.WriteString("/// explicit Token::End lexeme; later tokens are rejected by the parser.\n")
	b.WriteString("class LexemeSource {\n")
	b.WriteString("public:\n")
	b.WriteString("    virtual ~LexemeSource() = default;\n")
	b.WriteString("    virtual bool next(Lexeme& lexeme) = 0;\n")
	b.WriteString("};\n\n")
	b.WriteString("/// Incrementally tokenizes UTF-8 source text.\n")
	b.WriteString("///\n")
	b.WriteString("/// The scanner stores a string_view into caller-owned input. Keep the input\n")
	b.WriteString("/// string alive while lexemes are being read. Calls that mutate scanner cursor\n")
	b.WriteString("/// state are serialized, so a shared Scanner can be consumed concurrently.\n")
	b.WriteString("class Scanner : public LexemeSource {\n")
	b.WriteString("public:\n")
	b.WriteString("    explicit Scanner(std::string_view input);\n")
	b.WriteString("    void include_hidden(bool include);\n")
	b.WriteString("    bool next(Lexeme& lexeme) override;\n")
	b.WriteString("    std::vector<Lexeme> all();\n\n")
	b.WriteString("private:\n")
	b.WriteString("    std::mutex gate_;\n")
	b.WriteString("    std::string_view input_;\n")
	b.WriteString("    std::size_t pos_ = 0;\n")
	b.WriteString("    int line_ = 1;\n")
	b.WriteString("    int column_ = 1;\n")
	b.WriteString("    bool include_hidden_ = false;\n")
	b.WriteString("};\n\n")
	b.WriteString("/// Tokenizes every visible token in UTF-8 source text.\n")
	b.WriteString("std::vector<Lexeme> tokenize(std::string_view input);\n\n")
	b.WriteString(closeNamespace(namespace))
	return b.String()
}

func renderScannerSource(namespace []string, source string, dfa *lex.DFA, tokens []string, tokenIDs map[string]string) string {
	tokenSet := map[string]bool{}
	for _, token := range tokens {
		tokenSet[token] = true
	}
	var transitions []string
	stateStart := make([]int, len(dfa.States))
	stateCount := make([]int, len(dfa.States))
	for _, state := range dfa.States {
		stateStart[state.ID] = len(transitions)
		for _, tr := range state.Transitions {
			for _, rr := range tr.Set.Normalize() {
				transitions = append(transitions, fmt.Sprintf("    {%d, %d, %d},\n", rr.Lo, rr.Hi, tr.Target))
				stateCount[state.ID]++
			}
		}
	}
	maxRule := 0
	for _, rule := range dfa.Rules {
		if rule.Index > maxRule {
			maxRule = rule.Index
		}
	}
	rules := make([]lex.Rule, maxRule+1)
	for _, rule := range dfa.Rules {
		rules[rule.Index] = rule
	}

	var b strings.Builder
	b.WriteString(generatedHeader(source, "scanner.cpp"))
	b.WriteString("#include \"scanner.hpp\"\n\n")
	b.WriteString("#include <algorithm>\n")
	b.WriteString("#include <array>\n")
	b.WriteString("#include <cstdint>\n")
	b.WriteString("#include <stdexcept>\n")
	b.WriteString("#include <string>\n")
	b.WriteString("#include <utility>\n\n")
	b.WriteString(openNamespace(namespace))
	b.WriteString("namespace {\n\n")
	b.WriteString("struct ScannerTransition { std::uint32_t lo; std::uint32_t hi; int target; };\n")
	b.WriteString("struct ScannerState { int accept; std::size_t start; std::size_t count; };\n")
	b.WriteString("struct RuleAction { Token token; bool skip; std::string_view channel; };\n")
	b.WriteString("struct MatchResult { int rule; std::size_t end; };\n")
	b.WriteString("struct DecodedRune { std::uint32_t value; std::size_t length; };\n")
	b.WriteString("struct Position { int line; int column; };\n\n")
	b.WriteString(fmt.Sprintf("static constexpr std::array<ScannerTransition, %d> scanner_transitions = {{\n", max(1, len(transitions))))
	if len(transitions) == 0 {
		b.WriteString("    {0, 0, 0},\n")
	} else {
		for _, tr := range transitions {
			b.WriteString(tr)
		}
	}
	b.WriteString("}};\n\n")
	b.WriteString(fmt.Sprintf("static constexpr std::array<ScannerState, %d> scanner_states = {{\n", max(1, len(dfa.States))))
	if len(dfa.States) == 0 {
		b.WriteString("    {0, 0, 0},\n")
	} else {
		for _, state := range dfa.States {
			b.WriteString(fmt.Sprintf("    {%d, %d, %d},\n", state.AcceptRule, stateStart[state.ID], stateCount[state.ID]))
		}
	}
	b.WriteString("}};\n\n")
	b.WriteString(fmt.Sprintf("static constexpr std::array<RuleAction, %d> rule_actions = {{\n", max(1, maxRule+1)))
	b.WriteString("    {Token::Error, true, \"\"},\n")
	for i := 1; i <= maxRule; i++ {
		rule := rules[i]
		token := "Token::Error"
		if tokenSet[rule.Token] {
			token = "Token::" + tokenIDs[rule.Token]
		}
		if comment := sourceComment(rule.Span); comment != "" {
			b.WriteString("    " + comment + "\n")
		}
		b.WriteString(fmt.Sprintf("    {%s, %t, %s},\n", token, rule.Skip, cppString(rule.Channel)))
	}
	b.WriteString("}};\n\n")
	b.WriteString(scannerRuntime())
	b.WriteString("} // namespace\n\n")
	b.WriteString(`Scanner::Scanner(std::string_view input) : input_(input) {}

void Scanner::include_hidden(bool include) {
    std::lock_guard<std::mutex> lock(gate_);
    include_hidden_ = include;
}

bool Scanner::next(Lexeme& lexeme) {
    std::lock_guard<std::mutex> lock(gate_);
    while (pos_ < input_.size()) {
        const auto start = pos_;
        const auto start_line = line_;
        const auto start_column = column_;
        const auto match = match_at(input_, pos_);
        if (match.rule <= 0) {
            throw std::runtime_error("no lexical rule matched offset " + std::to_string(pos_) + " near '" + preview(input_, pos_) + "'");
        }
        if (match.end == pos_) {
            throw std::runtime_error("lexer rule " + std::to_string(match.rule) + " matched empty input at offset " + std::to_string(pos_));
        }
        const auto action = rule_actions.at(static_cast<std::size_t>(match.rule));
        const auto end_position = advance_position(input_, pos_, match.end, line_, column_);
        lexeme = Lexeme{action.token, input_.substr(start, match.end - start), action.channel, start, match.end, start_line, start_column, end_position.line, end_position.column};
        pos_ = match.end;
        line_ = end_position.line;
        column_ = end_position.column;
        if (action.skip) {
            continue;
        }
        if (!action.channel.empty() && !include_hidden_) {
            continue;
        }
        return true;
    }
    lexeme = Lexeme{Token::End, std::string_view{}, std::string_view{}, pos_, pos_, line_, column_, line_, column_};
    return false;
}

std::vector<Lexeme> Scanner::all() {
    std::vector<Lexeme> output;
    Lexeme lexeme;
    while (next(lexeme)) {
        output.push_back(lexeme);
    }
    return output;
}

std::vector<Lexeme> tokenize(std::string_view input) {
    return Scanner(input).all();
}
`)
	b.WriteString(closeNamespace(namespace))
	return b.String()
}

func scannerRuntime() string {
	return `DecodedRune decode_utf8(std::string_view input, std::size_t pos) {
    const auto b0 = static_cast<unsigned char>(input[pos]);
    if (b0 < 0x80) {
        return DecodedRune{b0, 1};
    }
    if ((b0 & 0xe0) == 0xc0 && pos + 1 < input.size()) {
        const auto b1 = static_cast<unsigned char>(input[pos + 1]);
        if ((b1 & 0xc0) == 0x80) {
            const auto value = (static_cast<std::uint32_t>(b0 & 0x1f) << 6) | static_cast<std::uint32_t>(b1 & 0x3f);
            if (value >= 0x80) {
                return DecodedRune{value, 2};
            }
        }
    }
    if ((b0 & 0xf0) == 0xe0 && pos + 2 < input.size()) {
        const auto b1 = static_cast<unsigned char>(input[pos + 1]);
        const auto b2 = static_cast<unsigned char>(input[pos + 2]);
        if ((b1 & 0xc0) == 0x80 && (b2 & 0xc0) == 0x80) {
            const auto value = (static_cast<std::uint32_t>(b0 & 0x0f) << 12) | (static_cast<std::uint32_t>(b1 & 0x3f) << 6) | static_cast<std::uint32_t>(b2 & 0x3f);
            if (value >= 0x800 && (value < 0xd800 || value > 0xdfff)) {
                return DecodedRune{value, 3};
            }
        }
    }
    if ((b0 & 0xf8) == 0xf0 && pos + 3 < input.size()) {
        const auto b1 = static_cast<unsigned char>(input[pos + 1]);
        const auto b2 = static_cast<unsigned char>(input[pos + 2]);
        const auto b3 = static_cast<unsigned char>(input[pos + 3]);
        if ((b1 & 0xc0) == 0x80 && (b2 & 0xc0) == 0x80 && (b3 & 0xc0) == 0x80) {
            const auto value = (static_cast<std::uint32_t>(b0 & 0x07) << 18) | (static_cast<std::uint32_t>(b1 & 0x3f) << 12) | (static_cast<std::uint32_t>(b2 & 0x3f) << 6) | static_cast<std::uint32_t>(b3 & 0x3f);
            if (value >= 0x10000 && value <= 0x10ffff) {
                return DecodedRune{value, 4};
            }
        }
    }
    throw std::runtime_error("invalid UTF-8 input at byte offset " + std::to_string(pos));
}

Position advance_position(std::string_view input, std::size_t start, std::size_t end, int line, int column) {
    for (auto pos = start; pos < end;) {
        const auto decoded = decode_utf8(input, pos);
        pos += decoded.length;
        if (decoded.value == '\n') {
            ++line;
            column = 1;
        } else {
            ++column;
        }
    }
    return Position{line, column};
}

MatchResult match_at(std::string_view input, std::size_t start) {
    int state = 0;
    int best_rule = scanner_states[static_cast<std::size_t>(state)].accept;
    auto best_end = start;
    for (auto pos = start; pos < input.size();) {
        DecodedRune decoded{};
        try {
            decoded = decode_utf8(input, pos);
        } catch (const std::runtime_error&) {
            if (best_rule > 0) {
                break;
            }
            throw;
        }
        int next = -1;
        const auto current = scanner_states[static_cast<std::size_t>(state)];
        for (std::size_t i = 0; i < current.count; ++i) {
            const auto transition = scanner_transitions[current.start + i];
            if (decoded.value >= transition.lo && decoded.value <= transition.hi) {
                next = transition.target;
                break;
            }
        }
        if (next < 0) {
            break;
        }
        pos += decoded.length;
        state = next;
        if (scanner_states[static_cast<std::size_t>(state)].accept > 0) {
            best_rule = scanner_states[static_cast<std::size_t>(state)].accept;
            best_end = pos;
        }
    }
    return MatchResult{best_rule, best_end};
}

std::string preview(std::string_view input, std::size_t pos) {
    const auto end = std::min(input.size(), pos + static_cast<std::size_t>(16));
    return std::string(input.substr(pos, end - pos));
}

`
}

func renderParserHeader(namespace []string, source string, actions []SemanticAction) string {
	var b strings.Builder
	b.WriteString(generatedHeader(source, "parser.hpp"))
	b.WriteString("#pragma once\n\n")
	b.WriteString("#include \"scanner.hpp\"\n\n")
	b.WriteString("#include <any>\n")
	b.WriteString("#include <cstddef>\n")
	b.WriteString("#include <functional>\n")
	b.WriteString("#include <initializer_list>\n")
	b.WriteString("#include <stdexcept>\n")
	b.WriteString("#include <string>\n")
	b.WriteString("#include <string_view>\n")
	b.WriteString("#include <unordered_map>\n")
	b.WriteString("#include <utility>\n")
	b.WriteString("#include <vector>\n\n")
	b.WriteString(openNamespace(namespace))
	b.WriteString("/// Identifies one generated semantic reduction hook.\n")
	b.WriteString("enum class SemanticAction : int {\n")
	b.WriteString("    None = 0,\n")
	for _, action := range actions {
		b.WriteString(fmt.Sprintf("    %s = %d,\n", action.Constant, action.ID))
	}
	b.WriteString("};\n\n")
	b.WriteString("/// Runtime semantic value carried on the generated parser stack.\n")
	b.WriteString("using Value = std::any;\n\n")
	b.WriteString("/// One expected terminal or reporting group.\n")
	b.WriteString("struct ExpectedToken {\n    std::string_view symbol;\n    std::string_view display;\n    std::vector<std::string_view> members;\n};\n\n")
	b.WriteString("/// Describes how one syntax error was recovered.\n")
	b.WriteString("struct RecoveryAction {\n    std::string_view kind = \"none\";\n    std::size_t discarded = 0;\n};\n\n")
	b.WriteString("/// One source-rich syntax diagnostic.\n")
	b.WriteString("struct ParseDiagnostic {\n    int state = 0;\n    std::string_view unexpected;\n    std::string_view unexpected_display;\n    std::vector<ExpectedToken> expected;\n    std::size_t start = 0;\n    std::size_t end = 0;\n    int start_line = 1;\n    int start_column = 1;\n    int end_line = 1;\n    int end_column = 1;\n    RecoveryAction recovery;\n};\n\n")
	b.WriteString("/// A possibly partial parser value plus every syntax diagnostic.\n")
	b.WriteString("struct ParseResult {\n    Value value;\n    std::vector<ParseDiagnostic> diagnostics;\n    bool accepted = false;\n};\n\n")
	b.WriteString("/// Thrown by compatibility parse APIs when syntax diagnostics were produced.\n")
	b.WriteString("class ParseError : public std::runtime_error {\npublic:\n    explicit ParseError(std::vector<ParseDiagnostic> diagnostics);\n    const std::vector<ParseDiagnostic>& diagnostics() const noexcept;\nprivate:\n    std::vector<ParseDiagnostic> diagnostics_;\n};\n\n")
	b.WriteString("/// Describes one grammar rule reduction passed to handwritten semantics.\n")
	b.WriteString("struct Reduction {\n")
	b.WriteString("    int rule = 0;\n")
	b.WriteString("    std::string_view lhs;\n")
	b.WriteString("    std::vector<std::string_view> rhs;\n")
	b.WriteString("    std::vector<std::string_view> labels;\n")
	b.WriteString("    SemanticAction action_id = SemanticAction::None;\n")
	b.WriteString("    std::string_view action;\n")
	b.WriteString("    std::vector<Value> values;\n")
	b.WriteString("    const Value& value_for(std::string_view label) const;\n")
	b.WriteString("};\n\n")
	b.WriteString("/// Receives target-tagged action hooks during parser reductions.\n")
	b.WriteString("using Reducer = std::function<Value(const Reduction&)>;\n")
	b.WriteString("using ReductionHandler = std::function<Value(const Reduction&)>;\n\n")
	b.WriteString("/// Hashes generated semantic action IDs for reducer maps.\n")
	b.WriteString("struct SemanticActionHash {\n")
	b.WriteString("    std::size_t operator()(SemanticAction action) const noexcept;\n")
	b.WriteString("};\n\n")
	b.WriteString("using ReducerTable = std::unordered_map<SemanticAction, ReductionHandler, SemanticActionHash>;\n\n")
	b.WriteString("/// Adapts an existing token vector to the pull source parser API.\n")
	b.WriteString("class VectorLexemeSource final : public LexemeSource {\n")
	b.WriteString("public:\n")
	b.WriteString("    explicit VectorLexemeSource(const std::vector<Lexeme>& tokens);\n")
	b.WriteString("    bool next(Lexeme& lexeme) override;\n\n")
	b.WriteString("private:\n")
	b.WriteString("    const std::vector<Lexeme>& tokens_;\n")
	b.WriteString("    std::size_t pos_ = 0;\n")
	b.WriteString("};\n\n")
	b.WriteString("/// Dispatches reductions by generated semantic action ID.\n")
	b.WriteString("class ReducerMap {\n")
	b.WriteString("public:\n")
	b.WriteString("    ReducerMap() = default;\n")
	b.WriteString("    explicit ReducerMap(ReducerTable handlers);\n")
	b.WriteString("    ReducerMap(std::initializer_list<std::pair<const SemanticAction, ReductionHandler>> handlers);\n")
	b.WriteString("    Value operator()(const Reduction& ctx) const;\n")
	b.WriteString("    void validate_coverage() const;\n")
	b.WriteString("    ReducerTable& handlers() noexcept;\n")
	b.WriteString("    const ReducerTable& handlers() const noexcept;\n\n")
	b.WriteString("private:\n")
	b.WriteString("    ReducerTable handlers_;\n")
	b.WriteString("};\n\n")
	b.WriteString("/// Generated table-driven parser.\n")
	b.WriteString("///\n")
	b.WriteString("/// Parser calls use local stacks, so a Parser instance can be shared by\n")
	b.WriteString("/// concurrent callers when the installed reducer is also safe to call.\n")
	b.WriteString("class Parser {\n")
	b.WriteString("public:\n")
	b.WriteString("    explicit Parser(Reducer reducer = Reducer{});\n")
	b.WriteString("    explicit Parser(const ReducerMap& reducer);\n")
	b.WriteString("    void parse(const std::vector<Lexeme>& tokens) const;\n")
	b.WriteString("    void parse(LexemeSource& source) const;\n")
	b.WriteString("    Value parse_value(const std::vector<Lexeme>& tokens) const;\n\n")
	b.WriteString("    Value parse_value(LexemeSource& source) const;\n\n")
	b.WriteString("    ParseResult parse_recovering(const std::vector<Lexeme>& tokens) const;\n")
	b.WriteString("    ParseResult parse_recovering(LexemeSource& source) const;\n\n")
	b.WriteString("private:\n")
	b.WriteString("    Reducer reducer_;\n")
	b.WriteString("};\n\n")
	b.WriteString("/// Returns the source action label for an action ID.\n")
	b.WriteString("std::string_view semantic_action_name(SemanticAction action) noexcept;\n")
	b.WriteString("/// Looks up a generated action ID from the source action label.\n")
	b.WriteString("bool lookup_semantic_action(std::string_view name, SemanticAction& action) noexcept;\n")
	b.WriteString("/// Recognizes a token stream without user semantics.\n")
	b.WriteString("void parse(const std::vector<Lexeme>& tokens);\n")
	b.WriteString("/// Recognizes tokens pulled from source without user semantics.\n")
	b.WriteString("void parse(LexemeSource& source);\n")
	b.WriteString("/// Parses with an explicit reducer and returns the final semantic value.\n")
	b.WriteString("Value parse_value(const std::vector<Lexeme>& tokens, Reducer reducer = Reducer{});\n\n")
	b.WriteString("/// Parses tokens pulled from source with an explicit reducer and returns the final semantic value.\n")
	b.WriteString("Value parse_value(LexemeSource& source, Reducer reducer = Reducer{});\n\n")
	b.WriteString("/// Parses with a checked reducer map and returns the final semantic value.\n")
	b.WriteString("Value parse_value(const std::vector<Lexeme>& tokens, const ReducerMap& reducer);\n\n")
	b.WriteString("/// Parses tokens pulled from source with a checked reducer map and returns the final semantic value.\n")
	b.WriteString("Value parse_value(LexemeSource& source, const ReducerMap& reducer);\n\n")
	b.WriteString("/// Parses with grammar-directed recovery and returns every syntax diagnostic.\n")
	b.WriteString("ParseResult parse_recovering(const std::vector<Lexeme>& tokens, Reducer reducer = Reducer{});\n\n")
	b.WriteString("/// Parses tokens pulled from source with grammar-directed recovery.\n")
	b.WriteString("ParseResult parse_recovering(LexemeSource& source, Reducer reducer = Reducer{});\n\n")
	b.WriteString("/// Parses with grammar-directed recovery and a checked reducer map.\n")
	b.WriteString("ParseResult parse_recovering(const std::vector<Lexeme>& tokens, const ReducerMap& reducer);\n\n")
	b.WriteString("/// Parses tokens pulled from source with grammar-directed recovery and a checked reducer map.\n")
	b.WriteString("ParseResult parse_recovering(LexemeSource& source, const ReducerMap& reducer);\n\n")
	b.WriteString(closeNamespace(namespace))
	return b.String()
}

func renderTypedParserHeader(namespace []string, source string, manifest action.Manifest, actions []SemanticAction) string {
	var b strings.Builder
	b.WriteString(generatedHeader(source, "parser_typed.hpp"))
	b.WriteString("#pragma once\n\n")
	b.WriteString("#include \"parser.hpp\"\n\n")
	b.WriteString("#include <any>\n")
	b.WriteString("#include <functional>\n")
	b.WriteString("#include <stdexcept>\n")
	b.WriteString("#include <string>\n")
	b.WriteString("#include <string_view>\n")
	b.WriteString("#include <typeinfo>\n")
	b.WriteString("#include <utility>\n\n")
	b.WriteString(openNamespace(namespace))
	typedActions := typedCppManifestActions(manifest, actions)
	if len(typedActions) == 0 {
		b.WriteString("// No consistently typed semantic actions were declared for this grammar.\n\n")
		b.WriteString(closeNamespace(namespace))
		return b.String()
	}
	constants := semanticActionIDs(actions)
	b.WriteString("namespace typed_detail {\n\n")
	b.WriteString("inline std::string typed_error_prefix(const Reduction& ctx, std::string_view label) {\n")
	b.WriteString("    return \"rule \" + std::to_string(ctx.rule) + \" action \" + std::string(ctx.action) + \" label \" + std::string(label);\n")
	b.WriteString("}\n\n")
	b.WriteString("template <typename T>\n")
	b.WriteString("T semantic_value_as(const Reduction& ctx, std::string_view label) {\n")
	b.WriteString("    const auto& value = ctx.value_for(label);\n")
	b.WriteString("    try {\n")
	b.WriteString("        return std::any_cast<T>(value);\n")
	b.WriteString("    } catch (const std::bad_any_cast&) {\n")
	b.WriteString("        throw std::runtime_error(typed_error_prefix(ctx, label) + \" has incompatible semantic value; expected \" + typeid(T).name());\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("template <typename T>\n")
	b.WriteString("// Converts a boxed reducer result back to the declared typed result.\n")
	b.WriteString("// When this adapter is used for std::nullptr_t actions, boxed reducers must\n")
	b.WriteString("// return nullptr. Returning Value{} or `{}` creates an empty std::any, which\n")
	b.WriteString("// boxed-only code may ignore but this typed adapter rejects because it means\n")
	b.WriteString("// missing value rather than an intentional no-op value.\n")
	b.WriteString("T boxed_value_as(Value value, const Reduction& ctx) {\n")
	b.WriteString("    try {\n")
	b.WriteString("        return std::any_cast<T>(std::move(value));\n")
	b.WriteString("    } catch (const std::bad_any_cast&) {\n")
	b.WriteString("        throw std::runtime_error(\"typed boxed adapter for action \" + std::string(ctx.action) + \" returned incompatible value; expected \" + typeid(T).name());\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("template <>\n")
	b.WriteString("inline Value boxed_value_as<Value>(Value value, const Reduction&) {\n")
	b.WriteString("    return value;\n")
	b.WriteString("}\n\n")
	b.WriteString("} // namespace typed_detail\n\n")
	for _, semanticAction := range typedActions {
		constant := constants[semanticAction.Name]
		suffix := strings.TrimPrefix(constant, "SemanticAction::")
		contextType := suffix + "Reduction"
		handlerType := suffix + "Handler"
		factoryName := "typed_" + cppFunctionSuffix(semanticAction.Name)
		fields := typedCppFields(semanticAction.Rules[0])

		b.WriteString("/// Typed context for semantic action " + cppString(semanticAction.Name) + ".\n")
		b.WriteString("struct " + contextType + " {\n")
		b.WriteString("    Reduction reduction;\n")
		for _, field := range fields {
			b.WriteString("    " + field.Type + " " + field.Name + ";\n")
		}
		b.WriteString("};\n\n")
		b.WriteString("/// Handles one typed " + cppString(semanticAction.Name) + " reduction.\n")
		b.WriteString("using " + handlerType + " = std::function<" + semanticAction.ReturnType + "(const " + contextType + "&)>;\n\n")
		b.WriteString("/// Adapts a typed " + cppString(semanticAction.Name) + " handler to the boxed reducer ABI.\n")
		b.WriteString("inline ReductionHandler " + factoryName + "(" + handlerType + " handler) {\n")
		b.WriteString("    return [handler = std::move(handler)](const Reduction& ctx) -> Value {\n")
		b.WriteString("        if (ctx.action_id != " + constant + ") { throw std::runtime_error(\"typed reducer " + factoryName + " received action \" + std::string(ctx.action)); }\n")
		b.WriteString("        " + contextType + " typed{ctx")
		for _, field := range fields {
			b.WriteString(", typed_detail::semantic_value_as<" + field.Type + ">(ctx, " + cppString(field.Label) + ")")
		}
		b.WriteString("};\n")
		b.WriteString("        return Value{handler(typed)};\n")
		b.WriteString("    };\n")
		b.WriteString("}\n\n")
	}
	b.WriteString("/// Builds typed reducers that validate generated contexts before delegating to a boxed handler.\n")
	b.WriteString("inline ReducerMap typed_reducer_map_from_boxed(ReductionHandler handler) {\n")
	b.WriteString("    ReducerMap reducers;\n")
	for _, semanticAction := range typedActions {
		constant := constants[semanticAction.Name]
		suffix := strings.TrimPrefix(constant, "SemanticAction::")
		contextType := suffix + "Reduction"
		factoryName := "typed_" + cppFunctionSuffix(semanticAction.Name)
		b.WriteString("    reducers.handlers().emplace(" + constant + ", " + factoryName + "([handler](const " + contextType + "& ctx) -> " + semanticAction.ReturnType + " {\n")
		b.WriteString("        return typed_detail::boxed_value_as<" + semanticAction.ReturnType + ">(handler(ctx.reduction), ctx.reduction);\n")
		b.WriteString("    }));\n")
	}
	b.WriteString("    reducers.validate_coverage();\n")
	b.WriteString("    return reducers;\n")
	b.WriteString("}\n\n")
	b.WriteString("/// Builds typed reducers that validate generated contexts before delegating to a boxed reducer map.\n")
	b.WriteString("inline ReducerMap typed_reducer_map_from_boxed(const ReducerMap& boxed) {\n")
	b.WriteString("    boxed.validate_coverage();\n")
	b.WriteString("    return typed_reducer_map_from_boxed([boxed](const Reduction& ctx) -> Value { return boxed(ctx); });\n")
	b.WriteString("}\n\n")
	b.WriteString(closeNamespace(namespace))
	return b.String()
}

func renderParserSource(namespace []string, source string, project *spec.Spec, table *parse.Table, tokens []string, tokenIDs map[string]string, actions []SemanticAction) string {
	actionIDs := semanticActionIDs(actions)
	var b strings.Builder
	b.WriteString(generatedHeader(source, "parser.cpp"))
	b.WriteString("#include \"parser.hpp\"\n\n")
	b.WriteString("#include <algorithm>\n")
	b.WriteString("#include <array>\n")
	b.WriteString("#include <stdexcept>\n")
	b.WriteString("#include <string>\n")
	b.WriteString("#include <utility>\n\n")
	b.WriteString(openNamespace(namespace))
	b.WriteString("namespace {\n\n")
	b.WriteString("enum class ParserActionKind { Shift, Reduce, Accept };\n")
	b.WriteString("struct ParserActionEntry { std::string_view symbol; ParserActionKind kind; int state; int rule; };\n")
	b.WriteString("struct ParserGotoEntry { std::string_view symbol; int state; };\n")
	b.WriteString("struct ParserRow { std::size_t start; std::size_t count; };\n")
	b.WriteString("struct ParserRule { int id; std::string_view lhs; const std::string_view* rhs; std::size_t rhs_count; const std::string_view* labels; std::size_t label_count; SemanticAction action; };\n")
	b.WriteString("struct ParserExpectedEntry { int state; std::string_view symbol; std::string_view display; std::size_t member_start; std::size_t member_count; };\n")
	b.WriteString("struct ParserAliasEntry { std::string_view symbol; std::string_view display; };\n")
	b.WriteString("struct SemanticActionLookup { std::string_view name; SemanticAction action; };\n\n")
	b.WriteString(renderSemanticNames(actions))
	b.WriteString(renderParserRules(table, actionIDs))
	b.WriteString(renderParserActions(table))
	b.WriteString(renderParserGotos(table))
	b.WriteString(renderParserExpected(project, table))
	b.WriteString(parserRuntime())
	b.WriteString("} // namespace\n\n")
	b.WriteString(renderSemanticImplementation(actions))
	b.WriteString(renderParserImplementation())
	b.WriteString(closeNamespace(namespace))
	return b.String()
}

func renderSemanticNames(actions []SemanticAction) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("static constexpr std::array<std::string_view, %d> semantic_action_names = {{\n", len(actions)+1))
	b.WriteString("    \"\",\n")
	for _, action := range actions {
		b.WriteString("    " + cppString(action.Name) + ",\n")
	}
	b.WriteString("}};\n\n")
	lookup := append([]SemanticAction{}, actions...)
	sort.SliceStable(lookup, func(i, j int) bool { return lookup[i].Name < lookup[j].Name })
	b.WriteString(fmt.Sprintf("[[maybe_unused]] static constexpr std::array<SemanticActionLookup, %d> semantic_action_lookup = {{\n", max(1, len(lookup))))
	if len(lookup) == 0 {
		b.WriteString("    {\"\", SemanticAction::None},\n")
	} else {
		for _, action := range lookup {
			b.WriteString(fmt.Sprintf("    {%s, SemanticAction::%s},\n", cppString(action.Name), action.Constant))
		}
	}
	b.WriteString("}};\n\n")
	return b.String()
}

func renderParserRules(table *parse.Table, actionIDs map[string]string) string {
	maxRule := 0
	for _, rule := range table.Rules {
		if rule.ID > maxRule {
			maxRule = rule.ID
		}
	}
	rules := make([]parse.Rule, maxRule+1)
	present := make([]bool, maxRule+1)
	for _, rule := range table.Rules {
		rules[rule.ID] = rule
		present[rule.ID] = true
	}
	var b strings.Builder
	for i, rule := range rules {
		size := len(rule.RHS)
		if !present[i] {
			size = 0
		}
		if present[i] {
			b.WriteString(ruleSourceComment(rule, "cpp", "// "))
		}
		b.WriteString(fmt.Sprintf("static constexpr std::array<std::string_view, %d> parser_rule_%d_rhs = {{", size, i))
		for _, sym := range rule.RHS {
			b.WriteString(cppString(sym) + ", ")
		}
		b.WriteString("}};\n")
		b.WriteString(fmt.Sprintf("static constexpr std::array<std::string_view, %d> parser_rule_%d_labels = {{", size, i))
		for _, label := range rule.Labels {
			b.WriteString(cppString(label) + ", ")
		}
		for missing := len(rule.Labels); missing < size; missing++ {
			b.WriteString("\"\", ")
		}
		b.WriteString("}};\n")
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("static constexpr std::array<ParserRule, %d> parser_rules = {{\n", max(1, maxRule+1)))
	for i, rule := range rules {
		if !present[i] {
			b.WriteString(fmt.Sprintf("    {-1, \"\", parser_rule_%d_rhs.data(), parser_rule_%d_rhs.size(), parser_rule_%d_labels.data(), parser_rule_%d_labels.size(), SemanticAction::None},\n", i, i, i, i))
			continue
		}
		action := "SemanticAction::None"
		if id, ok := actionIDs[strings.TrimSpace(rule.Actions["cpp"])]; ok {
			action = id
		}
		b.WriteString(indentComment(ruleSourceComment(rule, "cpp", "// "), "    "))
		b.WriteString(fmt.Sprintf("    {%d, %s, parser_rule_%d_rhs.data(), parser_rule_%d_rhs.size(), parser_rule_%d_labels.data(), parser_rule_%d_labels.size(), %s},\n", rule.ID, cppString(rule.LHS), i, i, i, i, action))
	}
	if len(rules) == 0 {
		b.WriteString("    {-1, \"\", nullptr, 0, nullptr, 0, SemanticAction::None},\n")
	}
	b.WriteString("}};\n\n")
	return b.String()
}

func renderParserActions(table *parse.Table) string {
	var entries []string
	entryCount := 0
	rowStart := make([]int, len(table.States))
	rowCount := make([]int, len(table.States))
	for _, state := range table.States {
		actions := table.Actions[state.ID]
		var symbols []string
		for sym := range actions {
			symbols = append(symbols, sym)
		}
		sort.Strings(symbols)
		rowStart[state.ID] = entryCount
		for _, sym := range symbols {
			action := actions[sym]
			kind := "ParserActionKind::Accept"
			if action.Kind == parse.ActionShift {
				kind = "ParserActionKind::Shift"
			} else if action.Kind == parse.ActionReduce {
				kind = "ParserActionKind::Reduce"
				if rule, ok := ruleByID(table.Rules, action.Rule); ok {
					entries = append(entries, indentComment(ruleSourceComment(rule, "cpp", "// "), "    "))
				}
			}
			entries = append(entries, fmt.Sprintf("    {%s, %s, %d, %d},\n", cppString(sym), kind, action.State, action.Rule))
			entryCount++
			rowCount[state.ID]++
		}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("static constexpr std::array<ParserActionEntry, %d> parser_actions = {{\n", max(1, len(entries))))
	if len(entries) == 0 {
		b.WriteString("    {\"\", ParserActionKind::Accept, 0, 0},\n")
	} else {
		for _, entry := range entries {
			b.WriteString(entry)
		}
	}
	b.WriteString("}};\n\n")
	b.WriteString(fmt.Sprintf("static constexpr std::array<ParserRow, %d> parser_action_rows = {{\n", max(1, len(table.States))))
	if len(table.States) == 0 {
		b.WriteString("    {0, 0},\n")
	} else {
		for _, state := range table.States {
			b.WriteString(fmt.Sprintf("    {%d, %d},\n", rowStart[state.ID], rowCount[state.ID]))
		}
	}
	b.WriteString("}};\n\n")
	return b.String()
}

func renderParserGotos(table *parse.Table) string {
	var entries []string
	rowStart := make([]int, len(table.States))
	rowCount := make([]int, len(table.States))
	for _, state := range table.States {
		gotos := table.Gotos[state.ID]
		var symbols []string
		for sym := range gotos {
			symbols = append(symbols, sym)
		}
		sort.Strings(symbols)
		rowStart[state.ID] = len(entries)
		for _, sym := range symbols {
			entries = append(entries, fmt.Sprintf("    {%s, %d},\n", cppString(sym), gotos[sym]))
			rowCount[state.ID]++
		}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("static constexpr std::array<ParserGotoEntry, %d> parser_gotos = {{\n", max(1, len(entries))))
	if len(entries) == 0 {
		b.WriteString("    {\"\", 0},\n")
	} else {
		for _, entry := range entries {
			b.WriteString(entry)
		}
	}
	b.WriteString("}};\n\n")
	b.WriteString(fmt.Sprintf("static constexpr std::array<ParserRow, %d> parser_goto_rows = {{\n", max(1, len(table.States))))
	if len(table.States) == 0 {
		b.WriteString("    {0, 0},\n")
	} else {
		for _, state := range table.States {
			b.WriteString(fmt.Sprintf("    {%d, %d},\n", rowStart[state.ID], rowCount[state.ID]))
		}
	}
	b.WriteString("}};\n\n")
	return b.String()
}

func renderParserExpected(project *spec.Spec, table *parse.Table) string {
	var members []string
	var entries []string
	for _, state := range table.States {
		for _, expected := range table.Expected[state.ID] {
			start := len(members)
			for _, member := range expected.Members {
				members = append(members, member)
			}
			entries = append(entries, fmt.Sprintf("    {%d, %s, %s, %d, %d},\n", state.ID, cppString(expected.Symbol), cppString(expected.Display), start, len(expected.Members)))
		}
	}
	aliases := append([]spec.ExpectedTokenAlias(nil), project.Grammar.ExpectedTokens.Aliases...)
	sort.SliceStable(aliases, func(i, j int) bool { return aliases[i].Token < aliases[j].Token })

	var b strings.Builder
	b.WriteString(fmt.Sprintf("static constexpr std::array<std::string_view, %d> parser_expected_members = {{\n", max(1, len(members))))
	if len(members) == 0 {
		b.WriteString("    \"\",\n")
	} else {
		for _, member := range members {
			b.WriteString("    " + cppString(member) + ",\n")
		}
	}
	b.WriteString("}};\n\n")
	b.WriteString(fmt.Sprintf("static constexpr std::array<ParserExpectedEntry, %d> parser_expected = {{\n", max(1, len(entries))))
	if len(entries) == 0 {
		b.WriteString("    {-1, \"\", \"\", 0, 0},\n")
	} else {
		for _, entry := range entries {
			b.WriteString(entry)
		}
	}
	b.WriteString("}};\n\n")
	b.WriteString(fmt.Sprintf("static constexpr std::array<ParserAliasEntry, %d> parser_aliases = {{\n", max(1, len(aliases))))
	if len(aliases) == 0 {
		b.WriteString("    {\"\", \"\"},\n")
	} else {
		for _, alias := range aliases {
			b.WriteString(fmt.Sprintf("    {%s, %s},\n", cppString(alias.Token), cppString(alias.Label)))
		}
	}
	b.WriteString("}};\n\n")
	return b.String()
}

func parserRuntime() string {
	return `std::string_view terminal_name(Token token) noexcept {
    if (token == Token::End) {
        return "$";
    }
    return token_name(token);
}

class SourceCursor {
public:
    explicit SourceCursor(LexemeSource& source) : source_(source) {}

    std::string_view peek_symbol() {
        if (ready_) {
            return symbol_;
        }
        if (saw_eof_) {
            lookahead_ = eof_lexeme();
            symbol_ = "$";
            ready_ = true;
            return symbol_;
        }
        Lexeme lexeme{};
        if (!source_.next(lexeme)) {
            lookahead_ = eof_lexeme();
            symbol_ = "$";
            ready_ = true;
            saw_eof_ = true;
            return symbol_;
        }
        if (lexeme.token == Token::End) {
            lexeme = normalize_eof(lexeme);
            Lexeme extra{};
            if (source_.next(extra)) {
                throw std::runtime_error("token after EOF in lexeme source");
            }
            lookahead_ = lexeme;
            symbol_ = "$";
            ready_ = true;
            saw_eof_ = true;
            return symbol_;
        }
        last_ = lexeme;
        have_last_ = true;
        lookahead_ = lexeme;
        symbol_ = terminal_name(lexeme.token);
        ready_ = true;
        return symbol_;
    }

    Lexeme advance() {
        const auto symbol = peek_symbol();
        if (symbol == "$") {
            throw std::runtime_error("shift past end of input");
        }
        auto lexeme = lookahead_;
        ready_ = false;
        return lexeme;
    }

    Lexeme diagnostic_lexeme() const {
        return ready_ ? lookahead_ : eof_lexeme();
    }

private:
    Lexeme eof_lexeme() const {
        if (have_last_) {
            return Lexeme{Token::End, {}, {}, last_.end, last_.end, last_.end_line, last_.end_column, last_.end_line, last_.end_column};
        }
        return Lexeme{Token::End, {}, {}, 0, 0, 1, 1, 1, 1};
    }

    Lexeme normalize_eof(Lexeme lexeme) const {
        const auto fallback = eof_lexeme();
        if (lexeme.start_line <= 0) { lexeme.start_line = fallback.start_line; }
        if (lexeme.start_column <= 0) { lexeme.start_column = fallback.start_column; }
        if (lexeme.end_line <= 0) { lexeme.end_line = fallback.end_line; }
        if (lexeme.end_column <= 0) { lexeme.end_column = fallback.end_column; }
        if (lexeme.start == 0 && lexeme.end == 0 && have_last_) {
            lexeme.start = fallback.start;
            lexeme.end = fallback.end;
        }
        return lexeme;
    }

    LexemeSource& source_;
    Lexeme lookahead_{};
    Lexeme last_{};
    std::string_view symbol_ = "$";
    bool ready_ = false;
    bool saw_eof_ = false;
    bool have_last_ = false;
};

const ParserActionEntry* find_action(int state, std::string_view symbol) {
    if (state < 0 || static_cast<std::size_t>(state) >= parser_action_rows.size()) {
        return nullptr;
    }
    const auto row = parser_action_rows[static_cast<std::size_t>(state)];
    const auto first = parser_actions.begin() + static_cast<std::ptrdiff_t>(row.start);
    const auto last = first + static_cast<std::ptrdiff_t>(row.count);
    const auto it = std::lower_bound(first, last, symbol, [](const ParserActionEntry& entry, std::string_view value) {
        return entry.symbol < value;
    });
    return it != last && it->symbol == symbol ? &*it : nullptr;
}

bool find_goto(int state, std::string_view symbol, int& out) {
    if (state < 0 || static_cast<std::size_t>(state) >= parser_goto_rows.size()) {
        return false;
    }
    const auto row = parser_goto_rows[static_cast<std::size_t>(state)];
    const auto first = parser_gotos.begin() + static_cast<std::ptrdiff_t>(row.start);
    const auto last = first + static_cast<std::ptrdiff_t>(row.count);
    const auto it = std::lower_bound(first, last, symbol, [](const ParserGotoEntry& entry, std::string_view value) {
        return entry.symbol < value;
    });
    if (it == last || it->symbol != symbol) {
        return false;
    }
    out = it->state;
    return true;
}

Value default_reduce(const std::vector<Value>& values) {
    if (values.empty()) {
        return Value{};
    }
    if (values.size() == 1) {
        return values[0];
    }
    return values;
}

std::vector<std::string_view> rhs_symbols(const ParserRule& rule) {
    if (rule.rhs_count == 0) {
        return {};
    }
    return std::vector<std::string_view>(rule.rhs, rule.rhs + rule.rhs_count);
}

std::vector<std::string_view> label_symbols(const ParserRule& rule) {
    if (rule.label_count == 0) {
        return {};
    }
    return std::vector<std::string_view>(rule.labels, rule.labels + rule.label_count);
}

std::string_view unexpected_display(std::string_view symbol) {
    if (symbol == "$") {
        return "end of input";
    }
    const auto it = std::lower_bound(parser_aliases.begin(), parser_aliases.end(), symbol, [](const ParserAliasEntry& entry, std::string_view value) {
        return entry.symbol < value;
    });
    return it != parser_aliases.end() && it->symbol == symbol ? it->display : symbol;
}

std::vector<ExpectedToken> expected_tokens(int state) {
    std::vector<ExpectedToken> out;
    for (const auto& entry : parser_expected) {
        if (entry.state != state) {
            continue;
        }
        ExpectedToken expected{entry.symbol, entry.display, {}};
        expected.members.reserve(entry.member_count);
        for (std::size_t i = 0; i < entry.member_count; ++i) {
            expected.members.push_back(parser_expected_members[entry.member_start + i]);
        }
        out.push_back(std::move(expected));
    }
    return out;
}

Lexeme error_lexeme(const SourceCursor& cursor) {
    const auto source = cursor.diagnostic_lexeme();
    return Lexeme{Token::Error, {}, {}, source.start, source.start, source.start_line, source.start_column, source.start_line, source.start_column};
}

ParseDiagnostic make_diagnostic(int state, std::string_view unexpected, const SourceCursor& cursor) {
    const auto source = cursor.diagnostic_lexeme();
    return ParseDiagnostic{
        state,
        unexpected,
        unexpected_display(unexpected),
        expected_tokens(state),
        source.start,
        source.end,
        source.start_line,
        source.start_column,
        source.end_line,
        source.end_column,
        RecoveryAction{},
    };
}

Value current_value(const std::vector<Value>& values) {
    return values.empty() ? Value{} : values.back();
}

std::string format_parse_error(const std::vector<ParseDiagnostic>& diagnostics) {
    if (diagnostics.empty()) {
        return "parse error";
    }
    const auto& first = diagnostics.front();
    std::string message = "parse error at " + std::to_string(first.start_line) + ":" + std::to_string(first.start_column) + ": unexpected " + std::string(first.unexpected_display);
    if (!first.expected.empty()) {
        message += "; expected ";
        for (std::size_t i = 0; i < first.expected.size(); ++i) {
            if (i != 0) {
                message += ", ";
            }
            message += first.expected[i].display;
        }
    }
    if (diagnostics.size() > 1) {
        message += " (" + std::to_string(diagnostics.size()) + " diagnostics)";
    }
    return message;
}

`
}

func renderSemanticImplementation(actions []SemanticAction) string {
	var b strings.Builder
	b.WriteString(`std::size_t SemanticActionHash::operator()(SemanticAction action) const noexcept {
    return static_cast<std::size_t>(action);
}

const Value& Reduction::value_for(std::string_view label) const {
    for (std::size_t index = 0; index < labels.size(); ++index) {
        if (labels[index] == label) {
            if (index >= values.size()) {
                throw std::runtime_error("rule " + std::to_string(rule) + " action " + std::string(action) + " label " + std::string(label) + " has no semantic value");
            }
            return values[index];
        }
    }
    throw std::runtime_error("rule " + std::to_string(rule) + " action " + std::string(action) + " has no RHS label " + std::string(label));
}

ReducerMap::ReducerMap(ReducerTable handlers) : handlers_(std::move(handlers)) {}

ReducerMap::ReducerMap(std::initializer_list<std::pair<const SemanticAction, ReductionHandler>> handlers) : handlers_(handlers) {}

Value ReducerMap::operator()(const Reduction& ctx) const {
    const auto it = handlers_.find(ctx.action_id);
    if (it == handlers_.end()) {
        throw std::runtime_error("no reducer registered for action " + std::string(semantic_action_name(ctx.action_id)));
    }
    return it->second(ctx);
}

void ReducerMap::validate_coverage() const {
`)
	if len(actions) == 0 {
		b.WriteString(`}

`)
	} else {
		b.WriteString(fmt.Sprintf("    static constexpr std::array<SemanticAction, %d> required = {{\n", len(actions)))
		for _, action := range actions {
			b.WriteString("        SemanticAction::" + action.Constant + ",\n")
		}
		b.WriteString("    }};\n")
		b.WriteString(`    for (const auto& handler : handlers_) {
        bool known = false;
        for (const auto action : required) {
            if (handler.first == action) {
                known = true;
                break;
            }
        }
        if (!known) {
            throw std::runtime_error("unknown reducer registered for action " + std::to_string(static_cast<int>(handler.first)));
        }
    }
    for (const auto action : required) {
        if (handlers_.find(action) == handlers_.end()) {
            throw std::runtime_error("no reducer registered for action " + std::string(semantic_action_name(action)));
        }
    }
}

`)
	}
	b.WriteString(`
ReducerTable& ReducerMap::handlers() noexcept {
    return handlers_;
}

const ReducerTable& ReducerMap::handlers() const noexcept {
    return handlers_;
}

std::string_view semantic_action_name(SemanticAction action) noexcept {
    const auto index = static_cast<std::size_t>(action);
    return index < semantic_action_names.size() ? semantic_action_names[index] : std::string_view{"UNKNOWN"};
}

bool lookup_semantic_action(std::string_view name, SemanticAction& action) noexcept {
`)
	if len(actions) == 0 {
		b.WriteString(`    action = SemanticAction::None;
    return name.empty();
}

`)
		return b.String()
	}
	b.WriteString(`    const auto it = std::lower_bound(semantic_action_lookup.begin(), semantic_action_lookup.end(), name, [](const SemanticActionLookup& entry, std::string_view value) {
        return entry.name < value;
    });
    if (it == semantic_action_lookup.end() || it->name != name) {
        action = SemanticAction::None;
        return false;
    }
    action = it->action;
    return true;
}

`)
	return b.String()
}

func renderParserImplementation() string {
	return `ParseError::ParseError(std::vector<ParseDiagnostic> diagnostics)
    : std::runtime_error(format_parse_error(diagnostics)), diagnostics_(std::move(diagnostics)) {}

const std::vector<ParseDiagnostic>& ParseError::diagnostics() const noexcept {
    return diagnostics_;
}

Parser::Parser(Reducer reducer) : reducer_(std::move(reducer)) {}

Parser::Parser(const ReducerMap& reducer) : reducer_([reducer](const Reduction& ctx) { return reducer(ctx); }) {
    reducer.validate_coverage();
}

VectorLexemeSource::VectorLexemeSource(const std::vector<Lexeme>& tokens) : tokens_(tokens) {}

bool VectorLexemeSource::next(Lexeme& lexeme) {
    if (pos_ >= tokens_.size()) {
        return false;
    }
    lexeme = tokens_[pos_++];
    return true;
}

void Parser::parse(const std::vector<Lexeme>& tokens) const {
    (void)parse_value(tokens);
}

void Parser::parse(LexemeSource& source) const {
    (void)parse_value(source);
}

Value Parser::parse_value(const std::vector<Lexeme>& tokens) const {
    VectorLexemeSource source(tokens);
    return parse_value(source);
}

Value Parser::parse_value(LexemeSource& source) const {
    auto result = parse_recovering(source);
    if (!result.diagnostics.empty()) {
        throw ParseError(std::move(result.diagnostics));
    }
    return std::move(result.value);
}

ParseResult Parser::parse_recovering(const std::vector<Lexeme>& tokens) const {
    VectorLexemeSource source(tokens);
    return parse_recovering(source);
}

ParseResult Parser::parse_recovering(LexemeSource& source) const {
    std::vector<int> states;
    std::vector<Value> values;
    states.reserve(64);
    values.reserve(64);
    states.push_back(0);
    SourceCursor cursor(source);
    int recovering = 0;
    std::ptrdiff_t active_diagnostic = -1;
    ParseResult result;

    while (true) {
        const auto symbol = cursor.peek_symbol();
        const auto action = find_action(states.back(), symbol);
        if (action == nullptr) {
            if (recovering == 0) {
                result.diagnostics.push_back(make_diagnostic(states.back(), symbol, cursor));
                active_diagnostic = static_cast<std::ptrdiff_t>(result.diagnostics.size() - 1);
                bool recovered = false;
                while (!states.empty()) {
                    const auto error_action = find_action(states.back(), "error");
                    if (error_action != nullptr && error_action->kind == ParserActionKind::Shift) {
                        states.push_back(error_action->state);
                        values.push_back(error_lexeme(cursor));
                        recovering = 3;
                        result.diagnostics[static_cast<std::size_t>(active_diagnostic)].recovery.kind = "shift-error";
                        recovered = true;
                        break;
                    }
                    if (states.size() == 1) {
                        break;
                    }
                    states.pop_back();
                    if (!values.empty()) {
                        values.pop_back();
                    }
                }
                if (recovered) {
                    continue;
                }
                result.diagnostics[static_cast<std::size_t>(active_diagnostic)].recovery.kind = "abort";
                result.value = current_value(values);
                return result;
            }
            if (symbol == "$") {
                if (active_diagnostic >= 0) {
                    result.diagnostics[static_cast<std::size_t>(active_diagnostic)].recovery.kind = "abort";
                }
                result.value = current_value(values);
                return result;
            }
            (void)cursor.advance();
            if (active_diagnostic >= 0) {
                ++result.diagnostics[static_cast<std::size_t>(active_diagnostic)].recovery.discarded;
            }
            continue;
        }

        if (action->kind == ParserActionKind::Shift) {
            states.push_back(action->state);
            values.push_back(cursor.advance());
            if (recovering > 0) {
                --recovering;
                if (recovering == 0 && active_diagnostic >= 0) {
                    result.diagnostics[static_cast<std::size_t>(active_diagnostic)].recovery.kind = "recovered";
                    active_diagnostic = -1;
                }
            }
            continue;
        }

        if (action->kind == ParserActionKind::Reduce) {
            if (action->rule < 0 || static_cast<std::size_t>(action->rule) >= parser_rules.size()) {
                throw std::runtime_error("invalid reduction rule " + std::to_string(action->rule));
            }
            const auto rule = parser_rules[static_cast<std::size_t>(action->rule)];
            if (rule.id < 0) {
                throw std::runtime_error("missing reduction rule " + std::to_string(action->rule));
            }
            if (states.size() < rule.rhs_count + 1 || values.size() < rule.rhs_count) {
                throw std::runtime_error("parser stack underflow reducing rule " + std::to_string(rule.id));
            }
            auto rhs_start = values.end() - static_cast<std::ptrdiff_t>(rule.rhs_count);
            std::vector<Value> rhs_values(rhs_start, values.end());
            values.erase(rhs_start, values.end());
            states.resize(states.size() - rule.rhs_count);

            Value result;
            if (reducer_ && rule.action != SemanticAction::None) {
                Reduction ctx{rule.id, rule.lhs, rhs_symbols(rule), label_symbols(rule), rule.action, semantic_action_name(rule.action), rhs_values};
                result = reducer_(ctx);
            } else {
                result = default_reduce(rhs_values);
            }

            int goto_state = 0;
            if (!find_goto(states.back(), rule.lhs, goto_state)) {
                throw std::runtime_error("missing goto from state " + std::to_string(states.back()) + " on " + std::string(rule.lhs));
            }
            states.push_back(goto_state);
            values.push_back(std::move(result));
            continue;
        }

        if (action->kind == ParserActionKind::Accept) {
            if (active_diagnostic >= 0) {
                result.diagnostics[static_cast<std::size_t>(active_diagnostic)].recovery.kind = "recovered";
            }
            result.value = current_value(values);
            result.accepted = true;
            return result;
        }
    }
}

void parse(const std::vector<Lexeme>& tokens) {
    Parser{}.parse(tokens);
}

void parse(LexemeSource& source) {
    Parser{}.parse(source);
}

Value parse_value(const std::vector<Lexeme>& tokens, Reducer reducer) {
    return Parser(std::move(reducer)).parse_value(tokens);
}

Value parse_value(LexemeSource& source, Reducer reducer) {
    return Parser(std::move(reducer)).parse_value(source);
}

Value parse_value(const std::vector<Lexeme>& tokens, const ReducerMap& reducer) {
    return Parser(reducer).parse_value(tokens);
}

Value parse_value(LexemeSource& source, const ReducerMap& reducer) {
    return Parser(reducer).parse_value(source);
}

ParseResult parse_recovering(const std::vector<Lexeme>& tokens, Reducer reducer) {
    return Parser(std::move(reducer)).parse_recovering(tokens);
}

ParseResult parse_recovering(LexemeSource& source, Reducer reducer) {
    return Parser(std::move(reducer)).parse_recovering(source);
}

ParseResult parse_recovering(const std::vector<Lexeme>& tokens, const ReducerMap& reducer) {
    return Parser(reducer).parse_recovering(tokens);
}

ParseResult parse_recovering(LexemeSource& source, const ReducerMap& reducer) {
    return Parser(reducer).parse_recovering(source);
}

`
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
		out[action.Name] = "SemanticAction::" + action.Constant
	}
	return out
}

func typedCppManifestActions(manifest action.Manifest, actions []SemanticAction) []action.Action {
	constants := semanticActionIDs(actions)
	var out []action.Action
	for _, semanticAction := range manifest.Actions {
		if !semanticAction.Typed || len(semanticAction.Rules) == 0 {
			continue
		}
		if constants[semanticAction.Name] == "" {
			continue
		}
		out = append(out, semanticAction)
	}
	return out
}

type cppTypedField struct {
	Label string
	Name  string
	Type  string
}

func typedCppFields(rule action.Rule) []cppTypedField {
	used := map[string]int{}
	var fields []cppTypedField
	for _, operand := range rule.RHS {
		if operand.Label == "" {
			continue
		}
		base := cppMemberName(operand.Label)
		if base == "" {
			base = "value"
		}
		used[base]++
		name := base
		if used[base] > 1 {
			name = fmt.Sprintf("%s%d", base, used[base])
		}
		fieldType := strings.TrimSpace(operand.Type)
		if fieldType == "" {
			fieldType = "Value"
		}
		fields = append(fields, cppTypedField{
			Label: operand.Label,
			Name:  name,
			Type:  fieldType,
		})
	}
	return fields
}

func cppMemberName(name string) string {
	suffix := exportedIdentifierSuffix(name)
	if suffix == "" {
		suffix = "Value"
	}
	var b strings.Builder
	for i, r := range suffix {
		if i == 0 && r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	out := b.String()
	if !isValidCppIdentifier(out) {
		return "value_" + out
	}
	return out
}

func cppFunctionSuffix(name string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case isASCIIAlpha(r):
			b.WriteRune(toASCIILower(r))
			lastUnderscore = false
		case isASCIIDigit(r):
			if b.Len() == 0 {
				b.WriteString("n_")
			}
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		out = "action"
	}
	if !isValidCppIdentifier(out) {
		out = "action_" + out
	}
	return out
}

func tokenIdentifiers(tokens []string) map[string]string {
	used := map[string]int{"End": 1, "Error": 1}
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

func cppNamespace(specPackage string, outDirBase string) ([]string, error) {
	if specPackage != "" {
		parts := splitNamespace(specPackage)
		if len(parts) == 0 {
			return nil, fmt.Errorf("invalid C++ namespace %q", specPackage)
		}
		for _, part := range parts {
			if !isValidCppIdentifier(part) {
				return nil, fmt.Errorf("invalid C++ namespace %q", specPackage)
			}
		}
		return parts, nil
	}
	part := sanitizeIdentifier(outDirBase)
	if part == "" {
		part = "Generated"
	}
	return []string{"LangForge", "Generated", part}, nil
}

func splitNamespace(namespace string) []string {
	value := strings.ReplaceAll(namespace, "::", ".")
	parts := strings.Split(value, ".")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
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

func isValidCppIdentifier(identifier string) bool {
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
	switch identifier {
	case "alignas", "alignof", "and", "and_eq", "asm", "auto", "bitand", "bitor", "bool", "break", "case", "catch", "char", "char16_t", "char32_t", "class", "compl", "concept", "const", "consteval", "constexpr", "constinit", "const_cast", "continue", "co_await", "co_return", "co_yield", "decltype", "default", "delete", "do", "double", "dynamic_cast", "else", "enum", "explicit", "export", "extern", "false", "float", "for", "friend", "goto", "if", "inline", "int", "long", "mutable", "namespace", "new", "noexcept", "not", "not_eq", "nullptr", "operator", "or", "or_eq", "private", "protected", "public", "reflexpr", "register", "reinterpret_cast", "requires", "return", "short", "signed", "sizeof", "static", "static_assert", "static_cast", "struct", "switch", "synchronized", "template", "this", "thread_local", "throw", "true", "try", "typedef", "typeid", "typename", "union", "unsigned", "using", "virtual", "void", "volatile", "wchar_t", "while", "xor", "xor_eq":
		return false
	default:
		return true
	}
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

func toASCIILower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

func openNamespace(namespace []string) string {
	var b strings.Builder
	for _, part := range namespace {
		b.WriteString("namespace " + part + " {\n")
	}
	b.WriteString("\n")
	return b.String()
}

func closeNamespace(namespace []string) string {
	var b strings.Builder
	for i := len(namespace) - 1; i >= 0; i-- {
		b.WriteString("} // namespace " + namespace[i] + "\n")
	}
	return b.String()
}

func generatedHeader(source string, filename string) string {
	var b strings.Builder
	b.WriteString("// <auto-generated />\n")
	b.WriteString("// Code generated by lang-forge; DO NOT EDIT.\n")
	if filename != "" {
		b.WriteString("// File: " + filename + "\n")
	}
	if source != "" {
		b.WriteString("// Source: " + sanitizeSourceFile(source) + "\n")
	}
	b.WriteString("\n")
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

func cppString(value string) string {
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

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
