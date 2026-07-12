package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	ccodegen "github.com/russlank/lang-forge/internal/codegen/c"
	cppcodegen "github.com/russlank/lang-forge/internal/codegen/cpp"
	csharpcodegen "github.com/russlank/lang-forge/internal/codegen/csharp"
	gocodegen "github.com/russlank/lang-forge/internal/codegen/golang"
	"github.com/russlank/lang-forge/internal/diagnostics"
	"github.com/russlank/lang-forge/internal/lex"
	"github.com/russlank/lang-forge/internal/parse"
	"github.com/russlank/lang-forge/internal/spec"
	"github.com/russlank/lang-forge/internal/version"
)

const (
	// ExitOK indicates successful command execution.
	ExitOK = 0
	// ExitUsage indicates invalid CLI usage.
	ExitUsage = 2
	// ExitValidate indicates specification validation failed.
	ExitValidate = 3
	// ExitConflict indicates parser table construction found conflicts.
	ExitConflict = 4
	// ExitIO indicates file or output writing failed.
	ExitIO = 5
)

// Run dispatches the lang-forge CLI and returns a process-style exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	_ = ctx
	if len(args) == 0 {
		printUsage(stderr)
		return ExitUsage
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, version.String())
		return ExitOK
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "inspect":
		return runInspect(args[1:], stdout, stderr)
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return ExitOK
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return ExitUsage
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `lang-forge - modern Lex/Yacc-style compiler tooling

Usage:
  lang-forge version
  lang-forge validate --spec grammar.lf
  lang-forge validate --lex lexer.l --yacc parser.y
  lang-forge inspect --spec grammar.lf --format text|json
  lang-forge generate --spec grammar.lf --target go --out ./generated
  lang-forge generate --spec grammar.lf --target csharp --out ./Generated
  lang-forge generate --spec grammar.lf --target c --out ./generated
  lang-forge generate --spec grammar.lf --target cpp --out ./generated

Common options:
  --verbosity N   diagnostics on stderr: 0 quiet, 1 stages, 2 decisions, 3 traces
  --verbose       same as --verbosity 1
  -v N            short form of --verbosity`)
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := addInputFlags(fs)
	verbosity := addVerbosityFlags(fs)
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	verbosityLevel, ok := verbosity.resolve(stderr)
	if !ok {
		return ExitUsage
	}
	result, code := loadAndBuild(input, stderr, buildOptions{Verbosity: verbosityLevel})
	if code != ExitOK {
		return code
	}
	if len(result.ParseTable.Conflicts) > 0 {
		printConflicts(stderr, result.ParseTable.Conflicts)
		return ExitConflict
	}
	fmt.Fprintf(stdout, "ok: %d lexer states, %d parser states, %d grammar rules\n", len(result.DFA.States), len(result.ParseTable.States), len(result.Grammar.Rules))
	return ExitOK
}

func runInspect(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := addInputFlags(fs)
	verbosity := addVerbosityFlags(fs)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	verbosityLevel, ok := verbosity.resolve(stderr)
	if !ok {
		return ExitUsage
	}
	result, code := loadAndBuild(input, stderr, buildOptions{Verbosity: verbosityLevel})
	if code != ExitOK {
		return code
	}
	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result.Summary()); err != nil {
			fmt.Fprintf(stderr, "write json: %v\n", err)
			return ExitIO
		}
	case "text":
		printTextSummary(stdout, result)
	default:
		fmt.Fprintf(stderr, "unknown inspect format %q\n", *format)
		return ExitUsage
	}
	if len(result.ParseTable.Conflicts) > 0 {
		return ExitConflict
	}
	return ExitOK
}

func runGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := addInputFlags(fs)
	verbosity := addVerbosityFlags(fs)
	target := fs.String("target", "go", "target backend: go, csharp, c, or cpp")
	outDir := fs.String("out", "", "output directory")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	verbosityLevel, ok := verbosity.resolve(stderr)
	if !ok {
		return ExitUsage
	}
	if *outDir == "" {
		fmt.Fprintln(stderr, "--out is required")
		return ExitUsage
	}
	result, code := loadAndBuild(input, stderr, buildOptions{Verbosity: verbosityLevel})
	if code != ExitOK {
		return code
	}
	if len(result.ParseTable.Conflicts) > 0 {
		printConflicts(stderr, result.ParseTable.Conflicts)
		return ExitConflict
	}
	logger := buildLogger{level: verbosityLevel, w: stderr}
	normalizedTarget := strings.ToLower(strings.TrimSpace(*target))
	logTarget := displayGenerateTarget(normalizedTarget)
	logger.Log(1, "generate: target=%s out=%s", logTarget, *outDir)
	var err error
	switch normalizedTarget {
	case "go":
		err = gocodegen.Generate(gocodegen.Input{Spec: result.Spec, DFA: result.DFA, Grammar: result.Grammar, ParseTable: result.ParseTable}, *outDir)
	case "csharp", "cs":
		err = csharpcodegen.Generate(csharpcodegen.Input{Spec: result.Spec, DFA: result.DFA, Grammar: result.Grammar, ParseTable: result.ParseTable}, *outDir)
	case "c":
		err = ccodegen.Generate(ccodegen.Input{Spec: result.Spec, DFA: result.DFA, Grammar: result.Grammar, ParseTable: result.ParseTable}, *outDir)
	case "cpp", "c++", "cplusplus":
		err = cppcodegen.Generate(cppcodegen.Input{Spec: result.Spec, DFA: result.DFA, Grammar: result.Grammar, ParseTable: result.ParseTable}, *outDir)
	default:
		fmt.Fprintf(stderr, "target %q is not implemented yet; available: go, csharp, c, cpp\n", *target)
		return ExitUsage
	}
	if err != nil {
		fmt.Fprintf(stderr, "generate: %v\n", err)
		return ExitIO
	}
	logger.Log(1, "generate: completed target=%s out=%s", logTarget, *outDir)
	fmt.Fprintf(stdout, "generated %s\n", *outDir)
	return ExitOK
}

func displayGenerateTarget(target string) string {
	switch target {
	case "":
		return "(empty)"
	case "csharp", "cs":
		return "csharp"
	case "cpp", "c++", "cplusplus":
		return "cpp"
	default:
		return target
	}
}

type inputFlags struct {
	specFile string
	lexFile  string
	yaccFile string
}

func addInputFlags(fs *flag.FlagSet) *inputFlags {
	input := &inputFlags{}
	fs.StringVar(&input.specFile, "spec", "", "combined .lf specification")
	fs.StringVar(&input.lexFile, "lex", "", "split lex file")
	fs.StringVar(&input.yaccFile, "yacc", "", "split yacc file")
	return input
}

// BuildResult is the complete in-memory output of loading and building a spec.
type BuildResult struct {
	Spec       *spec.Spec
	DFA        *lex.DFA
	Grammar    *parse.Grammar
	ParseTable *parse.Table
}

// Summary is the JSON shape emitted by the inspect command.
type Summary struct {
	Spec       *spec.Spec     `json:"spec"`
	Lexer      *lex.DFA       `json:"lexer"`
	Grammar    *parse.Grammar `json:"grammar"`
	ParseTable *parse.Table   `json:"parseTable"`
}

// Summary returns the inspectable build result.
func (r BuildResult) Summary() Summary {
	return Summary{Spec: r.Spec, Lexer: r.DFA, Grammar: r.Grammar, ParseTable: r.ParseTable}
}

