using LangForge.Examples.Templates.LayeredCompiler.Ast;

namespace LangForge.Examples.Templates.LayeredCompiler.Parsing;

/// <summary>Stable parser API for applications that consume the mini compiler language.</summary>
public interface IMiniCompilerParser
{
    /// <summary>Parses source text and returns a domain-level AST result.</summary>
    ParseResult<ProgramNode> Parse(string source);
}
