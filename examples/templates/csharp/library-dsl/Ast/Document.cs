namespace LangForge.Examples.Templates.LibraryDsl.Ast;

/// <summary>Stable AST root returned by the parser facade.</summary>
public sealed record Document(IReadOnlyList<Entry> Entries)
{
    /// <summary>Returns the latest value for each configured name.</summary>
    public IReadOnlyDictionary<string, Value> Settings()
    {
        var settings = new Dictionary<string, Value>(StringComparer.Ordinal);
        foreach (var entry in Entries)
        {
            settings[entry.Name] = entry.Value;
        }
        return settings;
    }
}

/// <summary>Identifies which grammar alternative produced an entry.</summary>
public enum EntryKind
{
    /// <summary>Entry : Set name=Ident Assign value=Value Semi.</summary>
    Set,

    /// <summary>Entry : Enable name=Ident Semi.</summary>
    Enable,
}

/// <summary>One top-level DSL statement.</summary>
public sealed record Entry(EntryKind Kind, string Name, Value Value);

/// <summary>Identifies which Value grammar alternative was reduced.</summary>
public enum ValueKind
{
    /// <summary>Value : token=Number.</summary>
    Number,

    /// <summary>Value : token=String.</summary>
    String,

    /// <summary>Value : token=Ident.</summary>
    Identifier,

    /// <summary>Implicit value used by enable statements.</summary>
    Boolean,
}

/// <summary>Assignment or enable value carried by the AST.</summary>
public sealed record Value(ValueKind Kind, string Text = "", int Number = 0, bool Boolean = false)
{
    /// <summary>Formats the value for demos and diagnostics.</summary>
    public override string ToString() => Kind switch
    {
        ValueKind.Number => Number.ToString(System.Globalization.CultureInfo.InvariantCulture),
        ValueKind.String => "\"" + Text + "\"",
        ValueKind.Identifier => Text,
        ValueKind.Boolean => Boolean.ToString(System.Globalization.CultureInfo.InvariantCulture).ToLowerInvariant(),
        _ => $"<unknown:{Kind}>",
    };
}
