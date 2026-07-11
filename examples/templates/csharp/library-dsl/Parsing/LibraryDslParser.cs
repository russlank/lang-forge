using LangForge.Examples.Templates.LibraryDsl.Ast;
using LangForge.Examples.Templates.LibraryDsl.Generated;
using LangForge.Examples.Templates.LibraryDsl.Semantics;

namespace LangForge.Examples.Templates.LibraryDsl.Parsing;

/// <summary>
/// Stable parser API for applications that consume the library DSL.
/// </summary>
/// <remarks>
/// Application code should depend on this interface instead of generated
/// parser classes. That keeps regenerated scanner/parser tables behind one
/// small facade and leaves the public API expressed in domain types.
/// </remarks>
public interface ILibraryDslParser
{
    /// <summary>
    /// Parses source text through the generated scanner lexeme source.
    /// </summary>
    /// <param name="source">DSL source text owned by the caller.</param>
    /// <returns>A domain-level result containing a document, diagnostics, or a partial document.</returns>
    ParseResult<Document> Parse(string source);
}

/// <summary>
/// Concrete parser facade that hides generated reducer and parser details.
/// </summary>
/// <remarks>
/// This class owns pure reducer wiring and creates a fresh generated parser
/// state for each parse call. The generated scanner/parser still do the syntax
/// work; this facade translates their output into <see cref="ParseResult{T}" />.
/// </remarks>
public sealed class LibraryDslParser : ILibraryDslParser
{
    private readonly ReducerMap reducers;

    /// <summary>Creates a parser facade with complete reducer coverage.</summary>
    public LibraryDslParser()
    {
        reducers = ReducerFactory.Create();
    }

    /// <inheritdoc />
    public ParseResult<Document> Parse(string source)
    {
        try
        {
            var parser = new Parser(reducers);
            var result = parser.ParseRecoveringLexemeSource(new Scanner(source));
            if (!result.Accepted || result.Diagnostics.Count != 0)
            {
                return ParseResult<Document>.Fail(DiagnosticFormatter.Format(result.Diagnostics), result.Value as Document);
            }
            return result.Value is Document document
                ? ParseResult<Document>.Ok(document)
                : ParseResult<Document>.Fail(new[] { $"parser final value has type {result.Value?.GetType().Name ?? "<null>"}, want Document" });
        }
        catch (Exception ex) when (ex is InvalidOperationException or FormatException or OverflowException)
        {
            return ParseResult<Document>.Fail(new[] { ex.Message });
        }
    }

    /// <summary>
    /// Compatibility/debug path for callers that have already tokenized input.
    /// </summary>
    /// <param name="tokens">A token collection produced by the generated scanner.</param>
    /// <returns>A domain-level result equivalent to <see cref="Parse(string)" />.</returns>
    /// <remarks>
    /// Prefer <see cref="Parse(string)" /> for production code. Token
    /// collections are useful when tests or tools need to inspect the scanner
    /// output before parsing.
    /// </remarks>
    public ParseResult<Document> ParseTokens(IReadOnlyList<Lexeme> tokens)
    {
        try
        {
            var parser = new Parser(reducers);
            var result = parser.ParseRecoveringInput(tokens);
            if (!result.Accepted || result.Diagnostics.Count != 0)
            {
                return ParseResult<Document>.Fail(DiagnosticFormatter.Format(result.Diagnostics), result.Value as Document);
            }
            return result.Value is Document document
                ? ParseResult<Document>.Ok(document)
                : ParseResult<Document>.Fail(new[] { $"parser final value has type {result.Value?.GetType().Name ?? "<null>"}, want Document" });
        }
        catch (Exception ex) when (ex is InvalidOperationException or FormatException or OverflowException)
        {
            return ParseResult<Document>.Fail(new[] { ex.Message });
        }
    }
}
