package lex

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ExprKind identifies one node kind in the regular-expression AST.
type ExprKind string

const (
	ExprEmpty  ExprKind = "empty"
	ExprSet    ExprKind = "set"
	ExprConcat ExprKind = "concat"
	ExprAlt    ExprKind = "alt"
	ExprStar   ExprKind = "star"
	ExprPlus   ExprKind = "plus"
	ExprOpt    ExprKind = "optional"
	ExprRef    ExprKind = "ref"
)

// Expr is the regular-expression AST used before NFA construction.
type Expr struct {
	Kind  ExprKind
	Set   RangeSet
	Name  string
	Left  *Expr
	Right *Expr
	Child *Expr
}

// Empty constructs an expression that accepts the empty string.
func Empty() *Expr { return &Expr{Kind: ExprEmpty} }

// SetExpr constructs an expression that accepts one rune from set.
func SetExpr(set RangeSet) *Expr { return &Expr{Kind: ExprSet, Set: set.Normalize()} }

// Ref constructs a reference to a named lexer definition.
func Ref(name string) *Expr { return &Expr{Kind: ExprRef, Name: name} }

// Concat constructs an expression that accepts a followed by b.
func Concat(a, b *Expr) *Expr { return &Expr{Kind: ExprConcat, Left: a, Right: b} }

// Alt constructs an expression that accepts either a or b.
func Alt(a, b *Expr) *Expr { return &Expr{Kind: ExprAlt, Left: a, Right: b} }

// Star constructs a zero-or-more repetition expression.
func Star(child *Expr) *Expr { return &Expr{Kind: ExprStar, Child: child} }

// Plus constructs a one-or-more repetition expression.
func Plus(child *Expr) *Expr { return &Expr{Kind: ExprPlus, Child: child} }

// Optional constructs a zero-or-one expression.
func Optional(child *Expr) *Expr { return &Expr{Kind: ExprOpt, Child: child} }

// LiteralRunes constructs a concatenation of single-rune expressions.
func LiteralRunes(runes []rune) *Expr { return concatRunes(runes) }

// LiteralString constructs a literal expression from a Go string.
func LiteralString(value string) *Expr { return concatRunes([]rune(value)) }
func concatRunes(runes []rune) *Expr {
	if len(runes) == 0 {
		return Empty()
	}
	expr := SetExpr(Single(runes[0]))
	for _, r := range runes[1:] {
		expr = Concat(expr, SetExpr(Single(r)))
	}
	return expr
}

// RegexParser parses the LangForge regex dialect.
type RegexParser struct {
	input  string
	pos    int
	domain RangeSet
}

// ParseRegex parses one lexical regular expression.
func ParseRegex(input string) (*Expr, error) {
	return ParseRegexWithDomain(input, UnicodeScalarDomain())
}

// ParseRegexWithDomain parses one lexical regular expression against a scanner
// domain. Domain-aware atoms such as `.` and negated classes use this set.
func ParseRegexWithDomain(input string, domain RangeSet) (*Expr, error) {
	if len(domain) == 0 {
		domain = UnicodeScalarDomain()
	}
	p := &RegexParser{input: input, domain: domain.Normalize()}
	expr, err := p.parseAlt()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if !p.eof() {
		return nil, p.errorf("unexpected input %q", p.remaining())
	}
	return expr, nil
}

// ExpandRefs replaces named definition references and detects recursion.
func ExpandRefs(expr *Expr, defs map[string]*Expr) (*Expr, error) {
	return expandRefs(expr, defs, map[string]bool{})
}

// Nullable reports whether the expression can accept the empty string.
func (e *Expr) Nullable() bool {
	if e == nil {
		return true
	}
	switch e.Kind {
	case ExprEmpty:
		return true
	case ExprSet, ExprRef:
		return false
	case ExprConcat:
		return e.Left.Nullable() && e.Right.Nullable()
	case ExprAlt:
		return e.Left.Nullable() || e.Right.Nullable()
	case ExprStar, ExprOpt:
		return true
	case ExprPlus:
		return e.Child.Nullable()
	default:
		return false
	}
}

