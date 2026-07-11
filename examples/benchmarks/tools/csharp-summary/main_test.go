package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMarkdownTable(t *testing.T) {
	rows := parseMarkdownTable("CalcParseBenchmarks", `
| Method | Mean | Error | StdDev | Allocated |
|--- |---:|---:|---:|---:|
| ParseFromStringScanner_TypedReducer | 1.500 ms | 0.100 ms | 0.200 ms | 128 B |
`)
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].Class != "CalcParseBenchmarks" || rows[0].Method != "ParseFromStringScanner_TypedReducer" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
	if rows[0].Mean != "1.500 ms" || rows[0].Allocated != "128 B" {
		t.Fatalf("unexpected metric fields: %+v", rows[0])
	}
}

func TestRenderSummaryUsesRelativePathsAndThroughput(t *testing.T) {
	repo := t.TempDir()
	artifacts := filepath.Join(repo, "dist", "benchmarks", "csharp")
	file := filepath.Join(artifacts, "results", "CalcParseBenchmarks-report-github.md")
	tokens := 1500
	md := renderSummary(repo, artifacts, "short", "*CalcParse*", []string{file}, []markdownRow{{
		Class:     "CalcParseBenchmarks",
		Method:    "ParseFromStringScanner_TypedReducer",
		Mean:      "2.000 ms",
		Error:     "0.100 ms",
		StdDev:    "0.200 ms",
		Allocated: "64 B",
	}}, map[string]workload{
		"CalcParseBenchmarks.ParseFromStringScanner_TypedReducer": {
			Class:  "CalcParseBenchmarks",
			Method: "ParseFromStringScanner_TypedReducer",
			Bytes:  1000,
			Tokens: &tokens,
			Lines:  10,
		},
	})
	for _, want := range []string{
		"- Artifacts: `dist/benchmarks/csharp`",
		"- `dist/benchmarks/csharp/results/CalcParseBenchmarks-report-github.md`",
		"| `CalcParseBenchmarks.ParseFromStringScanner_TypedReducer` | 2.000 ms | 0.100 ms | 0.200 ms | 0.50 | 750000 | 5000 | 64 B |",
		"quick (BenchmarkDotNet ShortRun)",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("summary missing %q:\n%s", want, md)
		}
	}
}

func TestBuildSummary(t *testing.T) {
	repo := t.TempDir()
	artifacts := filepath.Join(repo, "dist", "benchmarks", "csharp")
	results := filepath.Join(artifacts, "results")
	if err := os.MkdirAll(results, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifacts, "langforge-workloads.json"), append([]byte{0xef, 0xbb, 0xbf}, []byte(`{
  "workloads": [
    {"class":"ScannerBenchmarks","method":"StringScannerNext","bytes":2048,"tokens":512,"lines":20,"note":"fixture"}
  ]
}`)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(results, "LangForge.Examples.Benchmarks.CSharp.ScannerBenchmarks-report-github.md"), []byte(`
| Method | Mean | Error | StdDev | Allocated |
|--- |---:|---:|---:|---:|
| StringScannerNext | 1.000 ms | 0.010 ms | 0.020 ms | 64 B |
`), 0o644); err != nil {
		t.Fatal(err)
	}
	md, err := buildSummary(repo, artifacts, "medium", "*Scanner*")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "stable (BenchmarkDotNet MediumRun)") {
		t.Fatalf("medium mode not described:\n%s", md)
	}
	if !strings.Contains(md, "| `ScannerBenchmarks.StringScannerNext` | 1.000 ms | 0.010 ms | 0.020 ms | 2.05 | 512000 | 20000 | 64 B |") {
		t.Fatalf("summary did not include throughput row:\n%s", md)
	}
}
