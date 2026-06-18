package parse

import (
	"fmt"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/parseralgo"
)

type lr1ItemSet map[LR1Item]bool

// BuildCanonicalLR1 constructs the full canonical LR(1) parser table.
func BuildCanonicalLR1(g *Grammar) *Table {
	aug := augment(g)
	states, transitions := canonicalLR1(aug)
	return buildLR1Table(parseralgo.Canonical, aug, states, transitions)
}

// BuildLALR constructs a merged-core LALR(1) parser table.
func BuildLALR(g *Grammar) *Table {
	aug := augment(g)
	canonicalStates, canonicalTransitions := canonicalLR1(aug)
	coreIDs := map[string]int{}
	oldToMerged := map[int]int{}
	var merged []lr1ItemSet
	for oldID, state := range canonicalStates {
		key := lr1CoreKey(state)
		id, ok := coreIDs[key]
		if !ok {
			id = len(merged)
			coreIDs[key] = id
			merged = append(merged, lr1ItemSet{})
		}
		oldToMerged[oldID] = id
		for item := range state {
			merged[id][item] = true
		}
	}
	transitions := map[int]map[string]int{}
	for oldFrom, bySym := range canonicalTransitions {
		from := oldToMerged[oldFrom]
		if transitions[from] == nil {
			transitions[from] = map[string]int{}
		}
		for sym, oldTo := range bySym {
			to := oldToMerged[oldTo]
			if existing, exists := transitions[from][sym]; exists && existing != to {
				// Canonical LR(1) states with identical LR(0) cores should have
				// compatible goto cores. If this ever trips, keep the first edge so
				// table construction remains deterministic and conflict reporting can
				// still expose the grammar issue.
				continue
			}
			transitions[from][sym] = to
		}
	}
	return buildLR1Table(parseralgo.LALR, aug, merged, transitions)
}

// BuildIELR constructs a conservative IELR(1) parser table.
//
// LangForge builds IELR from canonical LR(1) states, then keeps an LALR-style
// merge only when the merged state has deterministic actions and transitions.
// Unsafe merges are split back to canonical states. This preserves canonical
// LR(1) recognition while retaining LALR-sized tables for grammars whose core
// merges are already safe.
func BuildIELR(g *Grammar) *Table {
	aug := augment(g)
	canonicalStates, canonicalTransitions := canonicalLR1(aug)
	partitions, oldToPartition := initialCorePartitions(canonicalStates)

	for {
		splitPartitions, splitMap, splitChanged := splitInadequatePartitions(aug, canonicalStates, canonicalTransitions, partitions)
		partitions = splitPartitions
		oldToPartition = splitMap

		refinedPartitions, refinedMap, refinedChanged := refinePartitionsByTransitionSignature(canonicalTransitions, partitions, oldToPartition)
		partitions = refinedPartitions
		oldToPartition = refinedMap

		if !splitChanged && !refinedChanged {
			break
		}
	}

	states, transitions := mergedLR1StatesAndTransitions(canonicalStates, canonicalTransitions, partitions, oldToPartition)
	return buildLR1Table(parseralgo.IELR, aug, states, transitions)
}

type lr1Partition struct {
	Members []int
}

func initialCorePartitions(states []lr1ItemSet) ([]lr1Partition, map[int]int) {
	coreIDs := map[string]int{}
	var partitions []lr1Partition
	for oldID, state := range states {
		key := lr1CoreKey(state)
		id, ok := coreIDs[key]
		if !ok {
			id = len(partitions)
			coreIDs[key] = id
			partitions = append(partitions, lr1Partition{})
		}
		partitions[id].Members = append(partitions[id].Members, oldID)
	}
	return partitions, partitionStateMap(partitions)
}

// splitInadequatePartitions rejects LALR-style core merges that would create a
// parser action conflict after LR(1) lookaheads are unioned.
func splitInadequatePartitions(g *Grammar, states []lr1ItemSet, transitions map[int]map[string]int, partitions []lr1Partition) ([]lr1Partition, map[int]int, bool) {
	out := make([]lr1Partition, 0, len(partitions))
	changed := false
	for _, partition := range partitions {
		if len(partition.Members) <= 1 {
			out = append(out, partition)
			continue
		}
		items := unionLR1Items(states, partition.Members)
		shifts := shiftTerminalsForMembers(g, transitions, partition.Members)
		if len(mergedLR1ActionConflicts(g, items, shifts)) == 0 {
			out = append(out, partition)
			continue
		}
		changed = true
		for _, member := range partition.Members {
			out = append(out, lr1Partition{Members: []int{member}})
		}
	}
	return out, partitionStateMap(out), changed
}

