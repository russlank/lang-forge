//go:build langforge_generated

package draw

import (
	"bytes"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestRenderSample(t *testing.T) {
	source := readSample(t)
	program, err := Compile(source)
	if err != nil {
		t.Fatal(err)
	}
	img, result, err := Render(program)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := img.Bounds().Dx(), 960; got != want {
		t.Fatalf("width = %d, want %d", got, want)
	}
	if got, want := img.Bounds().Dy(), 640; got != want {
		t.Fatalf("height = %d, want %d", got, want)
	}
	if len(result.Figures) != 5 {
		t.Fatalf("figures = %#v", result.Figures)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	if encoded.Len() == 0 {
		t.Fatal("expected encoded PNG bytes")
	}
	var report bytes.Buffer
	WriteReport(&report, "sample.draw", "sample.png", result)
	if !strings.Contains(report.String(), "Canvas: 960x640") || !strings.Contains(report.String(), "Operation summary:") {
		t.Fatalf("report missing expected content:\n%s", report.String())
	}
}

func TestRenderInlineLoop(t *testing.T) {
	source := `canvas 64,64;
background #000000;
x = 4;
fill #FFFFFF;
stroke #FFFFFF;
repdraw 4 (
  circle x,32,3;
  x = x + 12;
);`
	program, err := Compile(source)
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := Render(program)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, a := img.At(4, 32).RGBA(); a == 0 {
		t.Fatal("expected rendered pixel")
	}
}

func TestRejectsUndefinedFigure(t *testing.T) {
	program, err := Compile(`canvas 10,10; draw missing;`)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Render(program); err == nil {
		t.Fatal("expected undefined figure error")
	}
}

func TestRejectsMalformedSource(t *testing.T) {
	_, err := Compile(`canvas 10,10; line 0,0,1;`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func readSample(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("sample.draw")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
