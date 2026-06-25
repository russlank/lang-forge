package parse

import (
	"fmt"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/spec"
)

const (
	// EOF is the internal terminal name used for parser end-of-input.
	EOF = "$"
	// Error is the reserved parser-only terminal shifted during syntax recovery.
	Error = "error"
)

// Grammar is the normalized parser grammar used by table construction.
type Grammar struct {
	Start          string
	Terminals      map[string]bool
	Nonterminals   map[string]bool
	Rules          []Rule
	Spans          map[string]diagnostics.Span
	ExpectedTokens spec.ExpectedTokenSpec
}

// Rule is one normalized grammar production.
type Rule struct {
	ID      int               `json:"id"`
	LHS     string            `json:"lhs"`
	RHS     []string          `json:"rhs"`
	Labels  []string          `json:"labels,omitempty"`
	Actions map[string]string `json:"actions,omitempty"`
	Span    diagnostics.Span  `json:"span"`
}

// FromSpec converts a parsed project specification into a validated grammar.
func FromSpec(s spec.Spec) (*Grammar, diagnostics.List) {
	var diags diagnostics.List
	g := &Grammar{
		Start:          s.Grammar.Start,
		Terminals:      map[string]bool{EOF: true, Error: true},
		Nonterminals:   map[string]bool{},
		Spans:          map[string]diagnostics.Span{},
		ExpectedTokens: s.Grammar.ExpectedTokens,
	}
	for _, tok := range s.Tokens {
		if tok.Name == Error {
			diags.AddError("LF307", "`error` is reserved for parser recovery and must not be declared with %token", tok.Span)
			continue
		}
		g.Terminals[tok.Name] = true
	}
	for _, name := range s.TokenNames() {
		if name == Error {
			continue
		}
		g.Terminals[name] = true
	}
	for _, rule := range s.Lexer.Rules {
		if rule.Action.Kind == spec.ActionToken && rule.Action.Token == Error {
			diags.AddError("LF307", "`error` is reserved for parser recovery and cannot be emitted by the scanner", rule.Span)
		}
	}
	for _, rule := range s.Grammar.Rules {
		if rule.Name == Error {
			diags.AddError("LF308", "`error` is reserved for parser recovery and cannot be a rule name", rule.Span)
			continue
		}
		if g.Nonterminals[rule.Name] {
			// Multiple declarations are merged by appending alternatives.
		}
		g.Nonterminals[rule.Name] = true
		g.Spans[rule.Name] = rule.Span
	}
	for nt := range g.Nonterminals {
		if g.Terminals[nt] {
			diags.AddError("LF303", "symbol `"+nt+"` is both a token and a grammar rule name", g.Spans[nt])
		}
	}
	if g.Start == "" && len(s.Grammar.Rules) > 0 {
		g.Start = s.Grammar.Rules[0].Name
	}
	if g.Start == "" {
		diags.AddError("LF300", "grammar has no start symbol", diagnostics.Span{})
		return g, diags
	}
	if !g.Nonterminals[g.Start] {
		diags.AddError("LF301", "start symbol `"+g.Start+"` has no rule", diagnostics.Span{})
	}
	semanticTypes := map[string]bool{}
	for _, semanticType := range s.Semantics.Types {
		key := semanticType.Target + "\x00" + semanticType.Symbol
		if semanticTypes[key] {
			diags.AddError("LF306", "duplicate semantic type for target `"+semanticType.Target+"` and nonterminal `"+semanticType.Symbol+"`", semanticType.Span)
			continue
		}
		semanticTypes[key] = true
		if g.Terminals[semanticType.Symbol] {
			diags.AddError("LF304", "semantic type declarations apply to nonterminals; terminal `"+semanticType.Symbol+"` already has the generated Lexeme type", semanticType.Span)
			continue
		}
		if !g.Nonterminals[semanticType.Symbol] {
			diags.AddError("LF305", "semantic type references undefined nonterminal `"+semanticType.Symbol+"`", semanticType.Span)
		}
	}
	validateExpectedTokenSpec(g, &diags)
	nextID := 1
	for _, rule := range s.Grammar.Rules {
		if rule.Name == Error {
			continue
		}
		for _, alt := range rule.Alternatives {
			errorCount := 0
			errorIndex := -1
			for index, sym := range alt.Symbols {
				if !g.Terminals[sym] && !g.Nonterminals[sym] {
					diags.AddError("LF302", fmt.Sprintf("undefined grammar symbol `%s` in rule `%s`", sym, rule.Name), alt.Span)
				}
				if sym == Error {
					errorCount++
					errorIndex = index
					if index < len(alt.Labels) && alt.Labels[index] != "" {
						diags.AddError("LF313", "reserved `error` symbol cannot have a named RHS label", alt.Span)
					}
				}
			}
			if errorCount > 1 {
				diags.AddError("LF314", "a recovery alternative may contain reserved `error` only once", alt.Span)
			}
			if errorIndex >= 0 {
				hasSynchronizationTerminal := false
				for _, symbol := range alt.Symbols[errorIndex+1:] {
					if g.Terminals[symbol] && symbol != Error {
						hasSynchronizationTerminal = true
						break
					}
				}
				if !hasSynchronizationTerminal {
					diags.AddError("LF315", "reserved `error` must be followed by a synchronization terminal in the same alternative", alt.Span)
				}
			}
			g.Rules = append(g.Rules, Rule{
				ID:      nextID,
				LHS:     rule.Name,
				RHS:     append([]string(nil), alt.Symbols...),
				Labels:  append([]string(nil), alt.Labels...),
				Actions: cloneStringMap(alt.Actions),
				Span:    alt.Span,
			})
			nextID++
		}
	}
	return g, diags
}

