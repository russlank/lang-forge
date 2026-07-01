package parse

import (
	"fmt"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/diagnostics"
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
	State        int            `json:"state"`
	Symbol       string         `json:"symbol"`
	Existing     Action         `json:"existing"`
	Incoming     Action         `json:"incoming"`
	ExistingRule *ConflictRule  `json:"existingRule,omitempty"`
	IncomingRule *ConflictRule  `json:"incomingRule,omitempty"`
	Items        []Item         `json:"items"`
	ItemDetails  []ConflictItem `json:"itemDetails,omitempty"`
	Message      string         `json:"message"`
	Hint         string         `json:"hint,omitempty"`
}

// ConflictRule records source-rich information about a reduce/accept rule that
// participates in a parser-table conflict.
type ConflictRule struct {
	ID      int              `json:"id"`
	LHS     string           `json:"lhs"`
	RHS     []string         `json:"rhs"`
	Span    diagnostics.Span `json:"span"`
	Display string           `json:"display"`
}

// ConflictItem expands an LR item core with the source production it came from.
// It keeps the compact Item value for stable machine consumers while making
// text and JSON inspection useful to humans debugging a grammar.
type ConflictItem struct {
	Rule      int              `json:"rule"`
	Dot       int              `json:"dot"`
	LHS       string           `json:"lhs"`
	RHS       []string         `json:"rhs"`
	BeforeDot string           `json:"beforeDot,omitempty"`
	AfterDot  string           `json:"afterDot,omitempty"`
	Span      diagnostics.Span `json:"span"`
	Display   string           `json:"display"`
}

// State describes one generated parser automaton state.
type State struct {
	ID          int            `json:"id"`
	Items       []Item         `json:"items"`
	LR1Items    []LR1Item      `json:"lr1Items,omitempty"`
	Transitions map[string]int `json:"transitions,omitempty"`
}

// IELRReport summarizes how IELR moved between the compact LALR merge and the
// full canonical LR(1) automaton. It is intentionally small enough for inspect
// JSON while still explaining why a grammar needed more states than LALR.
type IELRReport struct {
	LALRStates      int               `json:"lalrStates"`
	IELRStates      int               `json:"ielrStates"`
	CanonicalStates int               `json:"canonicalStates"`
	AcceptedMerges  []IELRMergeReport `json:"acceptedMerges,omitempty"`
	RejectedMerges  []IELRMergeReport `json:"rejectedMerges,omitempty"`
}

// IELRMergeReport describes one LR(0)-core merge decision. Accepted decisions
// have CanonicalStates only. Rejected decisions also include ResultStates,
// Reason, and any candidate conflicts detected before the split.
type IELRMergeReport struct {
	Core            []Item         `json:"core"`
	CoreDetails     []ConflictItem `json:"coreDetails,omitempty"`
	CanonicalStates []int          `json:"canonicalStates"`
	ResultStates    [][]int        `json:"resultStates,omitempty"`
	Reason          string         `json:"reason,omitempty"`
	Conflicts       []Conflict     `json:"conflicts,omitempty"`
}

// Table is the generated parser automaton and action/goto table.
type Table struct {
	Algorithm     string                    `json:"algorithm"`
	States        []State                   `json:"states"`
	Actions       map[int]map[string]Action `json:"actions"`
	Gotos         map[int]map[string]int    `json:"gotos"`
	Conflicts     []Conflict                `json:"conflicts,omitempty"`
	IELR          *IELRReport               `json:"ielr,omitempty"`
	Rules         []Rule                    `json:"rules"`
	Expected      map[int][]ExpectedToken   `json:"expected,omitempty"`
	ErrorRecovery bool                      `json:"errorRecovery,omitempty"`
}

// ExpectedToken is one user-facing entry in a state's expected-token set.
// Members contains exact grammar terminals when the display entry represents
// a reporting group.
type ExpectedToken struct {
	Symbol  string   `json:"symbol,omitempty"`
	Display string   `json:"display"`
	Members []string `json:"members,omitempty"`
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
	return finalizeTable(table, aug)
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
	itemsList := sortedItems(items)
	t.Conflicts = append(t.Conflicts, Conflict{
		State:        state,
		Symbol:       symbol,
		Existing:     existing,
		Incoming:     action,
		ExistingRule: t.conflictRuleForAction(existing),
		IncomingRule: t.conflictRuleForAction(action),
		Items:        itemsList,
		ItemDetails:  t.conflictItemDetails(itemsList),
		Message:      fmt.Sprintf("%s/%s conflict on `%s` in state %d", existing.Kind, action.Kind, symbol, state),
		Hint:         conflictHint(existing, action, symbol),
	})
	// Preserve conventional Yacc-ish behavior by preferring shift if one side
	// shifts; otherwise keep the earlier reduce. The conflict remains fatal for
	// validate unless the caller explicitly allows conflicts later.
	if action.Kind == ActionShift {
		t.Actions[state][symbol] = action
	}
}

