using LangForge.Examples.Templates.LayeredCompiler.Ast;

namespace LangForge.Examples.Templates.LayeredCompiler.Compilation;

/// <summary>Compiles the domain AST into a tiny stack-machine program.</summary>
public static class MiniCompiler
{
    /// <summary>Compiles every statement in a parsed program.</summary>
    public static IReadOnlyList<Instruction> Compile(ProgramNode program)
    {
        var code = new List<Instruction>();
        foreach (var statement in program.Statements)
        {
            CompileStatement(statement, code);
        }
        return code;
    }

    private static void CompileStatement(StatementNode statement, List<Instruction> code)
    {
        switch (statement)
        {
            case PrintStatementNode print:
                CompileExpr(print.Expression, code);
                code.Add(new Instruction(OpCode.Print));
                break;
            default:
                throw new InvalidOperationException($"unsupported statement {statement.GetType().Name}");
        }
    }

    private static void CompileExpr(ExprNode expression, List<Instruction> code)
    {
        switch (expression)
        {
            case NumberExprNode number:
                code.Add(new Instruction(OpCode.Push, number.Value));
                break;
            case AddExprNode add:
                CompileExpr(add.Left, code);
                CompileExpr(add.Right, code);
                code.Add(new Instruction(OpCode.Add));
                break;
            default:
                throw new InvalidOperationException($"unsupported expression {expression.GetType().Name}");
        }
    }
}