func expandRefs(expr *Expr, defs map[string]*Expr, stack map[string]bool) (*Expr, error) {
	if expr == nil {
		return nil, nil
	}
	switch expr.Kind {
	case ExprRef:
		if stack[expr.Name] {
			return nil, fmt.Errorf("recursive lexer definition `%s`", expr.Name)
		}
		def, ok := defs[expr.Name]
		if !ok {
			return nil, fmt.Errorf("undefined lexer definition `%s`", expr.Name)
		}
		stack[expr.Name] = true
		out, err := expandRefs(def, defs, stack)
		delete(stack, expr.Name)
		return out, err
	case ExprConcat, ExprAlt:
		left, err := expandRefs(expr.Left, defs, stack)
		if err != nil {
			return nil, err
		}
		right, err := expandRefs(expr.Right, defs, stack)
		if err != nil {
			return nil, err
		}
		cp := *expr
		cp.Left = left
		cp.Right = right
		return &cp, nil
	case ExprStar, ExprPlus, ExprOpt:
		child, err := expandRefs(expr.Child, defs, stack)
		if err != nil {
			return nil, err
		}
		cp := *expr
		cp.Child = child
		return &cp, nil
	default:
		cp := *expr
		return &cp, nil
	}
}

func (p *RegexParser) parseAlt() (*Expr, error) {
	left, err := p.parseConcat()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		if !p.consume('|') {
			return left, nil
		}
		right, err := p.parseConcat()
		if err != nil {
			return nil, err
		}
		left = Alt(left, right)
	}
}

func (p *RegexParser) parseConcat() (*Expr, error) {
	var parts []*Expr
	for {
		p.skipSpace()
		if p.eof() || p.peek() == ')' || p.peek() == '|' {
			break
		}
		part, err := p.parseRepeat()
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return Empty(), nil
	}
	expr := parts[0]
	for _, part := range parts[1:] {
		expr = Concat(expr, part)
	}
	return expr, nil
}

func (p *RegexParser) parseRepeat() (*Expr, error) {
	expr, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		switch {
		case p.consume('*'):
			expr = Star(expr)
		case p.consume('+'):
			expr = Plus(expr)
		case p.consume('?'):
			expr = Optional(expr)
		default:
			return expr, nil
		}
	}
}

func (p *RegexParser) parseAtom() (*Expr, error) {
	p.skipSpace()
	if p.eof() {
		return nil, p.errorf("expected regex atom")
	}
	switch r := p.peek(); r {
	case '(':
		p.next()
		expr, err := p.parseAlt()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if !p.consume(')') {
			return nil, p.errorf("expected `)`")
		}
		return expr, nil
	case '[':
		return p.parseClass()
	case '\'', '"':
		return p.parseQuoted()
	case '.':
		p.next()
		return SetExpr(p.domain), nil
	case '\\':
		if p.hasPropertyEscape() {
			set, err := p.parsePropertyEscape()
			if err != nil {
				return nil, err
			}
			return SetExpr(set), nil
		}
		r, err := p.parseEscape(false)
		if err != nil {
			return nil, err
		}
		return SetExpr(Single(r)), nil
	default:
		if isIdentStart(r) {
			name := p.readIdent()
			return Ref(name), nil
		}
		p.next()
		return SetExpr(Single(r)), nil
	}
}

func (p *RegexParser) parseQuoted() (*Expr, error) {
	quote := p.next()
	var out []rune
	for !p.eof() {
		if p.peek() == quote {
			p.next()
			return LiteralRunes(out), nil
		}
		if p.peek() == '\\' {
			r, err := p.parseEscape(false)
			if err != nil {
				return nil, err
			}
			out = append(out, r)
			continue
		}
		out = append(out, p.next())
	}
	return nil, p.errorf("unterminated quoted literal")
}

func (p *RegexParser) parseClass() (*Expr, error) {
	if !p.consume('[') {
		return nil, p.errorf("expected `[`")
	}
	negated := p.consume('^')
	var set RangeSet
	for !p.eof() {
		p.skipClassSeparators()
		if p.consume(']') {
			if negated {
				set = p.domain.Difference(set)
			}
			return SetExpr(set), nil
		}
		if p.hasPropertyEscape() {
			property, err := p.parsePropertyEscape()
			if err != nil {
				return nil, err
			}
			p.skipClassSeparators()
			if p.peek() == '-' {
				return nil, p.errorf("Unicode property escapes cannot be range endpoints")
			}
			set = set.Union(property)
			continue
		}
		left, leftNumeric, leftEscaped, err := p.parseClassElement()
		if err != nil {
			return nil, err
		}
		p.skipClassSeparators()
		if p.consume('-') {
			p.skipClassSeparators()
			if p.hasPropertyEscape() {
				return nil, p.errorf("Unicode property escapes cannot be range endpoints")
			}
			right, rightNumeric, rightEscaped, err := p.parseClassElement()
			if err != nil {
				return nil, err
			}
			lo, hi := left, right
			if leftNumeric || rightNumeric || legacyByteRange(left, leftEscaped, right, rightEscaped) {
				lo = numericClassRune(left, leftNumeric)
				hi = numericClassRune(right, rightNumeric)
			}
			if hi < lo {
				return nil, p.errorf("descending range in character class")
			}
			set = append(set, Range{Lo: lo, Hi: hi})
		} else {
			set = append(set, Range{Lo: left, Hi: left})
		}
	}
	return nil, p.errorf("unterminated character class")
}

