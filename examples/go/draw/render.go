package draw

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"sort"
)

const maxRepDrawIterations = 20000

// RenderResult summarizes one rendered DRAW program.
type RenderResult struct {
	Width      int
	Height     int
	Variables  map[string]float64
	Figures    []string
	Operations []string
}

// RenderPNG renders a parsed DRAW program to a PNG file.
func RenderPNG(program *Program, outputPath string) (*RenderResult, error) {
	img, result, err := Render(program)
	if err != nil {
		return nil, err
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("create output: %w", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return result, nil
}

// Render executes a DRAW program and returns the rendered image.
func Render(program *Program) (*image.RGBA, *RenderResult, error) {
	r := renderer{
		vars: map[string]float64{
			"PI": math.Pi,
			"pi": math.Pi,
			"E":  math.E,
			"e":  math.E,
		},
		figures: map[string]FigureBlock{},
		style: drawStyle{
			stroke:    color.RGBA{R: 0x11, G: 0x18, B: 0x27, A: 255},
			fill:      color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 255},
			fillOn:    false,
			lineWidth: 1,
		},
	}
	for _, stmt := range program.Statements {
		if err := r.exec(stmt); err != nil {
			return nil, nil, err
		}
	}
	if r.img == nil {
		return nil, nil, fmt.Errorf("program did not create a canvas")
	}
	result := &RenderResult{
		Width:      r.img.Bounds().Dx(),
		Height:     r.img.Bounds().Dy(),
		Variables:  copyVars(r.vars),
		Figures:    sortedFigureNames(r.figures),
		Operations: append([]string(nil), r.operations...),
	}
	return r.img, result, nil
}

// WriteReport writes a concise render report.
func WriteReport(w io.Writer, sourcePath string, outputPath string, result *RenderResult) {
	fmt.Fprintf(w, "DRAW render report\n")
	fmt.Fprintf(w, "Source: %s\n", sourcePath)
	fmt.Fprintf(w, "Output: %s\n", outputPath)
	fmt.Fprintf(w, "Canvas: %dx%d\n", result.Width, result.Height)
	fmt.Fprintf(w, "Figures: %v\n", result.Figures)
	fmt.Fprintf(w, "Variables: %d\n", len(result.Variables))
	fmt.Fprintf(w, "\nOperation summary:\n")
	for _, item := range operationSummary(result.Operations) {
		fmt.Fprintf(w, "  %s: %d\n", item.Name, item.Count)
	}
}

type renderer struct {
	img        *image.RGBA
	vars       map[string]float64
	figures    map[string]FigureBlock
	style      drawStyle
	operations []string
}

type drawStyle struct {
	stroke    color.RGBA
	fill      color.RGBA
	fillOn    bool
	lineWidth float64
}

func (r *renderer) exec(stmt Statement) error {
	switch s := stmt.(type) {
	case CanvasStatement:
		width, err := r.eval(s.Width)
		if err != nil {
			return err
		}
		height, err := r.eval(s.Height)
		if err != nil {
			return err
		}
		w, h := int(math.Round(width)), int(math.Round(height))
		if w <= 0 || h <= 0 {
			return fmt.Errorf("canvas dimensions must be positive, got %dx%d", w, h)
		}
		if w > 4096 || h > 4096 {
			return fmt.Errorf("canvas dimensions too large, got %dx%d", w, h)
		}
		r.img = image.NewRGBA(image.Rect(0, 0, w, h))
		r.fillCanvas(color.RGBA{R: 255, G: 255, B: 255, A: 255})
		r.operations = append(r.operations, fmt.Sprintf("canvas %d,%d", w, h))
	case BackgroundStatement:
		if err := r.requireCanvas(); err != nil {
			return err
		}
		r.fillCanvas(s.Color)
		r.operations = append(r.operations, fmt.Sprintf("background %s", hexColor(s.Color)))
	case StrokeStatement:
		r.style.stroke = s.Color
	case FillStatement:
		r.style.fill = s.Color
		r.style.fillOn = s.Enabled
	case WidthStatement:
		width, err := r.eval(s.Value)
		if err != nil {
			return err
		}
		r.style.lineWidth = math.Max(1, width)
	case AssignStatement:
		value, err := r.eval(s.Expr)
		if err != nil {
			return err
		}
		r.vars[s.Name] = value
	case DefineFigureStatement:
		r.figures[s.Name] = s.Figure
		r.operations = append(r.operations, "define "+s.Name)
	case DrawStatement:
		if err := r.execFigureRef(s.Target); err != nil {
			return err
		}
	case RepDrawStatement:
		countValue, err := r.eval(s.Count)
		if err != nil {
			return err
		}
		count := int(math.Round(countValue))
		if count < 0 {
			return fmt.Errorf("repdraw count must be non-negative, got %d", count)
		}
		if count > maxRepDrawIterations {
			return fmt.Errorf("repdraw count %d exceeds limit %d", count, maxRepDrawIterations)
		}
		for i := 0; i < count; i++ {
			if err := r.execFigureRef(s.Target); err != nil {
				return err
			}
		}
	case PrimitiveStatement:
		if err := r.drawPrimitive(s); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported statement %T", stmt)
	}
	return nil
}

func (r *renderer) requireCanvas() error {
	if r.img == nil {
		return fmt.Errorf("drawing command used before canvas")
	}
	return nil
}

func (r *renderer) execFigureRef(ref FigureRef) error {
	switch f := ref.(type) {
	case NamedFigureRef:
		figure, ok := r.figures[f.Name]
		if !ok {
			return fmt.Errorf("undefined figure %q", f.Name)
		}
		return r.execFigure(figure)
	case InlineFigureRef:
		return r.execFigure(f.Figure)
	default:
		return fmt.Errorf("unsupported figure reference %T", ref)
	}
}

func (r *renderer) execFigure(figure FigureBlock) error {
	for _, stmt := range figure.Statements {
		if err := r.exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) drawPrimitive(stmt PrimitiveStatement) error {
	if err := r.requireCanvas(); err != nil {
		return err
	}
	args := make([]float64, len(stmt.Args))
	for i, expr := range stmt.Args {
		value, err := r.eval(expr)
		if err != nil {
			return err
		}
		args[i] = value
	}
	switch stmt.Kind {
	case "point":
		r.fillCircle(args[0], args[1], math.Max(1, r.style.lineWidth/2), r.style.stroke)
	case "line":
		r.drawLine(args[0], args[1], args[2], args[3], r.style.stroke, r.style.lineWidth)
	case "box":
		r.drawBox(args[0], args[1], args[2], args[3])
	case "circle":
		r.drawCircle(args[0], args[1], args[2])
	default:
		return fmt.Errorf("unsupported primitive %q", stmt.Kind)
	}
	r.operations = append(r.operations, fmt.Sprintf("%s %d args", stmt.Kind, len(args)))
	return nil
}

func (r *renderer) eval(expr Expr) (float64, error) {
	switch e := expr.(type) {
	case NumberExpr:
		return e.Value, nil
	case VariableExpr:
		value, ok := r.vars[e.Name]
		if !ok {
			return 0, fmt.Errorf("undefined variable %q", e.Name)
		}
		return value, nil
	case UnaryExpr:
		value, err := r.eval(e.X)
		if err != nil {
			return 0, err
		}
		if e.Op == "-" {
			return -value, nil
		}
		return 0, fmt.Errorf("unsupported unary operator %q", e.Op)
	case BinaryExpr:
		left, err := r.eval(e.Left)
		if err != nil {
			return 0, err
		}
		right, err := r.eval(e.Right)
		if err != nil {
			return 0, err
		}
		switch e.Op {
		case "+":
			return left + right, nil
		case "-":
			return left - right, nil
		case "*":
			return left * right, nil
		case "/":
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		default:
			return 0, fmt.Errorf("unsupported binary operator %q", e.Op)
		}
	case CallExpr:
		arg, err := r.eval(e.Arg)
		if err != nil {
			return 0, err
		}
		switch e.Name {
		case "sin":
			return math.Sin(arg), nil
		case "cos":
			return math.Cos(arg), nil
		case "tan":
			return math.Tan(arg), nil
		case "ln":
			return math.Log(arg), nil
		case "sqrt":
			return math.Sqrt(arg), nil
		case "sqr":
			return arg * arg, nil
		case "exp":
			return math.Exp(arg), nil
		default:
			return 0, fmt.Errorf("unsupported function %q", e.Name)
		}
	default:
		return 0, fmt.Errorf("unsupported expression %T", expr)
	}
}

func (r *renderer) fillCanvas(c color.RGBA) {
	if r.img == nil {
		return
	}
	for y := r.img.Bounds().Min.Y; y < r.img.Bounds().Max.Y; y++ {
		for x := r.img.Bounds().Min.X; x < r.img.Bounds().Max.X; x++ {
			r.img.SetRGBA(x, y, c)
		}
	}
}

func (r *renderer) drawBox(x1, y1, x2, y2 float64) {
	left, right := math.Min(x1, x2), math.Max(x1, x2)
	top, bottom := math.Min(y1, y2), math.Max(y1, y2)
	if r.style.fillOn {
		for y := int(math.Round(top)); y <= int(math.Round(bottom)); y++ {
			for x := int(math.Round(left)); x <= int(math.Round(right)); x++ {
				r.setPixel(x, y, r.style.fill)
			}
		}
	}
	r.drawLine(left, top, right, top, r.style.stroke, r.style.lineWidth)
	r.drawLine(right, top, right, bottom, r.style.stroke, r.style.lineWidth)
	r.drawLine(right, bottom, left, bottom, r.style.stroke, r.style.lineWidth)
	r.drawLine(left, bottom, left, top, r.style.stroke, r.style.lineWidth)
}

func (r *renderer) drawCircle(cx, cy, radius float64) {
	if radius < 0 {
		radius = -radius
	}
	if r.style.fillOn {
		r.fillCircle(cx, cy, radius, r.style.fill)
	}
	steps := int(math.Max(24, radius*8))
	width := math.Max(1, r.style.lineWidth)
	var prevX, prevY float64
	for i := 0; i <= steps; i++ {
		angle := 2 * math.Pi * float64(i) / float64(steps)
		x := cx + math.Cos(angle)*radius
		y := cy + math.Sin(angle)*radius
		if i > 0 {
			r.drawLine(prevX, prevY, x, y, r.style.stroke, width)
		}
		prevX, prevY = x, y
	}
}

func (r *renderer) fillCircle(cx, cy, radius float64, c color.RGBA) {
	if radius < 0 {
		radius = -radius
	}
	rr := radius * radius
	minX := int(math.Floor(cx - radius))
	maxX := int(math.Ceil(cx + radius))
	minY := int(math.Floor(cy - radius))
	maxY := int(math.Ceil(cy + radius))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= rr {
				r.setPixel(x, y, c)
			}
		}
	}
}

func (r *renderer) drawLine(x1, y1, x2, y2 float64, c color.RGBA, width float64) {
	dx := x2 - x1
	dy := y2 - y1
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if steps == 0 {
		r.fillCircle(x1, y1, math.Max(1, width/2), c)
		return
	}
	radius := math.Max(0.5, width/2)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		r.fillCircle(x1+dx*t, y1+dy*t, radius, c)
	}
}

func (r *renderer) setPixel(x, y int, c color.RGBA) {
	if r.img == nil || !image.Pt(x, y).In(r.img.Bounds()) {
		return
	}
	r.img.SetRGBA(x, y, c)
}

func copyVars(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sortedFigureNames(figures map[string]FigureBlock) []string {
	out := make([]string, 0, len(figures))
	for name := range figures {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

type operationCount struct {
	Name  string
	Count int
}

func operationSummary(operations []string) []operationCount {
	counts := map[string]int{}
	for _, op := range operations {
		counts[op]++
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]operationCount, 0, len(names))
	for _, name := range names {
		out = append(out, operationCount{Name: name, Count: counts[name]})
	}
	return out
}

func hexColor(c color.RGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}
