package golang

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	goparser "go/parser"
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

// Input contains all validated artifacts required by the Go backend.
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

// Manifest records high-level generation metadata.
type Manifest struct {
	Tool         string            `json:"tool"`
	Version      string            `json:"version"`
	Commit       string            `json:"commit"`
	BuildDate    string            `json:"buildDate,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	Source       string            `json:"source"`
	Target       string            `json:"target"`
	Package      string            `json:"package"`
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
	Constant string `json:"goConstant,omitempty"`
}

// Generate writes the Go scanner, parser, manifest, and table dump.
func Generate(input Input, outDir string) error {
	if input.Spec == nil || input.DFA == nil || input.Grammar == nil || input.ParseTable == nil {
		return errors.New("go codegen input is incomplete")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	pkg, err := goPackageName(input.Spec.Package, filepath.Base(outDir))
	if err != nil {
		return err
	}
	if err := validateSemanticTypes(input.Spec.Semantics, "go"); err != nil {
		return err
	}
	tokens := tokenNames(input)
	actionManifest := action.Build(input.Grammar, input.Spec.Semantics, "go")
	actions := semanticActionsFromNames(actionManifest.Names())
	manifest := Manifest{
		Tool:         version.Name,
		Version:      version.Version,
		Commit:       version.Commit,
		BuildDate:    version.BuildDate,
		Branch:       version.Branch,
		Source:       input.Spec.SourceFile,
		Target:       "go",
		Package:      pkg,
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
	if err := writeGoFile(filepath.Join(outDir, "tokens.go"), renderTokens(pkg, input.Spec.SourceFile, tokens)); err != nil {
		return err
	}
	if err := writeGoFile(filepath.Join(outDir, "scanner.go"), renderScanner(pkg, input.Spec.SourceFile, input.DFA, tokens)); err != nil {
		return err
	}
	if err := writeGoFile(filepath.Join(outDir, "parser.go"), renderParser(pkg, "parser.go", input.Spec, input.ParseTable, tokens, actions, actionManifest)); err != nil {
		return err
	}
	return nil
}

func validateSemanticTypes(semantics spec.SemanticSpec, target string) error {
	for _, semanticType := range semantics.Types {
		if semanticType.Target != target {
			continue
		}
		if _, err := goparser.ParseExpr(semanticType.Type); err != nil {
			return fmt.Errorf("invalid Go semantic type %q for %s: %w", semanticType.Type, semanticType.Symbol, err)
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

func renderTokens(pkg string, source string, tokens []string) string {
	var b strings.Builder
	b.WriteString(generatedHeader(pkg, source))
	b.WriteString("// Token identifies one terminal emitted by the scanner.\n")
	b.WriteString("type Token int\n\n")
	b.WriteString("const (\n")
	b.WriteString("\t// TokenEOF represents parser end-of-input.\n")
	b.WriteString("\tTokenEOF Token = iota\n")
	b.WriteString("\t// TokenError represents an unknown token value.\n")
	b.WriteString("\tTokenError\n")
	for _, tok := range tokens {
		b.WriteString("\tToken" + tok + "\n")
	}
	b.WriteString(")\n\n")
	b.WriteString("// String returns the grammar terminal name for t.\n")
	b.WriteString("func (t Token) String() string {\n")
	b.WriteString("\tswitch t {\n")
	b.WriteString("\tcase TokenEOF:\n\t\treturn \"EOF\"\n")
	b.WriteString("\tcase TokenError:\n\t\treturn \"ERROR\"\n")
	for _, tok := range tokens {
		b.WriteString("\tcase Token" + tok + ":\n\t\treturn \"" + tok + "\"\n")
	}
	b.WriteString("\tdefault:\n\t\treturn \"UNKNOWN\"\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return b.String()
}

func renderScanner(pkg string, source string, dfa *lex.DFA, tokens []string) string {
	tokenSet := map[string]bool{}
	for _, tok := range tokens {
		tokenSet[tok] = true
	}
	var b strings.Builder
	b.WriteString(generatedHeader(pkg, source))
	b.WriteString("import (\n\t\"fmt\"\n\t\"io\"\n\t\"strings\"\n\t\"sync\"\n\t\"unicode/utf8\"\n)\n\n")
	b.WriteString("// Lexeme is one scanner result with byte offsets and scalar positions.\n")
	b.WriteString("type Lexeme struct {\n\tToken Token\n\tText string\n\tChannel string\n\tStart int\n\tEnd int\n\tStartLine int\n\tStartColumn int\n\tEndLine int\n\tEndColumn int\n}\n\n")
	b.WriteString("// Scanner incrementally tokenizes an input string.\n//\n// Scanner methods are safe for concurrent use. Concurrent calls to Next share\n// one input cursor and therefore observe one serialized token stream.\n")
	b.WriteString("type Scanner struct {\n\tmu sync.Mutex\n\tinput string\n\tpos int\n\tline int\n\tcolumn int\n\tincludeHidden bool\n}\n\n")
	b.WriteString("// NewScanner creates a scanner for in-memory source text.\n")
	b.WriteString("func NewScanner(input string) *Scanner { return &Scanner{input: input, line: 1, column: 1} }\n\n")
	b.WriteString("// IncludeHidden controls whether channel lexemes are returned.\n")
	b.WriteString("func (s *Scanner) IncludeHidden(include bool) { s.mu.Lock(); defer s.mu.Unlock(); s.includeHidden = include }\n\n")
	b.WriteString("// Tokenize returns every visible lexeme in input.\n")
	b.WriteString("func Tokenize(input string) ([]Lexeme, error) { return NewScanner(input).All() }\n\n")
	b.WriteString("// All returns all visible lexemes until end-of-input.\n")
	b.WriteString("func (s *Scanner) All() ([]Lexeme, error) {\n\tvar out []Lexeme\n\tfor {\n\t\tlexeme, ok, err := s.Next()\n\t\tif err != nil { return nil, err }\n\t\tif !ok { return out, nil }\n\t\tout = append(out, lexeme)\n\t}\n}\n\n")
	b.WriteString("// Next returns the next visible lexeme, or ok=false at end-of-input.\n")
	b.WriteString("func (s *Scanner) Next() (Lexeme, bool, error) {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\tfor s.pos < len(s.input) {\n\t\tstart, startLine, startColumn := s.pos, s.line, s.column\n\t\trule, end, err := matchAt(s.input, s.pos)\n\t\tif err != nil { return Lexeme{}, false, err }\n\t\tif rule <= 0 { return Lexeme{}, false, fmt.Errorf(\"no lexical rule matched byte %d near %q\", s.pos, preview(s.input, s.pos)) }\n\t\tif end == s.pos { return Lexeme{}, false, fmt.Errorf(\"lexer rule %d matched empty input at byte %d\", rule, s.pos) }\n\t\taction := ruleActions[rule]\n\t\tendLine, endColumn := advancePosition(s.input, s.pos, end, s.line, s.column)\n\t\tlex := Lexeme{Token: action.token, Text: s.input[start:end], Channel: action.channel, Start: start, End: end, StartLine: startLine, StartColumn: startColumn, EndLine: endLine, EndColumn: endColumn}\n\t\ts.pos, s.line, s.column = end, endLine, endColumn\n\t\tif action.skip { continue }\n\t\tif action.channel != \"\" && !s.includeHidden { continue }\n\t\treturn lex, true, nil\n\t}\n\treturn Lexeme{Token: TokenEOF, Start: s.pos, End: s.pos, StartLine: s.line, StartColumn: s.column, EndLine: s.line, EndColumn: s.column}, false, nil\n}\n\n")
	b.WriteString(goReaderScannerRuntime())
	b.WriteString("type scannerTransition struct { lo rune; hi rune; target int }\n")
	b.WriteString("type scannerState struct { accept int; transitions []scannerTransition }\n")
	b.WriteString("type ruleAction struct { token Token; skip bool; channel string }\n\n")
	b.WriteString("var scannerStates = []scannerState{\n")
	for _, st := range dfa.States {
		b.WriteString(fmt.Sprintf("\t{accept: %d, transitions: []scannerTransition{", st.AcceptRule))
		for _, tr := range st.Transitions {
			for _, rr := range tr.Set.Normalize() {
				b.WriteString(fmt.Sprintf("{lo: %d, hi: %d, target: %d},", rr.Lo, rr.Hi, tr.Target))
			}
		}
		b.WriteString("}},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("var ruleActions = map[int]ruleAction{\n")
	for _, rule := range dfa.Rules {
		token := "TokenError"
		if tokenSet[rule.Token] {
			token = "Token" + rule.Token
		}
		if comment := sourceComment(rule.Span); comment != "" {
			b.WriteString("\t" + comment + "\n")
		}
		b.WriteString(fmt.Sprintf("\t%d: {token: %s, skip: %t, channel: %q},\n", rule.Index, token, rule.Skip, rule.Channel))
	}
	b.WriteString("}\n\n")
	b.WriteString("func matchAt(input string, start int) (int, int, error) {\n\tstateID := 0\n\tbestRule := scannerStates[stateID].accept\n\tbestEnd := start\n\tfor pos := start; pos < len(input); {\n\t\tr, size, err := decodeScannerRune(input, pos)\n\t\tif err != nil {\n\t\t\tif bestRule > 0 { break }\n\t\t\treturn 0, start, err\n\t\t}\n\t\tnext := -1\n\t\tfor _, tr := range scannerStates[stateID].transitions {\n\t\t\tif r >= tr.lo && r <= tr.hi { next = tr.target; break }\n\t\t}\n\t\tif next < 0 { break }\n\t\tpos += size\n\t\tstateID = next\n\t\tif scannerStates[stateID].accept > 0 { bestRule = scannerStates[stateID].accept; bestEnd = pos }\n\t}\n\treturn bestRule, bestEnd, nil\n}\n\n")
	b.WriteString("func decodeScannerRune(input string, pos int) (rune, int, error) {\n\tr, size := utf8.DecodeRuneInString(input[pos:])\n\tif r == utf8.RuneError && size == 1 && input[pos] >= utf8.RuneSelf { return 0, 0, fmt.Errorf(\"invalid UTF-8 at byte %d\", pos) }\n\treturn r, size, nil\n}\n\n")
	b.WriteString("func advancePosition(input string, start, end, line, column int) (int, int) {\n\tfor pos := start; pos < end; {\n\t\tr, size, err := decodeScannerRune(input, pos)\n\t\tif err != nil { return line, column }\n\t\tpos += size\n\t\tif r == '\\n' { line++; column = 1 } else { column++ }\n\t}\n\treturn line, column\n}\n\n")
	b.WriteString("func preview(input string, pos int) string {\n\tend := pos + 16\n\tif end > len(input) { end = len(input) }\n\treturn input[pos:end]\n}\n")
	return b.String()
}

func goReaderScannerRuntime() string {
	return `const defaultReaderScannerBufferSize = 4096
const defaultReaderScannerMaxBufferedLexemeBytes = 1048576

// ReaderScannerOption configures scanner constructors that pull source text from a reader.
type ReaderScannerOption func(*readerScannerOptions)

type readerScannerOptions struct {
	readBufferSize int
	maxBufferedLexemeBytes int
}

func defaultReaderScannerOptions() readerScannerOptions {
	return readerScannerOptions{readBufferSize: defaultReaderScannerBufferSize, maxBufferedLexemeBytes: defaultReaderScannerMaxBufferedLexemeBytes}
}

// WithReaderScannerBufferSize changes the temporary read buffer size used by NewReaderScanner.
func WithReaderScannerBufferSize(size int) ReaderScannerOption {
	return func(options *readerScannerOptions) {
		if size > 0 { options.readBufferSize = size }
	}
}

// WithMaxBufferedLexemeBytes limits how many bytes a reader-backed scanner may buffer while deciding one lexeme.
func WithMaxBufferedLexemeBytes(size int) ReaderScannerOption {
	return func(options *readerScannerOptions) {
		if size > 0 { options.maxBufferedLexemeBytes = size }
	}
}

// ReaderScanner tokenizes UTF-8 source text pulled synchronously from an io.Reader.
//
// The parser still pulls lexemes through LexemeSource. ReaderScanner is the scanner-side
// input adapter for files, stdin, pipes, and other readers. It reads more source text
// only when the DFA needs another rune to decide the current longest-match lexeme.
type ReaderScanner struct {
	mu sync.Mutex
	reader io.Reader
	buffer string
	base int
	line int
	column int
	includeHidden bool
	eof bool
	pendingErr error
	readBuffer []byte
	maxBufferedLexemeBytes int
}

// NewReaderScanner creates a scanner that pulls source text from reader on demand.
func NewReaderScanner(reader io.Reader, options ...ReaderScannerOption) *ReaderScanner {
	config := defaultReaderScannerOptions()
	for _, option := range options {
		if option != nil { option(&config) }
	}
	if config.readBufferSize <= 0 { config.readBufferSize = defaultReaderScannerBufferSize }
	if config.maxBufferedLexemeBytes <= 0 { config.maxBufferedLexemeBytes = defaultReaderScannerMaxBufferedLexemeBytes }
	return &ReaderScanner{
		reader: reader,
		line: 1,
		column: 1,
		readBuffer: make([]byte, config.readBufferSize),
		maxBufferedLexemeBytes: config.maxBufferedLexemeBytes,
	}
}

// TokenizeFromReader returns every visible lexeme read from reader.
func TokenizeFromReader(reader io.Reader, options ...ReaderScannerOption) ([]Lexeme, error) {
	return NewReaderScanner(reader, options...).All()
}

// IncludeHidden controls whether channel lexemes are returned.
func (s *ReaderScanner) IncludeHidden(include bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.includeHidden = include
}

// All returns all visible lexemes until end-of-input.
func (s *ReaderScanner) All() ([]Lexeme, error) {
	var out []Lexeme
	for {
		lexeme, ok, err := s.Next()
		if err != nil { return nil, err }
		if !ok { return out, nil }
		out = append(out, lexeme)
	}
}

// Next returns the next visible lexeme, or ok=false at end-of-input.
func (s *ReaderScanner) Next() (Lexeme, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if len(s.buffer) == 0 && !s.eof {
			if err := s.readMore(); err != nil { return Lexeme{}, false, err }
		}
		if len(s.buffer) == 0 && s.eof {
			return Lexeme{Token: TokenEOF, Start: s.base, End: s.base, StartLine: s.line, StartColumn: s.column, EndLine: s.line, EndColumn: s.column}, false, nil
		}
		start, startLine, startColumn := s.base, s.line, s.column
		rule, end, err := s.matchBuffered()
		if err != nil { return Lexeme{}, false, err }
		if rule <= 0 { return Lexeme{}, false, fmt.Errorf("no lexical rule matched byte %d near %q", s.base, preview(s.buffer, 0)) }
		if end == 0 { return Lexeme{}, false, fmt.Errorf("lexer rule %d matched empty input at byte %d", rule, s.base) }
		action := ruleActions[rule]
		endLine, endColumn := advancePosition(s.buffer, 0, end, s.line, s.column)
		text := strings.Clone(s.buffer[:end])
		s.buffer = s.buffer[end:]
		s.base += end
		s.line, s.column = endLine, endColumn
		if action.skip { continue }
		if action.channel != "" && !s.includeHidden { continue }
		return Lexeme{Token: action.token, Text: text, Channel: action.channel, Start: start, End: start + end, StartLine: startLine, StartColumn: startColumn, EndLine: endLine, EndColumn: endColumn}, true, nil
	}
}

