using System.Buffers.Binary;
using System.Globalization;
using System.IO.Compression;
using System.Text;
using LangForge.Examples.Draw.Generated;

namespace LangForge.Examples.Draw;

/// <summary>
/// Entry point for the C# DRAW example.
/// </summary>
internal static class Program
{
    /// <summary>
    /// Parses, renders, reports, and optionally runs edge-case assertions.
    /// </summary>
    private static void Main(string[] args)
    {
        var argsList = args.ToList();
        var assert = argsList.Remove("--assert");
        var logPath = ReadOption(argsList, "--log", "dist/draw-csharp-demo.log");
        var outputPath = ReadOption(argsList, "--output", "dist/sample-csharp.png");
        var inputPath = argsList.Count > 0 ? argsList[0] : "sample.draw";
        var source = File.ReadAllText(inputPath);

        if (assert)
        {
            RunAssertions(source, outputPath);
        }

        var result = RenderSource(source, outputPath);
        var reportText = ReportWriter.Build(inputPath, outputPath, result);
        Console.Write(reportText);
        Directory.CreateDirectory(Path.GetDirectoryName(logPath) ?? ".");
        File.WriteAllText(logPath, reportText);
    }

    private static string ReadOption(List<string> args, string name, string fallback)
    {
        var index = args.IndexOf(name);
        if (index < 0 || index + 1 >= args.Count)
        {
            return fallback;
        }
        var value = args[index + 1];
        args.RemoveAt(index + 1);
        args.RemoveAt(index);
        return value;
    }

    private static RenderResult RenderSource(string source, string outputPath)
    {
        var program = DrawParser.Parse(source);
        var result = DrawRenderer.Render(program);
        Directory.CreateDirectory(Path.GetDirectoryName(outputPath) ?? ".");
        PngWriter.Write(outputPath, result.Image);
        return result;
    }

    private static void RunAssertions(string source, string outputPath)
    {
        var result = RenderSource(source, outputPath);
        Check(result.Image.Width == 960 && result.Image.Height == 640, "expected 960x640 canvas");
        Check(result.Operations.Count > 100, "expected repeated drawing operations");
        Check(new FileInfo(outputPath).Length > 1000, "expected non-empty PNG output");

        var parser = new Parser(new ReducerFunc(DrawParser.Reduce));
        Parallel.For(0, 8, _ => parser.ParseValueInput(Scanner.Tokenize(source)));

        try
        {
            Scanner.Tokenize("canvas 1,@");
            throw new InvalidOperationException("expected scanner failure");
        }
        catch (InvalidOperationException ex) when (ex.Message.Contains("no lexical rule", StringComparison.Ordinal))
        {
        }

        try
        {
            Parser.Parse(Scanner.Tokenize("draw ;"));
            throw new InvalidOperationException("expected parser failure");
        }
        catch (InvalidOperationException ex) when (ex.Message.Contains("parse error", StringComparison.Ordinal))
        {
        }
    }

