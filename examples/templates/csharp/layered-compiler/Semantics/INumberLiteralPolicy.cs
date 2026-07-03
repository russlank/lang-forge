using System.Globalization;

namespace LangForge.Examples.Templates.LayeredCompiler.Semantics;

/// <summary>
/// Domain policy used by reducers when converting number lexemes into AST values.
/// </summary>
/// <remarks>
/// Applications can register another implementation in a dependency-injection
/// container when the grammar stays the same but literal validation needs to
/// follow project-specific rules.
/// </remarks>
public interface INumberLiteralPolicy
{
    /// <summary>Parses a token text value for the grammar action named by <paramref name="context" />.</summary>
    int Parse(string text, string context);
}

/// <summary>Default number policy for the template: decimal Int32 values only.</summary>
public sealed class DefaultNumberLiteralPolicy : INumberLiteralPolicy
{
    /// <inheritdoc />
    public int Parse(string text, string context)
    {
        if (!int.TryParse(text, NumberStyles.None, CultureInfo.InvariantCulture, out var value))
        {
            throw new InvalidOperationException($"{context} value {text} is not a valid Int32");
        }
        return value;
    }
}
