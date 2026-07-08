package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	repo := flag.String("repo", "../..", "repository root")
	artifacts := flag.String("artifacts", "../../dist/benchmarks/csharp", "BenchmarkDotNet artifacts directory")
	out := flag.String("out", "../../dist/benchmarks/csharp-benchmarks-summary.md", "Markdown summary output")
	job := flag.String("job", "short", "BenchmarkDotNet job mode")
	filter := flag.String("filter", "*CalcParse*", "BenchmarkDotNet filter")
	flag.Parse()

	summary, err := buildSummary(*repo, *artifacts, *job, *filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "csharp bench summary: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "csharp bench summary: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(summary), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "csharp bench summary: %v\n", err)
		os.Exit(1)
	}
}

type workloadPayload struct {
	Workloads []workload `json:"workloads"`
}

type workload struct {
	Class  string `json:"class"`
	Method string `json:"method"`
	Bytes  int    `json:"bytes"`
	Tokens *int   `json:"tokens"`
	Lines  int    `json:"lines"`
	Note   string `json:"note"`
}

type markdownRow struct {
	Class     string
	Method    string
	Mean      string
	Error     string
	StdDev    string
	Allocated string
}

func buildSummary(repo, artifacts, job, filter string) (string, error) {
	repoAbs, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	artifactsAbs, err := filepath.Abs(artifacts)
	if err != nil {
		return "", err
	}
	workloads, err := readWorkloads(filepath.Join(artifactsAbs, "langforge-workloads.json"))
	if err != nil {
		return "", err
	}
	markdownFiles, err := findMarkdownReports(artifactsAbs)
	if err != nil {
		return "", err
	}
	rows, err := readMarkdownRows(markdownFiles)
	if err != nil {
		return "", err
	}
	return renderSummary(repoAbs, artifactsAbs, job, filter, markdownFiles, rows, workloads), nil
}

func readWorkloads(path string) (map[string]workload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	var payload workloadPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	result := make(map[string]workload, len(payload.Workloads))
	for _, item := range payload.Workloads {
		result[item.Class+"."+item.Method] = item
	}
	return result, nil
}

func findMarkdownReports(artifacts string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(artifacts, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(entry.Name(), "-report-github.md") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func readMarkdownRows(files []string) ([]markdownRow, error) {
	var rows []markdownRow
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		className := classNameFromReport(file)
		parsed := parseMarkdownTable(className, string(data))
		rows = append(rows, parsed...)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Class == rows[j].Class {
			return rows[i].Method < rows[j].Method
		}
		return rows[i].Class < rows[j].Class
	})
	return rows, nil
}

func classNameFromReport(path string) string {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, "-report-github.md")
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
		return name[dot+1:]
	}
	return name
}

func parseMarkdownTable(className, content string) []markdownRow {
	lines := strings.Split(content, "\n")
	var columns []string
	var rows []markdownRow
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inTable {
			if strings.HasPrefix(trimmed, "| Method ") || strings.HasPrefix(trimmed, "| Method |") {
				columns = splitMarkdownRow(trimmed)
				inTable = true
			}
			continue
		}
		if trimmed == "" || !strings.HasPrefix(trimmed, "|") {
			break
		}
		if strings.Contains(trimmed, "---") {
			continue
		}
		values := splitMarkdownRow(trimmed)
		value := func(name string) string {
			index := columnIndex(columns, name)
			if index < 0 || index >= len(values) {
				return ""
			}
			return strings.Trim(values[index], "`")
		}
		method := value("Method")
		if method == "" {
			continue
		}
		rows = append(rows, markdownRow{
			Class:     className,
			Method:    method,
			Mean:      value("Mean"),
			Error:     value("Error"),
			StdDev:    value("StdDev"),
			Allocated: value("Allocated"),
		})
	}
	return rows
}

func splitMarkdownRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func columnIndex(columns []string, name string) int {
	for i, column := range columns {
		if strings.EqualFold(strings.TrimSpace(column), name) {
			return i
		}
	}
	return -1
}