func (s *ReaderScanner) readMore() error {
	if s.pendingErr != nil {
		err := s.pendingErr
		s.pendingErr = nil
		return err
	}
	if s.eof { return nil }
	if s.reader == nil { return fmt.Errorf("nil scanner reader") }
	n, err := s.reader.Read(s.readBuffer)
	if n > 0 {
		s.buffer += string(s.readBuffer[:n])
		if len(s.buffer) > s.maxBufferedLexemeBytes {
			return fmt.Errorf("scanner buffered lexeme exceeds %d bytes", s.maxBufferedLexemeBytes)
		}
	}
	if err != nil {
		if err == io.EOF {
			s.eof = true
			return nil
		}
		if n > 0 {
			s.pendingErr = err
			return nil
		}
		return err
	}
	if n == 0 {
		return io.ErrNoProgress
	}
	return nil
}

func (s *ReaderScanner) matchBuffered() (int, int, error) {
	stateID := 0
	bestRule := scannerStates[stateID].accept
	bestEnd := 0
	for pos := 0; ; {
		if pos >= len(s.buffer) {
			if s.eof || len(scannerStates[stateID].transitions) == 0 {
				return bestRule, bestEnd, nil
			}
			if err := s.readMore(); err != nil { return 0, 0, err }
			continue
		}
		r, size, needMore, err := decodeStreamingScannerRune(s.buffer, pos, s.eof, s.base)
		if needMore {
			if err := s.readMore(); err != nil { return 0, 0, err }
			continue
		}
		if err != nil {
			if bestRule > 0 { break }
			return 0, 0, err
		}
		next := -1
		for _, tr := range scannerStates[stateID].transitions {
			if r >= tr.lo && r <= tr.hi { next = tr.target; break }
		}
		if next < 0 { break }
		pos += size
		stateID = next
		if scannerStates[stateID].accept > 0 { bestRule = scannerStates[stateID].accept; bestEnd = pos }
	}
	return bestRule, bestEnd, nil
}

