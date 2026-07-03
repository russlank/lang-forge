namespace LangForge.Examples.Templates.LayeredCompiler.Parsing;

/// <summary>Domain-level parse result that keeps generated parser details behind the facade.</summary>
/// <param name="Value">Domain value produced by a successful parse, or a partial value when one is available.</param>
/// <param name="Diagnostics">Application-facing diagnostics.</param>
/// <param name="Accepted">True when the generated parser accepted the input.</param>
public sealed record ParseResult<T>(T? Value, IReadOnlyList<string> Diagnostics, bool Accepted)
{
    /// <summary>True when parsing accepted, produced a domain value, and has no diagnostics.</summary>
    public bool Success => Accepted && Diagnostics.Count == 0 && Value is not null;

    /// <summary>Creates a successful parse result.</summary>
    public static ParseResult<T> Ok(T value) => new(value, Array.Empty<string>(), true);

    /// <summary>Creates a failed parse result.</summary>
    public static ParseResult<T> Fail(IEnumerable<string> diagnostics, T? partial = default) =>
        new(partial, diagnostics.ToArray(), false);
}
