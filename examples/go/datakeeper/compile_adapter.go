//go:build langforge_generated

package datakeeper

// Compile parses source with the source-owned LangForge parser adapter and
// compiles the resulting AST to stack-machine code.
func Compile(source string) (*Script, *Executable, error) {
	ast, err := Parse(source)
	if err != nil {
		return nil, nil, err
	}
	exe, err := CompileScript(ast)
	if err != nil {
		return nil, nil, err
	}
	return ast, exe, nil
}
