using LangForge.Examples.Templates.LibraryDsl.Ast;
using LangForge.Examples.Templates.LibraryDsl.Generated;
using LangForge.Examples.Templates.LibraryDsl.Semantics;

namespace LangForge.Examples.Templates.LibraryDsl.Parsing;

/// <summary>Stable parser API for applications that consume the library DSL.</summary>
public interface ILibraryDslParser
{
    /// <summary>Parses source through the generated scanner token source.</summary>
    ParseResult<Document> Parse(string source);
}

/// <summary>Concrete parser facade that hides generated reducer and parser details.</summary>
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
            var result = parser.ParseRecoveringSource(new Scanner(source));
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

    /// <summary>Compatibility/debug path for callers that have already tokenized input.</summary>
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
