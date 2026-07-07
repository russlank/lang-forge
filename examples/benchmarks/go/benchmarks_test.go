//go:build langforge_generated

package gobenchmarks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	calc "github.com/russlank/lang-forge/examples/go/calc/generated"
	calcsem "github.com/russlank/lang-forge/examples/go/calc/semantics"
	draw "github.com/russlank/lang-forge/examples/go/draw"
	recovery "github.com/russlank/lang-forge/examples/go/parser-recovery/generated"
)

var (
	calcLargeSource     = makeCalcLargeSource(4096)
	calcLargeTokens     = mustCalcTokens(calcLargeSource)
	drawLargeSource     = makeDrawLargeSource(1200)
	recoveryLargeSource = makeRecoveryLargeSource(1500, 7)
	recoveryLargeTokens = mustRecoveryTokens(recoveryLargeSource)
	typedCalcReducer    = calc.ReducerFunc(calcsem.Reduce)
	boxedCalcReducer    = makeBoxedCalcReducer()
)

func TestBenchmarkFixtures(t *testing.T) {
	if _, err := calc.ParseWithReducerFromSource(calc.NewScanner(calcLargeSource), typedCalcReducer); err != nil {
		t.Fatalf("calc source parse failed: %v", err)
	}
	if _, err := calc.ParseWithReducer(calcLargeTokens, boxedCalcReducer); err != nil {
		t.Fatalf("calc token parse failed: %v", err)
	}
	if _, err := calc.ParseWithReducerFromSource(calc.NewScanner("1 / 0"), typedCalcReducer); err == nil {
		t.Fatal("calc reducer error was not propagated")
	}
	if _, err := calc.ParseWithReducerFromSource(calc.NewScanner("1 + @"), typedCalcReducer); err == nil {
		t.Fatal("calc lexical error was not propagated")
	}
	if _, err := draw.Parse(drawLargeSource); err != nil {
		t.Fatalf("draw parse failed: %v", err)
	}
	result, err := recovery.ParseRecoveringFromSource(recovery.NewScanner(recoveryLargeSource))
	if err != nil {
		t.Fatalf("recovery source parse failed: %v", err)
	}
	if !result.Accepted || len(result.Diagnostics) == 0 {
		t.Fatalf("recovery fixture accepted=%v diagnostics=%d, want accepted with diagnostics", result.Accepted, len(result.Diagnostics))
	}
}

func BenchmarkGeneratedScannerThroughput(b *testing.B) {
	b.Run("Next", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(calcLargeSource)))
		start := time.Now()
		for i := 0; i < b.N; i++ {
			scanner := calc.NewScanner(calcLargeSource)
			count := 0
			for {
				_, ok, err := scanner.Next()
				if err != nil {
					b.Fatal(err)
				}
				if !ok {
					break
				}
				count++
			}
			if count != len(calcLargeTokens) {
				b.Fatalf("token count = %d, want %d", count, len(calcLargeTokens))
			}
		}
		reportRate(b, len(calcLargeTokens), "tokens/s", start)
	})
	b.Run("All", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(calcLargeSource)))
		start := time.Now()
		for i := 0; i < b.N; i++ {
			tokens, err := calc.NewScanner(calcLargeSource).All()
			if err != nil {
				b.Fatal(err)
			}
			if len(tokens) != len(calcLargeTokens) {
				b.Fatalf("token count = %d, want %d", len(tokens), len(calcLargeTokens))
			}
		}
		reportRate(b, len(calcLargeTokens), "tokens/s", start)
	})
}

func BenchmarkCalcLargeParse(b *testing.B) {
	b.Run("SourceTypedReducer", func(b *testing.B) {
		benchmarkCalcSource(b, typedCalcReducer)
	})
	b.Run("TokenSliceTypedReducer", func(b *testing.B) {
		benchmarkCalcTokens(b, typedCalcReducer)
	})
	b.Run("SourceBoxedReducer", func(b *testing.B) {
		benchmarkCalcSource(b, boxedCalcReducer)
	})
	b.Run("TokenSliceBoxedReducer", func(b *testing.B) {
		benchmarkCalcTokens(b, boxedCalcReducer)
	})
}

func BenchmarkDrawLargeParse(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(drawLargeSource)))
	start := time.Now()
	for i := 0; i < b.N; i++ {
		program, err := draw.Parse(drawLargeSource)
		if err != nil {
			b.Fatal(err)
		}
		if len(program.Statements) == 0 {
			b.Fatal("DRAW parse produced no statements")
		}
	}
	reportRate(b, lineCount(drawLargeSource), "lines/s", start)
}

