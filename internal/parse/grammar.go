package parse

import (
	"fmt"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/spec"
)

// EOF is the internal terminal name used for parser end-of-input.
const EOF = "$"

// Grammar is the normalized parser grammar used by table construction.
type Grammar struct {
	Start        string
	Terminals    map[string]bool
	Nonterminals map[string]bool
	Rules        []Rule
	Spans        map[string]diagnostics.Span
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
		Start:        s.Grammar.Start,
		Terminals:    map[string]bool{EOF: true},
		Nonterminals: map[string]bool{},
		Spans:        map[string]diagnostics.Span{},
	}
	for _, tok := range s.Tokens {
		g.Terminals[tok.Name] = true
	}
	for _, name := range s.TokenNames() {
		g.Terminals[name] = true
	}
	for _, rule := range s.Grammar.Rules {
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
	nextID := 1
	for _, rule := range s.Grammar.Rules {
		for _, alt := range rule.Alternatives {
			for _, sym := range alt.Symbols {
				if !g.Terminals[sym] && !g.Nonterminals[sym] {
					diags.AddError("LF302", fmt.Sprintf("undefined grammar symbol `%s` in rule `%s`", sym, rule.Name), alt.Span)
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
