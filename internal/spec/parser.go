package spec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/parseralgo"
)

var (
	identRE     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	yaccTokenRE = regexp.MustCompile(`YACC_([A-Za-z_][A-Za-z0-9_]*)`)
)

// ParseCombined reads the modern single-file .lf specification format.
func ParseCombined(data []byte, filename string) (*Spec, diagnostics.List) {
	src := diagnostics.NewSource(filename, data)
	clean := stripBlockComments(src.Text)
	lines := splitLinesWithOffsets(clean)
	spec := &Spec{SourceFile: filename, Mode: ModeCombined, Scanner: DefaultScanner()}
	var diags diagnostics.List
	section := ""
	var lexerLines []linePart
	var parserLines []linePart
	for _, line := range lines {
		trimmed := strings.TrimSpace(line.Text)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "%%") {
			fields := strings.Fields(trimmed)
			if len(fields) < 2 {
				diags.AddError("LF001", "section marker must be `%% lexer` or `%% parser`", src.Span(line.Offset, line.Offset+len(line.Text)))
				continue
			}
			switch strings.ToLower(fields[1]) {
			case "lexer", "lex":
				section = "lexer"
			case "parser", "grammar", "yacc":
				section = "parser"
			default:
				diags.AddError("LF002", "unknown section `"+fields[1]+"`", src.Span(line.Offset, line.Offset+len(line.Text)))
			}
			continue
		}
		switch section {
		case "":
			parseDirective(spec, trimmed, src.Span(line.Offset, line.Offset+len(line.Text)), &diags)
		case "lexer":
			lexerLines = append(lexerLines, line)
		case "parser":
			parserLines = append(parserLines, line)
		}
	}
	spec.Lexer = parseLexerLines(lexerLines, src, &diags)
	spec.Grammar.Rules = parseGrammarLines(parserLines, src, &diags)
	if spec.Grammar.Start == "" && len(spec.Grammar.Rules) > 0 {
		spec.Grammar.Start = spec.Grammar.Rules[0].Name
	}
	return spec, diags
}

// ParseLex reads the legacy Pascal-oriented lex half used by UCDT examples.
func ParseLex(data []byte, filename string) (*Spec, diagnostics.List) {
	src := diagnostics.NewSource(filename, data)
	parts := splitPercentSections(stripBlockComments(src.Text))
	spec := &Spec{SourceFile: filename, Mode: ModeLexOnly, Scanner: DefaultScanner()}
	var diags diagnostics.List
	if len(parts) < 2 {
		diags.AddError("LF010", "lex file must contain at least one `%%` separator", src.Span(0, 0))
		return spec, diags
	}
	spec.Lexer.Definitions = parseLexDefinitions(parts[0].Text, src, parts[0].Offset, &diags)
	spec.Lexer.Rules = parseLegacyLexRules(parts[1].Text, src, parts[1].Offset, &diags)
	return spec, diags
}

// ParseYacc reads the legacy Pascal-oriented yacc half used by UCDT examples.
func ParseYacc(data []byte, filename string) (*Spec, diagnostics.List) {
	src := diagnostics.NewSource(filename, data)
	parts := splitPercentSections(stripBlockComments(src.Text))
	spec := &Spec{SourceFile: filename, Mode: ModeYaccOnly, Scanner: DefaultScanner()}
	var diags diagnostics.List
	if len(parts) < 2 {
		diags.AddError("LF020", "yacc file must contain at least one `%%` separator", src.Span(0, 0))
		return spec, diags
	}
	for _, line := range splitLinesWithOffsets(parts[0].Text) {
		trimmed := strings.TrimSpace(line.Text)
		if trimmed == "" {
			continue
		}
		parseDirective(spec, trimmed, src.Span(parts[0].Offset+line.Offset, parts[0].Offset+line.Offset+len(line.Text)), &diags)
	}
	spec.Grammar.Rules = parseGrammarText(stripHashBlocks(parts[1].Text), src, parts[1].Offset, &diags)
	if spec.Grammar.Start == "" && len(spec.Grammar.Rules) > 0 {
		spec.Grammar.Start = spec.Grammar.Rules[0].Name
	}
	return spec, diags
}