func decodeStreamingScannerRune(input string, pos int, eof bool, base int) (rune, int, bool, error) {
	if pos >= len(input) {
		return 0, 0, !eof, nil
	}
	if !utf8.FullRuneInString(input[pos:]) && !eof {
		return 0, 0, true, nil
	}
	r, size := utf8.DecodeRuneInString(input[pos:])
	if r == utf8.RuneError && size == 1 && input[pos] >= utf8.RuneSelf {
		return 0, 0, false, fmt.Errorf("invalid UTF-8 at byte %d", base+pos)
	}
	return r, size, false, nil
}

`
}

func renderParserImports(b *strings.Builder, mode spec.SemanticActionMode, includes []spec.SemanticInclude, actionManifest action.Manifest) {
	includes = requiredParserIncludes(mode, includes, actionManifest)
	if len(includes) == 0 {
		b.WriteString("import \"fmt\"\n\n")
		return
	}
	b.WriteString("import (\n\t\"fmt\"\n")
	for _, include := range includes {
		if include.Alias != "" {
			b.WriteString(fmt.Sprintf("\t%s %q\n", include.Alias, include.Path))
		} else {
			b.WriteString(fmt.Sprintf("\t%q\n", include.Path))
		}
	}
	b.WriteString(")\n\n")
}

func requiredParserIncludes(mode spec.SemanticActionMode, includes []spec.SemanticInclude, actionManifest action.Manifest) []spec.SemanticInclude {
	if mode == spec.SemanticModeInline {
		return includes
	}
	var required []spec.SemanticInclude
	for _, include := range includes {
		qualifier := include.Alias
		if qualifier == "" {
			qualifier = filepath.Base(include.Path)
		}
		if qualifier == "" || qualifier == "_" || qualifier == "." {
			continue
		}
		needle := qualifier + "."
		used := false
		for _, semanticAction := range actionManifest.Actions {
			if strings.Contains(semanticAction.ReturnType, needle) {
				used = true
			}
			for _, rule := range semanticAction.Rules {
				for _, operand := range rule.RHS {
					if strings.Contains(operand.Type, needle) {
						used = true
					}
				}
			}
		}
		if used {
			required = append(required, include)
		}
	}
	return required
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
	usedConstants := map[string]int{"SemanticActionNone": 1}
	out := make([]SemanticAction, 0, len(names))
	for _, name := range names {
		id := len(out) + 1
		out = append(out, SemanticAction{
			ID:       id,
			Name:     name,
			Constant: semanticActionConstant(name, id, usedConstants),
		})
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

func semanticActionExpr(name string, ids map[string]string) string {
	constant, ok := ids[strings.TrimSpace(name)]
	if !ok {
		return "SemanticActionNone"
	}
	return constant
}

func semanticActionConstant(name string, id int, used map[string]int) string {
	suffix := exportedIdentifierSuffix(name)
	if suffix == "" {
		suffix = fmt.Sprintf("Action%d", id)
	}
	base := "SemanticAction" + suffix
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

func renderSemanticActionDeclarations(b *strings.Builder, actions []SemanticAction) {
	b.WriteString("// SemanticAction identifies one generated semantic reduction hook.\n")
	b.WriteString("type SemanticAction int\n\n")
	b.WriteString("const (\n")
	b.WriteString("\t// SemanticActionNone marks grammar rules without target action text.\n")
	b.WriteString("\tSemanticActionNone SemanticAction = iota\n")
	for _, action := range actions {
		b.WriteString(fmt.Sprintf("\t// %s identifies the %q semantic action.\n", action.Constant, action.Name))
		b.WriteString("\t" + action.Constant + "\n")
	}
	b.WriteString(")\n\n")
	b.WriteString("var semanticActionNames = []string{\n")
	b.WriteString("\tSemanticActionNone: \"\",\n")
	for _, action := range actions {
		b.WriteString(fmt.Sprintf("\t%s: %q,\n", action.Constant, action.Name))
	}
	b.WriteString("}\n\n")
	b.WriteString("var semanticActionByName = map[string]SemanticAction{\n")
	for _, action := range actions {
		b.WriteString(fmt.Sprintf("\t%q: %s,\n", action.Name, action.Constant))
	}
	b.WriteString("}\n\n")
	b.WriteString("// String returns the source action label for a.\n")
	b.WriteString("func (a SemanticAction) String() string { return SemanticActionName(a) }\n\n")
	b.WriteString("// SemanticActionName returns the source action label for a.\n")
	b.WriteString("func SemanticActionName(a SemanticAction) string {\n\tif int(a) >= 0 && int(a) < len(semanticActionNames) { return semanticActionNames[a] }\n\treturn \"UNKNOWN\"\n}\n\n")
	b.WriteString("// LookupSemanticAction returns the generated action ID for name.\n")
	b.WriteString("func LookupSemanticAction(name string) (SemanticAction, bool) {\n\taction, ok := semanticActionByName[name]\n\treturn action, ok\n}\n\n")
}

func renderParser(pkg string, generatedFile string, project *spec.Spec, table *parse.Table, tokens []string, actions []SemanticAction, actionManifest action.Manifest) string {
	var b strings.Builder
	source := ""
	if project != nil {
		source = project.SourceFile
	}
	b.WriteString(generatedHeader(pkg, source))
	semantics := spec.SemanticSpec{}
	if project != nil {
		semantics = project.Semantics
	}
	goMode := semantics.ModeFor("go")
	goIncludes := semantics.IncludesFor("go")
	renderParserImports(&b, goMode, goIncludes, actionManifest)
	actionIDs := semanticActionIDs(actions)
	b.WriteString("// Parser recognizes the generated grammar over scanner lexemes.\n")
	b.WriteString("//\n// Parser instances are safe for concurrent Parse and ParseValue calls when\n// the installed Reducer is also safe for concurrent use.\n")
	b.WriteString("type Parser struct { reducer Reducer }\n\n")
	b.WriteString("// Value is one semantic value shifted or reduced by the parser.\n")
	b.WriteString("type Value any\n\n")
	b.WriteString("// ExpectedToken describes one expected terminal or reporting group.\n")
	b.WriteString("type ExpectedToken struct {\n\tSymbol string\n\tDisplay string\n\tMembers []string\n}\n\n")
	b.WriteString("// RecoveryAction records how the parser handled one syntax error.\n")
	b.WriteString("type RecoveryAction struct {\n\tKind string\n\tDiscarded int\n}\n\n")
	b.WriteString("// ParseDiagnostic describes one syntax error and its source range.\n")
	b.WriteString("type ParseDiagnostic struct {\n\tState int\n\tUnexpected string\n\tUnexpectedDisplay string\n\tExpected []ExpectedToken\n\tStart int\n\tEnd int\n\tStartLine int\n\tStartColumn int\n\tEndLine int\n\tEndColumn int\n\tRecovery RecoveryAction\n}\n\n")
	b.WriteString("// ParseResult contains a possibly partial semantic value and all syntax diagnostics.\n")
	b.WriteString("type ParseResult struct {\n\tValue Value\n\tDiagnostics []ParseDiagnostic\n\tAccepted bool\n}\n\n")
	b.WriteString("// ParseError reports one or more syntax diagnostics from fail-fast-compatible APIs.\n")
	b.WriteString("type ParseError struct { Diagnostics []ParseDiagnostic }\n\n")
	b.WriteString("// Error formats the first syntax diagnostic and the total diagnostic count.\n")
	b.WriteString("func (e *ParseError) Error() string {\n\tif e == nil || len(e.Diagnostics) == 0 { return \"parse error\" }\n\td := e.Diagnostics[0]\n\tmessage := fmt.Sprintf(\"parse error at %d:%d: unexpected %s\", d.StartLine, d.StartColumn, d.UnexpectedDisplay)\n\tif len(d.Expected) > 0 {\n\t\tnames := make([]string, len(d.Expected))\n\t\tfor i, expected := range d.Expected { names[i] = expected.Display }\n\t\tmessage += fmt.Sprintf(\"; expected %v\", names)\n\t}\n\tif len(e.Diagnostics) > 1 { message += fmt.Sprintf(\" (%d diagnostics)\", len(e.Diagnostics)) }\n\treturn message\n}\n\n")
	renderSemanticActionDeclarations(&b, actions)
	b.WriteString("// Reduction describes one grammar rule reduction.\n")
	b.WriteString("type Reduction struct {\n\tRule int\n\tLHS string\n\tRHS []string\n\tLabels []string\n\tActionID SemanticAction\n\tAction string\n\tValues []Value\n}\n\n")
	b.WriteString("// ValueFor returns the semantic value associated with a named RHS label.\n")
	b.WriteString("func (r Reduction) ValueFor(label string) (Value, error) {\n\tfor index, candidate := range r.Labels {\n\t\tif candidate == label {\n\t\t\tif index >= len(r.Values) { return nil, fmt.Errorf(\"rule %d action %q label %q has no semantic value\", r.Rule, r.Action, label) }\n\t\t\treturn r.Values[index], nil\n\t\t}\n\t}\n\treturn nil, fmt.Errorf(\"rule %d action %q has no RHS label %q\", r.Rule, r.Action, label)\n}\n\n")
	b.WriteString("// SemanticActionMode records how generated parser action text is handled.\n")
	b.WriteString("const SemanticActionMode = " + fmt.Sprintf("%q", string(goMode)) + "\n\n")
	b.WriteString("// SemanticInclude describes a handwritten package or library declared by the spec.\n")
	b.WriteString("type SemanticInclude struct { Target string; Alias string; Path string }\n\n")
	b.WriteString("// SemanticIncludes lists target-specific handwritten semantic dependencies.\n")
	b.WriteString("var SemanticIncludes = []SemanticInclude{\n")
	for _, include := range semantics.Includes {
		b.WriteString(fmt.Sprintf("\t{Target: %q, Alias: %q, Path: %q},\n", include.Target, include.Alias, include.Path))
	}
	b.WriteString("}\n\n")
	b.WriteString("// Reducer receives target-tagged action hooks during parser reductions.\n")
	b.WriteString("type Reducer interface { Reduce(Reduction) (Value, error) }\n\n")
	b.WriteString("// ReductionHandler handles one generated semantic action.\n")
	b.WriteString("type ReductionHandler func(Reduction) (Value, error)\n\n")
	b.WriteString("// ReducerMap dispatches reductions by generated semantic action ID.\n")
	b.WriteString("type ReducerMap map[SemanticAction]ReductionHandler\n\n")
	b.WriteString("// ValidateCoverage reports missing and unknown semantic action handlers.\n")
	b.WriteString("func (m ReducerMap) ValidateCoverage() error {\n\tvar missing []string\n\tfor action := SemanticAction(1); int(action) < len(semanticActionNames); action++ {\n\t\tif _, ok := m[action]; !ok { missing = append(missing, action.String()) }\n\t}\n\tfirstUnknown, hasUnknown := 0, false\n\tfor action := range m {\n\t\tif action <= SemanticActionNone || int(action) >= len(semanticActionNames) {\n\t\t\tif !hasUnknown || int(action) < firstUnknown { firstUnknown, hasUnknown = int(action), true }\n\t\t}\n\t}\n\tif len(missing) == 0 && !hasUnknown { return nil }\n\tif hasUnknown { return fmt.Errorf(\"semantic reducer coverage mismatch: missing=%v firstUnknown=%d\", missing, firstUnknown) }\n\treturn fmt.Errorf(\"semantic reducer coverage mismatch: missing=%v\", missing)\n}\n\n")
	b.WriteString("// Reduce dispatches ctx to the handler registered for ctx.ActionID.\n")
	b.WriteString("func (m ReducerMap) Reduce(ctx Reduction) (Value, error) {\n\thandler, ok := m[ctx.ActionID]\n\tif !ok { return nil, fmt.Errorf(\"no reducer registered for action %s\", ctx.ActionID) }\n\treturn handler(ctx)\n}\n\n")
	renderTypedReductionContexts(&b, actionManifest, actions)
	b.WriteString("// ReducerFunc adapts a function to the Reducer interface.\n")
	b.WriteString("type ReducerFunc func(Reduction) (Value, error)\n\n")
	b.WriteString("// Reduce calls f(ctx).\n")
	b.WriteString("func (f ReducerFunc) Reduce(ctx Reduction) (Value, error) { return f(ctx) }\n\n")
	b.WriteString("// Option configures a generated parser instance.\n")
	b.WriteString("type Option func(*Parser)\n\n")
	b.WriteString("// WithReducer installs a semantic reducer for target-tagged grammar actions.\n")
	b.WriteString("func WithReducer(reducer Reducer) Option { return func(p *Parser) { p.reducer = reducer } }\n\n")
	b.WriteString("// NewParser creates a parser instance configured by options.\n")
	b.WriteString("func NewParser(options ...Option) *Parser {\n\tp := &Parser{}\n\tfor _, option := range options { option(p) }\n\treturn p\n}\n\n")
	b.WriteString(`// LexemeSource is the synchronous pull interface consumed by generated parsers.
