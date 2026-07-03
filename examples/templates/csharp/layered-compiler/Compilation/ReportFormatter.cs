using System.Text;

namespace LangForge.Examples.Templates.LayeredCompiler.Compilation;

/// <summary>Formats compiler and runtime output for the thin demo program.</summary>
public static class ReportFormatter
{
    /// <summary>Builds a stable text report used by the demo and smoke test.</summary>
    public static string Format(
        string inputPath,
        string source,
        IReadOnlyList<Instruction> code,
        IReadOnlyList<int> output)
    {
        var report = new StringBuilder();
        report.AppendLine($"Layered C# compiler template: {inputPath}");
        report.AppendLine("source:");
        foreach (var line in source.Trim().Split('\n'))
        {
            report.AppendLine($"  {line}");
        }
        report.AppendLine("stack code:");
        for (var i = 0; i < code.Count; i++)
        {
            var instruction = code[i];
            report.AppendLine(instruction.Op == OpCode.Push
                ? $"  {i:00} push {instruction.Argument}"
                : $"  {i:00} {instruction.Op.ToString().ToLowerInvariant()}");
        }
        report.AppendLine($"output: [{string.Join(", ", output)}]");
        return report.ToString();
    }
}