func renderSummary(repo, artifacts, job, filter string, files []string, rows []markdownRow, workloads map[string]workload) string {
	var b strings.Builder
	b.WriteString("# C# Benchmark Summary\n\n")
	b.WriteString("## Environment\n\n")
	fmt.Fprintf(&b, "- Generated at: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "- Mode: %s\n", describeJob(job))
	fmt.Fprintf(&b, "- Filter: `%s`\n", filter)
	fmt.Fprintf(&b, "- Artifacts: `%s`\n", relativePath(repo, artifacts))
	b.WriteString("\n")
	b.WriteString("## Results\n\n")
	if len(rows) == 0 {
		b.WriteString("No BenchmarkDotNet Markdown result rows were found.\n\n")
	} else {
		b.WriteString("| Benchmark | Mean | Error | StdDev | MB/s | tokens/s | lines/s | Allocated |\n")
		b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
		for _, row := range rows {
			metric := workloads[row.Class+"."+row.Method]
			seconds, ok := parseDurationSeconds(row.Mean)
			fmt.Fprintf(&b, "| `%s.%s` | %s | %s | %s | %s | %s | %s | %s |\n",
				row.Class,
				row.Method,
				orDash(row.Mean),
				orDash(row.Error),
				orDash(row.StdDev),
				throughputMB(metric.Bytes, seconds, ok),
				throughputCount(metric.Tokens, seconds, ok),
				throughputLines(metric.Lines, seconds, ok),
				orDash(row.Allocated))
		}
		b.WriteString("\n")
	}
	b.WriteString("## BenchmarkDotNet Markdown Files\n\n")
	if len(files) == 0 {
		b.WriteString("- none found\n")
	} else {
		for _, file := range files {
			fmt.Fprintf(&b, "- `%s`\n", relativePath(repo, file))
		}
	}
	b.WriteString("\n")
	b.WriteString("BenchmarkDotNet `Error` can be large in quick/short mode because it uses few iterations. Use `CSHARP_BENCH_JOB=medium` or `default` for more stable conclusions.\n")
	return b.String()
}

func describeJob(job string) string {
	switch strings.ToLower(job) {
	case "short":
		return "quick (BenchmarkDotNet ShortRun)"
	case "medium":
		return "stable (BenchmarkDotNet MediumRun)"
	case "default":
		return "stable (BenchmarkDotNet DefaultJob)"
	default:
		return job
	}
}

func parseDurationSeconds(value string) (float64, bool) {
	parts := strings.Fields(strings.ReplaceAll(value, ",", ""))
	if len(parts) < 2 {
		return 0, false
	}
	number, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, false
	}
	switch parts[1] {
	case "ns":
		return number / 1_000_000_000, true
	case "us", "µs", "μs":
		return number / 1_000_000, true
	case "ms":
		return number / 1_000, true
	case "s":
		return number, true
	default:
		return 0, false
	}
}

func throughputMB(bytes int, seconds float64, ok bool) string {
	if !ok || bytes <= 0 || seconds <= 0 || math.IsNaN(seconds) {
		return "-"
	}
	return fmt.Sprintf("%.2f", float64(bytes)/seconds/1_000_000)
}

func throughputCount(tokens *int, seconds float64, ok bool) string {
	if !ok || tokens == nil || *tokens <= 0 || seconds <= 0 || math.IsNaN(seconds) {
		return "-"
	}
	return fmt.Sprintf("%.0f", float64(*tokens)/seconds)
}

func throughputLines(lines int, seconds float64, ok bool) string {
	if !ok || lines <= 0 || seconds <= 0 || math.IsNaN(seconds) {
		return "-"
	}
	return fmt.Sprintf("%.0f", float64(lines)/seconds)
}

func relativePath(repo, path string) string {
	rel, err := filepath.Rel(repo, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func orDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
