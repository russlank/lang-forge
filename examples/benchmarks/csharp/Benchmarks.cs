using BenchmarkDotNet.Attributes;

namespace LangForge.Examples.Benchmarks.CSharp;

/// <summary>Benchmarks generated scanner throughput over the calc workload.</summary>
[MemoryDiagnoser]
public class ScannerBenchmarks
{
    [Benchmark]
    public int StringScannerNext() => BenchmarkWorkloads.ScanStringScannerNext();

    [Benchmark]
    public int StringScannerMaterializeAll() => BenchmarkWorkloads.ScanStringScannerMaterializeAll();

    [Benchmark]
    public int TextReaderScannerNext() => BenchmarkWorkloads.ScanTextReaderScannerNext();
}

/// <summary>Benchmarks source parsing versus pre-tokenized parsing and typed versus boxed reducers.</summary>
[MemoryDiagnoser]
public class CalcParseBenchmarks
{
    [Benchmark]
    public double ParseFromStringScanner_TypedReducer() => BenchmarkWorkloads.CalcParseFromStringScanner(BenchmarkWorkloads.CalcTypedReducer);

    [Benchmark]
    public double ParsePreTokenized_TypedReducer() => BenchmarkWorkloads.CalcParsePreTokenized(BenchmarkWorkloads.CalcTypedReducer);

    [Benchmark]
    public double ParseFromStringScanner_BoxedReducer() => BenchmarkWorkloads.CalcParseFromStringScanner(BenchmarkWorkloads.CalcBoxedReducer);

    [Benchmark]
    public double ParsePreTokenized_BoxedReducer() => BenchmarkWorkloads.CalcParsePreTokenized(BenchmarkWorkloads.CalcBoxedReducer);

    [Benchmark]
    public double ParseFromTextReaderScanner_TypedReducer() => BenchmarkWorkloads.CalcParseFromTextReaderScanner(BenchmarkWorkloads.CalcTypedReducer);

    [Benchmark]
    public double ParseFromTextReaderScanner_BoxedReducer() => BenchmarkWorkloads.CalcParseFromTextReaderScanner(BenchmarkWorkloads.CalcBoxedReducer);
}

/// <summary>Benchmarks DRAW parsing through the handwritten AST-building facade.</summary>
[MemoryDiagnoser]
public class DrawParseBenchmarks
{
    [Benchmark]
    public int ParseFromStringScanner_BuildAst() => BenchmarkWorkloads.DrawParseFromStringScanner();
}

/// <summary>Benchmarks recovering parser runs with source and pre-tokenized inputs.</summary>
[MemoryDiagnoser]
public class RecoveryParseBenchmarks
{
    [Benchmark]
    public int ParseFromStringScanner() => BenchmarkWorkloads.RecoveryParseFromStringScanner();

    [Benchmark]
    public int ParsePreTokenized() => BenchmarkWorkloads.RecoveryParsePreTokenized();
}

/// <summary>
/// Placeholder for a future recognition-only benchmark that excludes semantic
/// reducer and AST/model construction work from the timed operation.
/// </summary>
public class CalcRecognitionBenchmarks
{
}
