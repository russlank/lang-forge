using System.Globalization;
using LangForge.Examples.Draw.Generated;

namespace LangForge.Examples.Draw;

/// <summary>
/// Parser adapter that maps generated reduction callbacks into a DRAW AST.
/// </summary>
internal static class DrawParser
{
    /// <summary>
    /// Converts DRAW source text into an AST using the generated scanner/parser.
    /// </summary>
    public static DrawProgram Parse(string source)
    {
        var value = Parser.ParseWithReducer(Scanner.Tokenize(source), new ReducerFunc(Reduce));
        return Arg<DrawProgram>(value, "program");
    }

    /// <summary>
    /// Dispatches generated semantic action IDs to AST-building helpers.
    /// </summary>
    public static object? Reduce(Reduction ctx)
    {
        return ctx.ActionID switch
        {
            SemanticAction.Program => new DrawProgram(Arg<List<Statement>>(ctx, 0, "statement list")),
            SemanticAction.Statements => Prepend(Arg<Statement>(ctx, 0, "statement"), Arg<List<Statement>>(ctx, 1, "tail statements")),
            SemanticAction.Figures => Prepend(Arg<Statement>(ctx, 0, "figure statement"), Arg<List<Statement>>(ctx, 1, "tail figures")),
            SemanticAction.StatementTailMore => Prepend(Arg<Statement>(ctx, 1, "statement"), Arg<List<Statement>>(ctx, 2, "tail statements")),
            SemanticAction.FigureTailMore => Prepend(Arg<Statement>(ctx, 1, "figure statement"), Arg<List<Statement>>(ctx, 2, "tail figures")),
            SemanticAction.StatementTailEmpty => new List<Statement>(),
            SemanticAction.FigureTailEmpty => new List<Statement>(),
            SemanticAction.Pass => ctx.Values[0],
            SemanticAction.Canvas => new CanvasStatement(Arg<Expr>(ctx, 1, "width"), Arg<Expr>(ctx, 3, "height")),
            SemanticAction.Background => new BackgroundStatement(Arg<ColorRgb>(ctx, 1, "color")),
            SemanticAction.Stroke => new StrokeStatement(Arg<ColorRgb>(ctx, 1, "color")),
            SemanticAction.Fill => new FillStatement(Arg<ColorRgb>(ctx, 1, "color"), true),
            SemanticAction.FillNone => new FillStatement(ColorRgb.Black, false),
            SemanticAction.Width => new WidthStatement(Arg<Expr>(ctx, 1, "line width")),
            SemanticAction.Assign => new AssignStatement(Text(ctx, 0, "variable name"), Arg<Expr>(ctx, 2, "assigned value")),
            SemanticAction.DefineFigure => new DefineFigureStatement(Text(ctx, 0, "figure name"), Arg<FigureBlock>(ctx, 2, "figure block")),
            SemanticAction.Draw => new DrawStatement(Arg<FigureRef>(ctx, 1, "figure reference")),
            SemanticAction.Repdraw => new RepDrawStatement(Arg<Expr>(ctx, 1, "repeat count"), Arg<FigureRef>(ctx, 2, "figure reference")),
            SemanticAction.FigureRefNamed => new NamedFigureRef(Text(ctx, 0, "figure name")),
            SemanticAction.FigureRefInline => new InlineFigureRef(Arg<FigureBlock>(ctx, 0, "inline figure")),
            SemanticAction.FigureBlock => new FigureBlock(Arg<List<Statement>>(ctx, 1, "figure statements")),
            SemanticAction.PrimitivePoint => Primitive("point", ctx, 1, 3),
            SemanticAction.PrimitiveLine => Primitive("line", ctx, 1, 3, 5, 7),
            SemanticAction.PrimitiveBox => Primitive("box", ctx, 1, 3, 5, 7),
            SemanticAction.PrimitiveCircle => Primitive("circle", ctx, 1, 3, 5),
            SemanticAction.Color => ParseColor(Text(ctx, 0, "color literal")),
            SemanticAction.Expr => FoldBinary(Arg<Expr>(ctx, 0, "left expression"), Arg<List<BinaryTail>>(ctx, 1, "expression tail")),
            SemanticAction.Term => FoldBinary(Arg<Expr>(ctx, 0, "left term"), Arg<List<BinaryTail>>(ctx, 1, "term tail")),
            SemanticAction.ExprTailAdd => BinaryTailList("+", ctx, 1, 2),
            SemanticAction.ExprTailSubtract => BinaryTailList("-", ctx, 1, 2),
            SemanticAction.ExprTailEmpty => new List<BinaryTail>(),
            SemanticAction.TermTailMultiply => BinaryTailList("*", ctx, 1, 2),
            SemanticAction.TermTailDivide => BinaryTailList("/", ctx, 1, 2),
            SemanticAction.TermTailEmpty => new List<BinaryTail>(),
            SemanticAction.UnaryNegate => new UnaryExpr("-", Arg<Expr>(ctx, 1, "operand")),
            SemanticAction.Number => new NumberExpr(double.Parse(Text(ctx, 0, "number"), CultureInfo.InvariantCulture)),
            SemanticAction.Variable => new VariableExpr(Text(ctx, 0, "variable name")),
            SemanticAction.Call => new CallExpr(Text(ctx, 0, "function name"), Arg<Expr>(ctx, 2, "argument")),
            SemanticAction.Group => ctx.Values[1],
            _ => DefaultReduce(ctx.Values),
        };
    }