func loadAndBuild(input *inputFlags, stderr io.Writer, options buildOptions) (*BuildResult, int) {
	logger := buildLogger{level: options.Verbosity, w: stderr}
	logInput(logger, input)
	parsed, diags, code := loadSpec(input)
	if code != ExitOK {
		for _, diag := range diags {
			fmt.Fprintln(stderr, diag.Error())
		}
		return nil, code
	}
	logSpec(logger, parsed)
	logger.Log(1, "lexer: building DFA encoding=%s invalid=%s rules=%d definitions=%d", displayValue(string(parsed.Scanner.WithDefaults().Encoding), string(spec.ScannerEncodingUTF8)), displayValue(string(parsed.Scanner.WithDefaults().Invalid), string(spec.ScannerInvalidError)), len(parsed.Lexer.Rules), len(parsed.Lexer.Definitions))
	dfa, lexDiags := lex.BuildFromSpecWithScanner(parsed.Lexer, parsed.Scanner)
	diags = append(diags, lexDiags...)
	if dfa != nil {
		logDFA(logger, dfa)
	}
	logger.Log(1, "grammar: normalizing start=%s algorithm=%s", displayValue(parsed.Grammar.Start, "first rule"), displayAlgorithm(parsed.Grammar.Algorithm))
	grammar, grammarDiags := parse.FromSpec(*parsed)
	diags = append(diags, grammarDiags...)
	if grammar != nil {
		logGrammar(logger, grammar)
	}
	if diags.HasErrors() {
		for _, diag := range diags {
			fmt.Fprintln(stderr, diag.Error())
		}
		return nil, ExitValidate
	}
	logger.Log(1, "parser: building table algorithm=%s", displayAlgorithm(parsed.Grammar.Algorithm))
	table := parse.Build(grammar, parsed.Grammar.Algorithm)
	logParseTable(logger, table)
	return &BuildResult{Spec: parsed, DFA: dfa, Grammar: grammar, ParseTable: table}, ExitOK
}

func loadSpec(input *inputFlags) (*spec.Spec, diagnostics.List, int) {
	switch {
	case input.specFile != "":
		data, err := os.ReadFile(input.specFile)
		if err != nil {
			return nil, diagnostics.List{{Severity: diagnostics.Error, Message: err.Error()}}, ExitIO
		}
		parsed, diags := spec.ParseCombined(data, input.specFile)
		if diags.HasErrors() {
			return parsed, diags, ExitValidate
		}
		return parsed, diags, ExitOK
	case input.lexFile != "" && input.yaccFile != "":
		lexData, err := os.ReadFile(input.lexFile)
		if err != nil {
			return nil, diagnostics.List{{Severity: diagnostics.Error, Message: err.Error()}}, ExitIO
		}
		yaccData, err := os.ReadFile(input.yaccFile)
		if err != nil {
			return nil, diagnostics.List{{Severity: diagnostics.Error, Message: err.Error()}}, ExitIO
		}
		lexSpec, lexDiags := spec.ParseLex(lexData, input.lexFile)
		yaccSpec, yaccDiags := spec.ParseYacc(yaccData, input.yaccFile)
		diags := append(lexDiags, yaccDiags...)
		merged := spec.MergeSplit(lexSpec, yaccSpec)
		if diags.HasErrors() {
			return merged, diags, ExitValidate
		}
		return merged, diags, ExitOK
	default:
		return nil, diagnostics.List{{Severity: diagnostics.Error, Message: "provide --spec or both --lex and --yacc"}}, ExitUsage
	}
}

func printTextSummary(w io.Writer, r *BuildResult) {
	fmt.Fprintf(w, "Spec: %s\n", r.Spec.SourceFile)
	fmt.Fprintf(w, "Tokens: %s\n", strings.Join(r.Spec.TokenNames(), ", "))
	fmt.Fprintf(w, "Lexer states: %d\n", len(r.DFA.States))
	fmt.Fprintf(w, "Grammar start: %s\n", r.Grammar.Start)
	fmt.Fprintf(w, "Parser algorithm: %s\n", r.ParseTable.Algorithm)
	fmt.Fprintf(w, "Grammar rules: %d\n", len(r.Grammar.Rules))
	fmt.Fprintf(w, "Parser states: %d\n", len(r.ParseTable.States))
	printIELRReport(w, r.ParseTable.IELR)
	if len(r.ParseTable.Conflicts) == 0 {
		fmt.Fprintln(w, "Conflicts: none")
		return
	}
	fmt.Fprintf(w, "Conflicts: %d\n", len(r.ParseTable.Conflicts))
	for _, conflict := range r.ParseTable.Conflicts {
		printConflict(w, conflict, "  ")
	}
}

