namespace LangForge.Examples.Templates.LibraryDsl.Parsing;

/// <summary>Domain-level parse result that keeps generated details behind the facade.</summary>
public sealed record ParseResult<T>(T? Value, IReadOnlyList<string> Diagnostics, bool Accepted)
{
    /// <summary>True when parsing accepted and produced a domain value.</summary>
    public bool Success => Accepted && Diagnostics.Count == 0 && Value is not null;

    /// <summary>Creates a successful parse result.</summary>
    public static ParseResult<T> Ok(T value) => new(value, Array.Empty<string>(), true);

    /// <summary>Creates a failed parse result.</summary>
    public static ParseResult<T> Fail(IEnumerable<string> diagnostics, T? partial = default) => new(partial, diagnostics.ToArray(), false);
}
