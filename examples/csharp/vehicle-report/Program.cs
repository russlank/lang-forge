using System.Globalization;
using System.Text;
using LangForge.Examples.VehicleReport.Generated;

// Vehicle report parsing demonstrates a data-extraction compiler front-end:
// generated parser tables validate syntax, while the reducer builds a typed
// model that reporting code can consume normally.
static Vehicle ParseVehicle(string source)
{
    var value = Parser.ParseWithReducer(Scanner.Tokenize(source), new ReducerFunc(Reduce));
    return (Vehicle)value!;
}

static object? Reduce(Reduction ctx)
{
    // The generated SemanticAction enum keeps dispatch fast and avoids matching
    // raw strings on every reduction.
    return ctx.ActionID switch
    {
        SemanticAction.Vehicle => new Vehicle(InfoArg(ctx, 3, "vehicle info")),
        SemanticAction.Info => new VehicleInfo(
            StringArg(ctx, 0, "model"),
            StringArg(ctx, 2, "license"),
            IntArg(ctx, 4, "distance"),
            FeatureList(ctx, 6, "features"),
            RepairList(ctx, 8, "repairs")),
        SemanticAction.FieldModel => DecodeQuoted(Text(ctx, 2, "model literal")),
        SemanticAction.FieldLicense => DecodeQuoted(Text(ctx, 2, "license literal")),
        SemanticAction.FieldDistance => int.Parse(Text(ctx, 2, "distance literal"), CultureInfo.InvariantCulture),
        SemanticAction.FieldFeatures => ctx.Values[3],
        SemanticAction.FeatureItems => Prepend(FeatureArg(ctx, 0, "feature"), FeatureList(ctx, 1, "feature tail")),
        SemanticAction.FeatureEmpty => new List<Feature>(),
        SemanticAction.FeatureTailMore => Prepend(FeatureArg(ctx, 1, "feature"), FeatureList(ctx, 2, "feature tail")),
        SemanticAction.FeatureTailEmpty => new List<Feature>(),
        SemanticAction.Feature => new Feature(Text(ctx, 0, "feature name"), DecodeQuoted(Text(ctx, 2, "feature value"))),
        SemanticAction.FieldRepairs => ctx.Values[3],
        SemanticAction.RepairItems => Prepend(RepairArg(ctx, 0, "repair"), RepairList(ctx, 1, "repair tail")),
        SemanticAction.RepairEmpty => new List<Repair>(),
        SemanticAction.RepairTailMore => Prepend(RepairArg(ctx, 1, "repair"), RepairList(ctx, 2, "repair tail")),
        SemanticAction.RepairTailEmpty => new List<Repair>(),
        SemanticAction.Repair => new Repair(DecodeQuoted(Text(ctx, 3, "repair date")), DecodeQuoted(Text(ctx, 7, "repair description"))),
        _ => DefaultReduce(ctx.Values),
    };
}

static List<T> Prepend<T>(T head, List<T> tail)
{
    var result = new List<T> { head };
    result.AddRange(tail);
    return result;
}

static object? DefaultReduce(IReadOnlyList<object?> values)
{
    return values.Count switch
    {
        0 => null,
        1 => values[0],
        _ => values.ToArray(),
    };
}

static T Arg<T>(Reduction ctx, int index, string name)
{
    // The grammar gives RHS values names, but current C# generated reducers are
    // still boxed. Centralizing the cast keeps each switch branch readable and
    // makes future typed-context migration mechanical.
    if (index < 0 || index >= ctx.Values.Count)
    {
        throw new InvalidOperationException($"rule {ctx.Rule} action {ctx.ActionID} is missing {name} at argument {index + 1}");
    }
    if (ctx.Values[index] is not T value)
    {
        throw new InvalidOperationException($"rule {ctx.Rule} action {ctx.ActionID} argument {index + 1} for {name} has type {ctx.Values[index]?.GetType().Name ?? "<null>"}, want {typeof(T).Name}");
    }
    return value;
}

static string Text(Reduction ctx, int index, string name) => Arg<Lexeme>(ctx, index, name).Text;

static string StringArg(Reduction ctx, int index, string name) => Arg<string>(ctx, index, name);

static int IntArg(Reduction ctx, int index, string name) => Arg<int>(ctx, index, name);

static VehicleInfo InfoArg(Reduction ctx, int index, string name) => Arg<VehicleInfo>(ctx, index, name);

