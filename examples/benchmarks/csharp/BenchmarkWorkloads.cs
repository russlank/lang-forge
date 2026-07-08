using System.Globalization;
using System.Text;
using System.Text.Json;
using CalcLexeme = LangForge.Examples.Calc.Generated.Lexeme;
using CalcParser = LangForge.Examples.Calc.Generated.Parser;
using CalcReducer = LangForge.Examples.Calc.Generated.IReducer;
using CalcReducerMap = LangForge.Examples.Calc.Generated.ReducerMap;
using CalcReduction = LangForge.Examples.Calc.Generated.Reduction;
using CalcScanner = LangForge.Examples.Calc.Generated.Scanner;
using CalcSemanticAction = LangForge.Examples.Calc.Generated.SemanticAction;
using DrawParser = LangForge.Examples.Draw.DrawParser;
using RecoveryLexeme = LangForge.Examples.ParserRecovery.Generated.Lexeme;
using RecoveryParseResult = LangForge.Examples.ParserRecovery.Generated.ParseResult;
using RecoveryParser = LangForge.Examples.ParserRecovery.Generated.Parser;
using RecoveryScanner = LangForge.Examples.ParserRecovery.Generated.Scanner;
using static LangForge.Examples.Calc.Generated.SemanticReducerContexts;

namespace LangForge.Examples.Benchmarks.CSharp;

/// <summary>
/// Shared deterministic workloads used by the C# BenchmarkDotNet classes.
/// </summary>
internal static class BenchmarkWorkloads
{
    public static readonly string CalcLargeSource = MakeCalcLargeSource(4096);
    public static readonly IReadOnlyList<CalcLexeme> CalcLargeTokens = CalcScanner.Tokenize(CalcLargeSource);
    public static readonly CalcReducer CalcTypedReducer = CreateTypedCalcReducer();
    public static readonly CalcReducer CalcBoxedReducer = CreateBoxedCalcReducer();

    public static readonly string DrawLargeSource = MakeDrawLargeSource(1200);

    public static readonly string RecoveryLargeSource = MakeRecoveryLargeSource(1500, 7);
    public static readonly IReadOnlyList<RecoveryLexeme> RecoveryLargeTokens = RecoveryScanner.Tokenize(RecoveryLargeSource);