func mergedLR1ActionConflicts(g *Grammar, items lr1ItemSet, shiftSymbols map[string]bool) []Conflict {
	table := &Table{
		Algorithm: parseralgo.IELR,
		Actions:   map[int]map[string]Action{},
		Gotos:     map[int]map[string]int{},
		Rules:     g.Rules,
	}
	core := coreItemSet(items)
	for sym := range shiftSymbols {
		table.setAction(0, sym, Action{Kind: ActionShift, State: 1}, core)
	}
	for item := range items {
		rule := g.Rules[item.Rule]
		if item.Dot != len(rule.RHS) {
			continue
		}
		if rule.ID == 0 {
			table.setAction(0, EOF, Action{Kind: ActionAccept, Rule: rule.ID}, core)
			continue
		}
		table.setAction(0, item.Lookahead, Action{Kind: ActionReduce, Rule: rule.ID}, core)
	}
	return table.Conflicts
}

// shiftTerminalsForMembers collects terminal shifts that would exist in a
// merged state. The concrete target state is irrelevant for action conflicts.
func shiftTerminalsForMembers(g *Grammar, transitions map[int]map[string]int, members []int) map[string]bool {
	out := map[string]bool{}
	for _, member := range members {
		for sym := range transitions[member] {
			if g.Terminals[sym] && sym != EOF {
				out[sym] = true
			}
		}
	}
	return out
}

// refinePartitionsByTransitionSignature ensures every remaining merged state
// has one deterministic transition target per grammar symbol.
func refinePartitionsByTransitionSignature(transitions map[int]map[string]int, partitions []lr1Partition, oldToPartition map[int]int) ([]lr1Partition, map[int]int, bool) {
	out := make([]lr1Partition, 0, len(partitions))
	changed := false
	for _, partition := range partitions {
		if len(partition.Members) <= 1 {
			out = append(out, partition)
			continue
		}
		groupIDs := map[string]int{}
		var groups []lr1Partition
		for _, member := range partition.Members {
			key := transitionSignature(transitions, member, oldToPartition)
			id, ok := groupIDs[key]
			if !ok {
				id = len(groups)
				groupIDs[key] = id
				groups = append(groups, lr1Partition{})
			}
			groups[id].Members = append(groups[id].Members, member)
		}
		if len(groups) > 1 {
			changed = true
		}
		out = append(out, groups...)
	}
	return out, partitionStateMap(out), changed
}

func transitionSignature(transitions map[int]map[string]int, state int, oldToPartition map[int]int) string {
	bySym := transitions[state]
	symbols := make([]string, 0, len(bySym))
	for sym := range bySym {
		symbols = append(symbols, sym)
	}
	sort.Strings(symbols)
	var b strings.Builder
	for _, sym := range symbols {
		b.WriteString(sym)
		b.WriteByte('=')
		b.WriteString(fmt.Sprint(oldToPartition[bySym[sym]]))
		b.WriteByte(';')
	}
	return b.String()
}

func mergedLR1StatesAndTransitions(states []lr1ItemSet, transitions map[int]map[string]int, partitions []lr1Partition, oldToPartition map[int]int) ([]lr1ItemSet, map[int]map[string]int) {
	merged := make([]lr1ItemSet, len(partitions))
	for id, partition := range partitions {
		merged[id] = unionLR1Items(states, partition.Members)
	}

	outTransitions := map[int]map[string]int{}
	for oldFrom, bySym := range transitions {
		from := oldToPartition[oldFrom]
		if outTransitions[from] == nil {
			outTransitions[from] = map[string]int{}
		}
		for sym, oldTo := range bySym {
			to := oldToPartition[oldTo]
			if existing, exists := outTransitions[from][sym]; exists && existing != to {
				continue
			}
			outTransitions[from][sym] = to
		}
	}
	return merged, outTransitions
}

func unionLR1Items(states []lr1ItemSet, members []int) lr1ItemSet {
	out := lr1ItemSet{}
	for _, member := range members {
		for item := range states[member] {
			out[item] = true
		}
	}
	return out
}

func partitionStateMap(partitions []lr1Partition) map[int]int {
	out := map[int]int{}
	for partitionID, partition := range partitions {
		for _, member := range partition.Members {
			out[member] = partitionID
		}
	}
	return out
}