//
// Next returns a visible grammar lexeme and true, or false when the source
// naturally reaches the end of input. A source may also return one explicit
// TokenEOF lexeme with ok=true; tokens returned after explicit EOF are errors.
type LexemeSource interface { Next() (Lexeme, bool, error) }

type lexemeSliceSource struct {
	input []Lexeme
	pos int
}

func newLexemeSliceSource(input []Lexeme) *lexemeSliceSource {
	return &lexemeSliceSource{input: input}
}

func (s *lexemeSliceSource) Next() (Lexeme, bool, error) {
	if s.pos >= len(s.input) { return Lexeme{}, false, nil }
	lexeme := s.input[s.pos]
	s.pos++
	return lexeme, true, nil
}

type parserLexemeCursor struct {
	source LexemeSource
	lookahead Lexeme
	symbol string
	ready bool
	sawEOF bool
	last Lexeme
	haveLast bool
}

func newParserLexemeCursor(source LexemeSource) (*parserLexemeCursor, error) {
	if source == nil { return nil, fmt.Errorf("nil lexeme source") }
	return &parserLexemeCursor{source: source}, nil
}

func (c *parserLexemeCursor) peekSymbol() (string, error) {
	if c.ready { return c.symbol, nil }
	if c.sawEOF {
		c.lookahead = c.eofLexeme()
		c.symbol = "$"
		c.ready = true
		return c.symbol, nil
	}
	lexeme, ok, err := c.source.Next()
	if err != nil { return "", err }
	if !ok {
		c.lookahead = c.eofLexeme()
		c.symbol = "$"
		c.ready = true
		c.sawEOF = true
		return c.symbol, nil
	}
	if lexeme.Token == TokenEOF {
		lexeme = c.normalizeEOFLexeme(lexeme)
		extra, more, err := c.source.Next()
		if err != nil { return "", err }
		if more {
			return "", fmt.Errorf("token after EOF in lexeme source: %s at %d:%d", terminalName(extra.Token), extra.StartLine, extra.StartColumn)
		}
		c.lookahead = lexeme
		c.symbol = "$"
		c.ready = true
		c.sawEOF = true
		return c.symbol, nil
	}
	c.last = lexeme
	c.haveLast = true
	c.lookahead = lexeme
	c.symbol = terminalName(lexeme.Token)
	c.ready = true
	return c.symbol, nil
}

