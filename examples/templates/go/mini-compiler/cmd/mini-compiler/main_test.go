//go:build langforge_generated

package main

import (
	"testing"

	minimodel "github.com/russlank/lang-forge/examples/templates/go/mini-compiler/model"
)

func TestMiniCompilerPipeline(t *testing.T) {
	p, err := parse("print 1 + 2;\nprint 40 + 2;")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	code := minimodel.CompileProgram(p)
	output, err := minimodel.Run(code)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(output) != 2 || output[0] != 3 || output[1] != 42 {
		t.Fatalf("unexpected output: %#v", output)
	}
	if _, err := parse("print 1 +;"); err == nil {
		t.Fatalf("expected parser error")
	}
}