func printIELRReport(w io.Writer, report *parse.IELRReport) {
	if report == nil {
		return
	}
	fmt.Fprintf(w, "IELR state counts: LALR=%d, IELR=%d, canonical=%d\n", report.LALRStates, report.IELRStates, report.CanonicalStates)
	fmt.Fprintf(w, "IELR merges: accepted=%d, rejected=%d\n", len(report.AcceptedMerges), len(report.RejectedMerges))
	for _, merge := range report.AcceptedMerges {
		fmt.Fprintf(w, "  accepted core %s from canonical states %s\n", ielrCoreSummary(merge), formatIntList(merge.CanonicalStates))
	}
	for _, merge := range report.RejectedMerges {
		fmt.Fprintf(w, "  rejected core %s from canonical states %s: %s", ielrCoreSummary(merge), formatIntList(merge.CanonicalStates), merge.Reason)
		if len(merge.ResultStates) > 0 {
			fmt.Fprintf(w, " -> %s", formatIntGroups(merge.ResultStates))
		}
		if len(merge.Conflicts) > 0 {
			fmt.Fprintf(w, " (%d candidate conflict(s))", len(merge.Conflicts))
		}
		fmt.Fprintln(w)
	}
}

func ielrCoreSummary(merge parse.IELRMergeReport) string {
	if len(merge.CoreDetails) == 0 {
		return fmt.Sprintf("%v", merge.Core)
	}
	parts := make([]string, 0, len(merge.CoreDetails))
	for _, detail := range merge.CoreDetails {
		parts = append(parts, detail.Display)
	}
	return strings.Join(parts, "; ")
}

func formatIntGroups(groups [][]int) string {
	parts := make([]string, 0, len(groups))
	for _, group := range groups {
		parts = append(parts, formatIntList(group))
	}
	return strings.Join(parts, " ")
}

func formatIntList(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func printConflicts(w io.Writer, conflicts []parse.Conflict) {
	for _, conflict := range conflicts {
		printConflict(w, conflict, "conflict: ")
	}
}

func printConflict(w io.Writer, conflict parse.Conflict, prefix string) {
	fmt.Fprintf(w, "%sstate %d on %s: %s\n", prefix, conflict.State, conflict.Symbol, conflict.Message)
	if conflict.Hint != "" {
		fmt.Fprintf(w, "%s  hint: %s\n", prefix, conflict.Hint)
	}
	printConflictRule(w, prefix, "existing", conflict.ExistingRule)
	printConflictRule(w, prefix, "incoming", conflict.IncomingRule)
	if len(conflict.ItemDetails) == 0 {
		return
	}
	fmt.Fprintf(w, "%s  state items:\n", prefix)
	for _, item := range conflict.ItemDetails {
		fmt.Fprintf(w, "%s    %s", prefix, item.Display)
		if item.Span.File != "" {
			fmt.Fprintf(w, " [%s]", item.Span)
		}
		fmt.Fprintln(w)
	}
}

func printConflictRule(w io.Writer, prefix, label string, rule *parse.ConflictRule) {
	if rule == nil {
		return
	}
	fmt.Fprintf(w, "%s  %s rule: %s", prefix, label, rule.Display)
	if rule.Span.File != "" {
		fmt.Fprintf(w, " [%s]", rule.Span)
	}
	fmt.Fprintln(w)
}