// MergeSplit combines separately parsed lex and yacc files into one Spec.
func MergeSplit(lexSpec, yaccSpec *Spec) *Spec {
	out := *yaccSpec
	out.Mode = ModeSplit
	out.Lexer = lexSpec.Lexer
	out.SourceFile = lexSpec.SourceFile + "+" + yaccSpec.SourceFile
	out.Semantics = mergeSemantics(lexSpec.Semantics, yaccSpec.Semantics)
	if out.Scanner.Encoding == "" {
		out.Scanner = lexSpec.Scanner
	}
	out.Scanner = out.Scanner.WithDefaults()
	return &out
}

func mergeSemantics(lexSemantics, yaccSemantics SemanticSpec) SemanticSpec {
	out := yaccSemantics
	if len(lexSemantics.Includes) > 0 {
		out.Includes = append(append([]SemanticInclude(nil), lexSemantics.Includes...), out.Includes...)
	}
	if len(lexSemantics.Modes) > 0 {
		modes := map[string]SemanticActionMode{}
		for target, mode := range lexSemantics.Modes {
			modes[target] = mode
		}
		for target, mode := range yaccSemantics.Modes {
			modes[target] = mode
		}
		out.Modes = modes
	}
	return out
}

func parseDirective(spec *Spec, line string, span diagnostics.Span, diags *diagnostics.List) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	switch strings.ToLower(fields[0]) {
	case "%target":
		if len(fields) != 2 {
			diags.AddError("LF101", "%target expects one value", span)
			return
		}
		spec.Target = fields[1]
	case "%package":
		if len(fields) != 2 {
			diags.AddError("LF102", "%package expects one value", span)
			return
		}
		spec.Package = fields[1]
	case "%start":
		if len(fields) != 2 {
			diags.AddError("LF103", "%start expects one symbol", span)
			return
		}
		spec.Grammar.Start = fields[1]
	case "%token":
		if len(fields) < 2 {
			diags.AddError("LF104", "%token expects at least one token name", span)
			return
		}
		for _, name := range fields[1:] {
			if !identRE.MatchString(name) {
				diags.AddError("LF105", "invalid token name `"+name+"`", span)
				continue
			}
			spec.Tokens = append(spec.Tokens, TokenDecl{Name: name, Span: span})
		}
	case "%type":
		if len(fields) != 2 {
			diags.AddError("LF106", "%type expects one parser algorithm: "+parseralgo.Allowed(), span)
			return
		}
		if algorithm, ok := parseralgo.Parse(fields[1]); ok {
			spec.Grammar.Algorithm = algorithm
		} else {
			diags.AddError("LF107", "unknown parser algorithm `"+fields[1]+"`; expected "+parseralgo.Allowed(), span)
		}
	case "%scanner":
		scanner, ok := parseScannerDirective(fields[1:], span, diags)
		if ok {
			spec.Scanner = scanner
		}
	case "%semantic":
		parseSemanticDirective(spec, fields[1:], span, diags)
	default:
		diags.AddError("LF100", "unknown directive `"+fields[0]+"`", span)
	}
}

