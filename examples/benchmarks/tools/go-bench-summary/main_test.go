package main

import (
	"strings"
	"testing"
)

func TestParseBenchmarkLine(t *testing.T) {
	row, ok := parseBenchmarkLine("BenchmarkCalcParse/ParseFromSource/TypedReducer-12         	     624	   1921843 ns/op	  16.07 MB/s	   8168420 tokens/s	     128 B/op	       2 allocs/op")
	if !ok {
		t.Fatal("benchmark line was not parsed")
	}
	if row.Name != "BenchmarkCalcParse/ParseFromSource/TypedReducer" {
		t.Fatalf("name = %q", row.Name)
	}
	if row.TimeNS != 1921843 {
		t.Fatalf("time = %v", row.TimeNS)
	}
	if row.MBs != "16.07" || row.TokensS != "8168420" || row.BPerOp != "128" || row.AllocsOp != "2" {
		t.Fatalf("unexpected row: %+v", row)
	}
}

func TestRenderMarkdown(t *testing.T) {
	md := renderMarkdown(report{
		GeneratedAt: "2026-07-09T00:00:00Z",
		GoVersion:   "go version go1.25.0 linux/amd64",
		GOOS:        "linux",
		GOARCH:      "amd64",
		Package:     "example/pkg",
		CPU:         "example cpu",
		Command:     "go test -bench .",
		Rows: []benchRow{{
			Name:     "BenchmarkScanner/StreamingNext",
			TimeNS:   1557907,
			MBs:      "19.82",
			TokensS:  "10079171",
			BPerOp:   "64",
			AllocsOp: "1",
		}},
	})
	for _, want := range []string{
		"# Go Benchmark Summary",
		"- GOOS: linux",
		"| `BenchmarkScanner/StreamingNext` | 1.558 ms/op | 19.82 | 10079171 | - | 64 | 1 |",
		"`ParseFromSource` includes scanner/token-source work.",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("summary missing %q:\n%s", want, md)
		}
	}
}
