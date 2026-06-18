package parse

import (
	"fmt"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/parseralgo"
)

// Item identifies an LR(0) item by grammar rule index and dot position.
type Item struct {
	Rule int `json:"rule"`
	Dot  int `json:"dot"`
}

// LR1Item identifies a canonical LR(1) item with one lookahead terminal.
type LR1Item struct {
	Rule      int    `json:"rule"`
	Dot       int    `json:"dot"`
	Lookahead string `json:"lookahead"`
}

// ActionKind classifies parser table actions.
type ActionKind string

const (
	ActionError  ActionKind = "error"
	ActionShift  ActionKind = "shift"
	ActionReduce ActionKind = "reduce"
	ActionAccept ActionKind = "accept"
)

// Action is one parser-table operation for a state/lookahead pair.
type Action struct {
	Kind  ActionKind `json:"kind"`
	State int        `json:"state,omitempty"`
	Rule  int        `json:"rule,omitempty"`
}

// Conflict records a parser-table conflict with enough context to diagnose it.
type Conflict struct {
	State    int    `json:"state"`
	Symbol   string `json:"symbol"`
	Existing Action `json:"existing"`
	Incoming Action `json:"incoming"`
	Items    []Item `json:"items"`
	Message  string `json:"message"`
}

// State describes one generated parser automaton state.
type State struct {
	ID          int            `json:"id"`
	Items       []Item         `json:"items"`
	LR1Items    []LR1Item      `json:"lr1Items,omitempty"`
	Transitions map[string]int `json:"transitions,omitempty"`
}

// Table is the generated parser automaton and action/goto table.
type Table struct {
	Algorithm string                    `json:"algorithm"`
	States    []State                   `json:"states"`
	Actions   map[int]map[string]Action `json:"actions"`
	Gotos     map[int]map[string]int    `json:"gotos"`
	Conflicts []Conflict                `json:"conflicts,omitempty"`
	Rules     []Rule                    `json:"rules"`
}

// Build constructs a parser table using the requested algorithm.
func Build(g *Grammar, algorithm string) *Table {
	normalized, ok := parseralgo.Normalize(algorithm)
	if !ok {
		normalized = parseralgo.Default
	}
	switch normalized {
	case parseralgo.LALR:
		return BuildLALR(g)
	case parseralgo.IELR:
		return BuildIELR(g)
	case parseralgo.Canonical:
		return BuildCanonicalLR1(g)
	case parseralgo.SLR:
		return BuildSLR(g)
	default:
		return BuildLALR(g)
	}
}

// BuildSLR constructs a compatibility-focused SLR(1) parser table.
func BuildSLR(g *Grammar) *Table {
	aug := augment(g)
	follow := aug.FollowSets()
	states, transitions := canonicalLR0(aug)
	table := &Table{
		Algorithm: parseralgo.SLR,
		Actions:   map[int]map[string]Action{},
		Gotos:     map[int]map[string]int{},
		Rules:     aug.Rules,
	}
	for i, itemSet := range states {
		state := State{ID: i, Items: sortedItems(itemSet), Transitions: map[string]int{}}
		for sym, to := range transitions[i] {
			state.Transitions[sym] = to
		}
		table.States = append(table.States, state)
	}
	for from, bySym := range transitions {
		for sym, to := range bySym {
			if aug.Terminals[sym] && sym != EOF {
				table.setAction(from, sym, Action{Kind: ActionShift, State: to}, states[from])
			} else if aug.Nonterminals[sym] {
				if table.Gotos[from] == nil {
					table.Gotos[from] = map[string]int{}
				}
				table.Gotos[from][sym] = to
			}
		}
	}
	for stateID, itemSet := range states {
		for item := range itemSet {
			rule := aug.Rules[item.Rule]
			if item.Dot != len(rule.RHS) {
				continue
			}
			if rule.LHS == augmentedStart(g.Start) {
				table.setAction(stateID, EOF, Action{Kind: ActionAccept, Rule: rule.ID}, itemSet)
				continue
			}
			for tok := range follow[rule.LHS] {
				table.setAction(stateID, tok, Action{Kind: ActionReduce, Rule: rule.ID}, itemSet)
			}
		}
	}
	return table
}

