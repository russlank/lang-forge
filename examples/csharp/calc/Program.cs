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
        [SemanticAction.Start] = ctx => NumberArg(ctx, 0, "expression"),
        [SemanticAction.Pass] = ctx => NumberArg(ctx, 0, "value"),
        [SemanticAction.Group] = ctx => NumberArg(ctx, 1, "grouped expression"),
        [SemanticAction.Number] = ctx => ParseNumber(LexemeArg(ctx, 0, "number")),
        [SemanticAction.Negate] = ctx => -NumberArg(ctx, 1, "operand"),
        [SemanticAction.Add] = ctx => NumberArg(ctx, 0, "left operand") + NumberArg(ctx, 2, "right operand"),
        [SemanticAction.Subtract] = ctx => NumberArg(ctx, 0, "left operand") - NumberArg(ctx, 2, "right operand"),
        [SemanticAction.Multiply] = ctx => NumberArg(ctx, 0, "left operand") * NumberArg(ctx, 2, "right operand"),
        [SemanticAction.Divide] = ctx =>
        {
            var right = NumberArg(ctx, 2, "right operand");
            if (right == 0.0)
            {
                throw new InvalidOperationException("division by zero");
            }
            return NumberArg(ctx, 0, "left operand") / right;
        },
    };
    return (double)Parser.ParseWithReducer(tokens, reducers)!;
}

static double ParseNumber(Lexeme lexeme)
{
    return double.Parse(lexeme.Text, CultureInfo.InvariantCulture);
}

static double NumberArg(Reduction ctx, int index, string name)
{
    return Arg<double>(ctx, index, name);
}

static Lexeme LexemeArg(Reduction ctx, int index, string name)
{
    return Arg<Lexeme>(ctx, index, name);
}

static T Arg<T>(Reduction ctx, int index, string name)
{
    if (index < 0 || index >= ctx.Values.Count)
    {
        throw new InvalidOperationException($"rule {ctx.Rule} action {ctx.Action} is missing {name} at argument {index + 1}");
    }
    var value = ctx.Values[index];
    if (value is T typed)
    {
        return typed;
    }
    throw new InvalidOperationException($"rule {ctx.Rule} action {ctx.Action} argument {index + 1} ({name}) has type {value?.GetType().Name ?? "null"}, expected {typeof(T).Name}");
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
    Parser.Parse(Scanner.Tokenize("1+2"));

    var parser = new Parser();
    Parallel.For(0, 16, _ => parser.ParseInput(Scanner.Tokenize("1 + 2 * (3 - 4.5)")));

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