func buildLR1Table(algorithm string, g *Grammar, states []lr1ItemSet, transitions map[int]map[string]int) *Table {
	table := &Table{
		Algorithm: algorithm,
		Actions:   map[int]map[string]Action{},
		Gotos:     map[int]map[string]int{},
		Rules:     g.Rules,
	}
	for i, itemSet := range states {
		state := State{ID: i, Items: sortedCoreItems(itemSet), LR1Items: sortedLR1Items(itemSet), Transitions: map[string]int{}}
		for sym, to := range transitions[i] {
			state.Transitions[sym] = to
		}
		table.States = append(table.States, state)
	}
	for from, bySym := range transitions {
		for sym, to := range bySym {
			if g.Terminals[sym] && sym != EOF {
				table.setAction(from, sym, Action{Kind: ActionShift, State: to}, coreItemSet(states[from]))
			} else if g.Nonterminals[sym] {
				if table.Gotos[from] == nil {
					table.Gotos[from] = map[string]int{}
				}
				table.Gotos[from][sym] = to
			}
		}
	}
	for stateID, itemSet := range states {
		for item := range itemSet {
			rule := g.Rules[item.Rule]
			if item.Dot != len(rule.RHS) {
				continue
			}
			if rule.ID == 0 {
				table.setAction(stateID, EOF, Action{Kind: ActionAccept, Rule: rule.ID}, coreItemSet(itemSet))
				continue
			}
			table.setAction(stateID, item.Lookahead, Action{Kind: ActionReduce, Rule: rule.ID}, coreItemSet(itemSet))
		}
	}
	return table
}

func canonicalLR1(g *Grammar) ([]lr1ItemSet, map[int]map[string]int) {
	start := closureLR1(g, lr1ItemSet{{Rule: 0, Dot: 0, Lookahead: EOF}: true})
	states := []lr1ItemSet{start}
	ids := map[string]int{lr1ItemSetKey(start): 0}
	transitions := map[int]map[string]int{}
	for i := 0; i < len(states); i++ {
		for _, sym := range g.Symbols() {
			next := gotoLR1(g, states[i], sym)
			if len(next) == 0 {
				continue
			}
			key := lr1ItemSetKey(next)
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

func closureLR1(g *Grammar, items lr1ItemSet) lr1ItemSet {
	first := g.FirstSets()
	nullable := g.Nullable()
	out := lr1ItemSet{}
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
			tail := append([]string{}, rule.RHS[item.Dot+1:]...)
			tail = append(tail, item.Lookahead)
			lookaheads := firstSequence(first, nullable, g.Terminals, tail)
			for idx, candidate := range g.Rules {
				if candidate.LHS != sym {
					continue
				}
				for la := range lookaheads {
					newItem := LR1Item{Rule: idx, Dot: 0, Lookahead: la}
					if !out[newItem] {
						out[newItem] = true
						changed = true
					}
				}
			}
		}
	}
	return out
}

func gotoLR1(g *Grammar, items lr1ItemSet, sym string) lr1ItemSet {
	moved := lr1ItemSet{}
	for item := range items {
		rule := g.Rules[item.Rule]
		if item.Dot < len(rule.RHS) && rule.RHS[item.Dot] == sym {
			moved[LR1Item{Rule: item.Rule, Dot: item.Dot + 1, Lookahead: item.Lookahead}] = true
		}
	}
	if len(moved) == 0 {
		return nil
	}
	return closureLR1(g, moved)
}

func firstSequence(first map[string]map[string]bool, nullable map[string]bool, terminals map[string]bool, symbols []string) map[string]bool {
	out := map[string]bool{}
	if len(symbols) == 0 {
		return out
	}
	for _, sym := range symbols {
		for tok := range first[sym] {
			out[tok] = true
		}
		if terminals[sym] || !nullable[sym] {
			return out
		}
	}
	return out
}

func sortedLR1Items(items lr1ItemSet) []LR1Item {
	out := make([]LR1Item, 0, len(items))
	for item := range items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rule != out[j].Rule {
			return out[i].Rule < out[j].Rule
		}
		if out[i].Dot != out[j].Dot {
			return out[i].Dot < out[j].Dot
		}
		return out[i].Lookahead < out[j].Lookahead
	})
	return out
}

func sortedCoreItems(items lr1ItemSet) []Item {
	seen := itemSet{}
	for item := range items {
		seen[Item{Rule: item.Rule, Dot: item.Dot}] = true
	}
	return sortedItems(seen)
}

func coreItemSet(items lr1ItemSet) itemSet {
	out := itemSet{}
	for item := range items {
		out[Item{Rule: item.Rule, Dot: item.Dot}] = true
	}
	return out
}

func lr1ItemSetKey(items lr1ItemSet) string {
	sorted := sortedLR1Items(items)
	var b strings.Builder
	for i, item := range sorted {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprintf("%d.%d.%s", item.Rule, item.Dot, item.Lookahead))
	}
	return b.String()
}

func lr1CoreKey(items lr1ItemSet) string {
	core := sortedCoreItems(items)
	var b strings.Builder
	for i, item := range core {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprintf("%d.%d", item.Rule, item.Dot))
	}
	return b.String()
}