func parseSemanticDirective(spec *Spec, fields []string, span diagnostics.Span, diags *diagnostics.List) {
	if len(fields) < 3 {
		diags.AddError("LF140", "%semantic expects `target mode reducer|inline` or `target import [alias] path`", span)
		return
	}
	target := strings.ToLower(fields[0])
	if !identRE.MatchString(target) {
		diags.AddError("LF141", "invalid semantic target `"+fields[0]+"`", span)
		return
	}
	switch strings.ToLower(fields[1]) {
	case "mode":
		if len(fields) != 3 {
			diags.AddError("LF140", "%semantic mode expects exactly one mode: reducer or inline", span)
			return
		}
		mode := SemanticActionMode(strings.ToLower(fields[2]))
		if mode != SemanticModeReducer && mode != SemanticModeInline {
			diags.AddError("LF142", "unknown semantic action mode `"+fields[2]+"`; expected reducer or inline", span)
			return
		}
		if spec.Semantics.Modes == nil {
			spec.Semantics.Modes = map[string]SemanticActionMode{}
		}
		spec.Semantics.Modes[target] = mode
	case "import", "include", "use":
		include, ok := parseSemanticInclude(target, fields[2:], span, diags)
		if ok {
			spec.Semantics.Includes = append(spec.Semantics.Includes, include)
		}
	default:
		diags.AddError("LF140", "unknown %semantic command `"+fields[1]+"`; expected mode or import", span)
	}
}

func parseSemanticInclude(target string, fields []string, span diagnostics.Span, diags *diagnostics.List) (SemanticInclude, bool) {
	if len(fields) != 1 && len(fields) != 2 {
		diags.AddError("LF143", "%semantic import expects `path` or `alias path`", span)
		return SemanticInclude{}, false
	}
	include := SemanticInclude{Target: target, Span: span}
	if len(fields) == 2 {
		include.Alias = fields[0]
		if !identRE.MatchString(include.Alias) && include.Alias != "_" && include.Alias != "." {
			diags.AddError("LF144", "invalid semantic import alias `"+include.Alias+"`", span)
			return SemanticInclude{}, false
		}
		include.Path = fields[1]
	} else {
		include.Path = fields[0]
	}
	path, err := unquoteSemanticPath(include.Path)
	if err != nil {
		diags.AddError("LF145", err.Error(), span)
		return SemanticInclude{}, false
	}
	include.Path = path
	return include, true
}

func unquoteSemanticPath(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("semantic import path is empty")
	}
	if strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "`") {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid semantic import path %s", value)
		}
		value = unquoted
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("semantic import path is empty")
	}
	return value, nil
}

func parseScannerDirective(fields []string, span diagnostics.Span, diags *diagnostics.List) (ScannerSpec, bool) {
	scanner := DefaultScanner()
	if len(fields) == 0 {
		diags.AddError("LF108", "%scanner expects `utf8` or key=value settings", span)
		return scanner, false
	}
	if len(fields) == 1 && !strings.Contains(fields[0], "=") {
		scanner.Encoding = ScannerEncoding(strings.ToLower(fields[0]))
		return validateScannerDirective(scanner, span, diags)
	}
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok || key == "" || value == "" {
			diags.AddError("LF108", "%scanner settings must use key=value form", span)
			return scanner, false
		}
		switch strings.ToLower(key) {
		case "encoding":
			scanner.Encoding = ScannerEncoding(strings.ToLower(value))
		case "invalid":
			scanner.Invalid = ScannerInvalidPolicy(strings.ToLower(value))
		case "newline":
			scanner.Newline = strings.ToLower(value)
		default:
			diags.AddError("LF108", "unknown %scanner setting `"+key+"`", span)
			return scanner, false
		}
	}
	return validateScannerDirective(scanner, span, diags)
}

func validateScannerDirective(scanner ScannerSpec, span diagnostics.Span, diags *diagnostics.List) (ScannerSpec, bool) {
	scanner = scanner.WithDefaults()
	if scanner.Encoding != ScannerEncodingUTF8 {
		diags.AddError("LF109", "unsupported scanner encoding `"+string(scanner.Encoding)+"`; supported: utf8", span)
		return scanner, false
	}
	if scanner.Invalid != ScannerInvalidError {
		diags.AddError("LF109", "unsupported scanner invalid-input policy `"+string(scanner.Invalid)+"`; supported: error", span)
		return scanner, false
	}
	if scanner.Newline != "" && scanner.Newline != "lf" {
		diags.AddError("LF109", "unsupported scanner newline policy `"+scanner.Newline+"`; supported: lf", span)
		return scanner, false
	}
	return scanner, true
}

