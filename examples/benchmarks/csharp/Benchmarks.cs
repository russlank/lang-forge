using BenchmarkDotNet.Attributes;
using BenchmarkDotNet.Jobs;

namespace LangForge.Examples.Benchmarks.CSharp;

/// <summary>Benchmarks generated scanner throughput over the calc workload.</summary>
[MemoryDiagnoser]
[ShortRunJob]
public class ScannerBenchmarks
{
    [Benchmark]
    public int StreamingNext() => BenchmarkWorkloads.ScanStreamingNext();

    [Benchmark]
    public int MaterializeAll() => BenchmarkWorkloads.ScanMaterializeAll();
}

/// <summary>Benchmarks source parsing versus pre-tokenized parsing and typed versus boxed reducers.</summary>
[MemoryDiagnoser]
[ShortRunJob]
public class CalcParseBenchmarks
{
    [Benchmark]
    public double ParseFromSource_TypedReducer() => BenchmarkWorkloads.CalcParseFromSource(BenchmarkWorkloads.CalcTypedReducer);

    [Benchmark]
    public double ParsePreTokenized_TypedReducer() => BenchmarkWorkloads.CalcParsePreTokenized(BenchmarkWorkloads.CalcTypedReducer);

    [Benchmark]
    public double ParseFromSource_BoxedReducer() => BenchmarkWorkloads.CalcParseFromSource(BenchmarkWorkloads.CalcBoxedReducer);

    [Benchmark]
    public double ParsePreTokenized_BoxedReducer() => BenchmarkWorkloads.CalcParsePreTokenized(BenchmarkWorkloads.CalcBoxedReducer);
}

/// <summary>Benchmarks DRAW parsing through the handwritten AST-building facade.</summary>
[MemoryDiagnoser]
[ShortRunJob]
public class DrawParseBenchmarks
{
    [Benchmark]
    public int ParseFromSource_BuildAst() => BenchmarkWorkloads.DrawParseFromSource();
}

/// <summary>Benchmarks recovering parser runs with source and pre-tokenized inputs.</summary>
[MemoryDiagnoser]
[ShortRunJob]
public class RecoveryParseBenchmarks
{
    [Benchmark]
    public int ParseFromSource() => BenchmarkWorkloads.RecoveryParseFromSource();

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
