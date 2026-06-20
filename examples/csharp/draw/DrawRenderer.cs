namespace LangForge.Examples.Draw;

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