func parseLexerLines(lines []linePart, src diagnostics.Source, diags *diagnostics.List) LexerSpec {
	var lex LexerSpec
	for _, line := range lines {
		trimmed := strings.TrimSpace(line.Text)
		if trimmed == "" {
			continue
		}
		span := src.Span(line.Offset, line.Offset+len(line.Text))
		if strings.Contains(trimmed, "=>") {
			rule, ok := parseModernLexRule(trimmed, span, diags)
			if ok {
				lex.Rules = append(lex.Rules, rule)
			}
			continue
		}
		if strings.Contains(trimmed, "=") {
			def, ok := parseLexDefinitionLine(trimmed, span, diags)
			if ok {
				lex.Definitions = append(lex.Definitions, def)
			}
			continue
		}
		diags.AddError("LF110", "expected lexer definition `NAME = pattern;` or rule `pattern => action`", span)
	}
	return lex
}

func parseLexDefinitions(text string, src diagnostics.Source, base int, diags *diagnostics.List) []LexDefinition {
	var defs []LexDefinition
	for _, stmt := range splitStatements(text, ';') {
		trimmed := strings.TrimSpace(stmt.Text)
		if trimmed == "" {
			continue
		}
		span := src.Span(base+stmt.Offset, base+stmt.Offset+len(stmt.Text))
		def, ok := parseLexDefinitionLine(trimmed+";", span, diags)
		if ok {
			defs = append(defs, def)
		}
	}
	return defs
}

func parseLexDefinitionLine(line string, span diagnostics.Span, diags *diagnostics.List) (LexDefinition, bool) {
	line = strings.TrimSuffix(strings.TrimSpace(line), ";")
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		diags.AddError("LF111", "lexer definition must contain `=`", span)
		return LexDefinition{}, false
	}
	name := strings.TrimSpace(parts[0])
	pattern := strings.TrimSpace(parts[1])
	if !identRE.MatchString(name) {
		diags.AddError("LF112", "invalid lexer definition name `"+name+"`", span)
		return LexDefinition{}, false
	}
	if pattern == "" {
		diags.AddError("LF113", "lexer definition `"+name+"` has empty pattern", span)
		return LexDefinition{}, false
	}
	return LexDefinition{Name: name, Pattern: pattern, Span: span}, true
}

func parseModernLexRule(line string, span diagnostics.Span, diags *diagnostics.List) (LexRule, bool) {
	parts := strings.SplitN(line, "=>", 2)
	if len(parts) != 2 {
		diags.AddError("LF120", "lexer rule must contain `=>`", span)
		return LexRule{}, false
	}
	pattern := strings.TrimSpace(parts[0])
	actionText := strings.TrimSpace(strings.TrimSuffix(parts[1], ";"))
	action, ok := parseLexAction(actionText, span, diags)
	if !ok {
		return LexRule{}, false
	}
	if pattern == "" {
		diags.AddError("LF121", "lexer rule has empty pattern", span)
		return LexRule{}, false
	}
	return LexRule{Pattern: pattern, Action: action, Span: span}, true
}

func parseLegacyLexRules(text string, src diagnostics.Source, base int, diags *diagnostics.List) []LexRule {
	var rules []LexRule
	for _, block := range splitLegacyRuleBlocks(text) {
		trimmed := strings.TrimSpace(block.Text)
		if trimmed == "" {
			continue
		}
		idx := indexLegacyLexRuleColon(trimmed)
		span := src.Span(base+block.Offset, base+block.Offset+len(block.Text))
		if idx < 0 {
			diags.AddError("LF122", "legacy lexer rule must contain `:`", span)
			continue
		}
		pattern := strings.TrimSpace(trimmed[:idx])
		raw := strings.TrimSpace(trimmed[idx+1:])
		if pattern == "" {
			diags.AddError("LF123", "legacy lexer rule has empty pattern", span)
			continue
		}
		action := legacyLexAction(raw)
		rules = append(rules, LexRule{Pattern: pattern, Action: action, Span: span})
	}
	return rules
}

