using System.Text;
using LangForge.Examples.DataKeeper.Generated;

// This example keeps the generated scanner/parser in Generated/ and expresses
// DataKeeper-specific lowering in ordinary C# reducer code.
static Script ParseScript(string source)
{
    var value = Parser.ParseWithReducer(Scanner.Tokenize(source), new ReducerFunc(Reduce));
    return (Script)value!;
}

static object? Reduce(Reduction ctx)
{
    // SemanticAction values are generated from {csharp: ...} labels in
    // datakeeper.lf. The reducer is the single place where those labels acquire
    // application behavior.
    return ctx.ActionID switch
    {
        SemanticAction.ProgramWithParameters => new Script(StringList(ctx, 0, "parameter list"), CommandList(ctx, 1, "command list")),
        SemanticAction.ProgramNoParameters => new Script([], CommandList(ctx, 0, "command list")),
        SemanticAction.ParametersList => ctx.Values[1],
        SemanticAction.ParametersDecl => Prepend(Text(ctx, 0, "parameter name"), StringList(ctx, 1, "parameter tail")),
        SemanticAction.ParametersTailMore => Prepend(Text(ctx, 1, "parameter name"), StringList(ctx, 2, "parameter tail")),
        SemanticAction.ParametersTailEmpty => new List<string>(),
        SemanticAction.CommandBlock => ctx.Values[1],
        SemanticAction.Statements => Prepend(CommandArg(ctx, 0, "statement"), CommandList(ctx, 1, "statement tail")),
        SemanticAction.StatementsTailMore => Prepend(CommandArg(ctx, 1, "statement"), CommandList(ctx, 2, "statement tail")),
        SemanticAction.StatementsTailEmpty => new List<Command>(),
        SemanticAction.StatementPass => ctx.Values[0],
        SemanticAction.Assign => new Command("assign", [Text(ctx, 0, "assignment name"), StringArg(ctx, 2, "assignment value")]),
        SemanticAction.Replace => new Command("replace", [Text(ctx, 2, "replace target"), StringArg(ctx, 4, "old value"), StringArg(ctx, 6, "new value")]),
        SemanticAction.Sqlrun => new Command("sqlrun", [StringArg(ctx, 2, "instance"), StringArg(ctx, 4, "script")]),
        SemanticAction.AddObject => new Command("addobject", [StringArg(ctx, 2, "parent"), StringArg(ctx, 4, "xml")]),
        SemanticAction.RemoveObject => new Command("removeobject", [StringArg(ctx, 2, "parent"), StringArg(ctx, 4, "name")]),
        SemanticAction.RunObjectsJob => new Command("runobjectsjob", [StringArg(ctx, 2, "parent"), StringArg(ctx, 4, "name"), StringArg(ctx, 6, "jobs tag")]),
        SemanticAction.ValueString => DecodeLiteral(Text(ctx, 0, "string literal")),
        SemanticAction.ValueNumber => Text(ctx, 0, "number literal"),
        SemanticAction.ValueIdent => "$" + Text(ctx, 0, "identifier value"),
        _ => DefaultReduce(ctx.Values),
    };
}

static List<T> Prepend<T>(T head, List<T> tail)
{
    var result = new List<T> { head };
    result.AddRange(tail);
    return result;
}

static object? DefaultReduce(IReadOnlyList<object?> values)
{
    return values.Count switch
    {
        0 => null,
        1 => values[0],
        _ => values.ToArray(),
    };
}

static T Arg<T>(Reduction ctx, int index, string name)
{
    if (index < 0 || index >= ctx.Values.Count)
    {
        throw new InvalidOperationException($"rule {ctx.Rule} action {ctx.ActionID} is missing {name} at argument {index + 1}");
    }
    if (ctx.Values[index] is not T value)
    {
        throw new InvalidOperationException($"rule {ctx.Rule} action {ctx.ActionID} argument {index + 1} for {name} has type {ctx.Values[index]?.GetType().Name ?? "<null>"}, want {typeof(T).Name}");
    }
    return value;
}

static string Text(Reduction ctx, int index, string name) => Arg<Lexeme>(ctx, index, name).Text;

static string StringArg(Reduction ctx, int index, string name) => Arg<string>(ctx, index, name);

static Command CommandArg(Reduction ctx, int index, string name) => Arg<Command>(ctx, index, name);

static List<string> StringList(Reduction ctx, int index, string name) => Arg<List<string>>(ctx, index, name);

static List<Command> CommandList(Reduction ctx, int index, string name) => Arg<List<Command>>(ctx, index, name);

static string DecodeLiteral(string text)
{
    if (text.StartsWith("#{", StringComparison.Ordinal) && text.EndsWith("#}", StringComparison.Ordinal))
    {
        return text[2..^2].Replace("##", "#", StringComparison.Ordinal);
    }
    if (text.StartsWith('"') && text.EndsWith('"'))
    {
        return text[1..^1].Replace("\\\"", "\"", StringComparison.Ordinal).Replace("\\\\", "\\", StringComparison.Ordinal);
    }
    return text;
}

static string BuildReport(Script script)
{
    var report = new StringBuilder();
    report.AppendLine("DataKeeper C# generated-parser demo");
    report.AppendLine("parameters:");
    foreach (var parameter in script.Parameters)
    {
        report.AppendLine($"  - {parameter}");
    }
    report.AppendLine("mock stack instructions:");
    for (var i = 0; i < script.Commands.Count; i++)
    {
        var command = script.Commands[i];
        report.AppendLine($"  {i + 1:00}: {command.Kind} {string.Join(" | ", command.Args)}");
    }
    return report.ToString();
}

static void Check(bool condition, string message)
{
    if (!condition)
    {
        throw new InvalidOperationException(message);
    }
}

static void RunAssertions(string source)
{
    var script = ParseScript(source);
    Check(script.Parameters.Count == 4, "expected four parameters");
    Check(script.Commands.Count == 8, $"expected eight commands, got {script.Commands.Count}");
    Check(script.Commands.Any(command => command.Kind == "runobjectsjob"), "expected runobjectsjob command");

    var parser = new Parser(new ReducerFunc(Reduce));
    Parallel.For(0, 8, _ => parser.ParseValueInput(Scanner.Tokenize(source)));

    try
    {
        Scanner.Tokenize("begin @ end");
        throw new InvalidOperationException("expected scanner failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("no lexical rule"))
    {
    }

    try
    {
        Parser.Parse(Scanner.Tokenize("begin end"));
        throw new InvalidOperationException("expected parser failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("parse error"))
    {
    }
}

var argsList = args.ToList();
var assert = argsList.Remove("--assert");
var logPath = "dist/datakeeper-csharp-demo.log";
var logIndex = argsList.IndexOf("--log");
if (logIndex >= 0 && logIndex + 1 < argsList.Count)
{
    logPath = argsList[logIndex + 1];
    argsList.RemoveAt(logIndex + 1);
    argsList.RemoveAt(logIndex);
}
var inputPath = argsList.Count > 0 ? argsList[0] : "sample.dks";
var source = File.ReadAllText(inputPath);
if (assert)
{
    RunAssertions(source);
}

var reportText = BuildReport(ParseScript(source));
Console.Write(reportText);
Directory.CreateDirectory(Path.GetDirectoryName(logPath) ?? ".");
File.WriteAllText(logPath, reportText);

/// <summary>Parsed script shape used by the mock DataKeeper compiler.</summary>
sealed record Script(List<string> Parameters, List<Command> Commands);

/// <summary>One mock stack-machine command emitted by a semantic reduction.</summary>
sealed record Command(string Kind, string[] Args);
