using System.Reflection;
using BenchmarkDotNet.Configs;
using BenchmarkDotNet.Exporters.Json;
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

        var config = DefaultConfig.Instance
            .WithArtifactsPath(artifactsPath)
            .AddExporter(JsonExporter.Full);

        BenchmarkWorkloads.ValidateFixtures();
        BenchmarkSwitcher.FromAssembly(Assembly.GetExecutingAssembly()).Run(options.BenchmarkDotNetArgs, config);
    }

    private sealed record BenchmarkOptions(string? ArtifactsPath, string[] BenchmarkDotNetArgs)
    {
        public static BenchmarkOptions Parse(string[] args)
        {
            string? artifacts = null;
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
                forwarded.Add(arg);
            }
            return new BenchmarkOptions(artifacts, forwarded.ToArray());
        }
    }
}
