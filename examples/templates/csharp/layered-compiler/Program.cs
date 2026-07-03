using LangForge.Examples.Templates.LayeredCompiler.Compilation;
using LangForge.Examples.Templates.LayeredCompiler.Parsing;

static void Require(bool condition, string message)
{
    if (!condition)
    {
        throw new InvalidOperationException(message);
    }
}

static void RunAssertions(IMiniCompilerParser parser, string source)
{
    var parsed = parser.Parse(source);
    Require(parsed.Success && parsed.Value is not null, string.Join("; ", parsed.Diagnostics));
    var output = StackMachine.Execute(MiniCompiler.Compile(parsed.Value!));
    Require(output.Count == 2 && output[0] == 3 && output[1] == 42, $"unexpected output: [{string.Join(", ", output)}]");

    var syntax = parser.Parse("print 1 +;");
    Require(!syntax.Success, "expected parser failure");
    Require(syntax.Diagnostics.Count > 0 && syntax.Diagnostics[0].Contains("unexpected", StringComparison.Ordinal),
        "wrong parser diagnostic");

    var reducer = parser.Parse("print 999999999999999999999999;");
    Require(!reducer.Success, "expected reducer failure");
    Require(reducer.Diagnostics.Count > 0 &&
            reducer.Diagnostics[0].Contains("action number", StringComparison.Ordinal) &&
            reducer.Diagnostics[0].Contains("label token", StringComparison.Ordinal),
        "wrong reducer diagnostic");
}

var argsList = args.ToList();
var assert = argsList.Remove("--assert");
var logPath = "dist/layered-csharp.log";
var logIndex = argsList.IndexOf("--log");
if (logIndex >= 0 && logIndex + 1 < argsList.Count)
{
    logPath = argsList[logIndex + 1];
    argsList.RemoveAt(logIndex + 1);
    argsList.RemoveAt(logIndex);
}

var inputPath = argsList.Count > 0 ? argsList[0] : "input.mini";
var source = File.ReadAllText(inputPath);
IMiniCompilerParser parser = new MiniCompilerParser();
if (assert)
{
    RunAssertions(parser, source);
}

var result = parser.Parse(source);
if (!result.Success || result.Value is null)
{
    Console.Error.WriteLine(string.Join(Environment.NewLine, result.Diagnostics));
    return 1;
}

var code = MiniCompiler.Compile(result.Value);
var output = StackMachine.Execute(code);
var report = ReportFormatter.Format(inputPath, source, code, output);
Console.Write(report);
Directory.CreateDirectory(Path.GetDirectoryName(logPath) ?? ".");
File.WriteAllText(logPath, report);
return 0;
