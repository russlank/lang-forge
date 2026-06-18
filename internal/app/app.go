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
  lang-forge generate --spec grammar.lf --target c --out ./generated`)
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := addInputFlags(fs)
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	result, code := loadAndBuild(input, stderr)
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
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	result, code := loadAndBuild(input, stderr)
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
	target := fs.String("target", "go", "target backend: go, csharp, or c")
	outDir := fs.String("out", "", "output directory")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *outDir == "" {
		fmt.Fprintln(stderr, "--out is required")
		return ExitUsage
	}
	result, code := loadAndBuild(input, stderr)
	if code != ExitOK {
		return code
	}
	if len(result.ParseTable.Conflicts) > 0 {
		printConflicts(stderr, result.ParseTable.Conflicts)
		return ExitConflict
	}
	var err error
	switch strings.ToLower(strings.TrimSpace(*target)) {
	case "go":
		err = gocodegen.Generate(gocodegen.Input{Spec: result.Spec, DFA: result.DFA, Grammar: result.Grammar, ParseTable: result.ParseTable}, *outDir)
	case "csharp", "cs":
		err = csharpcodegen.Generate(csharpcodegen.Input{Spec: result.Spec, DFA: result.DFA, Grammar: result.Grammar, ParseTable: result.ParseTable}, *outDir)
	case "c":
		err = ccodegen.Generate(ccodegen.Input{Spec: result.Spec, DFA: result.DFA, Grammar: result.Grammar, ParseTable: result.ParseTable}, *outDir)
	default:
		fmt.Fprintf(stderr, "target %q is not implemented yet; available: go, csharp, c\n", *target)
		return ExitUsage
	}
	if err != nil {
		fmt.Fprintf(stderr, "generate: %v\n", err)
		return ExitIO
	}
	fmt.Fprintf(stdout, "generated %s\n", *outDir)
	return ExitOK
}

type inputFlags struct {
	specFile string
	lexFile  string
	yaccFile string
}

func addInputFlags(fs *flag.FlagSet) *inputFlags {
	input := &inputFlags{}
	fs.StringVar(&input.specFile, "spec", "", "combined .lf specification")
	fs.StringVar(&input.lexFile, "lex", "", "legacy/split lex file")
	fs.StringVar(&input.yaccFile, "yacc", "", "legacy/split yacc file")
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

func loadAndBuild(input *inputFlags, stderr io.Writer) (*BuildResult, int) {
	parsed, diags, code := loadSpec(input)
	if code != ExitOK {
		for _, diag := range diags {
			fmt.Fprintln(stderr, diag.Error())
		}
		return nil, code
	}
	dfa, lexDiags := lex.BuildFromSpecWithScanner(parsed.Lexer, parsed.Scanner)
	diags = append(diags, lexDiags...)
	grammar, grammarDiags := parse.FromSpec(*parsed)
	diags = append(diags, grammarDiags...)
	if diags.HasErrors() {
		for _, diag := range diags {
			fmt.Fprintln(stderr, diag.Error())
		}
		return nil, ExitValidate
	}
	table := parse.Build(grammar, parsed.Grammar.Algorithm)
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
	if len(r.ParseTable.Conflicts) == 0 {
		fmt.Fprintln(w, "Conflicts: none")
		return
	}
	fmt.Fprintf(w, "Conflicts: %d\n", len(r.ParseTable.Conflicts))
	for _, conflict := range r.ParseTable.Conflicts {
		fmt.Fprintf(w, "  state %d on %s: %s\n", conflict.State, conflict.Symbol, conflict.Message)
	}
}

func printConflicts(w io.Writer, conflicts []parse.Conflict) {
	for _, conflict := range conflicts {
		fmt.Fprintf(w, "conflict: state %d on %s: %s\n", conflict.State, conflict.Symbol, conflict.Message)
	}
}
