using System.Text;

namespace LangForge.Examples.Draw;

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