func (c *parserLexemeCursor) advance() (Lexeme, error) {
	symbol, err := c.peekSymbol()
	if err != nil { return Lexeme{}, err }
	if symbol == "$" { return Lexeme{}, fmt.Errorf("shift past end of input") }
	lexeme := c.lookahead
	c.ready = false
	return lexeme, nil
}

func (c *parserLexemeCursor) diagnosticLexeme() Lexeme {
	if c.ready { return c.lookahead }
	return c.eofLexeme()
}

func (c *parserLexemeCursor) eofLexeme() Lexeme {
	if c.haveLast {
		return Lexeme{Token: TokenEOF, Start: c.last.End, End: c.last.End, StartLine: c.last.EndLine, StartColumn: c.last.EndColumn, EndLine: c.last.EndLine, EndColumn: c.last.EndColumn}
	}
	return Lexeme{Token: TokenEOF, StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 1}
}

func (c *parserLexemeCursor) normalizeEOFLexeme(lexeme Lexeme) Lexeme {
	fallback := c.eofLexeme()
	if lexeme.StartLine <= 0 { lexeme.StartLine = fallback.StartLine }
	if lexeme.StartColumn <= 0 { lexeme.StartColumn = fallback.StartColumn }
	if lexeme.EndLine <= 0 { lexeme.EndLine = fallback.EndLine }
	if lexeme.EndColumn <= 0 { lexeme.EndColumn = fallback.EndColumn }
	if lexeme.Start == 0 && lexeme.End == 0 && c.haveLast {
		lexeme.Start = fallback.Start
		lexeme.End = fallback.End
	}
	return lexeme
}

// Parse recognizes input with a parser configured by default options.
func Parse(input []Lexeme) error { return NewParser().Parse(input) }

// ParseFromLexemeSource recognizes lexemes pulled from source with a parser configured by default options.
func ParseFromLexemeSource(source LexemeSource) error { return NewParser().ParseFromLexemeSource(source) }

