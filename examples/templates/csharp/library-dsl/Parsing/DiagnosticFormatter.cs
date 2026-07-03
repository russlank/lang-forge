using LangForge.Examples.Templates.LibraryDsl.Generated;

namespace LangForge.Examples.Templates.LibraryDsl.Parsing;

/// <summary>Formats generated syntax diagnostics for application-facing errors.</summary>
public static class DiagnosticFormatter
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