    private static PrimitiveStatement Primitive(string kind, Reduction ctx, params int[] indexes)
    {
        return new PrimitiveStatement(kind, indexes.Select(index => Arg<Expr>(ctx, index, $"{kind} argument")).ToList());
    }

    private static List<BinaryTail> BinaryTailList(string op, Reduction ctx, int exprIndex, int tailIndex)
    {
        return Prepend(new BinaryTail(op, Arg<Expr>(ctx, exprIndex, "right expression")), Arg<List<BinaryTail>>(ctx, tailIndex, "tail expressions"));
    }

    private static Expr FoldBinary(Expr left, IReadOnlyList<BinaryTail> tails)
    {
        var result = left;
        foreach (var tail in tails)
        {
            result = new BinaryExpr(tail.Op, result, tail.Right);
        }
        return result;
    }

    private static List<T> Prepend<T>(T head, List<T> tail)
    {
        var result = new List<T> { head };
        result.AddRange(tail);
        return result;
    }

    private static object? DefaultReduce(IReadOnlyList<object?> values)
    {
        return values.Count switch
        {
            0 => null,
            1 => values[0],
            _ => values.ToArray(),
        };
    }

    private static string Text(Reduction ctx, int index, string name) => Arg<Lexeme>(ctx, index, name).Text;

    private static T Arg<T>(Reduction ctx, int index, string name)
    {
        if (index < 0 || index >= ctx.Values.Count)
        {
            throw new InvalidOperationException($"rule {ctx.Rule} action {ctx.Action} is missing {name} at argument {index + 1}");
        }
        return Arg<T>(ctx.Values[index], $"rule {ctx.Rule} action {ctx.Action} argument {index + 1} ({name})");
    }

    private static T Arg<T>(object? value, string name)
    {
        if (value is T typed)
        {
            return typed;
        }
        throw new InvalidOperationException($"{name} has type {value?.GetType().Name ?? "null"}, expected {typeof(T).Name}");
    }

    private static ColorRgb ParseColor(string text)
    {
        if (text.Length != 7 || text[0] != '#')
        {
            throw new InvalidOperationException($"invalid color {text}");
        }
        return new ColorRgb(
            Convert.ToByte(text[1..3], 16),
            Convert.ToByte(text[3..5], 16),
            Convert.ToByte(text[5..7], 16));
    }

    private sealed record BinaryTail(string Op, Expr Right);
}