func legacyLexAction(raw string) LexAction {
	if strings.Contains(raw, "LEX_Skip") || strings.Contains(raw, "AReturn := False") || strings.Contains(raw, "AReturn:=False") {
		return LexAction{Kind: ActionSkip, Raw: raw}
	}
	matches := yaccTokenRE.FindStringSubmatch(raw)
	if len(matches) == 2 {
		return LexAction{Kind: ActionToken, Token: matches[1], Raw: raw}
	}
	return LexAction{Kind: ActionRaw, Raw: raw}
}

func parseLexAction(text string, span diagnostics.Span, diags *diagnostics.List) (LexAction, bool) {
	switch {
	case text == "skip":
		return LexAction{Kind: ActionSkip}, true
	case strings.HasPrefix(text, "token(") && strings.HasSuffix(text, ")"):
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "token("), ")"))
		if !identRE.MatchString(name) {
			diags.AddError("LF124", "invalid token action name `"+name+"`", span)
			return LexAction{}, false
		}
		return LexAction{Kind: ActionToken, Token: name}, true
	case strings.HasPrefix(text, "channel(") && strings.HasSuffix(text, ")"):
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "channel("), ")"))
		if !identRE.MatchString(name) {
			diags.AddError("LF125", "invalid channel name `"+name+"`", span)
			return LexAction{}, false
		}
		return LexAction{Kind: ActionChannel, Channel: name}, true
	default:
		if strings.HasPrefix(text, "{") {
			return LexAction{Kind: ActionRaw, Raw: text}, true
		}
		diags.AddError("LF126", "unknown lexer action `"+text+"`", span)
		return LexAction{}, false
	}
}

func parseGrammarLines(lines []linePart, src diagnostics.Source, diags *diagnostics.List) []RuleSpec {
	var b strings.Builder
	offset := -1
	for _, line := range lines {
		if offset < 0 {
			offset = line.Offset
		}
		b.WriteString(line.Text)
		b.WriteByte('\n')
	}
	if offset < 0 {
		return nil
	}
	return parseGrammarText(b.String(), src, offset, diags)
}

func parseGrammarText(text string, src diagnostics.Source, base int, diags *diagnostics.List) []RuleSpec {
	var rules []RuleSpec
	for _, stmt := range splitStatements(text, ';') {
		trimmed := strings.TrimSpace(stmt.Text)
		if trimmed == "" {
			continue
		}
		span := src.Span(base+stmt.Offset, base+stmt.Offset+len(stmt.Text))
		rule, ok := parseRuleStatement(trimmed, span, diags)
		if ok {
			rules = append(rules, rule)
		}
	}
	return rules
}

func parseRuleStatement(stmt string, span diagnostics.Span, diags *diagnostics.List) (RuleSpec, bool) {
	idx := indexTopLevel(stmt, ':')
	if idx < 0 {
		diags.AddError("LF130", "grammar rule must contain `:`", span)
		return RuleSpec{}, false
	}
	name := strings.TrimSpace(stmt[:idx])
	if !identRE.MatchString(name) {
		diags.AddError("LF131", "invalid rule name `"+name+"`", span)
		return RuleSpec{}, false
	}
	rhs := strings.TrimSpace(stmt[idx+1:])
	parts := splitTopLevel(rhs, '|')
	if len(parts) == 0 {
		diags.AddError("LF132", "rule `"+name+"` has no alternatives", span)
		return RuleSpec{}, false
	}
	rule := RuleSpec{Name: name, Span: span}
	for _, part := range parts {
		altText := strings.TrimSpace(part)
		actions := map[string]string{}
		altText, actions = extractActionBlocks(altText)
		symbols := strings.Fields(altText)
		if len(symbols) == 1 && (symbols[0] == "e" || symbols[0] == "ε" || symbols[0] == "%empty") {
			symbols = nil
		}
		for _, sym := range symbols {
			if !identRE.MatchString(sym) {
				diags.AddError("LF133", fmt.Sprintf("invalid grammar symbol `%s` in rule `%s`", sym, name), span)
				return RuleSpec{}, false
			}
		}
		rule.Alternatives = append(rule.Alternatives, Alternative{Symbols: symbols, Actions: actions, Span: span})
	}
	return rule, true
}

