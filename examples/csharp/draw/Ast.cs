namespace LangForge.Examples.Draw;

/// <summary>Root node for a DRAW script.</summary>
internal sealed record DrawProgram(IReadOnlyList<Statement> Statements);

/// <summary>Base type for executable DRAW statements.</summary>
internal abstract record Statement;

/// <summary>Creates the target canvas.</summary>
internal sealed record CanvasStatement(Expr Width, Expr Height) : Statement;

/// <summary>Fills the canvas background.</summary>
internal sealed record BackgroundStatement(ColorRgb Color) : Statement;

/// <summary>Changes the active stroke color.</summary>
internal sealed record StrokeStatement(ColorRgb Color) : Statement;

/// <summary>Changes or disables the active fill style.</summary>
internal sealed record FillStatement(ColorRgb Color, bool Enabled) : Statement;

/// <summary>Changes the active line width.</summary>
internal sealed record WidthStatement(Expr Value) : Statement;

/// <summary>Assigns a numeric expression to a variable.</summary>
internal sealed record AssignStatement(string Name, Expr Value) : Statement;

/// <summary>Stores a reusable figure block.</summary>
internal sealed record DefineFigureStatement(string Name, FigureBlock Figure) : Statement;

/// <summary>Draws one named or inline figure.</summary>
internal sealed record DrawStatement(FigureRef Target) : Statement;

/// <summary>Draws a figure repeatedly.</summary>
internal sealed record RepDrawStatement(Expr Count, FigureRef Target) : Statement;

/// <summary>Draws one primitive shape.</summary>
internal sealed record PrimitiveStatement(string Kind, IReadOnlyList<Expr> Args) : Statement;

/// <summary>A reusable list of figure-local statements.</summary>
internal sealed record FigureBlock(IReadOnlyList<Statement> Statements);

/// <summary>Base type for figure references.</summary>
internal abstract record FigureRef;

/// <summary>Reference to a previously defined figure.</summary>
internal sealed record NamedFigureRef(string Name) : FigureRef;

/// <summary>Inline figure block reference.</summary>
internal sealed record InlineFigureRef(FigureBlock Figure) : FigureRef;

/// <summary>Base type for numeric expressions.</summary>
internal abstract record Expr;

/// <summary>Numeric literal expression.</summary>
internal sealed record NumberExpr(double Value) : Expr;

/// <summary>Variable lookup expression.</summary>
internal sealed record VariableExpr(string Name) : Expr;

/// <summary>Unary numeric expression.</summary>
internal sealed record UnaryExpr(string Op, Expr Value) : Expr;

/// <summary>Binary numeric expression.</summary>
internal sealed record BinaryExpr(string Op, Expr Left, Expr Right) : Expr;

/// <summary>Built-in single-argument function call.</summary>
internal sealed record CallExpr(string Name, Expr Arg) : Expr;

/// <summary>One deferred right-hand operation used while folding an expression.</summary>
internal sealed record BinaryTail(string Op, Expr Right);

/// <summary>RGB color used by the renderer and PNG writer.</summary>
internal readonly record struct ColorRgb(byte R, byte G, byte B)
{
    /// <summary>Black is used when fill is disabled.</summary>
    public static ColorRgb Black => new(0, 0, 0);

    /// <summary>Returns the color as a CSS-style hex string.</summary>
    public override string ToString() => $"#{R:X2}{G:X2}{B:X2}";
}
