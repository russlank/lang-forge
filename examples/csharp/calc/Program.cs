using System.Globalization;
using System.IO;
using LangForge.Examples.Calc.Generated;
using static LangForge.Examples.Calc.Generated.SemanticReducerContexts;

// The handwritten part of the calculator example is deliberately tiny:
// generated scanner/parser code handles syntax, while typed reducer handlers
// give each grammar action label its semantic meaning.
var reducers = CreateReducers();

static double EvalString(string source, ReducerMap reducers)
{
    return EvalReader(new StringReader(source), reducers);
}

static double EvalReader(TextReader source, ReducerMap reducers, TextReaderScannerOptions? options = null)
{
    using var scanner = Scanner.FromTextReader(source, options);
    return (double)Parser.ParseWithReducerFromLexemeSource(scanner, reducers)!;
}

static double EvalStream(Stream source, ReducerMap reducers, TextReaderScannerOptions? options = null)
{
    // Scanner.FromStream owns the StreamReader it creates, but leaves the
    // caller-owned stream open. Disposing the scanner releases decoder buffers
    // after the synchronous pull parse has finished.
    using var scanner = Scanner.FromStream(source, options: options);
    return (double)Parser.ParseWithReducerFromLexemeSource(scanner, reducers)!;
}

static ReducerMap CreateReducers()
{
    // The keys are generated from action labels in calc.lf. The comments repeat
    // the grammar alternatives so readers can move between the .lf file and
    // this handwritten semantic code without memorizing parser-table details.
    return new ReducerMap
    {
        // S : value=Expr {csharp: start}
        [SemanticAction.Start] = TypedStart(Start),
        // Expr : value=Term {csharp: pass}
        // Term : value=Factor {csharp: pass}
        [SemanticAction.Pass] = TypedPass(Pass),
        // Factor : LParen value=Expr RParen {csharp: group}
        [SemanticAction.Group] = TypedGroup(Group),
        // Factor : token=Number {csharp: number}
        [SemanticAction.Number] = TypedNumber(Number),
        // Factor : Minus value=Factor {csharp: negate}
        [SemanticAction.Negate] = TypedNegate(Negate),
        // Expr : left=Expr Plus right=Term {csharp: add}
        [SemanticAction.Add] = TypedAdd(Add),
        // Expr : left=Expr Minus right=Term {csharp: subtract}
        [SemanticAction.Subtract] = TypedSubtract(Subtract),
        // Term : left=Term Mul right=Factor {csharp: multiply}
        [SemanticAction.Multiply] = TypedMultiply(Multiply),
        // Term : left=Term Div right=Factor {csharp: divide}
        [SemanticAction.Divide] = TypedDivide(Divide),
    };
}

static double Start(StartReduction ctx) => ctx.Value;

static double Pass(PassReduction ctx) => ctx.Value;

static double Group(GroupReduction ctx) => ctx.Value;

static double Number(NumberReduction ctx) => ParseNumber(ctx.Token);

static double Negate(NegateReduction ctx) => -ctx.Value;

static double Add(AddReduction ctx) => ctx.Left + ctx.Right;

static double Subtract(SubtractReduction ctx) => ctx.Left - ctx.Right;

static double Multiply(MultiplyReduction ctx) => ctx.Left * ctx.Right;

static double Divide(DivideReduction ctx)
{
    if (ctx.Right == 0.0)
    {
        throw new InvalidOperationException("division by zero");
    }
    return ctx.Left / ctx.Right;
}

static double ParseNumber(Lexeme lexeme)
{
    return double.Parse(lexeme.Text, CultureInfo.InvariantCulture);
}

static void Check(bool condition, string message)
{
    if (!condition)
    {
        throw new InvalidOperationException(message);
    }
}