func extractActionBlocks(text string) (string, map[string]string) {
	actions := map[string]string{}
	for {
		start := strings.LastIndex(text, "{")
		if start < 0 {
			return strings.TrimSpace(text), actions
		}
		end := strings.LastIndex(text, "}")
		if end < start {
			return strings.TrimSpace(text), actions
		}
		body := strings.TrimSpace(text[start+1 : end])
		if idx := strings.Index(body, ":"); idx > 0 {
			lang := strings.TrimSpace(body[:idx])
			actions[lang] = strings.TrimSpace(body[idx+1:])
			text = strings.TrimSpace(text[:start] + " " + text[end+1:])
			continue
		}
		return strings.TrimSpace(text), actions
	}
}

type linePart struct {
	Text   string
	Offset int
}

func splitLinesWithOffsets(text string) []linePart {
	var out []linePart
	offset := 0
	for len(text) > 0 {
		idx := strings.IndexByte(text, '\n')
		if idx < 0 {
			out = append(out, linePart{Text: text, Offset: offset})
			break
		}
		out = append(out, linePart{Text: strings.TrimSuffix(text[:idx], "\r"), Offset: offset})
		text = text[idx+1:]
		offset += idx + 1
	}
	return out
}

func splitPercentSections(text string) []linePart {
	var out []linePart
	start := 0
	depthBracket := 0
	var quote byte
	escaped := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\\' {
			i++
			continue
		}
		switch c {
		case '\'', '"':
			if depthBracket == 0 {
				quote = c
			}
		case '[':
			depthBracket++
		case ']':
			if depthBracket > 0 {
				depthBracket--
			}
		case '%':
			if depthBracket == 0 && i+1 < len(text) && text[i+1] == '%' {
				out = append(out, linePart{Text: text[start:i], Offset: start})
				start = i + 2
				i++
			}
		}
	}
	out = append(out, linePart{Text: text[start:], Offset: start})
	return out
}

func splitStatements(text string, delim rune) []linePart {
	var out []linePart
	start := 0
	depthParen, depthBrace, depthBracket := 0, 0, 0
	var quote rune
	escaped := false
	for i, r := range text {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '\'', '"':
			if depthBracket == 0 {
				quote = r
			}
		case '(':
			depthParen++
		case ')':
			depthParen--
		case '{':
			depthBrace++
		case '}':
			depthBrace--
		case '[':
			depthBracket++
		case ']':
			depthBracket--
		default:
			if r == delim && depthParen == 0 && depthBrace == 0 && depthBracket == 0 {
				out = append(out, linePart{Text: text[start:i], Offset: start})
				start = i + len(string(r))
			}
		}
	}
	out = append(out, linePart{Text: text[start:], Offset: start})
	return out
}

func splitTopLevel(text string, delim rune) []string {
	var out []string
	start := 0
	depthParen, depthBrace, depthBracket := 0, 0, 0
	var quote rune
	escaped := false
	for i, r := range text {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '\'', '"':
			if depthBracket == 0 {
				quote = r
			}
		case '(':
			depthParen++
		case ')':
			depthParen--
		case '{':
			depthBrace++
		case '}':
			depthBrace--
		case '[':
			depthBracket++
		case ']':
			depthBracket--
		default:
			if r == delim && depthParen == 0 && depthBrace == 0 && depthBracket == 0 {
				out = append(out, strings.TrimSpace(text[start:i]))
				start = i + len(string(r))
			}
		}
	}
	out = append(out, strings.TrimSpace(text[start:]))
	return out
}

