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
        SemanticAction.Vehicle => new Vehicle((VehicleInfo)ctx.Values[3]!),
        SemanticAction.Info => new VehicleInfo(
            (string)ctx.Values[0]!,
            (string)ctx.Values[2]!,
            (int)ctx.Values[4]!,
            (List<Feature>)ctx.Values[6]!,
            (List<Repair>)ctx.Values[8]!),
        SemanticAction.FieldModel => DecodeQuoted(Text(ctx, 2)),
        SemanticAction.FieldLicense => DecodeQuoted(Text(ctx, 2)),
        SemanticAction.FieldDistance => int.Parse(Text(ctx, 2), CultureInfo.InvariantCulture),
        SemanticAction.FieldFeatures => ctx.Values[3],
        SemanticAction.FeatureItems => Prepend((Feature)ctx.Values[0]!, (List<Feature>)ctx.Values[1]!),
        SemanticAction.FeatureEmpty => new List<Feature>(),
        SemanticAction.FeatureTailMore => Prepend((Feature)ctx.Values[1]!, (List<Feature>)ctx.Values[2]!),
        SemanticAction.FeatureTailEmpty => new List<Feature>(),
        SemanticAction.Feature => new Feature(Text(ctx, 0), DecodeQuoted(Text(ctx, 2))),
        SemanticAction.FieldRepairs => ctx.Values[3],
        SemanticAction.RepairItems => Prepend((Repair)ctx.Values[0]!, (List<Repair>)ctx.Values[1]!),
        SemanticAction.RepairEmpty => new List<Repair>(),
        SemanticAction.RepairTailMore => Prepend((Repair)ctx.Values[1]!, (List<Repair>)ctx.Values[2]!),
        SemanticAction.RepairTailEmpty => new List<Repair>(),
        SemanticAction.Repair => new Repair(DecodeQuoted(Text(ctx, 3)), DecodeQuoted(Text(ctx, 7))),
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

static string Text(Reduction ctx, int index) => ((Lexeme)ctx.Values[index]!).Text;

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
