// Package version exposes build metadata for the CLI and generators.
package version

// Name is the CLI and manifest tool name.
const Name = "lang-forge"

var (
	// Version is set by release builds with -ldflags.
	Version = "dev"
	// Commit is set by release builds with -ldflags.
	Commit = "unknown"
	// BuildDate is set by release builds with -ldflags.
	BuildDate = "unknown"
	// Branch is set by release builds with -ldflags.
	Branch = "unknown"
)

// String returns a human-readable version line.
func String() string {
	return Name + " " + Version + " (commit " + Commit + ", branch " + Branch + ", built " + BuildDate + ")"
}
