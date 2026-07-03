using System.Globalization;
using System.Text;
using LangForge.Examples.Templates.LibraryDsl.Ast;
using LangForge.Examples.Templates.LibraryDsl.Generated;
using static LangForge.Examples.Templates.LibraryDsl.Generated.SemanticReducerContexts;

namespace LangForge.Examples.Templates.LibraryDsl.Semantics;

/// <summary>Creates generated reducer maps for the library DSL grammar.</summary>
public static class ReducerFactory
{
    /// <summary>Builds a reducer map with full generated coverage validation.</summary>
    public static ReducerMap Create() => new()
    {
        // Document : entries=Entries {csharp: document}
        [SemanticAction.Document] = TypedDocument(ctx => new Document(ctx.Entries)),

        // Entries : head=Entry tail=EntriesTail {csharp: entries}
        [SemanticAction.Entries] = TypedEntries(ctx => Prepend(ctx.Head, ctx.Tail)),

        // Entries : %empty {csharp: entries.empty}
        [SemanticAction.EntriesEmpty] = TypedEntriesEmpty(_ => new List<Entry>()),

        // EntriesTail : head=Entry tail=EntriesTail {csharp: entries.tail.more}
        [SemanticAction.EntriesTailMore] = TypedEntriesTailMore(ctx => Prepend(ctx.Head, ctx.Tail)),

        // EntriesTail : %empty {csharp: entries.tail.empty}
        [SemanticAction.EntriesTailEmpty] = TypedEntriesTailEmpty(_ => new List<Entry>()),

        // Entry : Set name=Ident Assign value=Value Semi {csharp: entry.set}
        [SemanticAction.EntrySet] = TypedEntrySet(ctx => new Entry(EntryKind.Set, ctx.Name.Text, ctx.Value)),

        // Entry : Enable name=Ident Semi {csharp: entry.enable}
        [SemanticAction.EntryEnable] = TypedEntryEnable(ctx => new Entry(EntryKind.Enable, ctx.Name.Text, new Value(ValueKind.Boolean, Boolean: true))),

        // Value : token=Number {csharp: value.number}
        [SemanticAction.ValueNumber] = TypedValueNumber(ReduceNumber),

        // Value : token=String {csharp: value.string}
        [SemanticAction.ValueString] = TypedValueString(ctx => new Value(ValueKind.String, Unquote(ctx.Token.Text))),

        // Value : token=Ident {csharp: value.ident}
        [SemanticAction.ValueIdent] = TypedValueIdent(ctx => new Value(ValueKind.Identifier, ctx.Token.Text)),
    };

    private static Value ReduceNumber(ValueNumberReduction ctx)
    {
        if (!int.TryParse(ctx.Token.Text, NumberStyles.None, CultureInfo.InvariantCulture, out var value))
        {
            throw new InvalidOperationException($"rule {ctx.Reduction.Rule} action {ctx.Reduction.Action} label token value {ctx.Token.Text} is not a valid Int32");
        }
        return new Value(ValueKind.Number, Number: value);
    }

    private static List<Entry> Prepend(Entry head, List<Entry> tail)
    {
        var result = new List<Entry> { head };
        result.AddRange(tail);
        return result;
    }

    private static string Unquote(string text)
    {
        if (text.Length < 2 || text[0] != '"' || text[^1] != '"')
        {
            throw new InvalidOperationException($"string literal {text} is not quoted");
        }
        var body = text[1..^1];
        var result = new StringBuilder();
        for (var i = 0; i < body.Length; i++)
        {
            if (body[i] == '\\')
            {
                i++;
                if (i >= body.Length)
                {
                    throw new InvalidOperationException($"string literal {text} ends with an escape");
                }
            }
            result.Append(body[i]);
        }
        return result.ToString();
    }
}