static void RunAssertions(ReducerMap reducers)
{
    // These checks exercise normal parsing, concurrent parser reuse, scanner
    // synchronization, stream-backed inputs, and common failure paths.
    Check(Math.Abs(EvalString("1 + 2 * (3 - 4.5)", reducers) - -2) < 0.0001, "wrong arithmetic result");
    Check(Math.Abs(EvalReader(new StringReader("1 + 2 * (3 - 4.5)"), reducers, new TextReaderScannerOptions { ReadBufferSize = 1 }) - -2) < 0.0001, "wrong chunked reader result");
    using (var memory = new MemoryStream(System.Text.Encoding.UTF8.GetBytes("7.5/2.5")))
    {
        Check(Math.Abs(EvalStream(memory, reducers, new TextReaderScannerOptions { ReadBufferSize = 1 }) - 3) < 0.0001, "wrong stream division result");
    }
    Check(Math.Abs(EvalString("7.5/2.5", reducers) - 3) < 0.0001, "wrong decimal division result");
    using (var syntaxScanner = Scanner.FromTextReader(new StringReader("1+2"), new TextReaderScannerOptions { ReadBufferSize = 1 }))
    {
        Parser.ParseFromLexemeSource(syntaxScanner);
    }
    Parser.Parse(Scanner.Tokenize("1+2"));

    var parser = new Parser();
    Parallel.For(0, 16, _ => parser.ParseLexemeSource(new Scanner("1 + 2 * (3 - 4.5)")));

    var shared = new Scanner("1 + 2 * (3 - 4.5)");
    var count = 0;
    Parallel.For(0, 4, _ =>
    {
        while (shared.Next(out var _))
        {
            Interlocked.Increment(ref count);
        }
    });
    Check(count == Scanner.Tokenize("1 + 2 * (3 - 4.5)").Count, $"shared scanner produced {count} tokens");

    try
    {
        EvalString("1/0", reducers);
        throw new InvalidOperationException("expected division-by-zero error");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("division by zero"))
    {
    }

    try
    {
        Scanner.Tokenize("1@");
        throw new InvalidOperationException("expected unmatched-input scanner error");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("no lexical rule"))
    {
    }

    try
    {
        EvalReader(new StringReader("1@"), reducers, new TextReaderScannerOptions { ReadBufferSize = 1 });
        throw new InvalidOperationException("expected source scanner error");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("no lexical rule"))
    {
    }

    try
    {
        Parser.ParseFromLexemeSource(new Scanner("1+"));
        throw new InvalidOperationException("expected parse error");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("parse error"))
    {
    }

    try
    {
        EvalReader(new FailingTextReader("1 + ", new InvalidOperationException("reader failed")), reducers, new TextReaderScannerOptions { ReadBufferSize = 1 });
        throw new InvalidOperationException("expected reader failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("reader failed"))
    {
    }

    try
    {
        Scanner.Tokenize(new StringReader("123"), new TextReaderScannerOptions { ReadBufferSize = 1, MaxBufferedLexemeLength = 2 });
        throw new InvalidOperationException("expected buffered-lexeme limit failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("buffered lexeme exceeds"))
    {
    }
}

var argsList = args.ToList();
var assert = argsList.Remove("--assert");
var inputPath = argsList.Count > 0 ? argsList[0] : "input.calc";
if (assert)
{
    RunAssertions(reducers);
}

var source = File.ReadAllText(inputPath);
using var input = File.OpenRead(inputPath);
var result = EvalStream(input, reducers);
var report = $"source: {source.Trim()}{Environment.NewLine}result: {result.ToString(CultureInfo.InvariantCulture)}{Environment.NewLine}";
Console.Write(report);
Directory.CreateDirectory("dist");
File.WriteAllText(Path.Combine("dist", "calc-csharp-demo.log"), report);

sealed class FailingTextReader : TextReader
{
    private string _remaining;
    private readonly InvalidOperationException _failure;

    public FailingTextReader(string prefix, InvalidOperationException failure)
    {
        _remaining = prefix;
        _failure = failure;
    }

    public override int Read(char[] buffer, int index, int count)
    {
        if (_remaining.Length == 0)
        {
            throw _failure;
        }
        var copied = Math.Min(count, _remaining.Length);
        _remaining.CopyTo(0, buffer, index, copied);
        _remaining = _remaining[copied..];
        return copied;
    }
}