func BenchmarkRecoveryParse(b *testing.B) {
	b.Run("Source", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(recoveryLargeSource)))
		start := time.Now()
		for i := 0; i < b.N; i++ {
			result, err := recovery.ParseRecoveringFromSource(recovery.NewScanner(recoveryLargeSource))
			if err != nil {
				b.Fatal(err)
			}
			if !result.Accepted || len(result.Diagnostics) == 0 {
				b.Fatalf("accepted=%v diagnostics=%d", result.Accepted, len(result.Diagnostics))
			}
		}
		reportRate(b, len(recoveryLargeTokens), "tokens/s", start)
	})
	b.Run("TokenSlice", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(recoveryLargeSource)))
		start := time.Now()
		for i := 0; i < b.N; i++ {
			result, err := recovery.ParseRecovering(recoveryLargeTokens)
			if err != nil {
				b.Fatal(err)
			}
			if !result.Accepted || len(result.Diagnostics) == 0 {
				b.Fatalf("accepted=%v diagnostics=%d", result.Accepted, len(result.Diagnostics))
			}
		}
		reportRate(b, len(recoveryLargeTokens), "tokens/s", start)
	})
}

func BenchmarkGeneratedArtifactMetrics(b *testing.B) {
	metrics := mustArtifactMetrics()
	b.ReportMetric(float64(metrics.CalcGeneratedBytes), "calc_generated_bytes")
	b.ReportMetric(float64(metrics.CalcParserStates), "calc_parser_states")
	b.ReportMetric(float64(metrics.CalcParserActions), "calc_parser_actions")
	b.ReportMetric(float64(metrics.CalcParserGotos), "calc_parser_gotos")
	b.ReportMetric(float64(metrics.CalcLexerStates), "calc_lexer_states")
	b.ReportMetric(float64(metrics.RecoveryGeneratedBytes), "recovery_generated_bytes")
	for i := 0; i < b.N; i++ {
	}
}

func benchmarkCalcSource(b *testing.B, reducer calc.Reducer) {
	b.ReportAllocs()
	b.SetBytes(int64(len(calcLargeSource)))
	start := time.Now()
	for i := 0; i < b.N; i++ {
		value, err := calc.ParseWithReducerFromSource(calc.NewScanner(calcLargeSource), reducer)
		if err != nil {
			b.Fatal(err)
		}
		if _, ok := value.(float64); !ok {
			b.Fatalf("value type = %T, want float64", value)
		}
	}
	reportRate(b, len(calcLargeTokens), "tokens/s", start)
}

func benchmarkCalcTokens(b *testing.B, reducer calc.Reducer) {
	b.ReportAllocs()
	b.SetBytes(int64(len(calcLargeSource)))
	start := time.Now()
	for i := 0; i < b.N; i++ {
		value, err := calc.ParseWithReducer(calcLargeTokens, reducer)
		if err != nil {
			b.Fatal(err)
		}
		if _, ok := value.(float64); !ok {
			b.Fatalf("value type = %T, want float64", value)
		}
	}
	reportRate(b, len(calcLargeTokens), "tokens/s", start)
}

func makeBoxedCalcReducer() calc.ReducerMap {
	return calc.ReducerMap{
		calc.SemanticActionStart: func(ctx calc.Reduction) (calc.Value, error) {
			return boxedFloatAt(ctx, 0, "value")
		},
		calc.SemanticActionPass: func(ctx calc.Reduction) (calc.Value, error) {
			return boxedFloatAt(ctx, 0, "value")
		},
		calc.SemanticActionGroup: func(ctx calc.Reduction) (calc.Value, error) {
			return boxedFloatAt(ctx, 1, "value")
		},
		calc.SemanticActionNumber: boxedNumber,
		calc.SemanticActionNegate: func(ctx calc.Reduction) (calc.Value, error) {
			value, err := boxedFloatAt(ctx, 1, "value")
			if err != nil {
				return nil, err
			}
			return -value, nil
		},
		calc.SemanticActionAdd: func(ctx calc.Reduction) (calc.Value, error) {
			left, right, err := boxedOperands(ctx)
			if err != nil {
				return nil, err
			}
			return left + right, nil
		},
		calc.SemanticActionSubtract: func(ctx calc.Reduction) (calc.Value, error) {
			left, right, err := boxedOperands(ctx)
			if err != nil {
				return nil, err
			}
			return left - right, nil
		},
		calc.SemanticActionMultiply: func(ctx calc.Reduction) (calc.Value, error) {
			left, right, err := boxedOperands(ctx)
			if err != nil {
				return nil, err
			}
			return left * right, nil
		},
		calc.SemanticActionDivide: boxedDivide,
	}
}

func boxedNumber(ctx calc.Reduction) (calc.Value, error) {
	lexeme, err := boxedAt[calc.Lexeme](ctx, 0, "token")
	if err != nil {
		return nil, err
	}
	return strconv.ParseFloat(lexeme.Text, 64)
}

func boxedDivide(ctx calc.Reduction) (calc.Value, error) {
	left, right, err := boxedOperands(ctx)
	if err != nil {
		return nil, err
	}
	if right == 0 {
		return nil, fmt.Errorf("division by zero")
	}
	return left / right, nil
}

func boxedOperands(ctx calc.Reduction) (float64, float64, error) {
	left, err := boxedFloatAt(ctx, 0, "left")
	if err != nil {
		return 0, 0, err
	}
	right, err := boxedFloatAt(ctx, 2, "right")
	if err != nil {
		return 0, 0, err
	}
	return left, right, nil
}

