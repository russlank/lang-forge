using LangForge.Examples.ParserRecovery.Generated;

/// <summary>
/// Runs the parser recovery teaching fixture with generated C# scanner/parser
/// code. The grammar contains the synchronization rule
/// <c>Statement : error Semi</c>, so this runner inspects diagnostics instead
/// of using exception-based parse failure handling.
/// </summary>
internal static class Program
{
    private static int Main(string[] args)
    {
        try
        {
            var options = Options.Parse(args);
            var source = File.ReadAllText(options.InputPath);
            var result = ParseSourceText(source);
            PrintResult(result);

            if (options.Assert)
            {
                Require(result.Accepted, "fixture should accept after recovery");
                Require(result.Diagnostics.Count == 2, $"expected 2 diagnostics, got {result.Diagnostics.Count}");
                Require(result.Diagnostics.Any(d => d.Recovery.Discarded > 0), "expected one recovery to discard a token");
                Require(result.Diagnostics.All(d => d.Expected.Any(e => e.Display == "number literal")), "expected number literal diagnostics");
            }
            return 0;
        }
        catch (Exception ex)
        {
            Console.Error.WriteLine(ex.Message);
            return 1;
        }
    }

    /// <summary>
    /// Preferred production-style path:
    /// source text -> generated scanner lexeme source -> recovering parser.
    /// </summary>
    private static ParseResult ParseSourceText(string source)
    {
        return Parser.ParseRecovering(new Scanner(source));
    }

    private static void PrintResult(ParseResult result)
    {
        Console.WriteLine($"accepted: {result.Accepted.ToString().ToLowerInvariant()}");
        for (var index = 0; index < result.Diagnostics.Count; index++)
        {
            var diagnostic = result.Diagnostics[index];
            Console.WriteLine(
                $"{index + 1}. {diagnostic.StartLine}:{diagnostic.StartColumn} unexpected {diagnostic.UnexpectedDisplay}; " +
                $"expected {ExpectedDisplay(diagnostic.Expected)}; " +
                $"recovery={diagnostic.Recovery.Kind} discarded={diagnostic.Recovery.Discarded}");
        }
    }

    private static string ExpectedDisplay(IReadOnlyList<ExpectedToken> expected)
    {
        return expected.Count == 0 ? "<none>" : string.Join(", ", expected.Select(token => token.Display));
    }

    private static void Require(bool condition, string message)
    {
        if (!condition)
        {
            throw new InvalidOperationException(message);
        }
    }

    private sealed record Options(string InputPath, bool Assert)
    {
        public static Options Parse(string[] args)
        {
            var inputPath = "input.recovery";
            var assert = false;
            for (var index = 0; index < args.Length; index++)
            {
                switch (args[index])
                {
                    case "--assert":
                        assert = true;
                        break;
                    case "--input":
                        if (++index >= args.Length)
                        {
                            throw new ArgumentException("missing value for --input");
                        }
                        inputPath = args[index];
                        break;
                    default:
                        inputPath = args[index];
                        break;
                }
            }
            return new Options(inputPath, assert);
        }
    }
}
