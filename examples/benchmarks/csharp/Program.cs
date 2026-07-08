using System.Reflection;
using BenchmarkDotNet.Configs;
using BenchmarkDotNet.Exporters.Json;
using BenchmarkDotNet.Jobs;
using BenchmarkDotNet.Running;

namespace LangForge.Examples.Benchmarks.CSharp;

internal static class Program
{
    private const string DefaultArtifactsPath = "../../../dist/benchmarks/csharp";

    private static void Main(string[] args)
    {
        var options = BenchmarkOptions.Parse(args);
        var artifactsPath = Path.GetFullPath(options.ArtifactsPath ?? DefaultArtifactsPath);
        Directory.CreateDirectory(artifactsPath);
        BenchmarkWorkloads.WriteMetrics(Path.Combine(artifactsPath, "langforge-workloads.json"));

        var config = ManualConfig.Create(DefaultConfig.Instance)
            .WithArtifactsPath(artifactsPath)
            .AddExporter(JsonExporter.Full)
            .AddJob(CreateJob(options.Job));

        BenchmarkWorkloads.ValidateFixtures();
        BenchmarkSwitcher.FromAssembly(Assembly.GetExecutingAssembly()).Run(options.BenchmarkDotNetArgs, config);
    }

    private static Job CreateJob(string job)
    {
        return job.ToLowerInvariant() switch
        {
            "short" => Job.ShortRun.WithId("quick-short"),
            "medium" => Job.MediumRun.WithId("stable-medium"),
            "default" => Job.Default.WithId("stable-default"),
            _ => throw new ArgumentException($"unknown --lf-job value {job}; expected short, medium, or default"),
        };
    }

    private sealed record BenchmarkOptions(string? ArtifactsPath, string Job, string[] BenchmarkDotNetArgs)
    {
        public static BenchmarkOptions Parse(string[] args)
        {
            string? artifacts = null;
            var job = "short";
            var forwarded = new List<string>();
            for (var index = 0; index < args.Length; index++)
            {
                var arg = args[index];
                if (arg == "--artifacts")
                {
                    if (++index >= args.Length)
                    {
                        throw new ArgumentException("missing value for --artifacts");
                    }
                    artifacts = args[index];
                    continue;
                }
                if (arg.StartsWith("--artifacts=", StringComparison.Ordinal))
                {
                    artifacts = arg["--artifacts=".Length..];
                    continue;
                }
                if (arg == "--lf-job")
                {
                    if (++index >= args.Length)
                    {
                        throw new ArgumentException("missing value for --lf-job");
                    }
                    job = args[index];
                    continue;
                }
                if (arg.StartsWith("--lf-job=", StringComparison.Ordinal))
                {
                    job = arg["--lf-job=".Length..];
                    continue;
                }
                forwarded.Add(arg);
            }
            return new BenchmarkOptions(artifacts, job, forwarded.ToArray());
        }
    }
}
