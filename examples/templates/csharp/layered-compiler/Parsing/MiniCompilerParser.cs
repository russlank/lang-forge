using LangForge.Examples.Templates.LayeredCompiler.Ast;
using LangForge.Examples.Templates.LayeredCompiler.Generated;
using LangForge.Examples.Templates.LayeredCompiler.Semantics;

namespace LangForge.Examples.Templates.LayeredCompiler.Parsing;

/// <summary>
/// Parser facade that hides generated scanner, parser, reducer, and diagnostic
/// types from application callers.
/// </summary>
public sealed class MiniCompilerParser : IMiniCompilerParser
{
    private readonly ReducerMap reducers;

    /// <summary>Creates a parser facade with the default semantic policies.</summary>
    public MiniCompilerParser()
        : this(new DefaultNumberLiteralPolicy())
    {
    }

    /// <summary>
    /// Creates a parser facade with injectable semantic policy dependencies.
    /// </summary>
    /// <remarks>
    /// A real application can register <see cref="IMiniCompilerParser" /> as
    /// <see cref="MiniCompilerParser" /> and register a custom
    /// <see cref="INumberLiteralPolicy" /> in its DI container.
    /// </remarks>
    public MiniCompilerParser(INumberLiteralPolicy numberPolicy)
    {
        reducers = ReducerFactory.Create(numberPolicy);
    }

    /// <inheritdoc />
    public ParseResult<ProgramNode> Parse(string source)
    {
        // Grammar-to-code map for readers:
        // grammar.lf declares action labels such as {csharp: print} and
        // {csharp: add}. ReducerFactory maps those generated SemanticAction
        // values to typed handlers. This facade keeps that generated contract
        // private and returns ProgramNode plus diagnostics to application code.
        try
        {
            var result = Parser.ParseRecovering(new Scanner(source), reducers);
            if (!result.Accepted || result.Diagnostics.Count != 0)
            {
                return ParseResult<ProgramNode>.Fail(DiagnosticFormatter.Format(result.Diagnostics), result.Value as ProgramNode);
            }

            return result.Value is ProgramNode program
                ? ParseResult<ProgramNode>.Ok(program)
                : ParseResult<ProgramNode>.Fail(new[] { $"parser final value has type {result.Value?.GetType().Name ?? "<null>"}, want ProgramNode" });
        }
        catch (Exception ex) when (ex is InvalidOperationException or FormatException or OverflowException)
        {
            return ParseResult<ProgramNode>.Fail(new[] { ex.Message });
        }
    }
}