func validateExpectedTokenSpec(g *Grammar, diags *diagnostics.List) {
	for _, alias := range g.ExpectedTokens.Aliases {
		if alias.Token == Error || !g.Terminals[alias.Token] {
			diags.AddError("LF309", "expected-token alias references undefined or reserved terminal `"+alias.Token+"`", alias.Span)
		}
	}
	grouped := map[string]string{}
	for _, group := range g.ExpectedTokens.Groups {
		for _, token := range group.Tokens {
			if token == Error || !g.Terminals[token] {
				diags.AddError("LF310", "expected-token group `"+group.Name+"` references undefined or reserved terminal `"+token+"`", group.Span)
				continue
			}
			if previous := grouped[token]; previous != "" {
				diags.AddError("LF311", "terminal `"+token+"` belongs to expected-token groups `"+previous+"` and `"+group.Name+"`", group.Span)
				continue
			}
			grouped[token] = group.Name
		}
	}
	hidden := map[string]bool{}
	for _, hiddenToken := range g.ExpectedTokens.Hidden {
		token := hiddenToken.Token
		if token == Error || !g.Terminals[token] {
			diags.AddError("LF312", "hidden expected token references undefined or reserved terminal `"+token+"`", hiddenToken.Span)
			continue
		}
		if hidden[token] {
			continue
		}
		hidden[token] = true
	}
}

// Nullable returns the set of nonterminals that can derive the empty string.
func (g *Grammar) Nullable() map[string]bool {
	nullable := map[string]bool{}
	changed := true
	for changed {
		changed = false
		for _, rule := range g.Rules {
			if nullable[rule.LHS] {
				continue
			}
			if len(rule.RHS) == 0 {
				nullable[rule.LHS] = true
				changed = true
				continue
			}
			all := true
			for _, sym := range rule.RHS {
				if g.Terminals[sym] || !nullable[sym] {
					all = false
					break
				}
			}
			if all {
				nullable[rule.LHS] = true
				changed = true
			}
		}
	}
	return nullable
}

// FirstSets returns FIRST sets keyed by terminal and nonterminal name.
func (g *Grammar) FirstSets() map[string]map[string]bool {
	nullable := g.Nullable()
	first := map[string]map[string]bool{}
	for t := range g.Terminals {
		first[t] = map[string]bool{t: true}
	}
	for nt := range g.Nonterminals {
		if first[nt] == nil {
			first[nt] = map[string]bool{}
		}
	}
	changed := true
	for changed {
		changed = false
		for _, rule := range g.Rules {
			dst := first[rule.LHS]
			for _, sym := range rule.RHS {
				for tok := range first[sym] {
					if !dst[tok] {
						dst[tok] = true
						changed = true
					}
				}
				if g.Terminals[sym] || !nullable[sym] {
					break
				}
			}
		}
	}
	return first
}

// FollowSets returns FOLLOW sets for all grammar nonterminals.
func (g *Grammar) FollowSets() map[string]map[string]bool {
	nullable := g.Nullable()
	first := g.FirstSets()
	follow := map[string]map[string]bool{}
	for nt := range g.Nonterminals {
		follow[nt] = map[string]bool{}
	}
	follow[g.Start][EOF] = true
	changed := true
	for changed {
		changed = false
		for _, rule := range g.Rules {
			trailer := cloneSet(follow[rule.LHS])
			for i := len(rule.RHS) - 1; i >= 0; i-- {
				sym := rule.RHS[i]
				if g.Nonterminals[sym] {
					for tok := range trailer {
						if !follow[sym][tok] {
							follow[sym][tok] = true
							changed = true
						}
					}
					if nullable[sym] {
						trailer = unionSet(trailer, first[sym])
					} else {
						trailer = cloneSet(first[sym])
					}
				} else {
					trailer = cloneSet(first[sym])
				}
			}
		}
	}
	return follow
}

// Symbols returns every known terminal and nonterminal in stable order.
func (g *Grammar) Symbols() []string {
	var out []string
	for t := range g.Terminals {
		out = append(out, t)
	}
	for nt := range g.Nonterminals {
		out = append(out, nt)
	}
	sort.Strings(out)
	return out
}

// String formats the grammar as numbered productions for diagnostics.
func (g *Grammar) String() string {
	var b strings.Builder
	for _, r := range g.Rules {
		b.WriteString(fmt.Sprintf("%d) %s ->", r.ID, r.LHS))
		for _, sym := range r.RHS {
			b.WriteByte(' ')
			b.WriteString(sym)
		}
		if len(r.RHS) == 0 {
			b.WriteString(" e")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func cloneSet(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func unionSet(a, b map[string]bool) map[string]bool {
	out := cloneSet(a)
	for k := range b {
		out[k] = true
	}
	return out
}