// ParseValue recognizes input and returns the final reduced semantic value.
func ParseValue(input []Lexeme) (Value, error) { return NewParser().ParseValue(input) }

// ParseValueFromLexemeSource recognizes lexemes pulled from source and returns the final reduced semantic value.
func ParseValueFromLexemeSource(source LexemeSource) (Value, error) {
	return NewParser().ParseValueFromLexemeSource(source)
}

// ParseRecovering recognizes input and returns all syntax diagnostics.
func ParseRecovering(input []Lexeme) (ParseResult, error) { return NewParser().ParseRecovering(input) }

// ParseRecoveringFromLexemeSource recognizes lexemes pulled from source and returns all syntax diagnostics.
func ParseRecoveringFromLexemeSource(source LexemeSource) (ParseResult, error) {
	return NewParser().ParseRecoveringFromLexemeSource(source)
}

// ParseWithReducer recognizes input using reducer for target-tagged grammar actions.
func ParseWithReducer(input []Lexeme, reducer Reducer) (Value, error) {
	return ParseWithReducerFromLexemeSource(newLexemeSliceSource(input), reducer)
}

// ParseWithReducerFromLexemeSource recognizes lexemes pulled from source using reducer for target-tagged grammar actions.
func ParseWithReducerFromLexemeSource(source LexemeSource, reducer Reducer) (Value, error) {
	return NewParser(WithReducer(reducer)).ParseValueFromLexemeSource(source)
}

// Parse recognizes input using this parser instance.
func (p *Parser) Parse(input []Lexeme) error { _, err := p.ParseValue(input); return err }

// ParseFromLexemeSource recognizes lexemes pulled from source using this parser instance.
func (p *Parser) ParseFromLexemeSource(source LexemeSource) error {
	_, err := p.ParseValueFromLexemeSource(source)
	return err
}

// ParseValue recognizes input using this parser instance and returns the final semantic value.
func (p *Parser) ParseValue(input []Lexeme) (Value, error) {
	return p.ParseValueFromLexemeSource(newLexemeSliceSource(input))
}

// ParseValueFromLexemeSource recognizes lexemes pulled from source using this parser instance.
func (p *Parser) ParseValueFromLexemeSource(source LexemeSource) (Value, error) {
	result, err := p.ParseRecoveringFromLexemeSource(source)
	if err != nil { return result.Value, err }
	if len(result.Diagnostics) > 0 { return result.Value, &ParseError{Diagnostics: result.Diagnostics} }
	return result.Value, nil
}

// ParseRecovering performs grammar-directed recovery and returns every syntax diagnostic.
func (p *Parser) ParseRecovering(input []Lexeme) (ParseResult, error) {
	return p.ParseRecoveringFromLexemeSource(newLexemeSliceSource(input))
}

// ParseRecoveringFromLexemeSource performs grammar-directed recovery over a pull lexeme source.
func (p *Parser) ParseRecoveringFromLexemeSource(source LexemeSource) (ParseResult, error) {
	result := ParseResult{}
	if p.reducer != nil {
		if validator, ok := p.reducer.(interface{ ValidateCoverage() error }); ok {
			if err := validator.ValidateCoverage(); err != nil { return result, err }
		}
	}
	cursor, err := newParserLexemeCursor(source)
	if err != nil { return result, err }
	states := []int{0}
	values := []Value{}
	recovering := 0
	activeDiagnostic := -1
	for {
		lookahead, err := cursor.peekSymbol()
		if err != nil { result.Value = parserCurrentValue(values); return result, err }
		action, ok := parserActions[states[len(states)-1]][lookahead]
		if !ok {
			if recovering == 0 {
				result.Diagnostics = append(result.Diagnostics, newParseDiagnostic(states[len(states)-1], lookahead, cursor))
				activeDiagnostic = len(result.Diagnostics) - 1
				recovered := false
				for len(states) > 0 {
					errorAction, canShiftError := parserActions[states[len(states)-1]]["error"]
					if canShiftError && errorAction.kind == "shift" {
						states = append(states, errorAction.state)
						values = append(values, parserErrorLexeme(cursor))
						recovering = 3
						result.Diagnostics[activeDiagnostic].Recovery.Kind = "shift-error"
						recovered = true
						break
					}
					if len(states) == 1 { break }
					states = states[:len(states)-1]
					if len(values) > 0 { values = values[:len(values)-1] }
				}
				if recovered { continue }
				result.Diagnostics[activeDiagnostic].Recovery.Kind = "abort"
				result.Value = parserCurrentValue(values)
				return result, nil
			}
			if lookahead == "$" {
				if activeDiagnostic >= 0 { result.Diagnostics[activeDiagnostic].Recovery.Kind = "abort" }
				result.Value = parserCurrentValue(values)
				return result, nil
			}
			if _, err := cursor.advance(); err != nil { result.Value = parserCurrentValue(values); return result, err }
			if activeDiagnostic >= 0 { result.Diagnostics[activeDiagnostic].Recovery.Discarded++ }
			continue
		}
		switch action.kind {
		case "shift":
			lexeme, err := cursor.advance()
			if err != nil { result.Value = parserCurrentValue(values); return result, fmt.Errorf("shift in state %d: %w", states[len(states)-1], err) }
			states = append(states, action.state)
			values = append(values, lexeme)
			if recovering > 0 {
				recovering--
				if recovering == 0 && activeDiagnostic >= 0 { result.Diagnostics[activeDiagnostic].Recovery.Kind = "recovered"; activeDiagnostic = -1 }
			}
		case "reduce":
			rule := parserRules[action.rule]
			if len(states) < rule.size + 1 { result.Value = parserCurrentValue(values); return result, fmt.Errorf("parser stack underflow reducing rule %d", action.rule) }
			if len(values) < rule.size { result.Value = parserCurrentValue(values); return result, fmt.Errorf("semantic value stack underflow reducing rule %d", action.rule) }
			rhs := append([]Value(nil), values[len(values)-rule.size:]...)
			values = values[:len(values)-rule.size]
			value, reduceErr := p.reduce(action.rule, rule, rhs)
			if reduceErr != nil { result.Value = parserCurrentValue(values); return result, reduceErr }
			states = states[:len(states)-rule.size]
			gotoState, exists := parserGotos[states[len(states)-1]][rule.lhs]
			if !exists { result.Value = parserCurrentValue(values); return result, fmt.Errorf("missing goto from state %d on %s", states[len(states)-1], rule.lhs) }
			states = append(states, gotoState)
			values = append(values, value)
		case "accept":
			if activeDiagnostic >= 0 { result.Diagnostics[activeDiagnostic].Recovery.Kind = "recovered" }
			result.Value = parserCurrentValue(values)
			result.Accepted = true
			return result, nil
		default:
			result.Value = parserCurrentValue(values)
			return result, fmt.Errorf("invalid parser action %q", action.kind)
		}
	}
}