func (p *RegexParser) parseClassElement() (rune, bool, bool, error) {
	if p.eof() {
		return 0, false, false, p.errorf("expected character class element")
	}
	if p.peek() == '\\' {
		r, err := p.parseEscape(true)
		return r, false, true, err
	}
	if unicode.IsDigit(p.peek()) {
		start := p.pos
		for !p.eof() && unicode.IsDigit(p.peek()) {
			p.next()
		}
		token := p.input[start:p.pos]
		if len(token) > 1 {
			n, err := strconv.Atoi(token)
			if err != nil {
				return 0, false, false, p.errorf("invalid numeric class element")
			}
			return rune(n), true, false, nil
		}
		return rune(token[0]), false, false, nil
	}
	return p.next(), false, false, nil
}

func numericClassRune(r rune, alreadyNumeric bool) rune {
	if alreadyNumeric {
		return r
	}
	if r >= '0' && r <= '9' {
		return r - '0'
	}
	return r
}

func legacyByteRange(left rune, leftEscaped bool, right rune, rightEscaped bool) bool {
	return !leftEscaped && unicode.IsDigit(left) && rightEscaped && !unicode.IsDigit(right)
}

func (p *RegexParser) parseEscape(legacyDigitAsChar bool) (rune, error) {
	if !p.consume('\\') {
		return 0, p.errorf("expected escape")
	}
	if p.eof() {
		return 0, p.errorf("trailing escape")
	}
	r := p.next()
	switch r {
	case 'n':
		return '\n', nil
	case 'r':
		return '\r', nil
	case 't':
		return '\t', nil
	case 'x':
		return p.parseFixedHex(2)
	case 'u':
		if p.consume('{') {
			return p.parseBracedHex()
		}
		return p.parseFixedHex(4)
	case 'U':
		return p.parseFixedHex(8)
	default:
		if legacyDigitAsChar && unicode.IsDigit(r) {
			return r, nil
		}
		return r, nil
	}
}

func (p *RegexParser) parseFixedHex(width int) (rune, error) {
	if p.pos+width > len(p.input) {
		return 0, p.errorf("short hex escape")
	}
	raw := p.input[p.pos : p.pos+width]
	n, err := strconv.ParseInt(raw, 16, 32)
	if err != nil {
		return 0, p.errorf("invalid hex escape")
	}
	p.pos += width
	r := rune(n)
	if !IsUnicodeScalar(r) {
		return 0, p.errorf("hex escape is not a valid Unicode scalar value")
	}
	return r, nil
}

func (p *RegexParser) parseBracedHex() (rune, error) {
	start := p.pos
	for !p.eof() && p.peek() != '}' {
		p.next()
	}
	if p.eof() {
		return 0, p.errorf("unterminated braced hex escape")
	}
	raw := p.input[start:p.pos]
	p.next()
	if raw == "" || len(raw) > 8 {
		return 0, p.errorf("invalid braced hex escape")
	}
	n, err := strconv.ParseInt(raw, 16, 32)
	if err != nil {
		return 0, p.errorf("invalid braced hex escape")
	}
	r := rune(n)
	if !IsUnicodeScalar(r) {
		return 0, p.errorf("braced hex escape is not a valid Unicode scalar value")
	}
	return r, nil
}

func (p *RegexParser) hasPropertyEscape() bool {
	if p.eof() || p.input[p.pos] != '\\' || p.pos+1 >= len(p.input) {
		return false
	}
	return p.input[p.pos+1] == 'p' || p.input[p.pos+1] == 'P'
}

func (p *RegexParser) parsePropertyEscape() (RangeSet, error) {
	if !p.consume('\\') {
		return nil, p.errorf("expected Unicode property escape")
	}
	negated := false
	switch p.next() {
	case 'p':
	case 'P':
		negated = true
	default:
		return nil, p.errorf("expected Unicode property escape")
	}
	if !p.consume('{') {
		return nil, p.errorf("Unicode property escape expects `{name}`")
	}
	start := p.pos
	for !p.eof() && p.peek() != '}' {
		p.next()
	}
	if p.eof() {
		return nil, p.errorf("unterminated Unicode property escape")
	}
	name := p.input[start:p.pos]
	p.next()
	set, ok := unicodePropertySet(name)
	if !ok {
		return nil, p.errorf("unsupported Unicode property %q", name)
	}
	set = set.Intersection(p.domain)
	if negated {
		set = p.domain.Difference(set)
	}
	return set, nil
}

