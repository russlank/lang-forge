using LangForge.Examples.Templates.LayeredCompiler.Generated;

namespace LangForge.Examples.Templates.LayeredCompiler.Parsing;

/// <summary>Formats generated diagnostics into stable application-facing messages.</summary>
internal static class DiagnosticFormatter
{
    /// <summary>Formats every generated parser diagnostic on one line each.</summary>
    public static IReadOnlyList<string> Format(IReadOnlyList<ParseDiagnostic> diagnostics)
    {
        return diagnostics.Select(FormatOne).ToArray();
    }

    private static string FormatOne(ParseDiagnostic diagnostic)
    {
        var expected = diagnostic.Expected.Count == 0
            ? "no known continuation"
            : string.Join(", ", diagnostic.Expected.Select(item => item.Display));
        return $"{diagnostic.StartLine}:{diagnostic.StartColumn}: unexpected {diagnostic.UnexpectedDisplay}; expected {expected}";
    }
}
