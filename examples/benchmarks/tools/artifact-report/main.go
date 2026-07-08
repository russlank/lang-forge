package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	repo := flag.String("repo", "../..", "repository root")
	out := flag.String("out", "../../dist/benchmarks", "benchmark report directory")
	flag.Parse()

	rows, err := collect(*repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "artifact report: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "artifact report: %v\n", err)
		os.Exit(1)
	}
	if err := writeJSON(filepath.Join(*out, "generated-artifacts.json"), rows); err != nil {
		fmt.Fprintf(os.Stderr, "artifact report: %v\n", err)
		os.Exit(1)
	}
	if err := writeMarkdown(filepath.Join(*out, "generated-artifacts.md"), rows); err != nil {
		fmt.Fprintf(os.Stderr, "artifact report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("generated artifact reports written under %s\n", *out)
}

type artifactRow struct {
	Example        string `json:"example"`
	Target         string `json:"target"`
	GeneratedBytes int64  `json:"generatedBytes"`
	LexerStates    int    `json:"lexerStates"`
	ParserStates   int    `json:"parserStates"`
	ParserActions  int    `json:"parserActions"`
	ParserGotos    int    `json:"parserGotos"`
	GrammarRules   int    `json:"grammarRules"`
	Recovery       bool   `json:"recovery"`
	Path           string `json:"path"`
}

type generatedDir struct {
	Example string
	Target  string
	Path    string
}

type tablesFile struct {
	Lexer struct {
		States []json.RawMessage `json:"states"`
	} `json:"lexer"`
	ParseTable struct {
		States  []json.RawMessage                     `json:"states"`
		Actions map[string]map[string]json.RawMessage `json:"actions"`
		Gotos   map[string]map[string]int             `json:"gotos"`
		Rules   []struct {
			RHS []string `json:"rhs"`
		} `json:"rules"`
	} `json:"parseTable"`
}

func collect(repo string) ([]artifactRow, error) {
	dirs := []generatedDir{
		{Example: "calc", Target: "go", Path: "examples/go/calc/generated"},
		{Example: "draw", Target: "go", Path: "examples/go/draw/generated"},
		{Example: "parser-recovery", Target: "go", Path: "examples/go/parser-recovery/generated"},
		{Example: "calc", Target: "csharp", Path: "examples/csharp/calc/Generated"},
		{Example: "draw", Target: "csharp", Path: "examples/csharp/draw/Generated"},
		{Example: "parser-recovery", Target: "csharp", Path: "examples/csharp/parser-recovery/Generated"},
	}
	rows := make([]artifactRow, 0, len(dirs))
	for _, dir := range dirs {
		generatedPath := filepath.Join(repo, filepath.FromSlash(dir.Path))
		tables, err := readTables(filepath.Join(generatedPath, "langforge.tables.json"))
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", dir.Target, dir.Example, err)
		}
		size, err := generatedBytes(generatedPath)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", dir.Target, dir.Example, err)
		}
		rows = append(rows, artifactRow{
			Example:        dir.Example,
			Target:         dir.Target,
			GeneratedBytes: size,
			LexerStates:    len(tables.Lexer.States),
			ParserStates:   len(tables.ParseTable.States),
			ParserActions:  nestedEntryCount(tables.ParseTable.Actions),
			ParserGotos:    gotoEntryCount(tables.ParseTable.Gotos),
			GrammarRules:   len(tables.ParseTable.Rules),
			Recovery:       hasErrorProduction(tables.ParseTable.Rules) || hasErrorAction(tables.ParseTable.Actions),
			Path:           dir.Path,
		})
	}
	return rows, nil
}

func readTables(path string) (tablesFile, error) {
	var tables tablesFile
	data, err := os.ReadFile(path)
	if err != nil {
		return tables, err
	}
	if err := json.Unmarshal(data, &tables); err != nil {
		return tables, err
	}
	return tables, nil
}

func generatedBytes(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func nestedEntryCount(entries map[string]map[string]json.RawMessage) int {
	count := 0
	for _, row := range entries {
		count += len(row)
	}
	return count
}

func gotoEntryCount(entries map[string]map[string]int) int {
	count := 0
	for _, row := range entries {
		count += len(row)
	}
	return count
}

func hasErrorProduction(rules []struct {
	RHS []string `json:"rhs"`
}) bool {
	for _, rule := range rules {
		for _, symbol := range rule.RHS {
			if symbol == "error" {
				return true
			}
		}
	}
	return false
}

func hasErrorAction(actions map[string]map[string]json.RawMessage) bool {
	for _, row := range actions {
		if _, ok := row["error"]; ok {
			return true
		}
	}
	return false
}

func writeJSON(path string, rows []artifactRow) error {
	data, err := json.MarshalIndent(struct {
		GeneratedAt string        `json:"generatedAt"`
		Artifacts   []artifactRow `json:"artifacts"`
	}{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Artifacts:   rows,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeMarkdown(path string, rows []artifactRow) error {
	var b strings.Builder
	b.WriteString("# Generated Artifact Metrics\n\n")
	b.WriteString("Generated on: ")
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString("\n\n")
	b.WriteString("| Example | Target | Generated bytes | Lexer states | Parser states | Parser actions | Parser gotos | Grammar rules | Recovery |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---|\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %d | %d | %t |\n",
			row.Example,
			row.Target,
			row.GeneratedBytes,
			row.LexerStates,
			row.ParserStates,
			row.ParserActions,
			row.ParserGotos,
			row.GrammarRules,
			row.Recovery)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
