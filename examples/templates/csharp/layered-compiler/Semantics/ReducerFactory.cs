using LangForge.Examples.Templates.LayeredCompiler.Ast;
using LangForge.Examples.Templates.LayeredCompiler.Generated;
using static LangForge.Examples.Templates.LayeredCompiler.Generated.SemanticReducerContexts;

namespace LangForge.Examples.Templates.LayeredCompiler.Semantics;

/// <summary>Creates generated reducer maps from handwritten domain semantics.</summary>
internal static class ReducerFactory
{
    /// <summary>Builds a reducer map with complete generated action coverage.</summary>
    public static ReducerMap Create(INumberLiteralPolicy numberPolicy) => new()
    {
        // Program : statements=Statements {csharp: program}
        [SemanticAction.Program] = TypedProgram(ctx => new ProgramNode(ctx.Statements)),

        // Statements : head=Statement tail=StatementsTail {csharp: statements}
        [SemanticAction.Statements] = TypedStatements(ctx => Prepend(ctx.Head, ctx.Tail)),

        // StatementsTail : head=Statement tail=StatementsTail {csharp: statements.tail.more}
        [SemanticAction.StatementsTailMore] = TypedStatementsTailMore(ctx => Prepend(ctx.Head, ctx.Tail)),

        // StatementsTail : %empty {csharp: statements.tail.empty}
        [SemanticAction.StatementsTailEmpty] = TypedStatementsTailEmpty(_ => new List<StatementNode>()),

        // Statement : Print expr=Expr Semi {csharp: print}
        [SemanticAction.Print] = TypedPrint(ctx => new PrintStatementNode(ctx.Expr)),

        // Expr : left=Expr Plus right=Term {csharp: add}
        [SemanticAction.Add] = TypedAdd(ctx => new AddExprNode(ctx.Left, ctx.Right)),

        // Expr : value=Term {csharp: pass}
        [SemanticAction.Pass] = TypedPass(ctx => ctx.Value),

        // Term : token=Number {csharp: number}
        [SemanticAction.Number] = TypedNumber(ctx => new NumberExprNode(ParseNumber(ctx, numberPolicy))),
    };

    private static int ParseNumber(NumberReduction ctx, INumberLiteralPolicy numberPolicy)
    {
        var context = $"rule {ctx.Reduction.Rule} action {ctx.Reduction.Action} label token";
        return numberPolicy.Parse(ctx.Token.Text, context);
    }

    private static List<T> Prepend<T>(T head, List<T> tail)
    {
        var result = new List<T> { head };
        result.AddRange(tail);
        return result;
    }
}