    private static void Check(bool condition, string message)
    {
        if (!condition)
        {
            throw new InvalidOperationException(message);
        }
    }
}

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
        return (DrawProgram)value!;
    }

    /// <summary>
    /// Dispatches generated semantic action IDs to AST-building helpers.
    /// </summary>
    public static object? Reduce(Reduction ctx)
    {
        return ctx.ActionID switch
        {
            SemanticAction.Program => new DrawProgram((List<Statement>)ctx.Values[0]!),
            SemanticAction.Statements => Prepend((Statement)ctx.Values[0]!, (List<Statement>)ctx.Values[1]!),
            SemanticAction.Figures => Prepend((Statement)ctx.Values[0]!, (List<Statement>)ctx.Values[1]!),
            SemanticAction.StatementTailMore => Prepend((Statement)ctx.Values[1]!, (List<Statement>)ctx.Values[2]!),
            SemanticAction.FigureTailMore => Prepend((Statement)ctx.Values[1]!, (List<Statement>)ctx.Values[2]!),
            SemanticAction.StatementTailEmpty => new List<Statement>(),
            SemanticAction.FigureTailEmpty => new List<Statement>(),
            SemanticAction.Pass => ctx.Values[0],
            SemanticAction.Canvas => new CanvasStatement((Expr)ctx.Values[1]!, (Expr)ctx.Values[3]!),
            SemanticAction.Background => new BackgroundStatement((ColorRgb)ctx.Values[1]!),
            SemanticAction.Stroke => new StrokeStatement((ColorRgb)ctx.Values[1]!),
            SemanticAction.Fill => new FillStatement((ColorRgb)ctx.Values[1]!, true),
            SemanticAction.FillNone => new FillStatement(ColorRgb.Black, false),
            SemanticAction.Width => new WidthStatement((Expr)ctx.Values[1]!),
            SemanticAction.Assign => new AssignStatement(Text(ctx, 0), (Expr)ctx.Values[2]!),
            SemanticAction.DefineFigure => new DefineFigureStatement(Text(ctx, 0), (FigureBlock)ctx.Values[2]!),
            SemanticAction.Draw => new DrawStatement((FigureRef)ctx.Values[1]!),
            SemanticAction.Repdraw => new RepDrawStatement((Expr)ctx.Values[1]!, (FigureRef)ctx.Values[2]!),
            SemanticAction.FigureRefNamed => new NamedFigureRef(Text(ctx, 0)),
            SemanticAction.FigureRefInline => new InlineFigureRef((FigureBlock)ctx.Values[0]!),
            SemanticAction.FigureBlock => new FigureBlock((List<Statement>)ctx.Values[1]!),
            SemanticAction.PrimitivePoint => Primitive("point", ctx, 1, 3),
            SemanticAction.PrimitiveLine => Primitive("line", ctx, 1, 3, 5, 7),
            SemanticAction.PrimitiveBox => Primitive("box", ctx, 1, 3, 5, 7),
            SemanticAction.PrimitiveCircle => Primitive("circle", ctx, 1, 3, 5),
            SemanticAction.Color => ParseColor(Text(ctx, 0)),
            SemanticAction.Expr => FoldBinary((Expr)ctx.Values[0]!, (List<BinaryTail>)ctx.Values[1]!),
            SemanticAction.Term => FoldBinary((Expr)ctx.Values[0]!, (List<BinaryTail>)ctx.Values[1]!),
            SemanticAction.ExprTailAdd => BinaryTailList("+", ctx, 1, 2),
            SemanticAction.ExprTailSubtract => BinaryTailList("-", ctx, 1, 2),
            SemanticAction.ExprTailEmpty => new List<BinaryTail>(),
            SemanticAction.TermTailMultiply => BinaryTailList("*", ctx, 1, 2),
            SemanticAction.TermTailDivide => BinaryTailList("/", ctx, 1, 2),
            SemanticAction.TermTailEmpty => new List<BinaryTail>(),
            SemanticAction.UnaryNegate => new UnaryExpr("-", (Expr)ctx.Values[1]!),
            SemanticAction.Number => new NumberExpr(double.Parse(Text(ctx, 0), CultureInfo.InvariantCulture)),
            SemanticAction.Variable => new VariableExpr(Text(ctx, 0)),
            SemanticAction.Call => new CallExpr(Text(ctx, 0), (Expr)ctx.Values[2]!),
            SemanticAction.Group => ctx.Values[1],
            _ => DefaultReduce(ctx.Values),
        };
    }

    private static PrimitiveStatement Primitive(string kind, Reduction ctx, params int[] indexes)
    {
        return new PrimitiveStatement(kind, indexes.Select(index => (Expr)ctx.Values[index]!).ToList());
    }

    private static List<BinaryTail> BinaryTailList(string op, Reduction ctx, int exprIndex, int tailIndex)
    {
        return Prepend(new BinaryTail(op, (Expr)ctx.Values[exprIndex]!), (List<BinaryTail>)ctx.Values[tailIndex]!);
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

    private static string Text(Reduction ctx, int index) => ((Lexeme)ctx.Values[index]!).Text;

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

/// <summary>RGB color used by the renderer and PNG writer.</summary>
internal readonly record struct ColorRgb(byte R, byte G, byte B)
{
    /// <summary>Black is used when fill is disabled.</summary>
    public static ColorRgb Black => new(0, 0, 0);

    /// <summary>Returns the color as a CSS-style hex string.</summary>
    public override string ToString() => $"#{R:X2}{G:X2}{B:X2}";
}

/// <summary>In-memory RGB image.</summary>
internal sealed class ImageBuffer
{
    private readonly ColorRgb[] _pixels;

    /// <summary>Creates a white image with the requested size.</summary>
    public ImageBuffer(int width, int height)
    {
        Width = width;
        Height = height;
        _pixels = Enumerable.Repeat(new ColorRgb(255, 255, 255), width * height).ToArray();
    }

    /// <summary>Image width in pixels.</summary>
    public int Width { get; }

    /// <summary>Image height in pixels.</summary>
    public int Height { get; }

    /// <summary>Sets one pixel if the coordinates are inside the image.</summary>
    public void SetPixel(int x, int y, ColorRgb color)
    {
        if (x < 0 || y < 0 || x >= Width || y >= Height)
        {
            return;
        }
        _pixels[y * Width + x] = color;
    }

    /// <summary>Returns one pixel without bounds checks.</summary>
    public ColorRgb GetPixel(int x, int y) => _pixels[y * Width + x];

    /// <summary>Fills every pixel with a color.</summary>
    public void Fill(ColorRgb color)
    {
        Array.Fill(_pixels, color);
    }
}

/// <summary>Rendering result plus metadata useful for logs and tests.</summary>
internal sealed record RenderResult(ImageBuffer Image, IReadOnlyList<string> Figures, IReadOnlyList<string> Operations);

/// <summary>Executes the DRAW AST and paints it into an in-memory image.</summary>
internal sealed class DrawRenderer
{
    private const int MaxRepDrawIterations = 20000;
    private readonly Dictionary<string, double> _vars = new(StringComparer.Ordinal) { ["PI"] = Math.PI, ["pi"] = Math.PI, ["E"] = Math.E, ["e"] = Math.E };
    private readonly Dictionary<string, FigureBlock> _figures = new(StringComparer.Ordinal);
    private readonly List<string> _operations = [];
    private ColorRgb _stroke = new(0x11, 0x18, 0x27);
    private ColorRgb _fill = new(0xff, 0xff, 0xff);
    private bool _fillOn;
    private double _lineWidth = 1;
    private ImageBuffer? _image;

    /// <summary>Renders a DRAW program.</summary>
    public static RenderResult Render(DrawProgram program)
    {
        var renderer = new DrawRenderer();
        foreach (var statement in program.Statements)
        {
            renderer.Execute(statement);
        }
        if (renderer._image is null)
        {
            throw new InvalidOperationException("program did not create a canvas");
        }
        return new RenderResult(renderer._image, renderer._figures.Keys.Order(StringComparer.Ordinal).ToList(), renderer._operations);
    }

    private void Execute(Statement statement)
    {
        switch (statement)
        {
            case CanvasStatement canvas:
                var width = PositiveDimension(Evaluate(canvas.Width), "width");
                var height = PositiveDimension(Evaluate(canvas.Height), "height");
                _image = new ImageBuffer(width, height);
                _image.Fill(new ColorRgb(255, 255, 255));
                _operations.Add($"canvas {width},{height}");
                break;
            case BackgroundStatement background:
                RequireCanvas().Fill(background.Color);
                _operations.Add($"background {background.Color}");
                break;
            case StrokeStatement stroke:
                _stroke = stroke.Color;
                break;
            case FillStatement fill:
                _fill = fill.Color;
                _fillOn = fill.Enabled;
                break;
            case WidthStatement widthStatement:
                _lineWidth = Math.Max(1, Evaluate(widthStatement.Value));
                break;
            case AssignStatement assign:
                _vars[assign.Name] = Evaluate(assign.Value);
                break;
            case DefineFigureStatement define:
                _figures[define.Name] = define.Figure;
                _operations.Add("define " + define.Name);
                break;
            case DrawStatement draw:
                ExecuteFigureRef(draw.Target);
                break;
            case RepDrawStatement repDraw:
                var count = (int)Math.Round(Evaluate(repDraw.Count));
                if (count < 0 || count > MaxRepDrawIterations)
                {
                    throw new InvalidOperationException($"repdraw count {count} is outside 0..{MaxRepDrawIterations}");
                }
                for (var i = 0; i < count; i++)
                {
                    ExecuteFigureRef(repDraw.Target);
                }
                break;
            case PrimitiveStatement primitive:
                DrawPrimitive(primitive);
                break;
            default:
                throw new InvalidOperationException($"unsupported statement {statement.GetType().Name}");
        }
    }

    private static int PositiveDimension(double value, string name)
    {
        var rounded = (int)Math.Round(value);
        if (rounded <= 0 || rounded > 4096)
        {
            throw new InvalidOperationException($"canvas {name} must be in 1..4096, got {rounded}");
        }
        return rounded;
    }

    private ImageBuffer RequireCanvas() => _image ?? throw new InvalidOperationException("drawing command used before canvas");

    private void ExecuteFigureRef(FigureRef reference)
    {
        switch (reference)
        {
            case NamedFigureRef named when _figures.TryGetValue(named.Name, out var figure):
                ExecuteFigure(figure);
                break;
            case NamedFigureRef named:
                throw new InvalidOperationException($"undefined figure {named.Name}");
            case InlineFigureRef inline:
                ExecuteFigure(inline.Figure);
                break;
            default:
                throw new InvalidOperationException($"unsupported figure reference {reference.GetType().Name}");
        }
    }

    private void ExecuteFigure(FigureBlock figure)
    {
        foreach (var statement in figure.Statements)
        {
            Execute(statement);
        }
    }

    private void DrawPrimitive(PrimitiveStatement primitive)
    {
        var args = primitive.Args.Select(Evaluate).ToArray();
        switch (primitive.Kind)
        {
            case "point":
                FillCircle(args[0], args[1], Math.Max(1, _lineWidth / 2), _stroke);
                break;
            case "line":
                DrawLine(args[0], args[1], args[2], args[3], _stroke, _lineWidth);
                break;
            case "box":
                DrawBox(args[0], args[1], args[2], args[3]);
                break;
            case "circle":
                DrawCircle(args[0], args[1], args[2]);
                break;
            default:
                throw new InvalidOperationException($"unsupported primitive {primitive.Kind}");
        }
        _operations.Add($"{primitive.Kind} {args.Length} args");
    }

    private double Evaluate(Expr expression)
    {
        return expression switch
        {
            NumberExpr number => number.Value,
            VariableExpr variable when _vars.TryGetValue(variable.Name, out var value) => value,
            VariableExpr variable => throw new InvalidOperationException($"undefined variable {variable.Name}"),
            UnaryExpr { Op: "-", Value: var value } => -Evaluate(value),
            BinaryExpr binary => EvaluateBinary(binary),
            CallExpr call => EvaluateCall(call),
            _ => throw new InvalidOperationException($"unsupported expression {expression.GetType().Name}"),
        };
    }

    private double EvaluateBinary(BinaryExpr binary)
    {
        var left = Evaluate(binary.Left);
        var right = Evaluate(binary.Right);
        return binary.Op switch
        {
            "+" => left + right,
            "-" => left - right,
            "*" => left * right,
            "/" when right != 0 => left / right,
            "/" => throw new InvalidOperationException("division by zero"),
            _ => throw new InvalidOperationException($"unsupported operator {binary.Op}"),
        };
    }

    private double EvaluateCall(CallExpr call)
    {
        var arg = Evaluate(call.Arg);
        return call.Name switch
        {
            "sin" => Math.Sin(arg),
            "cos" => Math.Cos(arg),
            "tan" => Math.Tan(arg),
            "ln" => Math.Log(arg),
            "sqrt" => Math.Sqrt(arg),
            "sqr" => arg * arg,
            "exp" => Math.Exp(arg),
            _ => throw new InvalidOperationException($"unsupported function {call.Name}"),
        };
    }

    private void DrawBox(double x1, double y1, double x2, double y2)
    {
        var left = Math.Min(x1, x2);
        var right = Math.Max(x1, x2);
        var top = Math.Min(y1, y2);
        var bottom = Math.Max(y1, y2);
        if (_fillOn)
        {
            for (var y = (int)Math.Round(top); y <= (int)Math.Round(bottom); y++)
            {
                for (var x = (int)Math.Round(left); x <= (int)Math.Round(right); x++)
                {
                    RequireCanvas().SetPixel(x, y, _fill);
                }
            }
        }
        DrawLine(left, top, right, top, _stroke, _lineWidth);
        DrawLine(right, top, right, bottom, _stroke, _lineWidth);
        DrawLine(right, bottom, left, bottom, _stroke, _lineWidth);
        DrawLine(left, bottom, left, top, _stroke, _lineWidth);
    }

    private void DrawCircle(double cx, double cy, double radius)
    {
        radius = Math.Abs(radius);
        if (_fillOn)
        {
            FillCircle(cx, cy, radius, _fill);
        }
        var steps = (int)Math.Max(24, radius * 8);
        var width = Math.Max(1, _lineWidth);
        var prevX = 0.0;
        var prevY = 0.0;
        for (var i = 0; i <= steps; i++)
        {
            var angle = 2 * Math.PI * i / steps;
            var x = cx + Math.Cos(angle) * radius;
            var y = cy + Math.Sin(angle) * radius;
            if (i > 0)
            {
                DrawLine(prevX, prevY, x, y, _stroke, width);
            }
            prevX = x;
            prevY = y;
        }
    }

    private void FillCircle(double cx, double cy, double radius, ColorRgb color)
    {
        radius = Math.Abs(radius);
        var rr = radius * radius;
        for (var y = (int)Math.Floor(cy - radius); y <= (int)Math.Ceiling(cy + radius); y++)
        {
            for (var x = (int)Math.Floor(cx - radius); x <= (int)Math.Ceiling(cx + radius); x++)
            {
                var dx = x - cx;
                var dy = y - cy;
                if (dx * dx + dy * dy <= rr)
                {
                    RequireCanvas().SetPixel(x, y, color);
                }
            }
        }
    }

    private void DrawLine(double x1, double y1, double x2, double y2, ColorRgb color, double width)
    {
        var dx = x2 - x1;
        var dy = y2 - y1;
        var steps = (int)Math.Max(Math.Abs(dx), Math.Abs(dy));
        if (steps == 0)
        {
            FillCircle(x1, y1, Math.Max(1, width / 2), color);
            return;
        }
        var radius = Math.Max(0.5, width / 2);
        for (var i = 0; i <= steps; i++)
        {
            var t = (double)i / steps;
            FillCircle(x1 + dx * t, y1 + dy * t, radius, color);
        }
    }
}

/// <summary>Writes render reports for the C# DRAW example.</summary>
internal static class ReportWriter
{
    /// <summary>Builds a concise report for console and log output.</summary>
    public static string Build(string sourcePath, string outputPath, RenderResult result)
    {
        var report = new StringBuilder();
        report.AppendLine("DRAW C# render report");
        report.AppendLine($"Source: {sourcePath}");
        report.AppendLine($"Output: {outputPath}");
        report.AppendLine($"Canvas: {result.Image.Width}x{result.Image.Height}");
        report.AppendLine($"Figures: [{string.Join(", ", result.Figures)}]");
        report.AppendLine();
        report.AppendLine("Operation summary:");
        foreach (var item in result.Operations.GroupBy(op => op).OrderBy(group => group.Key, StringComparer.Ordinal))
        {
            report.AppendLine($"  {item.Key}: {item.Count()}");
        }
        return report.ToString();
    }
}

/// <summary>Minimal PNG encoder for RGB images.</summary>
internal static class PngWriter
{
    private static readonly byte[] Signature = [137, 80, 78, 71, 13, 10, 26, 10];

    /// <summary>Writes the image as a PNG file.</summary>
    public static void Write(string path, ImageBuffer image)
    {
        using var file = File.Create(path);
        file.Write(Signature);
        WriteChunk(file, "IHDR", BuildHeader(image.Width, image.Height));
        WriteChunk(file, "IDAT", CompressScanlines(image));
        WriteChunk(file, "IEND", []);
    }

    private static byte[] BuildHeader(int width, int height)
    {
        var data = new byte[13];
        BinaryPrimitives.WriteInt32BigEndian(data.AsSpan(0, 4), width);
        BinaryPrimitives.WriteInt32BigEndian(data.AsSpan(4, 4), height);
        data[8] = 8;  // 8-bit channel depth
        data[9] = 2;  // truecolor RGB
        data[10] = 0; // deflate compression
        data[11] = 0; // adaptive filtering
        data[12] = 0; // no interlace
        return data;
    }

    private static byte[] CompressScanlines(ImageBuffer image)
    {
        using var raw = new MemoryStream();
        for (var y = 0; y < image.Height; y++)
        {
            raw.WriteByte(0); // PNG filter type 0: none
            for (var x = 0; x < image.Width; x++)
            {
                var pixel = image.GetPixel(x, y);
                raw.WriteByte(pixel.R);
                raw.WriteByte(pixel.G);
                raw.WriteByte(pixel.B);
            }
        }
        using var compressed = new MemoryStream();
        using (var zlib = new ZLibStream(compressed, CompressionLevel.Fastest, leaveOpen: true))
        {
            raw.Position = 0;
            raw.CopyTo(zlib);
        }
        return compressed.ToArray();
    }

    private static void WriteChunk(Stream stream, string type, byte[] data)
    {
        Span<byte> length = stackalloc byte[4];
        BinaryPrimitives.WriteInt32BigEndian(length, data.Length);
        stream.Write(length);
        var typeBytes = Encoding.ASCII.GetBytes(type);
        stream.Write(typeBytes);
        stream.Write(data);
        var crc = Crc32(typeBytes, data);
        Span<byte> crcBytes = stackalloc byte[4];
        BinaryPrimitives.WriteUInt32BigEndian(crcBytes, crc);
        stream.Write(crcBytes);
    }

    private static uint Crc32(byte[] type, byte[] data)
    {
        var crc = 0xffffffffu;
        foreach (var b in type)
        {
            crc = UpdateCrc(crc, b);
        }
        foreach (var b in data)
        {
            crc = UpdateCrc(crc, b);
        }
        return crc ^ 0xffffffffu;
    }

    private static uint UpdateCrc(uint crc, byte b)
    {
        crc ^= b;
        for (var i = 0; i < 8; i++)
        {
            crc = (crc & 1) == 1 ? 0xedb88320u ^ (crc >> 1) : crc >> 1;
        }
        return crc;
    }
}
