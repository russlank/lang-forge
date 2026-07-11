using LangForge.Examples.Draw.Generated;

namespace LangForge.Examples.Draw;

/// <summary>
/// Command-line entry point for the C# DRAW example.
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

        var parser = new Parser(DrawParser.CreateReducers());
        Parallel.For(0, 8, _ => parser.ParseValueLexemeSource(new Scanner(source)));

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
            Parser.ParseFromLexemeSource(new Scanner("draw ;"));
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