func (t *Table) conflictRuleForAction(action Action) *ConflictRule {
	if action.Kind != ActionReduce && action.Kind != ActionAccept {
		return nil
	}
	rule, ok := t.ruleByID(action.Rule)
	if !ok {
		return nil
	}
	info := conflictRuleInfo(rule)
	return &info
}

func (t *Table) conflictItemDetails(items []Item) []ConflictItem {
	out := make([]ConflictItem, 0, len(items))
	for _, item := range items {
		if item.Rule < 0 || item.Rule >= len(t.Rules) {
			continue
		}
		rule := t.Rules[item.Rule]
		detail := ConflictItem{
			Rule:    rule.ID,
			Dot:     item.Dot,
			LHS:     rule.LHS,
			RHS:     append([]string(nil), rule.RHS...),
			Span:    rule.Span,
			Display: formatRuleWithDot(rule, item.Dot),
		}
		if item.Dot > 0 && item.Dot <= len(rule.RHS) {
			detail.BeforeDot = rule.RHS[item.Dot-1]
		}
		if item.Dot >= 0 && item.Dot < len(rule.RHS) {
			detail.AfterDot = rule.RHS[item.Dot]
		}
		out = append(out, detail)
	}
	return out
}

func (t *Table) ruleByID(id int) (Rule, bool) {
	for _, rule := range t.Rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return Rule{}, false
}

func conflictRuleInfo(rule Rule) ConflictRule {
	return ConflictRule{
		ID:      rule.ID,
		LHS:     rule.LHS,
		RHS:     append([]string(nil), rule.RHS...),
		Span:    rule.Span,
		Display: formatRule(rule),
	}
}

func formatRule(rule Rule) string {
	if len(rule.RHS) == 0 {
		return fmt.Sprintf("%s -> e", rule.LHS)
	}
	return fmt.Sprintf("%s -> %s", rule.LHS, strings.Join(rule.RHS, " "))
}

func formatRuleWithDot(rule Rule, dot int) string {
	parts := make([]string, 0, len(rule.RHS)+1)
	for i, sym := range rule.RHS {
		if i == dot {
			parts = append(parts, "•")
		}
		parts = append(parts, sym)
	}
	if dot >= len(rule.RHS) {
		parts = append(parts, "•")
	}
	if len(parts) == 1 && parts[0] == "•" {
		return fmt.Sprintf("%s -> •", rule.LHS)
	}
	return fmt.Sprintf("%s -> %s", rule.LHS, strings.Join(parts, " "))
}

func conflictHint(existing, incoming Action, symbol string) string {
	if existing.Kind == ActionShift || incoming.Kind == ActionShift {
		return fmt.Sprintf("shift/reduce conflict: parser can shift `%s` or reduce a completed rule before seeing it", symbol)
	}
	if existing.Kind == ActionReduce && incoming.Kind == ActionReduce {
		return fmt.Sprintf("reduce/reduce conflict: two completed rules can reduce on `%s`", symbol)
	}
	return fmt.Sprintf("multiple parser actions compete on `%s`", symbol)
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
		Start:          start,
		Terminals:      terminals,
		Nonterminals:   nonterminals,
		Rules:          rules,
		Spans:          g.Spans,
		ExpectedTokens: g.ExpectedTokens,
	}
}

func finalizeTable(table *Table, grammar *Grammar) *Table {
	table.Expected = map[int][]ExpectedToken{}
	aliases := map[string]string{}
	for _, alias := range grammar.ExpectedTokens.Aliases {
		aliases[alias.Token] = alias.Label
	}
	hidden := map[string]bool{}
	for _, token := range grammar.ExpectedTokens.Hidden {
		hidden[token.Token] = true
	}
	for _, state := range table.States {
		available := map[string]bool{}
		for symbol := range table.Actions[state.ID] {
			if symbol == Error {
				table.ErrorRecovery = true
				continue
			}
			if !hidden[symbol] {
				available[symbol] = true
			}
		}
		var expected []ExpectedToken
		for _, group := range grammar.ExpectedTokens.Groups {
			var members []string
			for _, token := range group.Tokens {
				if available[token] {
					members = append(members, token)
				}
			}
			if len(members) < 2 {
				continue
			}
			expected = append(expected, ExpectedToken{Display: group.Name, Members: members})
			for _, token := range members {
				delete(available, token)
			}
		}
		symbols := make([]string, 0, len(available))
		for symbol := range available {
			symbols = append(symbols, symbol)
		}
		sort.Strings(symbols)
		for _, symbol := range symbols {
			display := aliases[symbol]
			if display == "" {
				display = symbol
				if symbol == EOF {
					display = "end of input"
				}
			}
			expected = append(expected, ExpectedToken{Symbol: symbol, Display: display})
		}
		if len(expected) > 0 {
			table.Expected[state.ID] = expected
		}
	}
	return table
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