`)
	b.WriteString("func (p *Parser) reduce(ruleID int, rule parserRule, values []Value) (Value, error) {\n\tctx := reductionContext(ruleID, rule, values)\n\tif p.reducer != nil && rule.action != SemanticActionNone {\n\t\treturn p.reducer.Reduce(ctx)\n\t}\n")
	if goMode == spec.SemanticModeInline {
		b.WriteString("\tif rule.action != SemanticActionNone {\n\t\treturn reduceInline(ctx)\n\t}\n")
	}
	b.WriteString("\treturn defaultReduce(values), nil\n}\n\n")
	b.WriteString("func reductionContext(ruleID int, rule parserRule, values []Value) Reduction {\n\treturn Reduction{Rule: ruleID, LHS: rule.lhs, RHS: append([]string(nil), rule.rhs...), Labels: append([]string(nil), rule.labels...), ActionID: rule.action, Action: rule.action.String(), Values: append([]Value(nil), values...)}\n}\n\n")
	if goMode == spec.SemanticModeInline {
		renderInlineReducers(&b, generatedFile, table)
	}
	b.WriteString("func defaultReduce(values []Value) Value {\n\tswitch len(values) {\n\tcase 0:\n\t\treturn nil\n\tcase 1:\n\t\treturn values[0]\n\tdefault:\n\t\treturn append([]Value(nil), values...)\n\t}\n}\n\n")
	b.WriteString("func parserCurrentValue(values []Value) Value {\n\tif len(values) == 0 { return nil }\n\treturn values[len(values)-1]\n}\n\n")
	b.WriteString("func newParseDiagnostic(state int, unexpected string, cursor *parserLexemeCursor) ParseDiagnostic {\n\tlexeme := cursor.diagnosticLexeme()\n\texpected := append([]ExpectedToken(nil), parserExpected[state]...)\n\tfor index := range expected { expected[index].Members = append([]string(nil), expected[index].Members...) }\n\treturn ParseDiagnostic{State: state, Unexpected: unexpected, UnexpectedDisplay: parserUnexpectedDisplay(unexpected), Expected: expected, Start: lexeme.Start, End: lexeme.End, StartLine: lexeme.StartLine, StartColumn: lexeme.StartColumn, EndLine: lexeme.EndLine, EndColumn: lexeme.EndColumn}\n}\n\n")
	b.WriteString("func parserErrorLexeme(cursor *parserLexemeCursor) Lexeme {\n\tlexeme := cursor.diagnosticLexeme()\n\tlexeme.Token = TokenError\n\tlexeme.Text = \"\"\n\tlexeme.Channel = \"\"\n\tlexeme.End = lexeme.Start\n\tlexeme.EndLine = lexeme.StartLine\n\tlexeme.EndColumn = lexeme.StartColumn\n\treturn lexeme\n}\n\n")
	b.WriteString("func parserUnexpectedDisplay(symbol string) string {\n\tif symbol == \"$\" { return \"end of input\" }\n\tif display, ok := parserTokenAliases[symbol]; ok { return display }\n\treturn symbol\n}\n\n")
	b.WriteString("func terminalName(tok Token) string {\n\tswitch tok {\n\tcase TokenEOF:\n\t\treturn \"$\"\n")
	for _, tok := range tokens {
		b.WriteString("\tcase Token" + tok + ":\n\t\treturn \"" + tok + "\"\n")
	}
	b.WriteString("\tdefault:\n\t\treturn \"ERROR\"\n\t}\n}\n\n")
	b.WriteString("type parserAction struct { kind string; state int; rule int }\n")
	b.WriteString("type parserRule struct { lhs string; rhs []string; labels []string; size int; action SemanticAction }\n\n")
	b.WriteString("var parserActions = map[int]map[string]parserAction{\n")
	actionStates := sortedActionStates(table.Actions)
	for _, state := range actionStates {
		b.WriteString(fmt.Sprintf("\t%d: {\n", state))
		syms := sortedActionSymbols(table.Actions[state])
		for _, sym := range syms {
			action := table.Actions[state][sym]
			if action.Kind == parse.ActionReduce {
				if rule, ok := ruleByID(table.Rules, action.Rule); ok {
					b.WriteString(indentComment(ruleSourceComment(rule, "go", "// "), "\t\t"))
				}
			}
			b.WriteString(fmt.Sprintf("\t\t%q: {kind: %q, state: %d, rule: %d},\n", sym, action.Kind, action.State, action.Rule))
		}
		b.WriteString("\t},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("var parserTokenAliases = map[string]string{\n")
	for _, alias := range project.Grammar.ExpectedTokens.Aliases {
		b.WriteString(fmt.Sprintf("\t%q: %q,\n", alias.Token, alias.Label))
	}
	b.WriteString("}\n\n")
	b.WriteString("var parserGotos = map[int]map[string]int{\n")
	gotoStates := sortedGotoStates(table.Gotos)
	for _, state := range gotoStates {
		b.WriteString(fmt.Sprintf("\t%d: {", state))
		syms := sortedGotoSymbols(table.Gotos[state])
		for _, sym := range syms {
			b.WriteString(fmt.Sprintf("%q: %d,", sym, table.Gotos[state][sym]))
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("var parserExpected = map[int][]ExpectedToken{\n")
	for _, state := range sortedExpectedStates(table.Expected) {
		b.WriteString(fmt.Sprintf("\t%d: {", state))
		for _, expected := range table.Expected[state] {
			b.WriteString(fmt.Sprintf("{Symbol: %q, Display: %q, Members: %s},", expected.Symbol, expected.Display, renderStringSlice(expected.Members)))
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n\n")
	b.WriteString("var parserRules = map[int]parserRule{\n")
	for _, rule := range table.Rules {
		b.WriteString(indentComment(ruleSourceComment(rule, "go", "// "), "\t"))
		b.WriteString(fmt.Sprintf("\t%d: {lhs: %q, rhs: %s, labels: %s, size: %d, action: %s},\n", rule.ID, rule.LHS, renderStringSlice(rule.RHS), renderStringSlice(rule.Labels), len(rule.RHS), semanticActionExpr(rule.Actions["go"], actionIDs)))
	}
	b.WriteString("}\n")
	return b.String()
}

func renderTypedReductionContexts(b *strings.Builder, manifest action.Manifest, actions []SemanticAction) {
	if len(manifest.Actions) == 0 {
		return
	}
	constants := semanticActionIDs(actions)
	b.WriteString("// semanticValueAs reads and type-checks one named reduction value.\n")
	b.WriteString("func semanticValueAs[T any](ctx Reduction, label string) (T, error) {\n\tvar zero T\n\tvalue, err := ctx.ValueFor(label)\n\tif err != nil { return zero, err }\n\ttyped, ok := value.(T)\n\tif !ok { return zero, fmt.Errorf(\"rule %d action %q label %q has type %T, want %T\", ctx.Rule, ctx.Action, label, value, zero) }\n\treturn typed, nil\n}\n\n")
	for _, semanticAction := range manifest.Actions {
		if !semanticAction.Typed || len(semanticAction.Rules) == 0 {
			continue
		}
		constant := constants[semanticAction.Name]
		if constant == "" {
			continue
		}
		suffix := strings.TrimPrefix(constant, "SemanticAction")
		contextType := suffix + "Reduction"
		handlerType := suffix + "Handler"
		converter := contextType + "From"
		adapter := "Typed" + suffix
		fields := typedFields(semanticAction.Rules[0])

		b.WriteString(fmt.Sprintf("// %s is the generated typed context for the %q semantic action.\n", contextType, semanticAction.Name))
		b.WriteString("type " + contextType + " struct {\n\tReduction Reduction\n")
		for _, field := range fields {
			b.WriteString(fmt.Sprintf("\t%s %s\n", field.Name, field.Type))
		}
		b.WriteString("}\n\n")
		b.WriteString(fmt.Sprintf("// %s validates and converts an untyped reduction context.\n", converter))
		b.WriteString(fmt.Sprintf("func %s(ctx Reduction) (%s, error) {\n", converter, contextType))
		b.WriteString(fmt.Sprintf("\tif ctx.ActionID != %s { return %s{}, fmt.Errorf(\"typed context %s requires action %%s, got %%s\", %s, ctx.ActionID) }\n", constant, contextType, contextType, constant))
		b.WriteString(fmt.Sprintf("\tresult := %s{Reduction: ctx}\n", contextType))
		for _, field := range fields {
			b.WriteString(fmt.Sprintf("\t%s, err := semanticValueAs[%s](ctx, %q)\n", field.Local, field.Type, field.Label))
			b.WriteString(fmt.Sprintf("\tif err != nil { return %s{}, err }\n", contextType))
			b.WriteString(fmt.Sprintf("\tresult.%s = %s\n", field.Name, field.Local))
		}
		b.WriteString("\treturn result, nil\n}\n\n")
		b.WriteString(fmt.Sprintf("// %s handles one typed %q reduction.\n", handlerType, semanticAction.Name))
		b.WriteString(fmt.Sprintf("type %s func(%s) (%s, error)\n\n", handlerType, contextType, semanticAction.ReturnType))
		b.WriteString(fmt.Sprintf("// %s adapts a typed handler to ReductionHandler.\n", adapter))
		b.WriteString(fmt.Sprintf("func %s(handler %s) ReductionHandler {\n", adapter, handlerType))
		b.WriteString(fmt.Sprintf("\treturn func(ctx Reduction) (Value, error) {\n\t\ttyped, err := %s(ctx)\n\t\tif err != nil { return nil, err }\n\t\treturn handler(typed)\n\t}\n}\n\n", converter))
	}
}

type typedField struct {
	Label string
	Name  string
	Local string
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
			Local: "value" + name,
			Type:  operand.Type,
		})
	}
	return fields
}

func renderInlineReducers(b *strings.Builder, generatedFile string, table *parse.Table) {
	b.WriteString("func reduceInline(ctx Reduction) (Value, error) {\n\tswitch ctx.Rule {\n")
	for _, rule := range table.Rules {
		action := strings.TrimSpace(rule.Actions["go"])
		if action == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("\tcase %d:\n", rule.ID))
		if directive := goLineDirective(rule.Span); directive != "" {
			b.WriteString(directive)
		}
		for _, line := range strings.Split(action, "\n") {
			trimmed := strings.TrimRight(line, " \t")
			if strings.TrimSpace(trimmed) == "" {
				continue
			}
			b.WriteString("\t\t" + trimmed + "\n")
		}
		if directive := generatedLineDirective(generatedFile); directive != "" {
			b.WriteString(directive)
		}
	}
	b.WriteString("\t}\n\treturn defaultReduce(ctx.Values), nil\n}\n\n")
}

func renderStringSlice(values []string) string {
	if len(values) == 0 {
		return "nil"
	}
	var b strings.Builder
	b.WriteString("[]string{")
	for _, value := range values {
		b.WriteString(fmt.Sprintf("%q,", value))
	}
	b.WriteString("}")
	return b.String()
}

func generatedHeader(pkg string, source string) string {
	var b strings.Builder
	b.WriteString("// Code generated by lang-forge; DO NOT EDIT.\n")
	if source != "" {
		b.WriteString("// Source: " + source + "\n")
	}
	b.WriteString("\npackage " + pkg + "\n\n")
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
	return fmt.Sprintf("%s:%d:%d", sanitizeLineDirectiveFile(span.File), span.Start.Line, column)
}

func goLineDirective(span diagnostics.Span) string {
	if span.File == "" || span.Start.Line <= 0 {
		return ""
	}
	column := span.Start.Column
	if column <= 0 {
		column = 1
	}
	return fmt.Sprintf("//line %s:%d:%d\n", sanitizeLineDirectiveFile(span.File), span.Start.Line, column)
}

func generatedLineDirective(filename string) string {
	if filename == "" {
		return ""
	}
	return fmt.Sprintf("//line %s:1:1\n", sanitizeLineDirectiveFile(filename))
}

func sanitizeLineDirectiveFile(filename string) string {
	filename = strings.ReplaceAll(filename, "\r", "_")
	filename = strings.ReplaceAll(filename, "\n", "_")
	return filename
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeGoFile(path string, source string) error {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return fmt.Errorf("format generated %s: %w", filepath.Base(path), err)
	}
	return os.WriteFile(path, formatted, 0o644)
}

func goPackageName(specPackage string, outDirBase string) (string, error) {
	if specPackage != "" {
		if !isValidGoPackageName(specPackage) {
			return "", fmt.Errorf("invalid Go package name %q", specPackage)
		}
		return specPackage, nil
	}
	pkg := sanitizePackage(outDirBase)
	if pkg == "" {
		return "", errors.New("could not determine Go package name")
	}
	if !isValidGoPackageName(pkg) {
		pkg = "langforge_" + pkg
	}
	return pkg, nil
}

func sanitizePackage(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r == '-' {
			r = '_'
		}
		if b.Len() == 0 {
			if isGoIdentStart(r) {
				b.WriteRune(r)
			}
			continue
		}
		if isGoIdentPart(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isValidGoPackageName(name string) bool {
	if name == "" || name == "_" || goKeywords[name] {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !isGoIdentStart(r) {
				return false
			}
			continue
		}
		if !isGoIdentPart(r) {
			return false
		}
	}
	return true
}

func isGoIdentStart(r rune) bool {
	return r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func isGoIdentPart(r rune) bool {
	return isGoIdentStart(r) || (r >= '0' && r <= '9')
}

var goKeywords = map[string]bool{
	"break":       true,
	"default":     true,
	"func":        true,
	"interface":   true,
	"select":      true,
	"case":        true,
	"defer":       true,
	"go":          true,
	"map":         true,
	"struct":      true,
	"chan":        true,
	"else":        true,
	"goto":        true,
	"package":     true,
	"switch":      true,
	"const":       true,
	"fallthrough": true,
	"if":          true,
	"range":       true,
	"type":        true,
	"continue":    true,
	"for":         true,
	"import":      true,
	"return":      true,
	"var":         true,
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
