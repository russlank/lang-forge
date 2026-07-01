package app

import (
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/lex"
	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/parseralgo"
	"github.com/russlank/lang-forge/internal/spec"
)

const maxVerbosity = 3

type verbosityFlags struct {
	level      int
	shortLevel int
	verbose    bool
}

func addVerbosityFlags(fs *flag.FlagSet) *verbosityFlags {
	flags := &verbosityFlags{}
	fs.IntVar(&flags.level, "verbosity", 0, "diagnostic verbosity: 0 quiet, 1 stages, 2 decisions, 3 traces")
	fs.IntVar(&flags.shortLevel, "v", 0, "short form of --verbosity")
	fs.BoolVar(&flags.verbose, "verbose", false, "enable stage-level diagnostics, equivalent to --verbosity 1")
	return flags
}

func (f *verbosityFlags) resolve(stderr io.Writer) (int, bool) {
	level := f.level
	if f.shortLevel > level {
		level = f.shortLevel
	}
	if f.verbose && level < 1 {
		level = 1
	}
	if level < 0 || level > maxVerbosity {
		fmt.Fprintf(stderr, "--verbosity must be between 0 and %d\n", maxVerbosity)
		return 0, false
	}
	return level, true
}

type buildOptions struct {
	Verbosity int
}

type buildLogger struct {
	level int
	w     io.Writer
}

func (l buildLogger) Enabled(level int) bool {
	return l.w != nil && l.level >= level
}

func (l buildLogger) Log(level int, format string, args ...any) {
	if !l.Enabled(level) {
		return
	}
	fmt.Fprintf(l.w, "[lf] "+format+"\n", args...)
}

func logInput(logger buildLogger, input *inputFlags) {
	switch {
	case input.specFile != "":
		logger.Log(1, "load: combined spec=%s", input.specFile)
	case input.lexFile != "" || input.yaccFile != "":
		logger.Log(1, "load: split lex=%s yacc=%s", displayValue(input.lexFile, "missing"), displayValue(input.yaccFile, "missing"))
	default:
		logger.Log(1, "load: waiting for --spec or --lex/--yacc")
	}
}

func logSpec(logger buildLogger, parsed *spec.Spec) {
	if parsed == nil {
		return
	}
	tokens := parsed.TokenNames()
	logger.Log(1, "spec: mode=%s source=%s target=%s package=%s tokens=%d lexerRules=%d grammarAlternatives=%d", displayValue(string(parsed.Mode), "unknown"), displayValue(parsed.SourceFile, "inline"), displayValue(parsed.Target, "unspecified"), displayValue(parsed.Package, "unspecified"), len(tokens), len(parsed.Lexer.Rules), countGrammarAlternatives(parsed))
	if logger.Enabled(2) {
		logger.Log(2, "spec: tokens=%s", strings.Join(tokens, ", "))
		logger.Log(2, "semantics: modes=%s includes=%d types=%d actionLabels=%d", formatSemanticModes(parsed.Semantics.Modes), len(parsed.Semantics.Includes), len(parsed.Semantics.Types), countSemanticActionLabels(parsed))
		for i, rule := range parsed.Lexer.Rules {
			logger.Log(2, "lexer rule %d: pattern=%q action=%s source=%s", i+1, rule.Pattern, formatLexAction(rule.Action), formatSpan(rule.Span))
		}
	}
}

func logDFA(logger buildLogger, dfa *lex.DFA) {
	logger.Log(1, "lexer: DFA states=%d transitions=%d acceptingStates=%d visibleRules=%d skippedRules=%d channelRules=%d", len(dfa.States), countDFATransitions(dfa), countAcceptingStates(dfa), countVisibleRules(dfa), countSkippedRules(dfa), countChannelRules(dfa))
	if !logger.Enabled(3) {
		return
	}
	for _, state := range dfa.States {
		logger.Log(3, "lexer state %d: acceptRule=%d transitions=%s", state.ID, state.AcceptRule, formatDFATransitions(state.Transitions))
	}
}

func logGrammar(logger buildLogger, grammar *parse.Grammar) {
	logger.Log(1, "grammar: start=%s terminals=%d nonterminals=%d rules=%d", grammar.Start, len(grammar.Terminals), len(grammar.Nonterminals), len(grammar.Rules))
	if !logger.Enabled(2) {
		return
	}
	for _, rule := range grammar.Rules {
		logger.Log(2, "grammar rule %d: %s source=%s", rule.ID, formatGrammarRule(rule), formatSpan(rule.Span))
	}
}

func logParseTable(logger buildLogger, table *parse.Table) {
	actions, shifts, reduces, accepts := countParserActions(table)
	logger.Log(1, "parser: table algorithm=%s states=%d actions=%d gotos=%d conflicts=%d recovery=%t", table.Algorithm, len(table.States), actions, countParserGotos(table), len(table.Conflicts), table.ErrorRecovery)
	if logger.Enabled(2) {
		logger.Log(2, "parser: actionKinds shifts=%d reduces=%d accepts=%d expectedRows=%d", shifts, reduces, accepts, countExpectedRows(table))
		if table.IELR != nil {
			logger.Log(2, "parser: IELR lalrStates=%d ielrStates=%d canonicalStates=%d acceptedMerges=%d rejectedMerges=%d", table.IELR.LALRStates, table.IELR.IELRStates, table.IELR.CanonicalStates, len(table.IELR.AcceptedMerges), len(table.IELR.RejectedMerges))
		}
	}
	if !logger.Enabled(3) {
		return
	}
	for _, state := range table.States {
		logger.Log(3, "parser state %d: actions=%s gotos=%s transitions=%s", state.ID, formatParserActions(table.Actions[state.ID]), formatParserGotos(table.Gotos[state.ID]), formatIntMap(state.Transitions))
	}
}

func displayValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func displayAlgorithm(value string) string {
	if strings.TrimSpace(value) == "" {
		return parseralgo.Default
	}
	return value
}

func countGrammarAlternatives(parsed *spec.Spec) int {
	count := 0
	for _, rule := range parsed.Grammar.Rules {
		count += len(rule.Alternatives)
	}
	return count
}

func countSemanticActionLabels(parsed *spec.Spec) int {
	count := 0
	for _, rule := range parsed.Grammar.Rules {
		for _, alternative := range rule.Alternatives {
			count += len(alternative.Actions)
		}
	}
	return count
}

func formatSemanticModes(modes map[string]spec.SemanticActionMode) string {
	if len(modes) == 0 {
		return "default-reducer"
	}
	keys := sortedKeys(modes)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+string(modes[key]))
	}
	return strings.Join(parts, ",")
}

func formatLexAction(action spec.LexAction) string {
	switch action.Kind {
	case spec.ActionToken:
		return "token(" + action.Token + ")"
	case spec.ActionSkip:
		return "skip"
	case spec.ActionChannel:
		return "channel(" + action.Channel + ")"
	case spec.ActionRaw:
		return "raw(" + action.Raw + ")"
	default:
		return displayValue(string(action.Kind), "unknown")
	}
}

func formatGrammarRule(rule parse.Rule) string {
	parts := make([]string, 0, len(rule.RHS))
	for i, symbol := range rule.RHS {
		if i < len(rule.Labels) && rule.Labels[i] != "" {
			parts = append(parts, rule.Labels[i]+":"+symbol)
			continue
		}
		parts = append(parts, symbol)
	}
	if len(parts) == 0 {
		parts = append(parts, "%empty")
	}
	return rule.LHS + " -> " + strings.Join(parts, " ") + formatRuleActions(rule.Actions)
}

func formatRuleActions(actions map[string]string) string {
	if len(actions) == 0 {
		return ""
	}
	keys := sortedKeys(actions)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+": "+actions[key])
	}
	return " {" + strings.Join(parts, ", ") + "}"
}

func formatSpan(span diagnostics.Span) string {
	if span.File == "" {
		return "unknown"
	}
	return span.String()
}

func countDFATransitions(dfa *lex.DFA) int {
	count := 0
	for _, state := range dfa.States {
		count += len(state.Transitions)
	}
	return count
}

func countAcceptingStates(dfa *lex.DFA) int {
	count := 0
	for _, state := range dfa.States {
		if state.AcceptRule > 0 {
			count++
		}
	}
	return count
}

func countVisibleRules(dfa *lex.DFA) int {
	count := 0
	for _, rule := range dfa.Rules {
		if !rule.Skip && rule.Channel == "" {
			count++
		}
	}
	return count
}

func countSkippedRules(dfa *lex.DFA) int {
	count := 0
	for _, rule := range dfa.Rules {
		if rule.Skip {
			count++
		}
	}
	return count
}

func countChannelRules(dfa *lex.DFA) int {
	count := 0
	for _, rule := range dfa.Rules {
		if rule.Channel != "" {
			count++
		}
	}
	return count
}

func formatDFATransitions(transitions []lex.DFATransition) string {
	if len(transitions) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(transitions))
	for _, transition := range transitions {
		parts = append(parts, fmt.Sprintf("%s->%d", transition.Set, transition.Target))
	}
	return strings.Join(parts, ",")
}

func countParserActions(table *parse.Table) (total, shifts, reduces, accepts int) {
	for _, bySymbol := range table.Actions {
		for _, action := range bySymbol {
			total++
			switch action.Kind {
			case parse.ActionShift:
				shifts++
			case parse.ActionReduce:
				reduces++
			case parse.ActionAccept:
				accepts++
			}
		}
	}
	return total, shifts, reduces, accepts
}

func countParserGotos(table *parse.Table) int {
	count := 0
	for _, bySymbol := range table.Gotos {
		count += len(bySymbol)
	}
	return count
}

func countExpectedRows(table *parse.Table) int {
	count := 0
	for _, expected := range table.Expected {
		if len(expected) > 0 {
			count++
		}
	}
	return count
}

func formatParserActions(actions map[string]parse.Action) string {
	if len(actions) == 0 {
		return "none"
	}
	keys := sortedKeys(actions)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+":"+formatParserAction(actions[key]))
	}
	return strings.Join(parts, ",")
}

func formatParserAction(action parse.Action) string {
	switch action.Kind {
	case parse.ActionShift:
		return fmt.Sprintf("shift(%d)", action.State)
	case parse.ActionReduce:
		return fmt.Sprintf("reduce(%d)", action.Rule)
	case parse.ActionAccept:
		return "accept"
	case parse.ActionError:
		return "error"
	default:
		return string(action.Kind)
	}
}

func formatParserGotos(gotos map[string]int) string {
	return formatIntMap(gotos)
}

func formatIntMap(values map[string]int) string {
	if len(values) == 0 {
		return "none"
	}
	keys := sortedKeys(values)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, values[key]))
	}
	return strings.Join(parts, ",")
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
