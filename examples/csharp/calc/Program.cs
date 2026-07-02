using System.Globalization;
using LangForge.Examples.Calc.Generated;
using static LangForge.Examples.Calc.Generated.SemanticReducerContexts;

// The handwritten part of the calculator example is deliberately tiny:
// generated scanner/parser code handles syntax, while typed reducer handlers
// give each grammar action label its semantic meaning.
static double Eval(string source)
{
    return (double)Parser.ParseWithReducerFromSource(new Scanner(source), CreateReducers())!;
}

static ReducerMap CreateReducers()
{
    return new ReducerMap
    {
        [SemanticAction.Start] = TypedStart(Start),
        [SemanticAction.Pass] = TypedPass(Pass),
        [SemanticAction.Group] = TypedGroup(Group),
        [SemanticAction.Number] = TypedNumber(Number),
        [SemanticAction.Negate] = TypedNegate(Negate),
        [SemanticAction.Add] = TypedAdd(Add),
        [SemanticAction.Subtract] = TypedSubtract(Subtract),
        [SemanticAction.Multiply] = TypedMultiply(Multiply),
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

static void RunAssertions()
{
    // These checks exercise normal parsing, concurrent parser reuse, scanner
    // synchronization, and the two most common failure paths.
    Check(Math.Abs(Eval("1 + 2 * (3 - 4.5)") - -2) < 0.0001, "wrong arithmetic result");
    Check(Math.Abs(Eval("7.5/2.5") - 3) < 0.0001, "wrong decimal division result");
    Parser.ParseFromSource(new Scanner("1+2"));
    Parser.Parse(Scanner.Tokenize("1+2"));

    var parser = new Parser();
    Parallel.For(0, 16, _ => parser.ParseSource(new Scanner("1 + 2 * (3 - 4.5)")));

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
        Eval("1/0");
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
        Parser.ParseFromSource(new Scanner("1+"));
        throw new InvalidOperationException("expected parse error");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("parse error"))
    {
    }
}

var argsList = args.ToList();
var assert = argsList.Remove("--assert");
var inputPath = argsList.Count > 0 ? argsList[0] : "input.calc";
if (assert)
{
    RunAssertions();
}

var source = File.ReadAllText(inputPath);
var result = Eval(source);
var report = $"source: {source.Trim()}{Environment.NewLine}result: {result.ToString(CultureInfo.InvariantCulture)}{Environment.NewLine}";
Console.Write(report);
Directory.CreateDirectory("dist");
File.WriteAllText(Path.Combine("dist", "calc-csharp-demo.log"), report);
