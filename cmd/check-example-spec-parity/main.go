package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type exampleFamily struct {
	Name               string
	Specs              []string
	AllowedDifferences []allowedDifference
}

type allowedDifference struct {
	Baseline string
	Path     string
	Reason   string
}

var actionTagPattern = regexp.MustCompile(`\{(?:go|csharp|c|cpp):`)

func main() {
	familyFlag := flag.String("family", "all", "example family to check: all or calc")
	flag.Parse()

	families := []exampleFamily{
		{
			Name: "calc",
			Specs: []string{
				"examples/go/calc/calc.lf",
				"examples/csharp/calc/calc.lf",
				"examples/c/calc/calc.lf",
				"examples/cpp/calc/calc.lf",
			},
		},
	}

	checked := 0
	for _, family := range families {
		if *familyFlag != "all" && *familyFlag != family.Name {
			continue
		}
		if err := checkFamily(family); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		checked++
	}
	if checked == 0 {
		fmt.Fprintf(os.Stderr, "unknown example family %q\n", *familyFlag)
		os.Exit(2)
	}
	fmt.Printf("example spec parity check passed for %d family set(s)\n", checked)
}

func checkFamily(family exampleFamily) error {
	var baselinePath string
	var baseline string
	for i, path := range family.Specs {
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: read %s: %w", family.Name, path, err)
		}
		normalized, err := normalizeSpec(string(body))
		if err != nil {
			return fmt.Errorf("%s: normalize %s: %w", family.Name, path, err)
		}
		if i == 0 {
			baselinePath = path
			baseline = normalized
			continue
		}
		if normalized != baseline {
			if reason, ok := allowedDifferenceReason(family, baselinePath, path); ok {
				fmt.Printf("%s parity allowed difference between %s and %s: %s\n", family.Name, baselinePath, path, reason)
				continue
			}
			line, want, got := firstDifference(baseline, normalized)
			return fmt.Errorf("%s spec parity mismatch between %s and %s at normalized line %d\nwant: %s\ngot:  %s", family.Name, baselinePath, path, line, want, got)
		}
	}
	fmt.Printf("%s parity ok (%d target specs)\n", family.Name, len(family.Specs))
	return nil
}

func allowedDifferenceReason(family exampleFamily, baseline string, path string) (string, bool) {
	for _, allowed := range family.AllowedDifferences {
		if allowed.Baseline == baseline && allowed.Path == path {
			if allowed.Reason == "" {
				return "no reason recorded", true
			}
			return allowed.Reason, true
		}
	}
	return "", false
}

func normalizeSpec(source string) (string, error) {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	lines := strings.Split(source, "\n")
	var directives []string
	var sections []string
	var current strings.Builder
	currentHeader := ""
	seenSection := false

	flushSection := func() {
		if currentHeader == "" {
			return
		}
		sections = append(sections, currentHeader+"\n"+current.String())
		current.Reset()
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if shouldSkipDirective(line) {
			continue
		}
		line = actionTagPattern.ReplaceAllString(line, "{ACTION:")
		if strings.HasPrefix(line, "%%") {
			flushSection()
			header, err := stripInsignificantWhitespace(line)
			if err != nil {
				return "", err
			}
			currentHeader = header
			seenSection = true
			continue
		}
		compact, err := stripInsignificantWhitespace(line)
		if err != nil {
			return "", err
		}
		if !seenSection && strings.HasPrefix(line, "%") {
			directives = append(directives, compact)
			continue
		}
		current.WriteString(compact)
	}
	flushSection()

	sort.Strings(directives)
	var out strings.Builder
	for _, directive := range directives {
		out.WriteString(directive)
		out.WriteByte('\n')
	}
	for _, section := range sections {
		out.WriteString(section)
		out.WriteByte('\n')
	}
	return out.String(), nil
}

func shouldSkipDirective(line string) bool {
	for _, prefix := range []string{"%target", "%package", "%semantic"} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func stripInsignificantWhitespace(line string) (string, error) {
	var out strings.Builder
	inString := false
	inClass := false
	escaped := false
	for _, r := range line {
		if inString {
			out.WriteRune(r)
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		if inClass {
			out.WriteRune(r)
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == ']' {
				inClass = false
			}
			continue
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case r == '"':
			inString = true
			out.WriteRune(r)
		case r == '[':
			inClass = true
			out.WriteRune(r)
		default:
			out.WriteRune(r)
		}
	}
	if inString {
		return "", errors.New("unterminated string literal")
	}
	if inClass {
		return "", errors.New("unterminated character class")
	}
	return out.String(), nil
}

func firstDifference(want string, got string) (int, string, string) {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	max := len(wantLines)
	if len(gotLines) > max {
		max = len(gotLines)
	}
	for i := 0; i < max; i++ {
		wantLine := "<missing>"
		gotLine := "<missing>"
		if i < len(wantLines) {
			wantLine = wantLines[i]
		}
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}
		if wantLine != gotLine {
			return i + 1, wantLine, gotLine
		}
	}
	return 0, "", ""
}
