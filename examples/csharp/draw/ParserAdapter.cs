using System.Globalization;
using LangForge.Examples.Draw.Generated;
using static LangForge.Examples.Draw.Generated.SemanticReducerContexts;

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
        var value = Parser.ParseWithReducerFromSource(new Scanner(source), CreateReducers());
        return value is DrawProgram program
            ? program
            : throw new InvalidOperationException($"parser returned {value?.GetType().Name ?? "<null>"} instead of DrawProgram");
    }

    /// <summary>
    /// Creates the semantic reducer map used by the generated parser.
    /// </summary>
    public static ReducerMap CreateReducers()
    {
        // The generated adapters validate the action ID, read named RHS labels,
        // and pass a typed context into each handwritten AST-building function.
        return new ReducerMap
        {
            [SemanticAction.Program] = TypedProgram(Program),
            [SemanticAction.Statements] = TypedStatements(Statements),
            [SemanticAction.StatementTailMore] = TypedStatementTailMore(StatementTailMore),
            [SemanticAction.StatementTailEmpty] = TypedStatementTailEmpty(StatementTailEmpty),
            [SemanticAction.Pass] = TypedPass(Pass),
            [SemanticAction.Canvas] = TypedCanvas(Canvas),
            [SemanticAction.Background] = TypedBackground(Background),
            [SemanticAction.Stroke] = TypedStroke(Stroke),
            [SemanticAction.Fill] = TypedFill(Fill),
            [SemanticAction.FillNone] = TypedFillNone(FillNone),
            [SemanticAction.Width] = TypedWidth(Width),
            [SemanticAction.Assign] = TypedAssign(Assign),
            [SemanticAction.DefineFigure] = TypedDefineFigure(DefineFigure),
            [SemanticAction.Draw] = TypedDraw(Draw),
            [SemanticAction.Repdraw] = TypedRepdraw(Repdraw),
            [SemanticAction.FigureRefNamed] = TypedFigureRefNamed(FigureRefNamed),
            [SemanticAction.FigureRefInline] = TypedFigureRefInline(FigureRefInline),
            [SemanticAction.FigureBlock] = TypedFigureBlock(FigureBlock),
            [SemanticAction.Figures] = TypedFigures(Figures),
            [SemanticAction.FigureTailMore] = TypedFigureTailMore(FigureTailMore),
            [SemanticAction.FigureTailEmpty] = TypedFigureTailEmpty(FigureTailEmpty),
            [SemanticAction.PrimitivePoint] = TypedPrimitivePoint(PrimitivePoint),
            [SemanticAction.PrimitiveLine] = TypedPrimitiveLine(PrimitiveLine),
            [SemanticAction.PrimitiveBox] = TypedPrimitiveBox(PrimitiveBox),
            [SemanticAction.PrimitiveCircle] = TypedPrimitiveCircle(PrimitiveCircle),
            [SemanticAction.Color] = TypedColor(Color),
            [SemanticAction.Expr] = TypedExpr(Expr),
            [SemanticAction.ExprTailAdd] = TypedExprTailAdd(ExprTailAdd),
            [SemanticAction.ExprTailSubtract] = TypedExprTailSubtract(ExprTailSubtract),
            [SemanticAction.ExprTailEmpty] = TypedExprTailEmpty(ExprTailEmpty),
            [SemanticAction.Term] = TypedTerm(Term),
            [SemanticAction.TermTailMultiply] = TypedTermTailMultiply(TermTailMultiply),
            [SemanticAction.TermTailDivide] = TypedTermTailDivide(TermTailDivide),
            [SemanticAction.TermTailEmpty] = TypedTermTailEmpty(TermTailEmpty),
            [SemanticAction.UnaryNegate] = TypedUnaryNegate(UnaryNegate),
            [SemanticAction.ExprPass] = TypedExprPass(ExprPass),
            [SemanticAction.Number] = TypedNumber(Number),
            [SemanticAction.Variable] = TypedVariable(Variable),
            [SemanticAction.Call] = TypedCall(Call),
            [SemanticAction.Group] = TypedGroup(Group),
        };
    }

    private static DrawProgram Program(ProgramReduction ctx) => new(ctx.Statements);

    private static List<Statement> Statements(StatementsReduction ctx) => Prepend(ctx.Head, ctx.Tail);

    private static List<Statement> StatementTailMore(StatementTailMoreReduction ctx) => Prepend(ctx.Head, ctx.Tail);

    private static List<Statement> StatementTailEmpty(StatementTailEmptyReduction ctx) => [];

    private static Statement Pass(PassReduction ctx) => ctx.Value;

    private static Statement Canvas(CanvasReduction ctx) => new CanvasStatement(ctx.Width, ctx.Height);

    private static Statement Background(BackgroundReduction ctx) => new BackgroundStatement(ctx.Color);

    private static Statement Stroke(StrokeReduction ctx) => new StrokeStatement(ctx.Color);

    private static Statement Fill(FillReduction ctx) => new FillStatement(ctx.Color, true);

    private static Statement FillNone(FillNoneReduction ctx) => new FillStatement(ColorRgb.Black, false);

    private static Statement Width(WidthReduction ctx) => new WidthStatement(ctx.Value);

    private static Statement Assign(AssignReduction ctx) => new AssignStatement(ctx.Name.Text, ctx.Value);

    private static Statement DefineFigure(DefineFigureReduction ctx) => new DefineFigureStatement(ctx.Name.Text, ctx.Figure);

    private static Statement Draw(DrawReduction ctx) => new DrawStatement(ctx.Target);

    private static Statement Repdraw(RepdrawReduction ctx) => new RepDrawStatement(ctx.Count, ctx.Target);

    private static FigureRef FigureRefNamed(FigureRefNamedReduction ctx) => new NamedFigureRef(ctx.Name.Text);

    private static FigureRef FigureRefInline(FigureRefInlineReduction ctx) => new InlineFigureRef(ctx.Figure);

    private static FigureBlock FigureBlock(FigureBlockReduction ctx) => new(ctx.Statements);

    private static List<Statement> Figures(FiguresReduction ctx) => Prepend(ctx.Head, ctx.Tail);

    private static List<Statement> FigureTailMore(FigureTailMoreReduction ctx) => Prepend(ctx.Head, ctx.Tail);

    private static List<Statement> FigureTailEmpty(FigureTailEmptyReduction ctx) => [];

    private static Statement PrimitivePoint(PrimitivePointReduction ctx) => Primitive("point", ctx.X, ctx.Y);

    private static Statement PrimitiveLine(PrimitiveLineReduction ctx) => Primitive("line", ctx.X1, ctx.Y1, ctx.X2, ctx.Y2);

    private static Statement PrimitiveBox(PrimitiveBoxReduction ctx) => Primitive("box", ctx.X1, ctx.Y1, ctx.X2, ctx.Y2);

    private static Statement PrimitiveCircle(PrimitiveCircleReduction ctx) => Primitive("circle", ctx.Cx, ctx.Cy, ctx.Radius);

    private static ColorRgb Color(ColorReduction ctx) => ParseColor(ctx.Literal.Text);

    private static Expr Expr(ExprReduction ctx) => FoldBinary(ctx.Left, ctx.Tail);

    private static List<BinaryTail> ExprTailAdd(ExprTailAddReduction ctx) => BinaryTailList("+", ctx.Right, ctx.Tail);

    private static List<BinaryTail> ExprTailSubtract(ExprTailSubtractReduction ctx) => BinaryTailList("-", ctx.Right, ctx.Tail);

    private static List<BinaryTail> ExprTailEmpty(ExprTailEmptyReduction ctx) => [];

    private static Expr Term(TermReduction ctx) => FoldBinary(ctx.Left, ctx.Tail);

    private static List<BinaryTail> TermTailMultiply(TermTailMultiplyReduction ctx) => BinaryTailList("*", ctx.Right, ctx.Tail);

    private static List<BinaryTail> TermTailDivide(TermTailDivideReduction ctx) => BinaryTailList("/", ctx.Right, ctx.Tail);

    private static List<BinaryTail> TermTailEmpty(TermTailEmptyReduction ctx) => [];

    private static Expr UnaryNegate(UnaryNegateReduction ctx) => new UnaryExpr("-", ctx.Operand);

    private static Expr ExprPass(ExprPassReduction ctx) => ctx.Value;

    private static Expr Number(NumberReduction ctx) => new NumberExpr(double.Parse(ctx.Token.Text, CultureInfo.InvariantCulture));

    private static Expr Variable(VariableReduction ctx) => new VariableExpr(ctx.Name.Text);

    private static Expr Call(CallReduction ctx) => new CallExpr(ctx.Function.Text, ctx.Argument);

    private static Expr Group(GroupReduction ctx) => ctx.Value;

    private static PrimitiveStatement Primitive(string kind, params Expr[] args)
    {
        return new PrimitiveStatement(kind, args.ToList());
    }

    private static List<BinaryTail> BinaryTailList(string op, Expr right, List<BinaryTail> tail)
    {
        return Prepend(new BinaryTail(op, right), tail);
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
}
