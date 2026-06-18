package spec

import "github.com/russlank/lang-forge/internal/diagnostics"

// Spec is the target-neutral intermediate representation for one language tool.
type Spec struct {
	SourceFile string       `json:"sourceFile,omitempty"`
	Target     string       `json:"target,omitempty"`
	Package    string       `json:"package,omitempty"`
	Scanner    ScannerSpec  `json:"scanner"`
	Semantics  SemanticSpec `json:"semantics,omitempty"`
	Tokens     []TokenDecl  `json:"tokens,omitempty"`
	Lexer      LexerSpec    `json:"lexer"`
	Grammar    GrammarSpec  `json:"grammar"`
	Spans      []NamedSpan  `json:"-"`
	Mode       InputMode    `json:"mode"`
}

// InputMode records whether a Spec came from the combined or split format.
type InputMode string

const (
	ModeCombined InputMode = "combined"
	ModeSplit    InputMode = "split"
	ModeLexOnly  InputMode = "lex-only"
	ModeYaccOnly InputMode = "yacc-only"
)

// ScannerEncoding names the source encoding consumed by generated scanners.
type ScannerEncoding string

const (
	// ScannerEncodingUTF8 decodes source text as checked UTF-8.
	ScannerEncodingUTF8 ScannerEncoding = "utf8"
)

// ScannerInvalidPolicy controls malformed source encoding handling.
type ScannerInvalidPolicy string

const (
	// ScannerInvalidError reports malformed source encoding as a scanner error.
	ScannerInvalidError ScannerInvalidPolicy = "error"
)

// ScannerSpec captures target-neutral scanner source decoding settings.
type ScannerSpec struct {
	Encoding ScannerEncoding      `json:"encoding"`
	Invalid  ScannerInvalidPolicy `json:"invalid"`
	Newline  string               `json:"newline,omitempty"`
}

// DefaultScanner returns LangForge's canonical scanner settings.
func DefaultScanner() ScannerSpec {
	return ScannerSpec{Encoding: ScannerEncodingUTF8, Invalid: ScannerInvalidError}
}

// WithDefaults returns s with omitted scanner settings filled in.
func (s ScannerSpec) WithDefaults() ScannerSpec {
	out := s
	if out.Encoding == "" {
		out.Encoding = ScannerEncodingUTF8
	}
	if out.Invalid == "" {
		out.Invalid = ScannerInvalidError
	}
	return out
}

// SemanticActionMode controls how target-tagged parser action text is used.
type SemanticActionMode string

const (
	// SemanticModeReducer treats action text as reducer-dispatch labels.
	SemanticModeReducer SemanticActionMode = "reducer"
	// SemanticModeInline treats action text as target-language code in generated output.
	SemanticModeInline SemanticActionMode = "inline"
)

// SemanticSpec captures target-specific semantic action integration settings.
type SemanticSpec struct {
	Includes []SemanticInclude             `json:"includes,omitempty"`
	Modes    map[string]SemanticActionMode `json:"modes,omitempty"`
}

// SemanticInclude declares a target-specific handwritten package/library used by semantics.
type SemanticInclude struct {
	Target string           `json:"target"`
	Alias  string           `json:"alias,omitempty"`
	Path   string           `json:"path"`
	Span   diagnostics.Span `json:"span"`
}

// ModeFor returns the configured action mode for target, defaulting to reducer callbacks.
func (s SemanticSpec) ModeFor(target string) SemanticActionMode {
	if s.Modes == nil || s.Modes[target] == "" {
		return SemanticModeReducer
	}
	return s.Modes[target]
}

// IncludesFor returns semantic includes for target in declaration order.
func (s SemanticSpec) IncludesFor(target string) []SemanticInclude {
	var out []SemanticInclude
	for _, include := range s.Includes {
		if include.Target == target {
			out = append(out, include)
		}
	}
	return out
}

// NamedSpan associates a source span with a named language element.
type NamedSpan struct {
	Name string
	Span diagnostics.Span
}

// TokenDecl is an explicit %token declaration.
type TokenDecl struct {
	Name string           `json:"name"`
	Span diagnostics.Span `json:"span"`
}

// LexerSpec contains reusable lexical definitions and ordered matching rules.
type LexerSpec struct {
	Definitions []LexDefinition `json:"definitions,omitempty"`
	Rules       []LexRule       `json:"rules,omitempty"`
}

// LexDefinition is a named regex fragment that rules can reference.
type LexDefinition struct {
	Name    string           `json:"name"`
	Pattern string           `json:"pattern"`
	Span    diagnostics.Span `json:"span"`
}

// LexActionKind classifies what a lexer rule should do after matching.
type LexActionKind string

const (
	ActionToken   LexActionKind = "token"
	ActionSkip    LexActionKind = "skip"
	ActionChannel LexActionKind = "channel"
	ActionRaw     LexActionKind = "raw"
)

// LexAction describes a token, skip, hidden channel, or raw legacy action.
type LexAction struct {
	Kind    LexActionKind `json:"kind"`
	Token   string        `json:"token,omitempty"`
	Channel string        `json:"channel,omitempty"`
	Raw     string        `json:"raw,omitempty"`
}

// LexRule is one ordered lexical match rule.
type LexRule struct {
	Name    string           `json:"name,omitempty"`
	Pattern string           `json:"pattern"`
	Action  LexAction        `json:"action"`
	Span    diagnostics.Span `json:"span"`
}

// GrammarSpec contains parser settings and grammar productions.
type GrammarSpec struct {
	Start     string     `json:"start,omitempty"`
	Algorithm string     `json:"algorithm,omitempty"`
	Rules     []RuleSpec `json:"rules,omitempty"`
}

// RuleSpec is a user-facing grammar production before normalization.
type RuleSpec struct {
	Name         string        `json:"name"`
	Alternatives []Alternative `json:"alternatives"`
	Span         diagnostics.Span
}

// Alternative is one right-hand side for a grammar production.
type Alternative struct {
	Symbols []string          `json:"symbols"`
	Actions map[string]string `json:"actions,omitempty"`
	Span    diagnostics.Span  `json:"span"`
}

// TokenNames returns the de-duplicated visible token names known to the spec.
func (s Spec) TokenNames() []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range s.Tokens {
		if !seen[t.Name] {
			seen[t.Name] = true
			out = append(out, t.Name)
		}
	}
	for _, r := range s.Lexer.Rules {
		if r.Action.Kind == ActionToken && r.Action.Token != "" && !seen[r.Action.Token] {
			seen[r.Action.Token] = true
			out = append(out, r.Action.Token)
		}
	}
	return out
}
