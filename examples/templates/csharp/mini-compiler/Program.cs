using System.Globalization;
using System.Text;
using LangForge.Examples.Templates.MiniCompiler.Generated;

static ProgramNode ParseProgram(string source)
{
    var value = Parser.ParseWithReducer(Scanner.Tokenize(source), new ReducerFunc(Reduce));
    return CastArg<ProgramNode>(value, "program");
}

static object? Reduce(Reduction ctx)
{
    return ctx.ActionID switch
    {
        SemanticAction.Program => new ProgramNode(StatementList(ctx, 0, "statements")),
        SemanticAction.Statements => Prepend(StatementArg(ctx, 0, "statement"), StatementList(ctx, 1, "statement tail")),
        SemanticAction.StatementsTailMore => Prepend(StatementArg(ctx, 0, "statement"), StatementList(ctx, 1, "statement tail")),
        SemanticAction.StatementsTailEmpty => new List<StatementNode>(),
        SemanticAction.Print => new StatementNode(ExprArg(ctx, 1, "print expression")),
        SemanticAction.Add => new AddExpr(ExprArg(ctx, 0, "left operand"), ExprArg(ctx, 2, "right operand")),
        SemanticAction.Pass => ctx.Values[0],
        SemanticAction.Number => new NumberExpr(int.Parse(Text(ctx, 0, "number literal"), CultureInfo.InvariantCulture)),
        _ => ctx.Values.Count == 1 ? ctx.Values[0] : null,
    };
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

static T CastArg<T>(object? value, string name)
{
    if (value is not T typed)
    {
        throw new InvalidOperationException($"{name} has type {value?.GetType().Name ?? "<null>"}, want {typeof(T).Name}");
    }
    return typed;
}

static T ReductionArg<T>(Reduction ctx, int index, string name)
{
    if (index < 0 || index >= ctx.Values.Count)
    {
        throw new InvalidOperationException($"rule {ctx.Rule} action {ctx.ActionID} is missing {name}");
    }
    return CastArg<T>(ctx.Values[index], $"{name} at argument {index + 1}");
}

static string Text(Reduction ctx, int index, string name) => ReductionArg<Lexeme>(ctx, index, name).Text;

static Expr ExprArg(Reduction ctx, int index, string name) => ReductionArg<Expr>(ctx, index, name);

static StatementNode StatementArg(Reduction ctx, int index, string name) => ReductionArg<StatementNode>(ctx, index, name);

static List<StatementNode> StatementList(Reduction ctx, int index, string name) => ReductionArg<List<StatementNode>>(ctx, index, name);

static List<T> Prepend<T>(T head, List<T> tail)
{
    var result = new List<T> { head };
    result.AddRange(tail);
    return result;
}

static void RunAssertions(string source)
{
    var output = Run(Compile(ParseProgram(source)));
    if (output.Count != 2 || output[0] != 3 || output[1] != 42)
    {
        throw new InvalidOperationException($"unexpected output: [{string.Join(", ", output)}]");
    }
    try
    {
        ParseProgram("print 1 +;");
        throw new InvalidOperationException("expected parser failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("parse error", StringComparison.Ordinal))
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
    RunAssertions(source);
}
var program = ParseProgram(source);
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
