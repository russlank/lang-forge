package lex

import (
	"fmt"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/spec"
)

// Rule is a normalized lexer rule after regex parsing and action lowering.
type Rule struct {
	Pattern string           `json:"pattern"`
	Token   string           `json:"token,omitempty"`
	Skip    bool             `json:"skip,omitempty"`
	Channel string           `json:"channel,omitempty"`
	Index   int              `json:"index"`
	Span    diagnostics.Span `json:"span"`
}

// DFA is the minimized deterministic automaton used by generated scanners.
type DFA struct {
	Start   int           `json:"start"`
	Scanner ScannerConfig `json:"scanner"`
	States  []DFAState    `json:"states"`
	Rules   []Rule        `json:"rules"`
}

// DFAState is one deterministic lexer state.
type DFAState struct {
	ID          int             `json:"id"`
	AcceptRule  int             `json:"acceptRule,omitempty"`
	Transitions []DFATransition `json:"transitions,omitempty"`
}

// DFATransition moves a DFA state to another state for a rune range set.
type DFATransition struct {
	Set    RangeSet `json:"set"`
	Target int      `json:"target"`
}

// Token is one tokenized lexeme produced by the in-process scanner.
type Token struct {
	Name        string `json:"name"`
	Lexeme      string `json:"lexeme"`
	Channel     string `json:"channel,omitempty"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	StartLine   int    `json:"startLine,omitempty"`
	StartColumn int    `json:"startColumn,omitempty"`
	EndLine     int    `json:"endLine,omitempty"`
	EndColumn   int    `json:"endColumn,omitempty"`
}

// BuildFromSpec compiles lexer definitions and rules into a minimized DFA.
func BuildFromSpec(lexer spec.LexerSpec) (*DFA, diagnostics.List) {
	return BuildFromSpecWithScanner(lexer, spec.DefaultScanner())
}

// BuildFromSpecWithScanner compiles lexer definitions and rules for scanner.
func BuildFromSpecWithScanner(lexer spec.LexerSpec, scanner spec.ScannerSpec) (*DFA, diagnostics.List) {
	var diags diagnostics.List
	config, err := scannerConfigFromSpec(scanner)
	if err != nil {
		diags.AddError("LF207", err.Error(), diagnostics.Span{})
		return nil, diags
	}
	defs := map[string]*Expr{}
	for _, def := range lexer.Definitions {
		expr, err := ParseRegexWithDomain(def.Pattern, config.Domain)
		if err != nil {
			diags.AddError("LF200", err.Error(), def.Span)
			continue
		}
		if _, exists := defs[def.Name]; exists {
			diags.AddError("LF201", "duplicate lexer definition `"+def.Name+"`", def.Span)
			continue
		}
		defs[def.Name] = expr
	}
	var compiled []compiledRule
	var rules []Rule
	for i, rule := range lexer.Rules {
		expr, err := ParseRegexWithDomain(rule.Pattern, config.Domain)
		if err != nil {
			diags.AddError("LF202", err.Error(), rule.Span)
			continue
		}
		expr, err = ExpandRefs(expr, defs)
		if err != nil {
			diags.AddError("LF203", err.Error(), rule.Span)
			continue
		}
		if err := validateScannerDomain(expr, config); err != nil {
			diags.AddError("LF205", err.Error(), rule.Span)
			continue
		}
		if expr.Nullable() {
			diags.AddError("LF206", "lexer rule must consume at least one scanner symbol", rule.Span)
			continue
		}
		outRule := Rule{Pattern: rule.Pattern, Index: i + 1, Span: rule.Span}
		switch rule.Action.Kind {
		case spec.ActionToken:
			outRule.Token = rule.Action.Token
		case spec.ActionSkip:
			outRule.Skip = true
		case spec.ActionChannel:
			outRule.Channel = rule.Action.Channel
			outRule.Token = rule.Action.Channel
		default:
			outRule.Token = fmt.Sprintf("RULE_%d", i+1)
		}
		rules = append(rules, outRule)
		compiled = append(compiled, compiledRule{expr: expr, rule: outRule})
	}
	if diags.HasErrors() {
		return nil, diags
	}
	dfa, err := buildDFA(compiled, rules, config)
	if err != nil {
		diags.AddError("LF204", err.Error(), diagnostics.Span{})
		return nil, diags
	}
	return dfa, diags
}

func validateScannerDomain(expr *Expr, config ScannerConfig) error {
	if expr == nil {
		return nil
	}
	switch expr.Kind {
	case ExprSet:
		if len(expr.Set.Normalize()) == 0 {
			return fmt.Errorf("regex character class is empty")
		}
		if !expr.Set.IsSubsetOf(config.Domain) {
			return fmt.Errorf("regex range %s is outside the %s scanner domain", expr.Set, config.Encoding)
		}
	case ExprConcat, ExprAlt:
		if err := validateScannerDomain(expr.Left, config); err != nil {
			return err
		}
		return validateScannerDomain(expr.Right, config)
	case ExprStar, ExprPlus, ExprOpt:
		return validateScannerDomain(expr.Child, config)
	}
	return nil
}

// Tokenize returns all visible tokens for input, optionally including channels.
func (d *DFA) Tokenize(input string, includeHidden bool) ([]Token, error) {
	var tokens []Token
	line, column := 1, 1
	for pos := 0; pos < len(input); {
		startLine, startColumn := line, column
		ruleIndex, end, err := d.Match(input, pos)
		if err != nil {
			return nil, err
		}
		if ruleIndex <= 0 {
			return nil, fmt.Errorf("no lexical rule matched byte %d near %q", pos, input[pos:minInt(len(input), pos+16)])
		}
		rule := d.Rules[ruleIndex-1]
		lexeme := input[pos:end]
		endLine, endColumn := advanceScalarPosition(input, pos, end, line, column)
		if !rule.Skip && (rule.Channel == "" || includeHidden) {
			tokens = append(tokens, Token{
				Name:        rule.Token,
				Lexeme:      lexeme,
				Channel:     rule.Channel,
				Start:       pos,
				End:         end,
				StartLine:   startLine,
				StartColumn: startColumn,
				EndLine:     endLine,
				EndColumn:   endColumn,
			})
		}
		if end == pos {
			return nil, fmt.Errorf("lexer rule %d matched empty input at byte %d", ruleIndex, pos)
		}
		line, column = endLine, endColumn
		pos = end
	}
	return tokens, nil
}

// Match returns the best rule index and end byte offset for input[start:].
func (d *DFA) Match(input string, start int) (int, int, error) {
	stateID := d.Start
	bestRule := 0
	bestEnd := start
	if d.States[stateID].AcceptRule > 0 {
		bestRule = d.States[stateID].AcceptRule
	}
	for pos := start; pos < len(input); {
		r, size, err := decodeUTF8ScannerRune(input, pos)
		if err != nil {
			if bestRule > 0 {
				break
			}
			return 0, start, err
		}
		next := -1
		for _, tr := range d.States[stateID].Transitions {
			if tr.Set.Contains(r) {
				next = tr.Target
				break
			}
		}
		if next < 0 {
			break
		}
		pos += size
		stateID = next
		if d.States[stateID].AcceptRule > 0 {
			bestRule = d.States[stateID].AcceptRule
			bestEnd = pos
		}
	}
	return bestRule, bestEnd, nil
}

type compiledRule struct {
	expr *Expr
	rule Rule
}

type nfa struct {
	transitions map[int][]nfaTransition
	accepts     map[int]int
	nextState   int
	start       int
}

type nfaTransition struct {
	to  int
	set RangeSet
}

func buildDFA(rules []compiledRule, outRules []Rule, config ScannerConfig) (*DFA, error) {
	n := &nfa{transitions: map[int][]nfaTransition{}, accepts: map[int]int{}}
	n.start = n.newState()
	var labels []RangeSet
	for _, rule := range rules {
		start, end := n.build(rule.expr, &labels)
		n.addEpsilon(n.start, start)
		n.accepts[end] = rule.rule.Index
	}
	partitions := Partition(labels, config.Domain)
	startSet := n.epsilonClosure(intSet{n.start: true})
	stateIDs := map[string]int{setKey(startSet): 0}
	states := []intSet{startSet}
	dfa := &DFA{Start: 0, Scanner: config, Rules: outRules}
	for i := 0; i < len(states); i++ {
		current := states[i]
		dfaState := DFAState{ID: i, AcceptRule: n.bestAccept(current)}
		for _, part := range partitions {
			moved := n.move(current, part)
			if len(moved) == 0 {
				continue
			}
			closed := n.epsilonClosure(moved)
			key := setKey(closed)
			id, ok := stateIDs[key]
			if !ok {
				id = len(states)
				stateIDs[key] = id
				states = append(states, closed)
			}
			dfaState.Transitions = append(dfaState.Transitions, DFATransition{Set: part, Target: id})
		}
		dfa.States = append(dfa.States, dfaState)
	}
	return dfa.Minimize(), nil
}

func (d *DFA) Minimize() *DFA {
	if d == nil || len(d.States) <= 1 {
		return d
	}
	alphabet := d.alphabet()
	classes := make([]int, len(d.States))
	classIDs := map[int]int{}
	for i, st := range d.States {
		id, ok := classIDs[st.AcceptRule]
		if !ok {
			id = len(classIDs)
			classIDs[st.AcceptRule] = id
		}
		classes[i] = id
	}
	for {
		next := make([]int, len(d.States))
		signatures := map[string]int{}
		for i := range d.States {
			sig := d.stateSignature(i, classes, alphabet)
			id, ok := signatures[sig]
			if !ok {
				id = len(signatures)
				signatures[sig] = id
			}
			next[i] = id
		}
		if sameClasses(classes, next) {
			classes = next
			break
		}
		classes = next
	}
	blocks := map[int][]int{}
	for state, class := range classes {
		blocks[class] = append(blocks[class], state)
	}
	type blockInfo struct {
		class int
		min   int
	}
	var ordered []blockInfo
	for class, states := range blocks {
		sort.Ints(states)
		ordered = append(ordered, blockInfo{class: class, min: states[0]})
	}
	startClass := classes[d.Start]
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].class == startClass {
			return true
		}
		if ordered[j].class == startClass {
			return false
		}
		return ordered[i].min < ordered[j].min
	})
	classToNew := map[int]int{}
	for i, block := range ordered {
		classToNew[block.class] = i
	}
	minimized := &DFA{Start: classToNew[startClass], Scanner: d.Scanner, Rules: d.Rules}
	for newID, block := range ordered {
		rep := blocks[block.class][0]
		state := DFAState{ID: newID, AcceptRule: d.States[rep].AcceptRule}
		for _, tr := range d.States[rep].Transitions {
			state.Transitions = append(state.Transitions, DFATransition{Set: tr.Set, Target: classToNew[classes[tr.Target]]})
		}
		minimized.States = append(minimized.States, state)
	}
	return minimized
}

func (d *DFA) alphabet() []RangeSet {
	seen := map[string]RangeSet{}
	for _, st := range d.States {
		for _, tr := range st.Transitions {
			seen[tr.Set.String()] = tr.Set
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]RangeSet, 0, len(keys))
	for _, key := range keys {
		out = append(out, seen[key])
	}
	return out
}

func (d *DFA) stateSignature(state int, classes []int, alphabet []RangeSet) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("accept:%d", d.States[state].AcceptRule))
	for _, sym := range alphabet {
		target := -1
		for _, tr := range d.States[state].Transitions {
			if tr.Set.String() == sym.String() {
				target = classes[tr.Target]
				break
			}
		}
		b.WriteString(fmt.Sprintf("|%s:%d", sym.String(), target))
	}
	return b.String()
}

func sameClasses(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (n *nfa) build(expr *Expr, labels *[]RangeSet) (int, int) {
	switch expr.Kind {
	case ExprEmpty:
		start, end := n.newState(), n.newState()
		n.addEpsilon(start, end)
		return start, end
	case ExprSet:
		start, end := n.newState(), n.newState()
		n.addTransition(start, end, expr.Set)
		*labels = append(*labels, expr.Set)
		return start, end
	case ExprConcat:
		leftStart, leftEnd := n.build(expr.Left, labels)
		rightStart, rightEnd := n.build(expr.Right, labels)
		n.addEpsilon(leftEnd, rightStart)
		return leftStart, rightEnd
	case ExprAlt:
		start, end := n.newState(), n.newState()
		leftStart, leftEnd := n.build(expr.Left, labels)
		rightStart, rightEnd := n.build(expr.Right, labels)
		n.addEpsilon(start, leftStart)
		n.addEpsilon(start, rightStart)
		n.addEpsilon(leftEnd, end)
		n.addEpsilon(rightEnd, end)
		return start, end
	case ExprStar:
		start, end := n.newState(), n.newState()
		childStart, childEnd := n.build(expr.Child, labels)
		n.addEpsilon(start, end)
		n.addEpsilon(start, childStart)
		n.addEpsilon(childEnd, childStart)
		n.addEpsilon(childEnd, end)
		return start, end
	case ExprPlus:
		start, end := n.newState(), n.newState()
		childStart, childEnd := n.build(expr.Child, labels)
		n.addEpsilon(start, childStart)
		n.addEpsilon(childEnd, childStart)
		n.addEpsilon(childEnd, end)
		return start, end
	case ExprOpt:
		start, end := n.newState(), n.newState()
		childStart, childEnd := n.build(expr.Child, labels)
		n.addEpsilon(start, childStart)
		n.addEpsilon(start, end)
		n.addEpsilon(childEnd, end)
		return start, end
	default:
		panic("unsupported regex expression after expansion: " + expr.Kind)
	}
}

func (n *nfa) newState() int {
	id := n.nextState
	n.nextState++
	return id
}

func (n *nfa) addEpsilon(from, to int) {
	n.transitions[from] = append(n.transitions[from], nfaTransition{to: to})
}

func (n *nfa) addTransition(from, to int, set RangeSet) {
	n.transitions[from] = append(n.transitions[from], nfaTransition{to: to, set: set.Normalize()})
}

type intSet map[int]bool

func (n *nfa) epsilonClosure(start intSet) intSet {
	out := intSet{}
	stack := make([]int, 0, len(start))
	for state := range start {
		out[state] = true
		stack = append(stack, state)
	}
	for len(stack) > 0 {
		state := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, tr := range n.transitions[state] {
			if len(tr.set) == 0 && !out[tr.to] {
				out[tr.to] = true
				stack = append(stack, tr.to)
			}
		}
	}
	return out
}

func (n *nfa) move(states intSet, set RangeSet) intSet {
	out := intSet{}
	for state := range states {
		for _, tr := range n.transitions[state] {
			if len(tr.set) > 0 && tr.set.Intersects(set) {
				out[tr.to] = true
			}
		}
	}
	return out
}

func (n *nfa) bestAccept(states intSet) int {
	best := 0
	for state := range states {
		if rule := n.accepts[state]; rule > 0 && (best == 0 || rule < best) {
			best = rule
		}
	}
	return best
}

func setKey(set intSet) string {
	ids := make([]int, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprint(id))
	}
	return b.String()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
