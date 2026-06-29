package c

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/russlank/lang-forge/internal/action"
	"github.com/russlank/lang-forge/internal/lex"
	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
	"github.com/russlank/lang-forge/internal/version"
)

// Input contains all validated artifacts required by the C backend.
type Input struct {
	Spec       *spec.Spec
	DFA        *lex.DFA
	Grammar    *parse.Grammar
	ParseTable *parse.Table
}

// Summary is the machine-readable table dump written next to generated C code.
type Summary struct {
	Spec       *spec.Spec     `json:"spec"`
	Lexer      *lex.DFA       `json:"lexer"`
	Grammar    *parse.Grammar `json:"grammar"`
	ParseTable *parse.Table   `json:"parseTable"`
}

// Manifest records high-level C generation metadata.
type Manifest struct {
	Tool         string            `json:"tool"`
	Version      string            `json:"version"`
	Commit       string            `json:"commit"`
	BuildDate    string            `json:"buildDate,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	Source       string            `json:"source"`
	Target       string            `json:"target"`
	Prefix       string            `json:"prefix"`
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
	Constant string `json:"cConstant,omitempty"`
}

// Generate writes the C scanner, parser, headers, manifest, and table dump.
func Generate(input Input, outDir string) error {
	if input.Spec == nil || input.DFA == nil || input.Grammar == nil || input.ParseTable == nil {
		return errors.New("c codegen input is incomplete")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	prefix := cPrefix(input.Spec.Package, filepath.Base(outDir))
	tokens := tokenNames(input)
	tokenIDs := tokenIdentifiers(prefix, tokens)
	actionManifest := action.Build(input.Grammar, input.Spec.Semantics, "c")
	actions := semanticActionsFromNames(actionManifest.Names(), prefix)
	manifest := Manifest{
		Tool:         version.Name,
		Version:      version.Version,
		Commit:       version.Commit,
		BuildDate:    version.BuildDate,
		Branch:       version.Branch,
		Source:       input.Spec.SourceFile,
		Target:       "c",
		Prefix:       prefix,
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
		"tokens.h":       renderTokensHeader(prefix, input.Spec.SourceFile, tokens, tokenIDs),
		"scanner.h":      renderScannerHeader(prefix, input.Spec.SourceFile),
		"scanner.c":      renderScannerSource(prefix, input.Spec.SourceFile, input.DFA, tokens, tokenIDs),
		"parser.h":       renderParserHeader(prefix, input.Spec.SourceFile, actions),
		"parser_typed.h": renderTypedParserHeader(prefix, input.Spec.SourceFile, actionManifest, actions),
		"parser.c":       renderParserSource(prefix, input.Spec.SourceFile, input.Spec, input.ParseTable, tokens, tokenIDs, actions),
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

func renderTokensHeader(prefix string, source string, tokens []string, tokenIDs map[string]string) string {
	guard := headerGuard(prefix, "TOKENS")
	var b strings.Builder
	b.WriteString(cHeader(source, "tokens.h"))
	b.WriteString("#ifndef " + guard + "\n#define " + guard + "\n\n")
	b.WriteString("#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n")
	b.WriteString("typedef enum " + prefix + "_token {\n")
	b.WriteString("    " + tokenEOF(prefix) + " = 0,\n")
	b.WriteString("    " + tokenError(prefix) + " = 1,\n")
	for i, token := range tokens {
		b.WriteString(fmt.Sprintf("    %s = %d,\n", tokenIDs[token], i+2))
	}
	b.WriteString("} " + prefix + "_token;\n\n")
	b.WriteString("const char *" + prefix + "_token_name(" + prefix + "_token token);\n\n")
	b.WriteString("#ifdef __cplusplus\n}\n#endif\n\n#endif\n")
	return b.String()
}

func renderScannerHeader(prefix string, source string) string {
	guard := headerGuard(prefix, "SCANNER")
	var b strings.Builder
	b.WriteString(cHeader(source, "scanner.h"))
	b.WriteString("#ifndef " + guard + "\n#define " + guard + "\n\n")
	b.WriteString("#include <stddef.h>\n#include \"tokens.h\"\n\n")
	b.WriteString("#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n")
	b.WriteString("typedef struct " + prefix + "_error {\n    char message[256];\n} " + prefix + "_error;\n\n")
	b.WriteString("typedef struct " + prefix + "_lexeme {\n    " + prefix + "_token token;\n    const char *text;\n    size_t length;\n    const char *channel;\n    size_t start;\n    size_t end;\n    int start_line;\n    int start_column;\n    int end_line;\n    int end_column;\n} " + prefix + "_lexeme;\n\n")
	b.WriteString("typedef struct " + prefix + "_scanner {\n    const char *input;\n    size_t length;\n    size_t pos;\n    int line;\n    int column;\n    int include_hidden;\n} " + prefix + "_scanner;\n\n")
	b.WriteString("void " + prefix + "_scanner_init(" + prefix + "_scanner *scanner, const char *input);\n")
	b.WriteString("void " + prefix + "_scanner_include_hidden(" + prefix + "_scanner *scanner, int include_hidden);\n")
	b.WriteString("int " + prefix + "_scanner_next(" + prefix + "_scanner *scanner, " + prefix + "_lexeme *lexeme, int *ok, " + prefix + "_error *error);\n")
	b.WriteString("int " + prefix + "_tokenize(const char *input, " + prefix + "_lexeme **out, size_t *count, " + prefix + "_error *error);\n")
	b.WriteString("void " + prefix + "_free_lexemes(" + prefix + "_lexeme *lexemes);\n\n")
	b.WriteString("#ifdef __cplusplus\n}\n#endif\n\n#endif\n")
	return b.String()
}

func renderScannerSource(prefix string, source string, dfa *lex.DFA, tokens []string, tokenIDs map[string]string) string {
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
	var b strings.Builder
	b.WriteString(cHeader(source, "scanner.c"))
	b.WriteString("#include \"scanner.h\"\n\n#include <stdint.h>\n#include <stdio.h>\n#include <stdlib.h>\n#include <string.h>\n\n")
	b.WriteString("typedef struct { uint32_t lo; uint32_t hi; int target; } " + prefix + "_transition;\n")
	b.WriteString("typedef struct { int accept; size_t start; size_t count; } " + prefix + "_state;\n")
	b.WriteString("typedef struct { " + prefix + "_token token; int skip; const char *channel; } " + prefix + "_rule_action;\n\n")
	b.WriteString("static const " + prefix + "_transition " + prefix + "_transitions[] = {\n")
	if len(transitions) == 0 {
		b.WriteString("    {0, 0, 0},\n")
	} else {
		for _, tr := range transitions {
			b.WriteString(tr)
		}
	}
	b.WriteString("};\n\n")
	b.WriteString("static const " + prefix + "_state " + prefix + "_states[] = {\n")
	for _, state := range dfa.States {
		b.WriteString(fmt.Sprintf("    {%d, %d, %d},\n", state.AcceptRule, stateStart[state.ID], stateCount[state.ID]))
	}
	b.WriteString("};\n\n")
	b.WriteString("static const " + prefix + "_rule_action " + prefix + "_rule_actions[] = {\n    {" + tokenError(prefix) + ", 1, \"\"},\n")
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
	for i := 1; i <= maxRule; i++ {
		rule := rules[i]
		token := tokenError(prefix)
		if tokenSet[rule.Token] {
			token = tokenIDs[rule.Token]
		}
		b.WriteString(fmt.Sprintf("    {%s, %d, %s},\n", token, boolInt(rule.Skip), cString(rule.Channel)))
	}
	b.WriteString("};\n\n")
	b.WriteString(renderTokenName(prefix, tokens, tokenIDs))
	b.WriteString(`
static void lf_set_error(` + prefix + `_error *error, const char *message) {
    if (error != NULL) {
        snprintf(error->message, sizeof(error->message), "%s", message);
    }
}

static void lf_clear_error(` + prefix + `_error *error) {
    if (error != NULL) {
        error->message[0] = '\0';
    }
}

static int lf_decode_utf8(const char *input, size_t length, size_t pos, uint32_t *out, size_t *size, ` + prefix + `_error *error) {
    unsigned char b0 = (unsigned char)input[pos];
    if (b0 < 0x80) { *out = b0; *size = 1; return 1; }
    if ((b0 & 0xe0) == 0xc0 && pos + 1 < length) {
        unsigned char b1 = (unsigned char)input[pos + 1];
        if ((b1 & 0xc0) == 0x80) {
            uint32_t value = ((uint32_t)(b0 & 0x1f) << 6) | (uint32_t)(b1 & 0x3f);
            if (value >= 0x80) { *out = value; *size = 2; return 1; }
        }
    }
    if ((b0 & 0xf0) == 0xe0 && pos + 2 < length) {
        unsigned char b1 = (unsigned char)input[pos + 1];
        unsigned char b2 = (unsigned char)input[pos + 2];
        if ((b1 & 0xc0) == 0x80 && (b2 & 0xc0) == 0x80) {
            uint32_t value = ((uint32_t)(b0 & 0x0f) << 12) | ((uint32_t)(b1 & 0x3f) << 6) | (uint32_t)(b2 & 0x3f);
            if (value >= 0x800 && (value < 0xd800 || value > 0xdfff)) { *out = value; *size = 3; return 1; }
        }
    }
    if ((b0 & 0xf8) == 0xf0 && pos + 3 < length) {
        unsigned char b1 = (unsigned char)input[pos + 1];
        unsigned char b2 = (unsigned char)input[pos + 2];
        unsigned char b3 = (unsigned char)input[pos + 3];
        if ((b1 & 0xc0) == 0x80 && (b2 & 0xc0) == 0x80 && (b3 & 0xc0) == 0x80) {
            uint32_t value = ((uint32_t)(b0 & 0x07) << 18) | ((uint32_t)(b1 & 0x3f) << 12) | ((uint32_t)(b2 & 0x3f) << 6) | (uint32_t)(b3 & 0x3f);
            if (value >= 0x10000 && value <= 0x10ffff) { *out = value; *size = 4; return 1; }
        }
    }
    lf_set_error(error, "invalid UTF-8 input");
    return 0;
}

static void lf_advance(const char *input, size_t start, size_t end, int *line, int *column) {
    size_t pos = start;
    while (pos < end) {
        uint32_t r = 0;
        size_t size = 1;
        ` + prefix + `_error ignored;
        ignored.message[0] = '\0';
        if (!lf_decode_utf8(input, end, pos, &r, &size, &ignored)) { return; }
        pos += size;
        if (r == '\n') { (*line)++; *column = 1; } else { (*column)++; }
    }
}

static int lf_match_at(const char *input, size_t length, size_t start, int *rule, size_t *end, ` + prefix + `_error *error) {
    int state = 0;
    int best_rule = ` + prefix + `_states[state].accept;
    size_t best_end = start;
    size_t pos = start;
    while (pos < length) {
        uint32_t r = 0;
        size_t size = 0;
        if (!lf_decode_utf8(input, length, pos, &r, &size, error)) {
            if (best_rule > 0) { break; }
            return 0;
        }
        int next = -1;
        ` + prefix + `_state st = ` + prefix + `_states[state];
        for (size_t i = 0; i < st.count; i++) {
            ` + prefix + `_transition tr = ` + prefix + `_transitions[st.start + i];
            if (r >= tr.lo && r <= tr.hi) { next = tr.target; break; }
        }
        if (next < 0) { break; }
        pos += size;
        state = next;
        if (` + prefix + `_states[state].accept > 0) {
            best_rule = ` + prefix + `_states[state].accept;
            best_end = pos;
        }
    }
    *rule = best_rule;
    *end = best_end;
    return 1;
}

void ` + prefix + `_scanner_init(` + prefix + `_scanner *scanner, const char *input) {
    scanner->input = input == NULL ? "" : input;
    scanner->length = strlen(scanner->input);
    scanner->pos = 0;
    scanner->line = 1;
    scanner->column = 1;
    scanner->include_hidden = 0;
}

void ` + prefix + `_scanner_include_hidden(` + prefix + `_scanner *scanner, int include_hidden) {
    scanner->include_hidden = include_hidden;
}

int ` + prefix + `_scanner_next(` + prefix + `_scanner *scanner, ` + prefix + `_lexeme *lexeme, int *ok, ` + prefix + `_error *error) {
    lf_clear_error(error);
    if (ok != NULL) { *ok = 0; }
    while (scanner->pos < scanner->length) {
        size_t start = scanner->pos;
        int start_line = scanner->line;
        int start_column = scanner->column;
        int rule = 0;
        size_t end = start;
        if (!lf_match_at(scanner->input, scanner->length, start, &rule, &end, error)) { return 0; }
        if (rule <= 0) { lf_set_error(error, "no lexical rule matched input"); return 0; }
        if (end == start) { lf_set_error(error, "lexer rule matched empty input"); return 0; }
        ` + prefix + `_rule_action action = ` + prefix + `_rule_actions[rule];
        int end_line = scanner->line;
        int end_column = scanner->column;
        lf_advance(scanner->input, start, end, &end_line, &end_column);
        scanner->pos = end;
        scanner->line = end_line;
        scanner->column = end_column;
        if (action.skip) { continue; }
        if (action.channel[0] != '\0' && !scanner->include_hidden) { continue; }
        lexeme->token = action.token;
        lexeme->text = scanner->input + start;
        lexeme->length = end - start;
        lexeme->channel = action.channel;
        lexeme->start = start;
        lexeme->end = end;
        lexeme->start_line = start_line;
        lexeme->start_column = start_column;
        lexeme->end_line = end_line;
        lexeme->end_column = end_column;
        *ok = 1;
        return 1;
    }
    lexeme->token = ` + tokenEOF(prefix) + `;
    lexeme->text = scanner->input + scanner->pos;
    lexeme->length = 0;
    lexeme->channel = "";
    lexeme->start = scanner->pos;
    lexeme->end = scanner->pos;
    lexeme->start_line = scanner->line;
    lexeme->start_column = scanner->column;
    lexeme->end_line = scanner->line;
    lexeme->end_column = scanner->column;
    *ok = 0;
    return 1;
}

int ` + prefix + `_tokenize(const char *input, ` + prefix + `_lexeme **out, size_t *count, ` + prefix + `_error *error) {
    lf_clear_error(error);
    if (out == NULL || count == NULL) { lf_set_error(error, "tokenize output arguments are required"); return 0; }
    *out = NULL;
    *count = 0;
    ` + prefix + `_scanner scanner;
    ` + prefix + `_scanner_init(&scanner, input);
    size_t capacity = 16;
    size_t used = 0;
    ` + prefix + `_lexeme *items = (` + prefix + `_lexeme *)malloc(capacity * sizeof(` + prefix + `_lexeme));
    if (items == NULL) { lf_set_error(error, "out of memory"); return 0; }
    while (1) {
        ` + prefix + `_lexeme lexeme;
        int ok = 0;
        if (!` + prefix + `_scanner_next(&scanner, &lexeme, &ok, error)) { free(items); return 0; }
        if (!ok) { break; }
        if (used == capacity) {
            capacity *= 2;
            ` + prefix + `_lexeme *next = (` + prefix + `_lexeme *)realloc(items, capacity * sizeof(` + prefix + `_lexeme));
            if (next == NULL) { free(items); lf_set_error(error, "out of memory"); return 0; }
            items = next;
        }
        items[used++] = lexeme;
    }
    *out = items;
    *count = used;
    return 1;
}

void ` + prefix + `_free_lexemes(` + prefix + `_lexeme *lexemes) {
    free(lexemes);
}
`)
	return b.String()
}

func renderParserHeader(prefix string, source string, actions []SemanticAction) string {
	guard := headerGuard(prefix, "PARSER")
	var b strings.Builder
	b.WriteString(cHeader(source, "parser.h"))
	b.WriteString("#ifndef " + guard + "\n#define " + guard + "\n\n")
	b.WriteString("#include <stddef.h>\n#include \"scanner.h\"\n\n")
	b.WriteString("#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n")
	b.WriteString("typedef enum " + prefix + "_semantic_action {\n    " + actionNone(prefix) + " = 0,\n")
	for _, action := range actions {
		b.WriteString(fmt.Sprintf("    %s = %d,\n", action.Constant, action.ID))
	}
	b.WriteString("} " + prefix + "_semantic_action;\n\n")
	b.WriteString("typedef void *" + prefix + "_value;\n\n")
	b.WriteString("typedef struct " + prefix + "_reduction {\n    int rule;\n    const char *lhs;\n    const char **rhs;\n    size_t rhs_count;\n    const char **labels;\n    size_t label_count;\n    " + prefix + "_semantic_action action_id;\n    const char *action;\n    " + prefix + "_value *values;\n} " + prefix + "_reduction;\n\n")
	b.WriteString("typedef " + prefix + "_value (*" + prefix + "_reduce_fn)(const " + prefix + "_reduction *ctx, void *user, " + prefix + "_error *error);\n\n")
	b.WriteString("/* One expected terminal or reporting group. */\n")
	b.WriteString("typedef struct " + prefix + "_expected_token {\n    const char *symbol;\n    const char *display;\n    const char *const *members;\n    size_t member_count;\n} " + prefix + "_expected_token;\n\n")
	b.WriteString("/* One source-rich parser syntax diagnostic. */\n")
	b.WriteString("typedef struct " + prefix + "_parse_diagnostic {\n    int state;\n    const char *unexpected;\n    const char *unexpected_display;\n    const " + prefix + "_expected_token *expected;\n    size_t expected_count;\n    size_t start;\n    size_t end;\n    int start_line;\n    int start_column;\n    int end_line;\n    int end_column;\n    const char *recovery;\n    size_t discarded;\n} " + prefix + "_parse_diagnostic;\n\n")
	b.WriteString("/* A possibly partial semantic value plus owned syntax diagnostics. */\n")
	b.WriteString("typedef struct " + prefix + "_parse_result {\n    " + prefix + "_value value;\n    " + prefix + "_parse_diagnostic *diagnostics;\n    size_t diagnostic_count;\n    int accepted;\n} " + prefix + "_parse_result;\n\n")
	b.WriteString("const char *" + prefix + "_semantic_action_name(" + prefix + "_semantic_action action);\n")
	b.WriteString("int " + prefix + "_reduction_value_for(const " + prefix + "_reduction *ctx, const char *label, " + prefix + "_value *out, " + prefix + "_error *error);\n")
	b.WriteString("int " + prefix + "_parse(const " + prefix + "_lexeme *tokens, size_t count, " + prefix + "_error *error);\n")
	b.WriteString("int " + prefix + "_parse_value(const " + prefix + "_lexeme *tokens, size_t count, " + prefix + "_reduce_fn reducer, void *user, " + prefix + "_value *out, " + prefix + "_error *error);\n\n")
	b.WriteString("int " + prefix + "_parse_recovering(const " + prefix + "_lexeme *tokens, size_t count, " + prefix + "_parse_result *result, " + prefix + "_error *error);\n")
	b.WriteString("int " + prefix + "_parse_value_recovering(const " + prefix + "_lexeme *tokens, size_t count, " + prefix + "_reduce_fn reducer, void *user, " + prefix + "_parse_result *result, " + prefix + "_error *error);\n")
	b.WriteString("void " + prefix + "_parse_result_init(" + prefix + "_parse_result *result);\n")
	b.WriteString("void " + prefix + "_parse_result_free(" + prefix + "_parse_result *result);\n\n")
	b.WriteString("#ifdef __cplusplus\n}\n#endif\n\n#endif\n")
	return b.String()
}

func renderTypedParserHeader(prefix string, source string, manifest action.Manifest, actions []SemanticAction) string {
	guard := headerGuard(prefix, "PARSER_TYPED")
	var b strings.Builder
	b.WriteString(cHeader(source, "parser_typed.h"))
	b.WriteString("#ifndef " + guard + "\n#define " + guard + "\n\n")
	b.WriteString("#include \"parser.h\"\n\n")
	b.WriteString("#include <stdio.h>\n\n")
	b.WriteString("#ifdef __cplusplus\nextern \"C\" {\n#endif\n\n")
	typedActions := typedCManifestActions(manifest, actions)
	if len(typedActions) == 0 {
		b.WriteString("/* No consistently typed semantic actions were declared for this grammar. */\n\n")
		b.WriteString("#ifdef __cplusplus\n}\n#endif\n\n#endif\n")
		return b.String()
	}
	constants := semanticActionIDs(actions)
	for _, semanticAction := range typedActions {
		actionName := cMemberName(semanticAction.Name)
		contextType := prefix + "_" + actionName + "_reduction"
		handlerType := prefix + "_" + actionName + "_handler"
		b.WriteString("/* Typed context for semantic action " + cString(semanticAction.Name) + ". */\n")
		b.WriteString("typedef struct " + contextType + " {\n")
		b.WriteString("    const " + prefix + "_reduction *reduction;\n")
		for _, field := range typedCFields(prefix, semanticAction.Rules[0]) {
			b.WriteString("    " + field.Type + " " + field.Name + ";\n")
		}
		b.WriteString("} " + contextType + ";\n\n")
		b.WriteString("typedef " + prefix + "_value (*" + handlerType + ")(const " + contextType + " *ctx, void *user, " + prefix + "_error *error);\n\n")
	}
	b.WriteString("typedef struct " + prefix + "_typed_reducer {\n")
	b.WriteString("    void *user;\n")
	for _, semanticAction := range typedActions {
		actionName := cMemberName(semanticAction.Name)
		b.WriteString("    " + prefix + "_" + actionName + "_handler " + actionName + ";\n")
	}
	b.WriteString("} " + prefix + "_typed_reducer;\n\n")
	b.WriteString("typedef struct " + prefix + "_boxed_typed_reducer {\n")
	b.WriteString("    " + prefix + "_reduce_fn reducer;\n")
	b.WriteString("    void *user;\n")
	b.WriteString("} " + prefix + "_boxed_typed_reducer;\n\n")
	for _, semanticAction := range typedActions {
		actionName := cMemberName(semanticAction.Name)
		contextType := prefix + "_" + actionName + "_reduction"
		b.WriteString("static inline " + prefix + "_value " + prefix + "_" + actionName + "_boxed_typed_handler(const " + contextType + " *ctx, void *user, " + prefix + "_error *error) {\n")
		b.WriteString("    const " + prefix + "_boxed_typed_reducer *boxed = (const " + prefix + "_boxed_typed_reducer *)user;\n")
		b.WriteString("    if (boxed == NULL || boxed->reducer == NULL) { if (error != NULL) { snprintf(error->message, sizeof(error->message), \"boxed typed reducer is required\"); } return NULL; }\n")
		b.WriteString("    return boxed->reducer(ctx->reduction, boxed->user, error);\n")
		b.WriteString("}\n\n")
	}
	b.WriteString("static inline " + prefix + "_typed_reducer " + prefix + "_typed_reducer_from_boxed(" + prefix + "_boxed_typed_reducer *storage, " + prefix + "_reduce_fn reducer, void *user) {\n")
	b.WriteString("    if (storage != NULL) { storage->reducer = reducer; storage->user = user; }\n")
	b.WriteString("    " + prefix + "_typed_reducer typed;\n")
	b.WriteString("    typed.user = storage;\n")
	for _, semanticAction := range typedActions {
		actionName := cMemberName(semanticAction.Name)
		b.WriteString("    typed." + actionName + " = " + prefix + "_" + actionName + "_boxed_typed_handler;\n")
	}
	b.WriteString("    return typed;\n")
	b.WriteString("}\n\n")
	b.WriteString("static inline int " + prefix + "_typed_reducer_validate(const " + prefix + "_typed_reducer *reducer, " + prefix + "_error *error) {\n")
	b.WriteString("    if (reducer == NULL) { if (error != NULL) { snprintf(error->message, sizeof(error->message), \"typed reducer is required\"); } return 0; }\n")
	for _, semanticAction := range typedActions {
		actionName := cMemberName(semanticAction.Name)
		b.WriteString("    if (reducer->" + actionName + " == NULL) { if (error != NULL) { snprintf(error->message, sizeof(error->message), \"typed reducer missing handler " + actionName + "\"); } return 0; }\n")
	}
	b.WriteString("    return 1;\n")
	b.WriteString("}\n\n")
	b.WriteString("static inline " + prefix + "_value " + prefix + "_typed_reduce(const " + prefix + "_reduction *ctx, void *user, " + prefix + "_error *error) {\n")
	b.WriteString("    const " + prefix + "_typed_reducer *reducer = (const " + prefix + "_typed_reducer *)user;\n")
	b.WriteString("    if (!" + prefix + "_typed_reducer_validate(reducer, error)) { return NULL; }\n")
	b.WriteString("    switch (ctx->action_id) {\n")
	for _, semanticAction := range typedActions {
		actionName := cMemberName(semanticAction.Name)
		contextType := prefix + "_" + actionName + "_reduction"
		b.WriteString("    case " + constants[semanticAction.Name] + ": {\n")
		b.WriteString("        " + contextType + " typed;\n")
		b.WriteString("        typed.reduction = ctx;\n")
		for _, field := range typedCFields(prefix, semanticAction.Rules[0]) {
			b.WriteString("        " + prefix + "_value " + field.Local + " = NULL;\n")
			b.WriteString("        if (!" + prefix + "_reduction_value_for(ctx, " + cString(field.Label) + ", &" + field.Local + ", error)) { return NULL; }\n")
			if !field.Pointer {
				b.WriteString("        if (" + field.Local + " == NULL) { if (error != NULL) { snprintf(error->message, sizeof(error->message), \"typed reducer label " + field.Name + " has null value\"); } return NULL; }\n")
			}
			b.WriteString("        typed." + field.Name + " = " + fieldCValueExpr(field, field.Local) + ";\n")
		}
		b.WriteString("        return reducer->" + actionName + "(&typed, reducer->user, error);\n")
		b.WriteString("    }\n")
	}
	b.WriteString("    case " + actionNone(prefix) + ":\n")
	b.WriteString("        return ctx->rhs_count == 1 ? ctx->values[0] : NULL;\n")
	b.WriteString("    default:\n")
	b.WriteString("        if (error != NULL) { snprintf(error->message, sizeof(error->message), \"no typed reducer handler for action %s\", ctx->action); }\n")
	b.WriteString("        return NULL;\n")
	b.WriteString("    }\n")
	b.WriteString("}\n\n")
	b.WriteString("static inline int " + prefix + "_parse_value_typed(const " + prefix + "_lexeme *tokens, size_t count, const " + prefix + "_typed_reducer *reducer, " + prefix + "_value *out, " + prefix + "_error *error) {\n")
	b.WriteString("    if (!" + prefix + "_typed_reducer_validate(reducer, error)) { return 0; }\n")
	b.WriteString("    return " + prefix + "_parse_value(tokens, count, " + prefix + "_typed_reduce, (void *)reducer, out, error);\n")
	b.WriteString("}\n\n")
	b.WriteString("static inline int " + prefix + "_parse_value_recovering_typed(const " + prefix + "_lexeme *tokens, size_t count, const " + prefix + "_typed_reducer *reducer, " + prefix + "_parse_result *result, " + prefix + "_error *error) {\n")
	b.WriteString("    if (!" + prefix + "_typed_reducer_validate(reducer, error)) { return 0; }\n")
	b.WriteString("    return " + prefix + "_parse_value_recovering(tokens, count, " + prefix + "_typed_reduce, (void *)reducer, result, error);\n")
	b.WriteString("}\n\n")
	b.WriteString("#ifdef __cplusplus\n}\n#endif\n\n#endif\n")
	return b.String()
}

func renderParserSource(prefix string, source string, project *spec.Spec, table *parse.Table, tokens []string, tokenIDs map[string]string, actions []SemanticAction) string {
	actionIDs := semanticActionIDs(actions)
	var b strings.Builder
	b.WriteString(cHeader(source, "parser.c"))
	b.WriteString("#include \"parser.h\"\n\n#include <stdio.h>\n#include <stdlib.h>\n#include <string.h>\n\n")
	b.WriteString("typedef enum { LF_ACT_SHIFT, LF_ACT_REDUCE, LF_ACT_ACCEPT } lf_action_kind;\n")
	b.WriteString("typedef struct { const char *symbol; lf_action_kind kind; int state; int rule; } lf_action_entry;\n")
	b.WriteString("typedef struct { size_t start; size_t count; } lf_row;\n")
	b.WriteString("typedef struct { const char *symbol; int state; } lf_goto_entry;\n")
	b.WriteString("typedef struct { int id; const char *lhs; const char **rhs; size_t rhs_count; const char **labels; size_t label_count; " + prefix + "_semantic_action action; } lf_rule;\n\n")
	b.WriteString("typedef struct { size_t start; size_t count; } lf_expected_row;\n")
	b.WriteString("typedef struct { const char *symbol; const char *display; } lf_alias_entry;\n\n")
	b.WriteString(renderActionName(prefix, actions))
	b.WriteString(renderParserRules(prefix, table, actionIDs))
	b.WriteString(renderParserActions(prefix, table))
	b.WriteString(renderParserGotos(prefix, table))
	b.WriteString(renderParserExpected(prefix, project, table))
	b.WriteString(`
static void lf_set_error(` + prefix + `_error *error, const char *message) {
    if (error != NULL) {
        snprintf(error->message, sizeof(error->message), "%s", message);
    }
}

static void lf_clear_error(` + prefix + `_error *error) {
    if (error != NULL) {
        error->message[0] = '\0';
    }
}

static const char *lf_lookahead(const ` + prefix + `_lexeme *tokens, size_t count, size_t pos) {
    if (pos >= count) { return "$"; }
    if (tokens[pos].token == ` + tokenEOF(prefix) + `) { return "$"; }
    return ` + prefix + `_token_name(tokens[pos].token);
}

static const lf_action_entry *lf_find_action(int state, const char *symbol) {
    if (state < 0 || (size_t)state >= sizeof(` + prefix + `_action_rows) / sizeof(` + prefix + `_action_rows[0])) { return NULL; }
    lf_row row = ` + prefix + `_action_rows[state];
    for (size_t i = 0; i < row.count; i++) {
        const lf_action_entry *entry = &` + prefix + `_actions[row.start + i];
        if (strcmp(entry->symbol, symbol) == 0) { return entry; }
    }
    return NULL;
}

static int lf_find_goto(int state, const char *symbol, int *out) {
    if (state < 0 || (size_t)state >= sizeof(` + prefix + `_goto_rows) / sizeof(` + prefix + `_goto_rows[0])) { return 0; }
    lf_row row = ` + prefix + `_goto_rows[state];
    for (size_t i = 0; i < row.count; i++) {
        const lf_goto_entry *entry = &` + prefix + `_gotos[row.start + i];
        if (strcmp(entry->symbol, symbol) == 0) { *out = entry->state; return 1; }
    }
    return 0;
}

static int lf_push_state(int **states, size_t *count, size_t *capacity, int state, ` + prefix + `_error *error) {
    if (*count == *capacity) {
        *capacity *= 2;
        int *next = (int *)realloc(*states, *capacity * sizeof(int));
        if (next == NULL) { lf_set_error(error, "out of memory"); return 0; }
        *states = next;
    }
    (*states)[(*count)++] = state;
    return 1;
}

static int lf_push_value(` + prefix + `_value **values, size_t *count, size_t *capacity, ` + prefix + `_value value, ` + prefix + `_error *error) {
    if (*count == *capacity) {
        *capacity *= 2;
        ` + prefix + `_value *next = (` + prefix + `_value *)realloc(*values, *capacity * sizeof(` + prefix + `_value));
        if (next == NULL) { lf_set_error(error, "out of memory"); return 0; }
        *values = next;
    }
    (*values)[(*count)++] = value;
    return 1;
}

static const char *lf_unexpected_display(const char *symbol) {
    if (strcmp(symbol, "$") == 0) { return "end of input"; }
    for (size_t i = 0; i < ` + prefix + `_alias_count; i++) {
        if (strcmp(` + prefix + `_aliases[i].symbol, symbol) == 0) { return ` + prefix + `_aliases[i].display; }
    }
    return symbol;
}

static int lf_append_diagnostic(` + prefix + `_parse_result *result, int state, const char *unexpected, const ` + prefix + `_lexeme *tokens, size_t count, size_t pos, ` + prefix + `_error *error) {
    size_t next_count = result->diagnostic_count + 1;
    ` + prefix + `_parse_diagnostic *next = (` + prefix + `_parse_diagnostic *)realloc(result->diagnostics, next_count * sizeof(` + prefix + `_parse_diagnostic));
    if (next == NULL) { lf_set_error(error, "out of memory"); return 0; }
    result->diagnostics = next;
    ` + prefix + `_parse_diagnostic *diagnostic = &result->diagnostics[result->diagnostic_count];
    memset(diagnostic, 0, sizeof(*diagnostic));
    diagnostic->state = state;
    diagnostic->unexpected = unexpected;
    diagnostic->unexpected_display = lf_unexpected_display(unexpected);
    if (state >= 0 && (size_t)state < sizeof(` + prefix + `_expected_rows) / sizeof(` + prefix + `_expected_rows[0])) {
        lf_expected_row row = ` + prefix + `_expected_rows[state];
        diagnostic->expected = &` + prefix + `_expected_tokens[row.start];
        diagnostic->expected_count = row.count;
    }
    if (pos < count) {
        diagnostic->start = tokens[pos].start;
        diagnostic->end = tokens[pos].end;
        diagnostic->start_line = tokens[pos].start_line;
        diagnostic->start_column = tokens[pos].start_column;
        diagnostic->end_line = tokens[pos].end_line;
        diagnostic->end_column = tokens[pos].end_column;
    } else if (count > 0) {
        diagnostic->start = tokens[count - 1].end;
        diagnostic->end = tokens[count - 1].end;
        diagnostic->start_line = tokens[count - 1].end_line;
        diagnostic->start_column = tokens[count - 1].end_column;
        diagnostic->end_line = tokens[count - 1].end_line;
        diagnostic->end_column = tokens[count - 1].end_column;
    } else {
        diagnostic->start_line = diagnostic->end_line = 1;
        diagnostic->start_column = diagnostic->end_column = 1;
    }
    diagnostic->recovery = "none";
    result->diagnostic_count = next_count;
    return 1;
}

static void lf_format_parse_error(` + prefix + `_error *error, const ` + prefix + `_parse_result *result) {
    if (error == NULL || result->diagnostic_count == 0) { return; }
    const ` + prefix + `_parse_diagnostic *diagnostic = &result->diagnostics[0];
    snprintf(error->message, sizeof(error->message), "parse error at %d:%d: unexpected %s", diagnostic->start_line, diagnostic->start_column, diagnostic->unexpected_display);
}

void ` + prefix + `_parse_result_init(` + prefix + `_parse_result *result) {
    if (result == NULL) { return; }
    result->value = NULL;
    result->diagnostics = NULL;
    result->diagnostic_count = 0;
    result->accepted = 0;
}

void ` + prefix + `_parse_result_free(` + prefix + `_parse_result *result) {
    if (result == NULL) { return; }
    free(result->diagnostics);
    ` + prefix + `_parse_result_init(result);
}

int ` + prefix + `_parse(const ` + prefix + `_lexeme *tokens, size_t count, ` + prefix + `_error *error) {
    return ` + prefix + `_parse_value(tokens, count, NULL, NULL, NULL, error);
}

int ` + prefix + `_parse_value(const ` + prefix + `_lexeme *tokens, size_t count, ` + prefix + `_reduce_fn reducer, void *user, ` + prefix + `_value *out, ` + prefix + `_error *error) {
    ` + prefix + `_parse_result result = {0};
    int ok = ` + prefix + `_parse_value_recovering(tokens, count, reducer, user, &result, error);
    if (!ok) { ` + prefix + `_parse_result_free(&result); return 0; }
    if (out != NULL) { *out = result.value; }
    if (result.diagnostic_count != 0) {
        lf_format_parse_error(error, &result);
        ` + prefix + `_parse_result_free(&result);
        return 0;
    }
    ` + prefix + `_parse_result_free(&result);
    return 1;
}

int ` + prefix + `_parse_recovering(const ` + prefix + `_lexeme *tokens, size_t count, ` + prefix + `_parse_result *result, ` + prefix + `_error *error) {
    return ` + prefix + `_parse_value_recovering(tokens, count, NULL, NULL, result, error);
}

int ` + prefix + `_parse_value_recovering(const ` + prefix + `_lexeme *tokens, size_t count, ` + prefix + `_reduce_fn reducer, void *user, ` + prefix + `_parse_result *result, ` + prefix + `_error *error) {
    lf_clear_error(error);
    if (result == NULL) { lf_set_error(error, "parser result is required"); return 0; }
    ` + prefix + `_parse_result_init(result);
    if (tokens == NULL && count != 0) { lf_set_error(error, "parser tokens are required"); return 0; }
    size_t state_capacity = 64, state_count = 0;
    size_t value_capacity = 64, value_count = 0;
    int *states = (int *)malloc(state_capacity * sizeof(int));
    ` + prefix + `_value *values = (` + prefix + `_value *)malloc(value_capacity * sizeof(` + prefix + `_value));
    if (states == NULL || values == NULL) { free(states); free(values); lf_set_error(error, "out of memory"); return 0; }
    states[state_count++] = 0;
    size_t pos = 0;
    int recovering = 0;
    size_t active_diagnostic = 0;
    int has_active_diagnostic = 0;
    while (1) {
        const char *lookahead = lf_lookahead(tokens, count, pos);
        const lf_action_entry *action = lf_find_action(states[state_count - 1], lookahead);
        if (action == NULL) {
            if (recovering == 0) {
                if (!lf_append_diagnostic(result, states[state_count - 1], lookahead, tokens, count, pos, error)) { free(states); free(values); return 0; }
                active_diagnostic = result->diagnostic_count - 1;
                has_active_diagnostic = 1;
                int recovered = 0;
                while (state_count > 0) {
                    const lf_action_entry *error_action = lf_find_action(states[state_count - 1], "error");
                    if (error_action != NULL && error_action->kind == LF_ACT_SHIFT) {
                        if (!lf_push_state(&states, &state_count, &state_capacity, error_action->state, error)) { free(states); free(values); return 0; }
                        if (!lf_push_value(&values, &value_count, &value_capacity, NULL, error)) { free(states); free(values); return 0; }
                        recovering = 3;
                        result->diagnostics[active_diagnostic].recovery = "shift-error";
                        recovered = 1;
                        break;
                    }
                    if (state_count == 1) { break; }
                    state_count--;
                    if (value_count > 0) { value_count--; }
                }
                if (recovered) { continue; }
                result->diagnostics[active_diagnostic].recovery = "abort";
                result->value = value_count == 0 ? NULL : values[value_count - 1];
                free(states);
                free(values);
                return 1;
            }
            if (strcmp(lookahead, "$") == 0) {
                if (has_active_diagnostic) { result->diagnostics[active_diagnostic].recovery = "abort"; }
                result->value = value_count == 0 ? NULL : values[value_count - 1];
                free(states);
                free(values);
                return 1;
            }
            pos++;
            if (has_active_diagnostic) { result->diagnostics[active_diagnostic].discarded++; }
            continue;
        }
        if (action->kind == LF_ACT_SHIFT) {
            if (pos >= count) { free(states); free(values); lf_set_error(error, "shift past end of input"); return 0; }
            if (!lf_push_state(&states, &state_count, &state_capacity, action->state, error)) { free(states); free(values); return 0; }
            if (!lf_push_value(&values, &value_count, &value_capacity, (` + prefix + `_value)&tokens[pos], error)) { free(states); free(values); return 0; }
            pos++;
            if (recovering > 0) {
                recovering--;
                if (recovering == 0 && has_active_diagnostic) {
                    result->diagnostics[active_diagnostic].recovery = "recovered";
                    has_active_diagnostic = 0;
                }
            }
            continue;
        }
        if (action->kind == LF_ACT_REDUCE) {
            const lf_rule *rule = &` + prefix + `_rules[action->rule];
            if (state_count < rule->rhs_count + 1 || value_count < rule->rhs_count) { free(states); free(values); lf_set_error(error, "parser stack underflow"); return 0; }
            ` + prefix + `_value *rhs_values = (` + prefix + `_value *)calloc(rule->rhs_count == 0 ? 1 : rule->rhs_count, sizeof(` + prefix + `_value));
            if (rhs_values == NULL) { free(states); free(values); lf_set_error(error, "out of memory"); return 0; }
            for (size_t i = 0; i < rule->rhs_count; i++) {
                rhs_values[i] = values[value_count - rule->rhs_count + i];
            }
            ` + prefix + `_value result = NULL;
            if (reducer != NULL && rule->action != ` + actionNone(prefix) + `) {
                ` + prefix + `_reduction ctx = {rule->id, rule->lhs, rule->rhs, rule->rhs_count, rule->labels, rule->label_count, rule->action, ` + prefix + `_semantic_action_name(rule->action), rhs_values};
                result = reducer(&ctx, user, error);
                if (error != NULL && error->message[0] != '\0') { free(rhs_values); free(states); free(values); return 0; }
            } else if (rule->rhs_count == 1) {
                result = rhs_values[0];
            }
            value_count -= rule->rhs_count;
            state_count -= rule->rhs_count;
            int goto_state = 0;
            if (!lf_find_goto(states[state_count - 1], rule->lhs, &goto_state)) { free(rhs_values); free(states); free(values); lf_set_error(error, "missing goto"); return 0; }
            free(rhs_values);
            if (!lf_push_state(&states, &state_count, &state_capacity, goto_state, error)) { free(states); free(values); return 0; }
            if (!lf_push_value(&values, &value_count, &value_capacity, result, error)) { free(states); free(values); return 0; }
            continue;
        }
        if (action->kind == LF_ACT_ACCEPT) {
            if (pos < count && !(tokens[pos].token == ` + tokenEOF(prefix) + ` && pos + 1 == count)) {
                free(states); free(values); lf_set_error(error, "tokens after EOF"); return 0;
            }
            if (has_active_diagnostic) { result->diagnostics[active_diagnostic].recovery = "recovered"; }
            result->value = value_count == 0 ? NULL : values[value_count - 1];
            result->accepted = 1;
            free(states);
            free(values);
            return 1;
        }
    }
}
`)
	return b.String()
}

func renderTokenName(prefix string, tokens []string, tokenIDs map[string]string) string {
	var b strings.Builder
	b.WriteString("const char *" + prefix + "_token_name(" + prefix + "_token token) {\n    switch (token) {\n")
	b.WriteString("    case " + tokenEOF(prefix) + ": return \"EOF\";\n")
	b.WriteString("    case " + tokenError(prefix) + ": return \"ERROR\";\n")
	for _, token := range tokens {
		b.WriteString(fmt.Sprintf("    case %s: return %s;\n", tokenIDs[token], cString(token)))
	}
	b.WriteString("    default: return \"UNKNOWN\";\n    }\n}\n\n")
	return b.String()
}

func renderActionName(prefix string, actions []SemanticAction) string {
	var b strings.Builder
	b.WriteString("const char *" + prefix + "_semantic_action_name(" + prefix + "_semantic_action action) {\n    switch (action) {\n")
	b.WriteString("    case " + actionNone(prefix) + ": return \"\";\n")
	for _, action := range actions {
		b.WriteString(fmt.Sprintf("    case %s: return %s;\n", action.Constant, cString(action.Name)))
	}
	b.WriteString("    default: return \"UNKNOWN\";\n    }\n}\n\n")
	b.WriteString("int " + prefix + "_reduction_value_for(const " + prefix + "_reduction *ctx, const char *label, " + prefix + "_value *out, " + prefix + "_error *error) {\n")
	b.WriteString("    if (out != NULL) { *out = NULL; }\n")
	b.WriteString("    if (ctx == NULL || label == NULL) { if (error != NULL) { snprintf(error->message, sizeof(error->message), \"reduction and label are required\"); } return 0; }\n")
	b.WriteString("    for (size_t i = 0; i < ctx->label_count; i++) {\n")
	b.WriteString("        if (ctx->labels[i] != NULL && strcmp(ctx->labels[i], label) == 0) {\n")
	b.WriteString("            if (i >= ctx->rhs_count) { if (error != NULL) { snprintf(error->message, sizeof(error->message), \"reduction label has no value\"); } return 0; }\n")
	b.WriteString("            if (out != NULL) { *out = ctx->values[i]; }\n")
	b.WriteString("            return 1;\n")
	b.WriteString("        }\n")
	b.WriteString("    }\n")
	b.WriteString("    if (error != NULL) { snprintf(error->message, sizeof(error->message), \"reduction label was not found\"); }\n")
	b.WriteString("    return 0;\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderParserRules(prefix string, table *parse.Table, actionIDs map[string]string) string {
	var b strings.Builder
	for _, rule := range table.Rules {
		b.WriteString(fmt.Sprintf("static const char *%s_rule_%d_rhs[] = {", prefix, rule.ID))
		if len(rule.RHS) == 0 {
			b.WriteString("NULL")
		} else {
			for _, sym := range rule.RHS {
				b.WriteString(cString(sym) + ", ")
			}
		}
		b.WriteString("};\n")
		b.WriteString(fmt.Sprintf("static const char *%s_rule_%d_labels[] = {", prefix, rule.ID))
		if len(rule.Labels) == 0 {
			b.WriteString("NULL")
		} else {
			for _, label := range rule.Labels {
				b.WriteString(cString(label) + ", ")
			}
		}
		b.WriteString("};\n")
	}
	b.WriteString("\nstatic const lf_rule " + prefix + "_rules[] = {\n")
	for _, rule := range table.Rules {
		action := actionNone(prefix)
		if id, ok := actionIDs[strings.TrimSpace(rule.Actions["c"])]; ok {
			action = id
		}
		b.WriteString(fmt.Sprintf("    {%d, %s, %s_rule_%d_rhs, %d, %s_rule_%d_labels, %d, %s},\n", rule.ID, cString(rule.LHS), prefix, rule.ID, len(rule.RHS), prefix, rule.ID, len(rule.Labels), action))
	}
	b.WriteString("};\n\n")
	return b.String()
}

func renderParserActions(prefix string, table *parse.Table) string {
	var entries []string
	rowStart := make([]int, len(table.States))
	rowCount := make([]int, len(table.States))
	for _, state := range table.States {
		actions := table.Actions[state.ID]
		var symbols []string
		for sym := range actions {
			symbols = append(symbols, sym)
		}
		sort.Strings(symbols)
		rowStart[state.ID] = len(entries)
		for _, sym := range symbols {
			action := actions[sym]
			kind := "LF_ACT_ACCEPT"
			if action.Kind == parse.ActionShift {
				kind = "LF_ACT_SHIFT"
			} else if action.Kind == parse.ActionReduce {
				kind = "LF_ACT_REDUCE"
			}
			entries = append(entries, fmt.Sprintf("    {%s, %s, %d, %d},\n", cString(sym), kind, action.State, action.Rule))
			rowCount[state.ID]++
		}
	}
	var b strings.Builder
	b.WriteString("static const lf_action_entry " + prefix + "_actions[] = {\n")
	if len(entries) == 0 {
		b.WriteString("    {\"\", LF_ACT_ACCEPT, 0, 0},\n")
	} else {
		for _, entry := range entries {
			b.WriteString(entry)
		}
	}
	b.WriteString("};\n\nstatic const lf_row " + prefix + "_action_rows[] = {\n")
	for _, state := range table.States {
		b.WriteString(fmt.Sprintf("    {%d, %d},\n", rowStart[state.ID], rowCount[state.ID]))
	}
	b.WriteString("};\n\n")
	return b.String()
}

func renderParserGotos(prefix string, table *parse.Table) string {
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
			entries = append(entries, fmt.Sprintf("    {%s, %d},\n", cString(sym), gotos[sym]))
			rowCount[state.ID]++
		}
	}
	var b strings.Builder
	b.WriteString("static const lf_goto_entry " + prefix + "_gotos[] = {\n")
	if len(entries) == 0 {
		b.WriteString("    {\"\", 0},\n")
	} else {
		for _, entry := range entries {
			b.WriteString(entry)
		}
	}
	b.WriteString("};\n\nstatic const lf_row " + prefix + "_goto_rows[] = {\n")
	for _, state := range table.States {
		b.WriteString(fmt.Sprintf("    {%d, %d},\n", rowStart[state.ID], rowCount[state.ID]))
	}
	b.WriteString("};\n\n")
	return b.String()
}

func renderParserExpected(prefix string, project *spec.Spec, table *parse.Table) string {
	var entries []string
	rowStart := make([]int, len(table.States))
	rowCount := make([]int, len(table.States))
	var b strings.Builder
	memberID := 0
	for _, state := range table.States {
		rowStart[state.ID] = len(entries)
		for _, expected := range table.Expected[state.ID] {
			memberName := "NULL"
			if len(expected.Members) > 0 {
				memberName = fmt.Sprintf("%s_expected_members_%d", prefix, memberID)
				b.WriteString("static const char *" + memberName + "[] = {")
				for _, member := range expected.Members {
					b.WriteString(cString(member) + ", ")
				}
				b.WriteString("};\n")
				memberID++
			}
			entries = append(entries, fmt.Sprintf("    {%s, %s, %s, %d},\n", cString(expected.Symbol), cString(expected.Display), memberName, len(expected.Members)))
			rowCount[state.ID]++
		}
	}
	if memberID > 0 {
		b.WriteString("\n")
	}
	b.WriteString("static const " + prefix + "_expected_token " + prefix + "_expected_tokens[] = {\n")
	if len(entries) == 0 {
		b.WriteString("    {\"\", \"\", NULL, 0},\n")
	} else {
		for _, entry := range entries {
			b.WriteString(entry)
		}
	}
	b.WriteString("};\n\n")
	b.WriteString("static const lf_expected_row " + prefix + "_expected_rows[] = {\n")
	for _, state := range table.States {
		b.WriteString(fmt.Sprintf("    {%d, %d},\n", rowStart[state.ID], rowCount[state.ID]))
	}
	if len(table.States) == 0 {
		b.WriteString("    {0, 0},\n")
	}
	b.WriteString("};\n\n")

	aliases := append([]spec.ExpectedTokenAlias(nil), project.Grammar.ExpectedTokens.Aliases...)
	sort.SliceStable(aliases, func(i, j int) bool { return aliases[i].Token < aliases[j].Token })
	b.WriteString("static const lf_alias_entry " + prefix + "_aliases[] = {\n")
	if len(aliases) == 0 {
		b.WriteString("    {\"\", \"\"},\n")
	} else {
		for _, alias := range aliases {
			b.WriteString(fmt.Sprintf("    {%s, %s},\n", cString(alias.Token), cString(alias.Label)))
		}
	}
	b.WriteString("};\n\n")
	b.WriteString(fmt.Sprintf("static const size_t %s_alias_count = %d;\n\n", prefix, len(aliases)))
	return b.String()
}

func semanticActions(rules []parse.Rule, target string, prefix string) []SemanticAction {
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
	return semanticActionsFromNames(names, prefix)
}

func semanticActionsFromNames(names []string, prefix string) []SemanticAction {
	used := map[string]int{"NONE": 1}
	out := make([]SemanticAction, 0, len(names))
	for _, name := range names {
		id := len(out) + 1
		out = append(out, SemanticAction{ID: id, Name: name, Constant: ""})
	}
	for i := range out {
		out[i].Constant = uniqueConstant(cIdentifierSuffix(out[i].Name), used, strings.ToUpper(prefix)+"_ACTION_")
	}
	return out
}

func semanticActionIDs(actions []SemanticAction) map[string]string {
	out := map[string]string{}
	for _, action := range actions {
		out[action.Name] = action.Constant
	}
	return out
}

func typedCManifestActions(manifest action.Manifest, actions []SemanticAction) []action.Action {
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

type cTypedField struct {
	Label   string
	Name    string
	Local   string
	Type    string
	Pointer bool
}

func typedCFields(prefix string, rule action.Rule) []cTypedField {
	used := map[string]int{}
	var fields []cTypedField
	for _, operand := range rule.RHS {
		if operand.Label == "" {
			continue
		}
		base := cMemberName(operand.Label)
		if base == "" {
			base = "value"
		}
		used[base]++
		name := base
		if used[base] > 1 {
			name = fmt.Sprintf("%s_%d", base, used[base])
		}
		fieldType := strings.TrimSpace(operand.Type)
		if fieldType == "" {
			fieldType = "void *"
		}
		pointer := cTypeIsPointer(fieldType)
		if fieldType == "lexeme" {
			fieldType = "const " + prefix + "_lexeme *"
			pointer = true
		}
		fields = append(fields, cTypedField{
			Label:   operand.Label,
			Name:    name,
			Local:   "value_" + name,
			Type:    fieldType,
			Pointer: pointer,
		})
	}
	return fields
}

func fieldCValueExpr(field cTypedField, local string) string {
	if field.Pointer {
		return "(" + field.Type + ")" + local
	}
	return "*((" + field.Type + " *)" + local + ")"
}

func cTypeIsPointer(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.Contains(trimmed, "*")
}

func tokenIdentifiers(prefix string, tokens []string) map[string]string {
	used := map[string]int{"EOF": 1, "ERROR": 1}
	out := map[string]string{}
	for _, token := range tokens {
		out[token] = uniqueConstant(cIdentifierSuffix(token), used, strings.ToUpper(prefix)+"_TOKEN_")
	}
	return out
}

func uniqueConstant(base string, used map[string]int, prefix string) string {
	if base == "" {
		base = "VALUE"
	}
	name := prefix + strings.ToUpper(base)
	if used[name] == 0 {
		used[name] = 1
		return name
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", name, i)
		if used[candidate] == 0 {
			used[candidate] = 1
			return candidate
		}
	}
}

func cPrefix(specPackage string, outDirBase string) string {
	value := specPackage
	if value == "" {
		value = outDirBase
	}
	var b strings.Builder
	for _, r := range value {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			if b.Len() == 0 && unicode.IsDigit(r) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "langforge_generated"
	}
	if first, _ := utf8FirstRune(out); unicode.IsDigit(first) {
		return "lf_" + out
	}
	return out
}

func utf8FirstRune(value string) (rune, bool) {
	for _, r := range value {
		return r, true
	}
	return 0, false
}

func cIdentifierSuffix(name string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToUpper(r))
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func cMemberName(name string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			if b.Len() == 0 && unicode.IsDigit(r) {
				b.WriteString("value_")
			}
			b.WriteRune(unicode.ToLower(r))
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		out = "value"
	}
	if cKeywords[out] {
		out = "value_" + out
	}
	return out
}

var cKeywords = map[string]bool{
	"auto": true, "break": true, "case": true, "char": true, "const": true, "continue": true,
	"default": true, "do": true, "double": true, "else": true, "enum": true, "extern": true,
	"float": true, "for": true, "goto": true, "if": true, "inline": true, "int": true,
	"long": true, "register": true, "restrict": true, "return": true, "short": true, "signed": true,
	"sizeof": true, "static": true, "struct": true, "switch": true, "typedef": true, "union": true,
	"unsigned": true, "void": true, "volatile": true, "while": true,
}

func cHeader(source, file string) string {
	return "/* Code generated by lang-forge; DO NOT EDIT.\n * File: " + file + "\n * Source: " + source + "\n */\n\n"
}

func headerGuard(prefix, name string) string {
	return strings.ToUpper(prefix) + "_" + name + "_H"
}

func tokenEOF(prefix string) string   { return strings.ToUpper(prefix) + "_TOKEN_EOF" }
func tokenError(prefix string) string { return strings.ToUpper(prefix) + "_TOKEN_ERROR" }
func actionNone(prefix string) string { return strings.ToUpper(prefix) + "_ACTION_NONE" }

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func cString(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			if r < 0x20 || r > 0x7e {
				b.WriteString(fmt.Sprintf("\\x%02x", r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
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
