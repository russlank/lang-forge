package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	input := flag.String("input", "", "raw go benchmark output")
	output := flag.String("out", "", "Markdown summary output")
	goTool := flag.String("go", "go", "Go tool used to run benchmarks")
	command := flag.String("command", "", "benchmark command shown in the report")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "go bench summary: --input and --out are required")
		os.Exit(2)
	}
	report, err := parseFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go bench summary: %v\n", err)
		os.Exit(1)
	}
	report.GoVersion = goVersion(*goTool)
	report.Command = *command
	report.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "go bench summary: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, []byte(renderMarkdown(report)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "go bench summary: %v\n", err)
		os.Exit(1)
	}
}

type report struct {
	GeneratedAt string
	GoVersion   string
	GOOS        string
	GOARCH      string
	Package     string
	CPU         string
	Command     string
	Rows        []benchRow
}

type benchRow struct {
	Name     string
	TimeNS   float64
	MBs      string
	TokensS  string
	LinesS   string
	BPerOp   string
	AllocsOp string
}

func parseFile(path string) (report, error) {
	file, err := os.Open(path)
	if err != nil {
		return report{}, err
	}
	defer file.Close()

	var r report
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "goos:"):
			r.GOOS = strings.TrimSpace(strings.TrimPrefix(line, "goos:"))
		case strings.HasPrefix(line, "goarch:"):
			r.GOARCH = strings.TrimSpace(strings.TrimPrefix(line, "goarch:"))
		case strings.HasPrefix(line, "pkg:"):
			r.Package = strings.TrimSpace(strings.TrimPrefix(line, "pkg:"))
		case strings.HasPrefix(line, "cpu:"):
			r.CPU = strings.TrimSpace(strings.TrimPrefix(line, "cpu:"))
		case strings.HasPrefix(line, "Benchmark"):
			row, ok := parseBenchmarkLine(line)
			if ok {
				r.Rows = append(r.Rows, row)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return report{}, err
	}
	return r, nil
}

var benchmarkSuffix = regexp.MustCompile(`-\d+$`)

func parseBenchmarkLine(line string) (benchRow, bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return benchRow{}, false
	}
	row := benchRow{Name: benchmarkSuffix.ReplaceAllString(fields[0], "")}
	for i := 2; i+1 < len(fields); i += 2 {
		value := fields[i]
		unit := fields[i+1]
		switch unit {
		case "ns/op":
			ns, err := strconv.ParseFloat(value, 64)
			if err == nil {
				row.TimeNS = ns
			}
		case "MB/s":
			row.MBs = value
		case "tokens/s":
			row.TokensS = value
		case "lines/s":
			row.LinesS = value
		case "B/op":
			row.BPerOp = value
		case "allocs/op":
			row.AllocsOp = value
		}
	}
	return row, true
}

func renderMarkdown(r report) string {
	var b strings.Builder
	b.WriteString("# Go Benchmark Summary\n\n")
	writeMetadata(&b, r)
	b.WriteString("## Results\n\n")
	if len(r.Rows) == 0 {
		b.WriteString("No benchmark rows were found in the raw output.\n")
		return b.String()
	}
	b.WriteString("| Benchmark | Time | MB/s | tokens/s | lines/s | B/op | allocs/op |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|\n")
	for _, row := range r.Rows {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s | %s |\n",
			row.Name,
			formatTime(row.TimeNS),
			orDash(row.MBs),
			orDash(row.TokensS),
			orDash(row.LinesS),
			orDash(row.BPerOp),
			orDash(row.AllocsOp))
	}
	b.WriteString("\n")
	b.WriteString("`ParseFromStringScanner` and `ParseFromReaderScanner` include scanner/lexeme-source work. `ParsePreTokenized` uses a lexeme collection prepared before the timed loop.\n\n")
	b.WriteString("Use `benchstat` with repeated runs before drawing conclusions; single-sample quick mode is for smoke and performance sanity checks.\n")
	return b.String()
}

func writeMetadata(b *strings.Builder, r report) {
	b.WriteString("## Environment\n\n")
	writeMetaRow(b, "Generated at", r.GeneratedAt)
	writeMetaRow(b, "Go version", r.GoVersion)
	writeMetaRow(b, "GOOS", r.GOOS)
	writeMetaRow(b, "GOARCH", r.GOARCH)
	writeMetaRow(b, "CPU", r.CPU)
	writeMetaRow(b, "Package", r.Package)
	if r.Command != "" {
		b.WriteString("\nCommand:\n\n")
		b.WriteString("```sh\n")
		b.WriteString(r.Command)
		b.WriteString("\n```\n")
	}
	b.WriteString("\n")
}

func writeMetaRow(b *strings.Builder, label, value string) {
	if value == "" {
		value = "unknown"
	}
	fmt.Fprintf(b, "- %s: %s\n", label, value)
}

func formatTime(ns float64) string {
	if ns <= 0 {
		return "-"
	}
	if ns >= 1_000_000 {
		return fmt.Sprintf("%.3f ms/op", ns/1_000_000)
	}
	return fmt.Sprintf("%.0f ns/op", ns)
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func goVersion(goTool string) string {
	cmd := exec.Command(goTool, "version")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "unknown"
	}
	return strings.TrimSpace(out.String())
}
