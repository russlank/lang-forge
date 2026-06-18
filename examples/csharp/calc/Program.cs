using System.Globalization;
using LangForge.Examples.Calc.Generated;

// The handwritten part of the calculator example is deliberately tiny:
// generated scanner/parser code handles syntax, while this reducer map owns the
// semantic meaning of each grammar action label.
static double Eval(string source)
{
    var tokens = Scanner.Tokenize(source);
    var reducers = new ReducerMap
    {
        [SemanticAction.Start] = ctx => ctx.Values[0],
        [SemanticAction.Pass] = ctx => ctx.Values[0],
        [SemanticAction.Group] = ctx => ctx.Values[1],
        [SemanticAction.Number] = ctx => double.Parse(((Lexeme)ctx.Values[0]!).Text, CultureInfo.InvariantCulture),
        [SemanticAction.Negate] = ctx => -(double)ctx.Values[1]!,
        [SemanticAction.Add] = ctx => (double)ctx.Values[0]! + (double)ctx.Values[2]!,
        [SemanticAction.Subtract] = ctx => (double)ctx.Values[0]! - (double)ctx.Values[2]!,
        [SemanticAction.Multiply] = ctx => (double)ctx.Values[0]! * (double)ctx.Values[2]!,
        [SemanticAction.Divide] = ctx => (double)ctx.Values[0]! / (double)ctx.Values[2]!,
    };
    return (double)Parser.ParseWithReducer(tokens, reducers)!;
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
    Check(Math.Abs(Eval("1+2*(3-4)") - -1) < 0.0001, "wrong arithmetic result");
    Parser.Parse(Scanner.Tokenize("1+2"));

    var parser = new Parser();
    Parallel.For(0, 16, _ => parser.ParseInput(Scanner.Tokenize("1+2*(3-4)")));

    var shared = new Scanner("1+2*(3-4)");
    var count = 0;
    Parallel.For(0, 4, _ =>
    {
        while (shared.Next(out var _))
        {
            Interlocked.Increment(ref count);
        }
    });
    Check(count == Scanner.Tokenize("1+2*(3-4)").Count, $"shared scanner produced {count} tokens");

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
        Parser.Parse(Scanner.Tokenize("1+"));
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
