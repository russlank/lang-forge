using System.Text;
using LangForge.Examples.Templates.LibraryDsl.Ast;
using LangForge.Examples.Templates.LibraryDsl.Parsing;

static string BuildReport(string inputPath, Document document)
{
    var report = new StringBuilder();
    report.AppendLine($"Library DSL C# template: {inputPath}");
    foreach (var entry in document.Entries)
    {
        report.AppendLine($"  {entry.Kind.ToString().ToLowerInvariant()} {entry.Name} = {entry.Value}");
    }
    return report.ToString();
}

static void Require(bool condition, string message)
{
    if (!condition)
    {
        throw new InvalidOperationException(message);
    }
}

static void RunAssertions(ILibraryDslParser parser)
{
    var result = parser.Parse("set retries = 3;\nset title = \"nightly\";\nenable audit;");
    Require(result.Success && result.Value is not null, string.Join("; ", result.Diagnostics));
    var document = result.Value!;
    var settings = document.Settings();
    Require(settings["retries"].Number == 3, "unexpected retries value");
    Require(settings["title"].Text == "nightly", "unexpected title value");
    Require(settings["audit"].Boolean, "expected audit flag");
    Require(!parser.Parse("set retries = ;").Success, "expected parser failure");
    Require(!parser.Parse("set retries = 999999999999999999999999;").Success, "expected reducer failure");
}

var argsList = args.ToList();
var assert = argsList.Remove("--assert");
var logPath = "dist/library-csharp.log";
var logIndex = argsList.IndexOf("--log");
if (logIndex >= 0 && logIndex + 1 < argsList.Count)
{
    logPath = argsList[logIndex + 1];
    argsList.RemoveAt(logIndex + 1);
    argsList.RemoveAt(logIndex);
}

var inputPath = argsList.Count > 0 ? argsList[0] : "input.dsl";
var source = File.ReadAllText(inputPath);
ILibraryDslParser parser = new LibraryDslParser();
if (assert)
{
    RunAssertions(parser);
}

var parsed = parser.Parse(source);
if (!parsed.Success || parsed.Value is null)
{
    Console.Error.WriteLine(string.Join(Environment.NewLine, parsed.Diagnostics));
    return 1;
}

var report = BuildReport(inputPath, parsed.Value);
Console.Write(report);
Directory.CreateDirectory(Path.GetDirectoryName(logPath) ?? ".");
File.WriteAllText(logPath, report);
return 0;
