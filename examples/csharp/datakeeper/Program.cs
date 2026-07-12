using System.Text;
using LangForge.Examples.DataKeeper.Generated;
using static LangForge.Examples.DataKeeper.Generated.SemanticReducerContexts;

// This example keeps the generated scanner/parser in Generated/ and expresses
// DataKeeper-specific lowering in ordinary typed C# reducer code.
var reducers = CreateReducers();

static Script ParseScript(string source, ReducerMap reducers)
{
    var value = Parser.ParseWithReducer(new Scanner(source), reducers);
    return value is Script script
        ? script
        : throw new InvalidOperationException($"parser returned {value?.GetType().Name ?? "<null>"} instead of Script");
}

static ReducerMap CreateReducers()
{
    // SemanticAction values are generated from {csharp: ...} labels in
    // datakeeper.lf. Each adapter converts the boxed parser stack into a typed
    // context whose property names match the RHS labels in the grammar.
    return new ReducerMap
    {
        [SemanticAction.ProgramWithParameters] = TypedProgramWithParameters(ProgramWithParameters),
        [SemanticAction.ProgramNoParameters] = TypedProgramNoParameters(ProgramNoParameters),
        [SemanticAction.ParametersList] = TypedParametersList(ParametersList),
        [SemanticAction.ParametersDecl] = TypedParametersDecl(ParametersDecl),
        [SemanticAction.ParametersTailMore] = TypedParametersTailMore(ParametersTailMore),
        [SemanticAction.ParametersTailEmpty] = TypedParametersTailEmpty(ParametersTailEmpty),
        [SemanticAction.CommandBlock] = TypedCommandBlock(CommandBlock),
        [SemanticAction.Statements] = TypedStatements(Statements),
        [SemanticAction.StatementsTailMore] = TypedStatementsTailMore(StatementsTailMore),
        [SemanticAction.StatementsTailEmpty] = TypedStatementsTailEmpty(StatementsTailEmpty),
        [SemanticAction.StatementPass] = TypedStatementPass(StatementPass),
        [SemanticAction.Assign] = TypedAssign(Assign),
        [SemanticAction.Replace] = TypedReplace(Replace),
        [SemanticAction.Sqlrun] = TypedSqlrun(Sqlrun),
        [SemanticAction.AddObject] = TypedAddObject(AddObject),
        [SemanticAction.RemoveObject] = TypedRemoveObject(RemoveObject),
        [SemanticAction.RunObjectsJob] = TypedRunObjectsJob(RunObjectsJob),
        [SemanticAction.ValueString] = TypedValueString(ValueString),
        [SemanticAction.ValueNumber] = TypedValueNumber(ValueNumber),
        [SemanticAction.ValueIdent] = TypedValueIdent(ValueIdent),
    };
}

static Script ProgramWithParameters(ProgramWithParametersReduction ctx) => new(ctx.Parameters, ctx.Block);

static Script ProgramNoParameters(ProgramNoParametersReduction ctx) => new([], ctx.Block);

static List<string> ParametersList(ParametersListReduction ctx) => ctx.Params;

static List<string> ParametersDecl(ParametersDeclReduction ctx) => Prepend(ctx.Name.Text, ctx.Tail);

static List<string> ParametersTailMore(ParametersTailMoreReduction ctx) => Prepend(ctx.Name.Text, ctx.Tail);

static List<string> ParametersTailEmpty(ParametersTailEmptyReduction ctx) => [];

static List<Command> CommandBlock(CommandBlockReduction ctx) => ctx.Statements;

static List<Command> Statements(StatementsReduction ctx) => Prepend(ctx.Head, ctx.Tail);

static List<Command> StatementsTailMore(StatementsTailMoreReduction ctx) => Prepend(ctx.Head, ctx.Tail);

static List<Command> StatementsTailEmpty(StatementsTailEmptyReduction ctx) => [];

static Command StatementPass(StatementPassReduction ctx) => ctx.Value;

static Command Assign(AssignReduction ctx) => new("assign", [ctx.Name.Text, ctx.Value]);

static Command Replace(ReplaceReduction ctx) => new("replace", [ctx.Target.Text, ctx.Old, ctx.New]);

static Command Sqlrun(SqlrunReduction ctx) => new("sqlrun", [ctx.Instance, ctx.Script]);

static Command AddObject(AddObjectReduction ctx) => new("addobject", [ctx.Parent, ctx.Xml]);

static Command RemoveObject(RemoveObjectReduction ctx) => new("removeobject", [ctx.Parent, ctx.Name]);

static Command RunObjectsJob(RunObjectsJobReduction ctx) => new("runobjectsjob", [ctx.Parent, ctx.Name, ctx.JobsTag]);

static string ValueString(ValueStringReduction ctx) => DecodeLiteral(ctx.Token.Text);

static string ValueNumber(ValueNumberReduction ctx) => ctx.Token.Text;

static string ValueIdent(ValueIdentReduction ctx) => "$" + ctx.Token.Text;

static List<T> Prepend<T>(T head, List<T> tail)
{
    var result = new List<T> { head };
    result.AddRange(tail);
    return result;
}

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

static void RunAssertions(string source, ReducerMap reducers)
{
    var script = ParseScript(source, reducers);
    Check(script.Parameters.Count == 4, "expected four parameters");
    Check(script.Commands.Count == 8, $"expected eight commands, got {script.Commands.Count}");
    Check(script.Commands.Any(command => command.Kind == "runobjectsjob"), "expected runobjectsjob command");

    Parallel.For(0, 8, _ => Parser.ParseWithReducer(new Scanner(source), reducers));

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
        Parser.Parse(new Scanner("begin end"));
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
    RunAssertions(source, reducers);
}

var reportText = BuildReport(ParseScript(source, reducers));
Console.Write(reportText);
Directory.CreateDirectory(Path.GetDirectoryName(logPath) ?? ".");
File.WriteAllText(logPath, reportText);

/// <summary>Parsed script shape used by the mock DataKeeper compiler.</summary>
sealed record Script(List<string> Parameters, List<Command> Commands);

/// <summary>One mock stack-machine command emitted by a semantic reduction.</summary>
sealed record Command(string Kind, string[] Args);
