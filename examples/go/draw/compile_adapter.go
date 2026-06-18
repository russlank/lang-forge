//go:build langforge_generated

package draw

// Compile parses source with the source-owned LangForge parser adapter.
func Compile(source string) (*Program, error) {
	return Parse(source)
}