    public static void WriteMetrics(string path)
    {
        var payload = new
        {
            generatedAt = DateTimeOffset.UtcNow,
            workloads = new[]
            {
                WorkloadMetric.ForSource(nameof(ScannerBenchmarks), nameof(ScannerBenchmarks.StreamingNext), CalcLargeSource, CalcLargeTokens.Count, "Streaming scanner.Next over the calc fixture."),
                WorkloadMetric.ForSource(nameof(ScannerBenchmarks), nameof(ScannerBenchmarks.MaterializeAll), CalcLargeSource, CalcLargeTokens.Count, "Materializes all calc tokens."),
                WorkloadMetric.ForSource(nameof(CalcParseBenchmarks), nameof(CalcParseBenchmarks.ParseFromSource_TypedReducer), CalcLargeSource, CalcLargeTokens.Count, "Source parse includes scanner/token-source work and typed reducer handlers."),
                WorkloadMetric.ForSource(nameof(CalcParseBenchmarks), nameof(CalcParseBenchmarks.ParsePreTokenized_TypedReducer), CalcLargeSource, CalcLargeTokens.Count, "Pre-tokenized parse uses tokens prepared outside the timed operation and typed reducer handlers."),
                WorkloadMetric.ForSource(nameof(CalcParseBenchmarks), nameof(CalcParseBenchmarks.ParseFromSource_BoxedReducer), CalcLargeSource, CalcLargeTokens.Count, "Source parse includes scanner/token-source work and boxed reducer handlers."),
                WorkloadMetric.ForSource(nameof(CalcParseBenchmarks), nameof(CalcParseBenchmarks.ParsePreTokenized_BoxedReducer), CalcLargeSource, CalcLargeTokens.Count, "Pre-tokenized parse uses tokens prepared outside the timed operation and boxed reducer handlers."),
                WorkloadMetric.ForSource(nameof(DrawParseBenchmarks), nameof(DrawParseBenchmarks.ParseFromSource_BuildAst), DrawLargeSource, null, "Source parse through the DRAW handwritten AST-building facade."),
                WorkloadMetric.ForSource(nameof(RecoveryParseBenchmarks), nameof(RecoveryParseBenchmarks.ParseFromSource), RecoveryLargeSource, RecoveryLargeTokens.Count, "Recovering source parse over malformed recovery fixture."),
                WorkloadMetric.ForSource(nameof(RecoveryParseBenchmarks), nameof(RecoveryParseBenchmarks.ParsePreTokenized), RecoveryLargeSource, RecoveryLargeTokens.Count, "Recovering parse over tokens prepared outside the timed operation."),
            },
        };
        var json = JsonSerializer.Serialize(payload, new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
            WriteIndented = true,
        });
        File.WriteAllText(path, json + Environment.NewLine, new UTF8Encoding(encoderShouldEmitUTF8Identifier: false));
    }

    public static void ValidateFixtures()
    {
        _ = CalcParseFromSource(CalcTypedReducer);
        _ = CalcParsePreTokenized(CalcBoxedReducer);
        _ = DrawParseFromSource();
        var diagnostics = RecoveryParseFromSource();
        if (diagnostics == 0)
        {
            throw new InvalidOperationException("recovery benchmark fixture should produce diagnostics");
        }
    }

    public static int ScanStreamingNext()
    {
        var scanner = new CalcScanner(CalcLargeSource);
        var count = 0;
        while (scanner.Next(out _))
        {
            count++;
        }
        return count;
    }

    public static int ScanMaterializeAll()
    {
        return CalcScanner.Tokenize(CalcLargeSource).Count;
    }

    public static double CalcParseFromSource(CalcReducer reducer)
    {
        // ParseFromSource includes tokenization in the measured operation:
        // source text -> generated scanner/token source -> parser -> reducer.
        return RequireDouble(CalcParser.ParseWithReducerFromSource(new CalcScanner(CalcLargeSource), reducer));
    }

    public static double CalcParsePreTokenized(CalcReducer reducer)
    {
        // ParsePreTokenized uses tokens materialized before the benchmark, so
        // it measures parser/reducer cost over an existing token collection.
        return RequireDouble(CalcParser.ParseWithReducer(CalcLargeTokens, reducer));
    }

    public static int DrawParseFromSource()
    {
        return DrawParser.Parse(DrawLargeSource).Statements.Count;
    }

    public static int RecoveryParseFromSource()
    {
        var result = RecoveryParser.ParseRecoveringFromSource(new RecoveryScanner(RecoveryLargeSource));
        RequireRecoveryResult(result);
        return result.Diagnostics.Count;
    }

    public static int RecoveryParsePreTokenized()
    {
        var result = RecoveryParser.ParseRecovering(RecoveryLargeTokens);
        RequireRecoveryResult(result);
        return result.Diagnostics.Count;
    }

    private static CalcReducerMap CreateTypedCalcReducer()
    {
        return new CalcReducerMap
        {
            [CalcSemanticAction.Start] = TypedStart(ctx => ctx.Value),
            [CalcSemanticAction.Pass] = TypedPass(ctx => ctx.Value),
            [CalcSemanticAction.Group] = TypedGroup(ctx => ctx.Value),
            [CalcSemanticAction.Number] = TypedNumber(ctx => ParseNumber(ctx.Token)),
            [CalcSemanticAction.Negate] = TypedNegate(ctx => -ctx.Value),
            [CalcSemanticAction.Add] = TypedAdd(ctx => ctx.Left + ctx.Right),
            [CalcSemanticAction.Subtract] = TypedSubtract(ctx => ctx.Left - ctx.Right),
            [CalcSemanticAction.Multiply] = TypedMultiply(ctx => ctx.Left * ctx.Right),
            [CalcSemanticAction.Divide] = TypedDivide(ctx => Divide(ctx.Left, ctx.Right)),
        };
    }

    private static CalcReducerMap CreateBoxedCalcReducer()
    {
        return new CalcReducerMap
        {
            [CalcSemanticAction.Start] = ctx => BoxedAt<double>(ctx, 0, "value"),
            [CalcSemanticAction.Pass] = ctx => BoxedAt<double>(ctx, 0, "value"),
            [CalcSemanticAction.Group] = ctx => BoxedAt<double>(ctx, 1, "value"),
            [CalcSemanticAction.Number] = ctx => ParseNumber(BoxedAt<CalcLexeme>(ctx, 0, "token")),
            [CalcSemanticAction.Negate] = ctx => -BoxedAt<double>(ctx, 1, "value"),
            [CalcSemanticAction.Add] = ctx => BoxedAt<double>(ctx, 0, "left") + BoxedAt<double>(ctx, 2, "right"),
            [CalcSemanticAction.Subtract] = ctx => BoxedAt<double>(ctx, 0, "left") - BoxedAt<double>(ctx, 2, "right"),
            [CalcSemanticAction.Multiply] = ctx => BoxedAt<double>(ctx, 0, "left") * BoxedAt<double>(ctx, 2, "right"),
            [CalcSemanticAction.Divide] = ctx => Divide(BoxedAt<double>(ctx, 0, "left"), BoxedAt<double>(ctx, 2, "right")),
        };
    }

    private static double Divide(double left, double right)
    {
        if (right == 0.0)
        {
            throw new InvalidOperationException("division by zero");
        }
        return left / right;
    }

    private static double ParseNumber(CalcLexeme lexeme)
    {
        return double.Parse(lexeme.Text, CultureInfo.InvariantCulture);
    }

    private static double RequireDouble(object? value)
    {
        return value is double typed
            ? typed
            : throw new InvalidOperationException($"parser returned {value?.GetType().Name ?? "<null>"} instead of double");
    }

    private static T BoxedAt<T>(CalcReduction ctx, int index, string label)
    {
        if (index < 0 || index >= ctx.Values.Count)
        {
            throw new InvalidOperationException($"action {ctx.Action} field {label} index {index}: value missing");
        }
        return ctx.Values[index] is T typed
            ? typed
            : throw new InvalidOperationException($"action {ctx.Action} field {label} index {index}: expected {typeof(T).Name}, got {ctx.Values[index]?.GetType().Name ?? "<null>"}");
    }

    private static void RequireRecoveryResult(RecoveryParseResult result)
    {
        if (!result.Accepted || result.Diagnostics.Count == 0)
        {
            throw new InvalidOperationException($"recovery result accepted={result.Accepted}, diagnostics={result.Diagnostics.Count}");
        }
    }

    private static string MakeCalcLargeSource(int terms)
    {
        var builder = new StringBuilder(terms * 14);
        builder.Append('1');
        for (var i = 1; i <= terms; i++)
        {
            var left = (i % 97) + 1;
            var right = (i % 13) + 1;
            switch (i % 6)
            {
                case 0:
                    builder.Append(CultureInfo.InvariantCulture, $" + ({left} * {right})");
                    break;
                case 1:
                    builder.Append(CultureInfo.InvariantCulture, $" - ({left + 10} / {right})");
                    break;
                case 2:
                    builder.Append(CultureInfo.InvariantCulture, $" + -{left}");
                    break;
                case 3:
                    builder.Append(CultureInfo.InvariantCulture, $" + ({left})");
                    break;
                case 4:
                    builder.Append(CultureInfo.InvariantCulture, $" + {left}");
                    break;
                default:
                    builder.Append(CultureInfo.InvariantCulture, $" - {left}");
                    break;
            }
        }
        return builder.ToString();
    }

    private static string MakeDrawLargeSource(int statements)
    {
        var builder = new StringBuilder(statements * 28);
        builder.AppendLine("canvas 640,480;");
        builder.AppendLine("background #ffffff;");
        builder.AppendLine("stroke #204060;");
        for (var i = 0; i < statements; i++)
        {
            var x = i % 640;
            var y = (i * 3) % 480;
            builder.Append(CultureInfo.InvariantCulture, $"line {x},{y},{(x + 17) % 640},{(y + 29) % 480};");
            builder.AppendLine();
        }
        return builder.ToString();
    }

    private static string MakeRecoveryLargeSource(int statements, int malformedEvery)
    {
        var builder = new StringBuilder(statements * 10);
        for (var i = 0; i < statements; i++)
        {
            if (malformedEvery > 0 && i % malformedEvery == 0)
            {
                builder.Append(CultureInfo.InvariantCulture, $"x{i}=;");
            }
            else
            {
                builder.Append(CultureInfo.InvariantCulture, $"x{i}={i};");
            }
            builder.AppendLine();
        }
        return builder.ToString();
    }

    private sealed record WorkloadMetric(
        string Class,
        string Method,
        int Bytes,
        int? Tokens,
        int Lines,
        string Note)
    {
        public static WorkloadMetric ForSource(string benchmarkClass, string method, string source, int? tokens, string note)
        {
            return new WorkloadMetric(
                benchmarkClass,
                method,
                Encoding.UTF8.GetByteCount(source),
                tokens,
                source.Count(static c => c == '\n'),
                note);
        }
    }
}