func (t *Table) setAction(state int, symbol string, action Action, items itemSet) {
	if t.Actions[state] == nil {
		t.Actions[state] = map[string]Action{}
	}
	existing, exists := t.Actions[state][symbol]
	if !exists || existing == action {
		t.Actions[state][symbol] = action
		return
	}
	t.Conflicts = append(t.Conflicts, Conflict{
		State:    state,
		Symbol:   symbol,
		Existing: existing,
		Incoming: action,
		Items:    sortedItems(items),
		Message:  fmt.Sprintf("%s/%s conflict on `%s` in state %d", existing.Kind, action.Kind, symbol, state),
	})
	// Preserve conventional Yacc-ish behavior by preferring shift if one side
	// shifts; otherwise keep the earlier reduce. The conflict remains fatal for
	// validate unless the caller explicitly allows conflicts later.
	if action.Kind == ActionShift {
		t.Actions[state][symbol] = action
	}
}

type itemSet map[Item]bool

func augment(g *Grammar) *Grammar {
	start := augmentedStart(g.Start)
	terminals := cloneBoolMap(g.Terminals)
	nonterminals := cloneBoolMap(g.Nonterminals)
	nonterminals[start] = true
	rules := []Rule{{ID: 0, LHS: start, RHS: []string{g.Start}}}
	rules = append(rules, g.Rules...)
	return &Grammar{
		Start:        start,
		Terminals:    terminals,
		Nonterminals: nonterminals,
		Rules:        rules,
		Spans:        g.Spans,
	}
}

func augmentedStart(start string) string {
	return start + "'"
}

func canonicalLR0(g *Grammar) ([]itemSet, map[int]map[string]int) {
	start := closure(g, itemSet{{Rule: 0, Dot: 0}: true})
	states := []itemSet{start}
	ids := map[string]int{itemSetKey(start): 0}
	transitions := map[int]map[string]int{}
	for i := 0; i < len(states); i++ {
		for _, sym := range g.Symbols() {
			next := gotoSet(g, states[i], sym)
			if len(next) == 0 {
				continue
			}
			key := itemSetKey(next)
			id, ok := ids[key]
			if !ok {
				id = len(states)
				ids[key] = id
				states = append(states, next)
			}
			if transitions[i] == nil {
				transitions[i] = map[string]int{}
			}
			transitions[i][sym] = id
		}
	}
	return states, transitions
}

func closure(g *Grammar, items itemSet) itemSet {
	out := itemSet{}
	for item := range items {
		out[item] = true
	}
	changed := true
	for changed {
		changed = false
		for item := range out {
			rule := g.Rules[item.Rule]
			if item.Dot >= len(rule.RHS) {
				continue
			}
			sym := rule.RHS[item.Dot]
			if !g.Nonterminals[sym] {
				continue
			}
			for idx, candidate := range g.Rules {
				if candidate.LHS != sym {
					continue
				}
				newItem := Item{Rule: idx, Dot: 0}
				if !out[newItem] {
					out[newItem] = true
					changed = true
				}
			}
		}
	}
	return out
}

func gotoSet(g *Grammar, items itemSet, sym string) itemSet {
	moved := itemSet{}
	for item := range items {
		rule := g.Rules[item.Rule]
		if item.Dot < len(rule.RHS) && rule.RHS[item.Dot] == sym {
			moved[Item{Rule: item.Rule, Dot: item.Dot + 1}] = true
		}
	}
	if len(moved) == 0 {
		return nil
	}
	return closure(g, moved)
}

func sortedItems(items itemSet) []Item {
	out := make([]Item, 0, len(items))
	for item := range items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rule == out[j].Rule {
			return out[i].Dot < out[j].Dot
		}
		return out[i].Rule < out[j].Rule
	})
	return out
}

func itemSetKey(items itemSet) string {
	sorted := sortedItems(items)
	var b strings.Builder
	for i, item := range sorted {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprintf("%d.%d", item.Rule, item.Dot))
	}
	return b.String()
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