func indexTopLevel(text string, target rune) int {
	depthParen, depthBrace, depthBracket := 0, 0, 0
	var quote rune
	escaped := false
	for i, r := range text {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '(':
			depthParen++
		case ')':
			depthParen--
		case '{':
			depthBrace++
		case '}':
			depthBrace--
		case '[':
			depthBracket++
		case ']':
			depthBracket--
		default:
			if r == target && depthParen == 0 && depthBrace == 0 && depthBracket == 0 {
				return i
			}
		}
	}
	return -1
}

func indexLegacyLexRuleColon(text string) int {
	depthParen, depthBrace, depthBracket := 0, 0, 0
	var quote rune
	escaped := false
	for i, r := range text {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '(':
			depthParen++
		case ')':
			depthParen--
		case '{':
			depthBrace++
		case '}':
			depthBrace--
		case '[':
			depthBracket++
		case ']':
			depthBracket--
		case ':':
			if depthParen == 0 && depthBrace == 0 && depthBracket == 0 {
				return i
			}
		}
	}
	return -1
}

func stripBlockComments(text string) string {
	var b strings.Builder
	var quote byte
	depthBracket := 0
	escaped := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			b.WriteByte(c)
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\\' {
			b.WriteByte(c)
			if i+1 < len(text) {
				i++
				b.WriteByte(text[i])
			}
			continue
		}
		if c == '\'' || c == '"' {
			if depthBracket == 0 {
				quote = c
			}
			b.WriteByte(c)
			continue
		}
		if c == '[' {
			depthBracket++
			b.WriteByte(c)
			continue
		}
		if c == ']' {
			if depthBracket > 0 {
				depthBracket--
			}
			b.WriteByte(c)
			continue
		}
		if depthBracket == 0 && i+1 < len(text) && c == '/' && text[i+1] == '*' {
			b.WriteString("  ")
			i += 2
			for i < len(text) {
				if i+1 < len(text) && text[i] == '*' && text[i+1] == '/' {
					b.WriteString("  ")
					i++
					break
				}
				if text[i] == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
				i++
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func stripHashBlocks(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); {
		if i+1 < len(text) && text[i] == '#' && text[i+1] == '{' {
			b.WriteString("  ")
			i += 2
			for i < len(text) {
				if i+1 < len(text) && text[i] == '#' && text[i+1] == '}' {
					b.WriteString("  ")
					i += 2
					break
				}
				if text[i] == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
				i++
			}
			continue
		}
		b.WriteByte(text[i])
		i++
	}
	return b.String()
}

func splitLegacyRuleBlocks(text string) []linePart {
	var out []linePart
	lines := splitLinesWithOffsets(text)
	var b strings.Builder
	start := -1
	inAction := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line.Text)
		if trimmed == "" {
			if b.Len() > 0 && !inAction {
				out = append(out, linePart{Text: b.String(), Offset: start})
				b.Reset()
				start = -1
			}
			continue
		}
		if start < 0 {
			start = line.Offset
		}
		b.WriteString(line.Text)
		b.WriteByte('\n')
		if strings.Contains(trimmed, "#{") {
			inAction = true
		}
		if strings.Contains(trimmed, "#}") {
			inAction = false
			out = append(out, linePart{Text: b.String(), Offset: start})
			b.Reset()
			start = -1
			continue
		}
		if !inAction && strings.HasSuffix(trimmed, ";") {
			out = append(out, linePart{Text: b.String(), Offset: start})
			b.Reset()
			start = -1
		}
	}
	if b.Len() > 0 {
		out = append(out, linePart{Text: b.String(), Offset: start})
	}
	return out
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdent(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
