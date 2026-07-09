using System.Globalization;
using System.Text;
using LangForge.Examples.VehicleReport.Generated;
using static LangForge.Examples.VehicleReport.Generated.SemanticReducerContexts;

// Vehicle report parsing demonstrates a data-extraction compiler front-end:
// generated parser tables validate syntax, while typed reducers build a model
// that reporting code can consume normally.
var reducers = CreateReducers();

static Vehicle ParseVehicle(string source, ReducerMap reducers)
{
    var value = Parser.ParseWithReducerFromSource(new Scanner(source), reducers);
    return value is Vehicle vehicle
        ? vehicle
        : throw new InvalidOperationException($"parser returned {value?.GetType().Name ?? "<null>"} instead of Vehicle");
}

static ReducerMap CreateReducers()
{
    // The generated SemanticAction enum keeps dispatch fast and avoids matching
    // raw strings on every reduction. The typed adapters also verify the named
    // RHS labels and convert them to C# values before invoking these handlers.
    return new ReducerMap
    {
        [SemanticAction.Vehicle] = TypedVehicle(BuildVehicle),
        [SemanticAction.Info] = TypedInfo(Info),
        [SemanticAction.FieldModel] = TypedFieldModel(FieldModel),
        [SemanticAction.FieldLicense] = TypedFieldLicense(FieldLicense),
        [SemanticAction.FieldDistance] = TypedFieldDistance(FieldDistance),
        [SemanticAction.FieldFeatures] = TypedFieldFeatures(FieldFeatures),
        [SemanticAction.FeatureItems] = TypedFeatureItems(FeatureItems),
        [SemanticAction.FeatureEmpty] = TypedFeatureEmpty(FeatureEmpty),
        [SemanticAction.FeatureTailMore] = TypedFeatureTailMore(FeatureTailMore),
        [SemanticAction.FeatureTailEmpty] = TypedFeatureTailEmpty(FeatureTailEmpty),
        [SemanticAction.Feature] = TypedFeature(BuildFeature),
        [SemanticAction.FieldRepairs] = TypedFieldRepairs(FieldRepairs),
        [SemanticAction.RepairItems] = TypedRepairItems(RepairItems),
        [SemanticAction.RepairEmpty] = TypedRepairEmpty(RepairEmpty),
        [SemanticAction.RepairTailMore] = TypedRepairTailMore(RepairTailMore),
        [SemanticAction.RepairTailEmpty] = TypedRepairTailEmpty(RepairTailEmpty),
        [SemanticAction.Repair] = TypedRepair(BuildRepair),
    };
}

static Vehicle BuildVehicle(VehicleReduction ctx) => new(ctx.Info);

static VehicleInfo Info(InfoReduction ctx) => new(ctx.Model, ctx.License, ctx.Distance, ctx.Features, ctx.Repairs);

static string FieldModel(FieldModelReduction ctx) => DecodeQuoted(ctx.Literal.Text);

static string FieldLicense(FieldLicenseReduction ctx) => DecodeQuoted(ctx.Literal.Text);

static int FieldDistance(FieldDistanceReduction ctx) => int.Parse(ctx.Literal.Text, CultureInfo.InvariantCulture);

static List<Feature> FieldFeatures(FieldFeaturesReduction ctx) => ctx.Items;

static List<Feature> FeatureItems(FeatureItemsReduction ctx) => Prepend(ctx.Head, ctx.Tail);

static List<Feature> FeatureEmpty(FeatureEmptyReduction ctx) => [];

static List<Feature> FeatureTailMore(FeatureTailMoreReduction ctx) => Prepend(ctx.Head, ctx.Tail);

static List<Feature> FeatureTailEmpty(FeatureTailEmptyReduction ctx) => [];

static Feature BuildFeature(FeatureReduction ctx) => new(ctx.Name.Text, DecodeQuoted(ctx.Value.Text));

static List<Repair> FieldRepairs(FieldRepairsReduction ctx) => ctx.Items;

static List<Repair> RepairItems(RepairItemsReduction ctx) => Prepend(ctx.Head, ctx.Tail);

static List<Repair> RepairEmpty(RepairEmptyReduction ctx) => [];

static List<Repair> RepairTailMore(RepairTailMoreReduction ctx) => Prepend(ctx.Head, ctx.Tail);

static List<Repair> RepairTailEmpty(RepairTailEmptyReduction ctx) => [];

static Repair BuildRepair(RepairReduction ctx) => new(DecodeQuoted(ctx.Date.Text), DecodeQuoted(ctx.Description.Text));

static List<T> Prepend<T>(T head, List<T> tail)
{
    var result = new List<T> { head };
    result.AddRange(tail);
    return result;
}

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

static void RunAssertions(string source, ReducerMap reducers)
{
    var vehicle = ParseVehicle(source, reducers);
    Check(vehicle.Info.Model == "KIA", "expected model KIA");
    Check(vehicle.Info.Features.Count == 4, "expected four features");
    Check(vehicle.Info.Repairs.Count == 3, "expected three repairs");

    var parser = new Parser(reducers);
    Parallel.For(0, 8, _ => parser.ParseValueSource(new Scanner(source)));

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
    var emptyVehicle = ParseVehicle(empty, reducers);
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
        Parser.ParseFromSource(new Scanner("car = { model = \"KIA\" }"));
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
    RunAssertions(source, reducers);
}

var reportText = BuildReport(ParseVehicle(source, reducers));
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