func boxedFloatAt(ctx calc.Reduction, index int, label string) (float64, error) {
	return boxedAt[float64](ctx, index, label)
}

func boxedAt[T any](ctx calc.Reduction, index int, label string) (T, error) {
	var zero T
	if index < 0 || index >= len(ctx.Values) {
		return zero, fmt.Errorf("action %q field %q index %d: value missing", ctx.Action, label, index)
	}
	value, ok := ctx.Values[index].(T)
	if !ok {
		return zero, fmt.Errorf("action %q field %q index %d: expected %T, got %T", ctx.Action, label, index, zero, ctx.Values[index])
	}
	return value, nil
}

func mustCalcTokens(source string) []calc.Lexeme {
	tokens, err := calc.Tokenize(source)
	if err != nil {
		panic(err)
	}
	return tokens
}

func mustRecoveryTokens(source string) []recovery.Lexeme {
	tokens, err := recovery.Tokenize(source)
	if err != nil {
		panic(err)
	}
	return tokens
}

func makeCalcLargeSource(terms int) string {
	var builder strings.Builder
	builder.Grow(terms * 14)
	builder.WriteString("1")
	for i := 1; i <= terms; i++ {
		left := (i % 97) + 1
		right := (i % 13) + 1
		switch i % 6 {
		case 0:
			fmt.Fprintf(&builder, " + (%d * %d)", left, right)
		case 1:
			fmt.Fprintf(&builder, " - (%d / %d)", left+10, right)
		case 2:
			fmt.Fprintf(&builder, " + -%d", left)
		case 3:
			fmt.Fprintf(&builder, " + (%d)", left)
		case 4:
			fmt.Fprintf(&builder, " + %d", left)
		default:
			fmt.Fprintf(&builder, " - %d", left)
		}
	}
	return builder.String()
}

func makeDrawLargeSource(statements int) string {
	var builder strings.Builder
	builder.Grow(statements * 28)
	builder.WriteString("canvas 640,480;\nbackground #ffffff;\nstroke #204060;\n")
	for i := 0; i < statements; i++ {
		x := i % 640
		y := (i * 3) % 480
		fmt.Fprintf(&builder, "line %d,%d,%d,%d;\n", x, y, (x+17)%640, (y+29)%480)
	}
	return builder.String()
}

func makeRecoveryLargeSource(statements, malformedEvery int) string {
	var builder strings.Builder
	builder.Grow(statements * 10)
	for i := 0; i < statements; i++ {
		if malformedEvery > 0 && i%malformedEvery == 0 {
			fmt.Fprintf(&builder, "x%d=;\n", i)
			continue
		}
		fmt.Fprintf(&builder, "x%d=%d;\n", i, i)
	}
	return builder.String()
}

func reportRate(b *testing.B, unitCount int, name string, start time.Time) {
	elapsed := time.Since(start).Seconds()
	if elapsed > 0 {
		b.ReportMetric(float64(unitCount)*float64(b.N)/elapsed, name)
	}
}

func lineCount(source string) int {
	return strings.Count(source, "\n")
}

type artifactMetrics struct {
	CalcGeneratedBytes     int64
	CalcParserStates       int
	CalcParserActions      int
	CalcParserGotos        int
	CalcLexerStates        int
	RecoveryGeneratedBytes int64
}

func mustArtifactMetrics() artifactMetrics {
	base := benchmarkDir()
	calcGenerated := filepath.Clean(filepath.Join(base, "..", "..", "go", "calc", "generated"))
	recoveryGenerated := filepath.Clean(filepath.Join(base, "..", "..", "go", "parser-recovery", "generated"))
	calcTables := readTables(filepath.Join(calcGenerated, "langforge.tables.json"))
	return artifactMetrics{
		CalcGeneratedBytes:     generatedBytes(calcGenerated),
		CalcParserStates:       len(calcTables.ParseTable.States),
		CalcParserActions:      nestedEntryCount(calcTables.ParseTable.Actions),
		CalcParserGotos:        nestedEntryCount(calcTables.ParseTable.Gotos),
		CalcLexerStates:        len(calcTables.Lexer.States),
		RecoveryGeneratedBytes: generatedBytes(recoveryGenerated),
	}
}

func benchmarkDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

type tablesFile struct {
	Lexer struct {
		States []json.RawMessage `json:"states"`
	} `json:"lexer"`
	ParseTable struct {
		States  []json.RawMessage                     `json:"states"`
		Actions map[string]map[string]json.RawMessage `json:"actions"`
		Gotos   map[string]map[string]json.RawMessage `json:"gotos"`
	} `json:"parseTable"`
}

func readTables(path string) tablesFile {
	var tables tablesFile
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, &tables); err != nil {
		panic(err)
	}
	return tables
}

func generatedBytes(dir string) int64 {
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
	if err != nil {
		panic(err)
	}
	return total
}

func nestedEntryCount(entries map[string]map[string]json.RawMessage) int {
	count := 0
	for _, row := range entries {
		count += len(row)
	}
	return count
}
