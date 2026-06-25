// Package action builds target-neutral semantic action metadata shared by
// code generators, generated reducer contexts, and coverage tooling.
package action

import (
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
)

// Manifest is the deterministic semantic-action contract for one target.
type Manifest struct {
	Target  string   `json:"target"`
	Actions []Action `json:"actions,omitempty"`
}

// Action groups every grammar rule that uses one reducer action label.
type Action struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Typed             bool   `json:"typed"`
	TypeIssue         string `json:"typeIssue,omitempty"`
	ReturnType        string `json:"returnType,omitempty"`
	Rules             []Rule `json:"rules"`
	ConsistentContext bool   `json:"consistentContext"`
}

// Rule describes one normalized production associated with an action.
type Rule struct {
	ID         int              `json:"id"`
	LHS        string           `json:"lhs"`
	ReturnType string           `json:"returnType,omitempty"`
	RHS        []Operand        `json:"rhs,omitempty"`
	Typed      bool             `json:"typed"`
	TypeIssue  string           `json:"typeIssue,omitempty"`
	Span       diagnostics.Span `json:"span"`
}

// Operand describes one RHS position, including its optional source label.
type Operand struct {
	Position int    `json:"position"`
	Symbol   string `json:"symbol"`
	Label    string `json:"label,omitempty"`
	Type     string `json:"type,omitempty"`
}

// Build creates a stable manifest in first-action-use and grammar-rule order.
func Build(grammar *parse.Grammar, semantics spec.SemanticSpec, target string) Manifest {
	manifest := Manifest{Target: target}
	if grammar == nil {
		return manifest
	}

	byName := map[string]int{}
	for _, grammarRule := range grammar.Rules {
		name := strings.TrimSpace(grammarRule.Actions[target])
		if name == "" {
			continue
		}
		index, ok := byName[name]
		if !ok {
			index = len(manifest.Actions)
			byName[name] = index
			manifest.Actions = append(manifest.Actions, Action{
				ID:   index + 1,
				Name: name,
			})
		}
		rule := buildRule(grammar, semantics, target, grammarRule)
		manifest.Actions[index].Rules = append(manifest.Actions[index].Rules, rule)
	}

	for index := range manifest.Actions {
		finalizeAction(&manifest.Actions[index])
	}
	return manifest
}

func buildRule(grammar *parse.Grammar, semantics spec.SemanticSpec, target string, grammarRule parse.Rule) Rule {
	rule := Rule{
		ID:   grammarRule.ID,
		LHS:  grammarRule.LHS,
		Span: grammarRule.Span,
	}
	rule.ReturnType, _ = semantics.TypeFor(target, grammarRule.LHS)
	for index, symbol := range grammarRule.RHS {
		operand := Operand{Position: index + 1, Symbol: symbol}
		if index < len(grammarRule.Labels) {
			operand.Label = grammarRule.Labels[index]
		}
		if grammar.Terminals[symbol] {
			operand.Type = terminalType(target)
		} else {
			operand.Type, _ = semantics.TypeFor(target, symbol)
		}
		rule.RHS = append(rule.RHS, operand)
	}

	switch {
	case rule.ReturnType == "":
		rule.TypeIssue = "missing semantic type for result nonterminal " + grammarRule.LHS
	default:
		for _, operand := range rule.RHS {
			if operand.Label != "" && operand.Type == "" {
				rule.TypeIssue = "missing semantic type for labeled nonterminal " + operand.Symbol
				break
			}
		}
	}
	rule.Typed = rule.TypeIssue == ""
	return rule
}

func finalizeAction(action *Action) {
	if len(action.Rules) == 0 {
		return
	}
	action.ReturnType = action.Rules[0].ReturnType
	action.ConsistentContext = true
	for _, rule := range action.Rules {
		if !rule.Typed {
			action.Typed = false
			action.ConsistentContext = false
			action.TypeIssue = rule.TypeIssue
			return
		}
	}
	first := contextSignature(action.Rules[0])
	for _, rule := range action.Rules[1:] {
		if contextSignature(rule) != first {
			action.ConsistentContext = false
			action.TypeIssue = "action is used by rules with different labeled value types"
			return
		}
	}
	action.Typed = true
}

func contextSignature(rule Rule) string {
	var parts []string
	parts = append(parts, rule.ReturnType)
	for _, operand := range rule.RHS {
		if operand.Label != "" {
			parts = append(parts, operand.Label+"="+operand.Type)
		}
	}
	return strings.Join(parts, "\x00")
}

func terminalType(target string) string {
	switch target {
	case "go", "csharp", "cpp":
		return "Lexeme"
	case "c":
		return "lexeme"
	default:
		return "lexeme"
	}
}

// Names returns action names in stable numeric ID order.
func (m Manifest) Names() []string {
	actions := append([]Action(nil), m.Actions...)
	sort.Slice(actions, func(i, j int) bool { return actions[i].ID < actions[j].ID })
	names := make([]string, 0, len(actions))
	for _, action := range actions {
		names = append(names, action.Name)
	}
	return names
}