func (p *RegexParser) skipSpace() {
	for !p.eof() && unicode.IsSpace(p.peek()) {
		p.next()
	}
}

func (p *RegexParser) skipClassSeparators() {
	for !p.eof() {
		r := p.peek()
		if unicode.IsSpace(r) || r == ',' {
			p.next()
			continue
		}
		return
	}
}

func (p *RegexParser) readIdent() string {
	start := p.pos
	p.next()
	for !p.eof() && isIdent(p.peek()) {
		p.next()
	}
	return p.input[start:p.pos]
}

func (p *RegexParser) consume(r rune) bool {
	if !p.eof() && p.peek() == r {
		p.next()
		return true
	}
	return false
}

func (p *RegexParser) peek() rune {
	r, _ := utf8.DecodeRuneInString(p.input[p.pos:])
	return r
}

func (p *RegexParser) next() rune {
	r, size := utf8.DecodeRuneInString(p.input[p.pos:])
	p.pos += size
	return r
}

func (p *RegexParser) eof() bool {
	return p.pos >= len(p.input)
}

func (p *RegexParser) remaining() string {
	return p.input[p.pos:]
}

func (p *RegexParser) errorf(format string, args ...any) error {
	return fmt.Errorf("regex at byte %d: %s", p.pos, fmt.Sprintf(format, args...))
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdent(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func unicodePropertySet(name string) (RangeSet, bool) {
	original := strings.TrimSpace(name)
	if original == "" {
		return nil, false
	}
	if table, ok := unicode.Categories[original]; ok {
		return rangeTableSet(table), true
	}
	if table, ok := unicode.Scripts[original]; ok {
		return rangeTableSet(table), true
	}
	if table, ok := unicode.Properties[original]; ok {
		return rangeTableSet(table), true
	}
	normalized := normalizePropertyName(original)
	switch normalized {
	case "l", "letter", "letters":
		return rangeTableSet(unicode.Letter), true
	case "n", "number", "numbers":
		return rangeTableSet(unicode.Number), true
	case "nd", "decimalnumber", "decimaldigit", "digit", "digits":
		return rangeTableSet(unicode.Nd), true
	case "white_space", "whitespace", "space", "spaces":
		return rangeTableSet(unicode.White_Space), true
	}
	for key, table := range unicode.Categories {
		if normalizePropertyName(key) == normalized {
			return rangeTableSet(table), true
		}
	}
	for key, table := range unicode.Scripts {
		if normalizePropertyName(key) == normalized {
			return rangeTableSet(table), true
		}
	}
	for key, table := range unicode.Properties {
		if normalizePropertyName(key) == normalized {
			return rangeTableSet(table), true
		}
	}
	return nil, false
}

func normalizePropertyName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, " ", "")
	return name
}

func rangeTableSet(table *unicode.RangeTable) RangeSet {
	var set RangeSet
	for _, r16 := range table.R16 {
		set = appendRangeWithStride(set, rune(r16.Lo), rune(r16.Hi), rune(r16.Stride))
	}
	for _, r32 := range table.R32 {
		set = appendRangeWithStride(set, rune(r32.Lo), rune(r32.Hi), rune(r32.Stride))
	}
	return set.Intersection(UnicodeScalarDomain())
}

func appendRangeWithStride(set RangeSet, lo, hi, stride rune) RangeSet {
	if stride <= 1 {
		return append(set, Range{Lo: lo, Hi: hi})
	}
	for r := lo; r <= hi; r += stride {
		set = append(set, Range{Lo: r, Hi: r})
		if hi-r < stride {
			break
		}
	}
	return set
}

func (e *Expr) String() string {
	if e == nil {
		return "<nil>"
	}
	switch e.Kind {
	case ExprEmpty:
		return "e"
	case ExprSet:
		return e.Set.String()
	case ExprRef:
		return e.Name
	case ExprConcat:
		return "(" + e.Left.String() + " " + e.Right.String() + ")"
	case ExprAlt:
		return "(" + e.Left.String() + "|" + e.Right.String() + ")"
	case ExprStar:
		return "(" + e.Child.String() + ")*"
	case ExprPlus:
		return "(" + e.Child.String() + ")+"
	case ExprOpt:
		return "(" + e.Child.String() + ")?"
	default:
		return strings.ToUpper(string(e.Kind))
	}
}