static Feature FeatureArg(Reduction ctx, int index, string name) => Arg<Feature>(ctx, index, name);

static Repair RepairArg(Reduction ctx, int index, string name) => Arg<Repair>(ctx, index, name);

static List<Feature> FeatureList(Reduction ctx, int index, string name) => Arg<List<Feature>>(ctx, index, name);

static List<Repair> RepairList(Reduction ctx, int index, string name) => Arg<List<Repair>>(ctx, index, name);

static string DecodeQuoted(string text) => text.Length >= 2 && text[0] == '"' && text[^1] == '"' ? text[1..^1] : text;

static string BuildReport(Vehicle vehicle)
{
    var report = new StringBuilder();
    report.AppendLine("Vehicle report C# generated-parser demo");
    report.AppendLine($"model: {vehicle.Info.Model}");
    report.AppendLine($"license: {vehicle.Info.License}");
    report.AppendLine($"distance: {vehicle.Info.Distance}");
    report.AppendLine("features:");
    foreach (var feature in vehicle.Info.Features)
    {
        report.AppendLine($"  - {feature.Name}: {feature.Value}");
    }
    report.AppendLine("repairs:");
    foreach (var repair in vehicle.Info.Repairs)
    {
        report.AppendLine($"  - {repair.Date}: {repair.Description}");
    }
    return report.ToString();
}

static void Check(bool condition, string message)
{
    if (!condition)
    {
        throw new InvalidOperationException(message);
    }
}

static void RunAssertions(string source)
{
    var vehicle = ParseVehicle(source);
    Check(vehicle.Info.Model == "KIA", "expected model KIA");
    Check(vehicle.Info.Features.Count == 4, "expected four features");
    Check(vehicle.Info.Repairs.Count == 3, "expected three repairs");

    var parser = new Parser(new ReducerFunc(Reduce));
    Parallel.For(0, 8, _ => parser.ParseValueInput(Scanner.Tokenize(source)));

    var empty = source.Replace(
        """
            color = "Red",
            central-lock = "yes",
            airbag = "yes",
            shape_style = "sedan"
        """,
        "",
        StringComparison.Ordinal).Replace(
        """
            ( date = "21-05-2009", description = "first service" ),
            ( date = "12-11-2010", description = "brake inspection" ),
            ( date = "11-03-2011", description = "winter tire replacement" )
        """,
        "",
        StringComparison.Ordinal);
    var emptyVehicle = ParseVehicle(empty);
    Check(emptyVehicle.Info.Features.Count == 0, "expected empty features to parse");
    Check(emptyVehicle.Info.Repairs.Count == 0, "expected empty repairs to parse");

    try
    {
        Scanner.Tokenize("car = @");
        throw new InvalidOperationException("expected scanner failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("no lexical rule"))
    {
    }

    try
    {
        Parser.Parse(Scanner.Tokenize("car = { model = \"KIA\" }"));
        throw new InvalidOperationException("expected parser failure");
    }
    catch (InvalidOperationException ex) when (ex.Message.Contains("parse error"))
    {
    }
}

var argsList = args.ToList();
var assert = argsList.Remove("--assert");
var logPath = "dist/vehicle-report-csharp-demo.log";
var logIndex = argsList.IndexOf("--log");
if (logIndex >= 0 && logIndex + 1 < argsList.Count)
{
    logPath = argsList[logIndex + 1];
    argsList.RemoveAt(logIndex + 1);
    argsList.RemoveAt(logIndex);
}
var inputPath = argsList.Count > 0 ? argsList[0] : "sample.vehicle";
var source = File.ReadAllText(inputPath);
if (assert)
{
    RunAssertions(source);
}

var reportText = BuildReport(ParseVehicle(source));
Console.Write(reportText);
Directory.CreateDirectory(Path.GetDirectoryName(logPath) ?? ".");
File.WriteAllText(logPath, reportText);

/// <summary>Root object produced from the vehicle report grammar.</summary>
sealed record Vehicle(VehicleInfo Info);

/// <summary>Normalized vehicle fields collected by semantic reductions.</summary>
sealed record VehicleInfo(string Model, string License, int Distance, List<Feature> Features, List<Repair> Repairs);

/// <summary>One named vehicle feature.</summary>
sealed record Feature(string Name, string Value);

/// <summary>One repair history entry.</summary>
sealed record Repair(string Date, string Description);
