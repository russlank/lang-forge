using System.Globalization;
using System.Text;
using LangForge.Examples.Templates.MiniCompiler.Generated;
using static LangForge.Examples.Templates.MiniCompiler.Generated.SemanticReducerContexts;

var reducers = CreateReducers();

static ProgramNode ParseProgram(string source, ReducerMap reducers)
{
    var value = Parser.ParseWithReducer(new Scanner(source), reducers);
    if (value is ProgramNode program)
    {
        return program;
    }
    throw new InvalidOperationException($"parser final value has type {value?.GetType().Name ?? "<null>"}, want ProgramNode");
}

static ReducerMap CreateReducers()
{
    // Each entry connects a `{csharp: ...}` grammar action to a typed handler.
    // The generated adapter validates labels such as `left=Expr` before the
    // handwritten semantic code runs.
    return new ReducerMap
    {
        [SemanticAction.Program] = TypedProgram(ReduceProgram),
        [SemanticAction.Statements] = TypedStatements(ReduceStatements),
        [SemanticAction.StatementsTailMore] = TypedStatementsTailMore(ReduceStatementsTailMore),
        [SemanticAction.StatementsTailEmpty] = TypedStatementsTailEmpty(_ => new List<StatementNode>()),
        [SemanticAction.Print] = TypedPrint(ctx => new StatementNode(ctx.Expr)),
        [SemanticAction.Add] = TypedAdd(ctx => new AddExpr(ctx.Left, ctx.Right)),
        [SemanticAction.Pass] = TypedPass(ctx => ctx.Value),
        [SemanticAction.Number] = TypedNumber(ctx => new NumberExpr(ParseNumber(ctx))),
    };
}

static ProgramNode ReduceProgram(ProgramReduction ctx) => new(ctx.Statements);

static List<StatementNode> ReduceStatements(StatementsReduction ctx) => Prepend(ctx.Head, ctx.Tail);

static List<StatementNode> ReduceStatementsTailMore(StatementsTailMoreReduction ctx) => Prepend(ctx.Head, ctx.Tail);

static int ParseNumber(NumberReduction ctx)
{
    try
    {
        return int.Parse(ctx.Token.Text, CultureInfo.InvariantCulture);
    }
    catch (Exception ex) when (ex is FormatException or OverflowException)
    {
        throw new InvalidOperationException(
            $"rule {ctx.Reduction.Rule} action {ctx.Reduction.Action} label token value {ctx.Token.Text} is not a valid Int32",
            ex);
    }
}

static List<Instruction> Compile(ProgramNode program)
{
    var code = new List<Instruction>();
    foreach (var statement in program.Statements)
    {
        CompileExpr(statement.Expr, code);
        code.Add(new Instruction("print"));
    }
    return code;
}

static void CompileExpr(Expr expr, List<Instruction> code)
{
    switch (expr)
    {
        case NumberExpr number:
            code.Add(new Instruction("push", number.Value));
            break;
        case AddExpr add:
            CompileExpr(add.Left, code);
            CompileExpr(add.Right, code);
            code.Add(new Instruction("add"));
            break;
        default:
            throw new InvalidOperationException($"unsupported expression {expr.GetType().Name}");
    }
}

static List<int> Run(IReadOnlyList<Instruction> code)
{
    var stack = new Stack<int>();
    var output = new List<int>();
    for (var pc = 0; pc < code.Count; pc++)
    {
        var instruction = code[pc];
        switch (instruction.Op)
        {
            case "push":
                stack.Push(instruction.Arg);
                break;
            case "add":
                if (stack.Count < 2)
                {
                    throw new InvalidOperationException($"pc {pc}: add needs two stack values");
                }
                stack.Push(stack.Pop() + stack.Pop());
                break;
            case "print":
                if (stack.Count < 1)
                {
                    throw new InvalidOperationException($"pc {pc}: print needs one stack value");
                }
                output.Add(stack.Pop());
                break;
            default:
                throw new InvalidOperationException($"pc {pc}: unknown instruction {instruction.Op}");
        }
    }
    return output;
}

static string BuildReport(string inputPath, string source, IReadOnlyList<Instruction> code, IReadOnlyList<int> output)
{
    var report = new StringBuilder();
    report.AppendLine($"Mini compiler C# template: {inputPath}");
    report.AppendLine("source:");
    foreach (var line in source.Trim().Split('\n'))
    {
        report.AppendLine($"  {line}");
    }
    report.AppendLine("stack code:");
    for (var i = 0; i < code.Count; i++)
    {
        var instruction = code[i];
        report.AppendLine(instruction.Op == "push" ? $"  {i:00} push {instruction.Arg}" : $"  {i:00} {instruction.Op}");
    }
    report.AppendLine($"output: [{string.Join(", ", output)}]");
    return report.ToString();
}

static List<T> Prepend<T>(T head, List<T> tail)
{
    var result = new List<T> { head };
    result.AddRange(tail);
    return result;
}

static void RunAssertions(string source, ReducerMap reducers)
{
    var output = Run(Compile(ParseProgram(source, reducers)));
    if (output.Count != 2 || output[0] != 3 || output[1] != 42)
    {
        throw new InvalidOperationException($"unexpected output: [{string.Join(", ", output)}]");
    }
    try
    {
        ParseProgram("print 1 +;", reducers);
        throw new InvalidOperationException("expected parser failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("parse error", StringComparison.Ordinal))
    {
    }
    try
    {
        ParseProgram("print 999999999999999999999999;", reducers);
        throw new InvalidOperationException("expected reducer failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("action number", StringComparison.Ordinal) && ex.Message.Contains("label token", StringComparison.Ordinal))
    {
    }
}

var argsList = args.ToList();
var assert = argsList.Remove("--assert");
var logPath = "dist/mini-csharp.log";
var logIndex = argsList.IndexOf("--log");
if (logIndex >= 0 && logIndex + 1 < argsList.Count)
{
    logPath = argsList[logIndex + 1];
    argsList.RemoveAt(logIndex + 1);
    argsList.RemoveAt(logIndex);
}
var inputPath = argsList.Count > 0 ? argsList[0] : "input.mini";
var source = File.ReadAllText(inputPath);
if (assert)
{
    RunAssertions(source, reducers);
}
var program = ParseProgram(source, reducers);
var code = Compile(program);
var output = Run(code);
var report = BuildReport(inputPath, source, code, output);
Console.Write(report);
Directory.CreateDirectory(Path.GetDirectoryName(logPath) ?? ".");
File.WriteAllText(logPath, report);

/// <summary>Root AST node for a mini-compiler source file.</summary>
sealed record ProgramNode(List<StatementNode> Statements);

/// <summary>One print statement in the mini language.</summary>
sealed record StatementNode(Expr Expr);

/// <summary>Base type for expression AST nodes.</summary>
abstract record Expr;

/// <summary>Integer literal expression.</summary>
sealed record NumberExpr(int Value) : Expr;

/// <summary>Binary addition expression.</summary>
sealed record AddExpr(Expr Left, Expr Right) : Expr;

/// <summary>One mock stack-machine instruction.</summary>
sealed record Instruction(string Op, int Arg = 0);
