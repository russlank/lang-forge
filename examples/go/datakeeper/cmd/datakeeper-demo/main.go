//go:build langforge_generated

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/russlank/lang-forge/examples/go/datakeeper"
)

func main() {
	var params paramFlags
	scriptPath := flag.String("script", "examples/go/datakeeper/sample.dks", "script file to compile and run")
	logPath := flag.String("log", "", "optional report file")
	flag.Var(&params, "param", "parameter value as NAME=value; may be repeated")
	flag.Parse()

	source, err := os.ReadFile(*scriptPath)
	if err != nil {
		exitf("read script: %v", err)
	}
	ast, executable, err := datakeeper.Compile(string(source))
	if err != nil {
		exitf("compile: %v", err)
	}
	values := params.values()
	for _, name := range ast.Parameters {
		if _, ok := values[name]; !ok {
			values[name] = "demo-" + strings.ToLower(name)
		}
	}
	result := executable.Run(values)
	var report bytes.Buffer
	datakeeper.WriteReport(&report, *scriptPath, ast, executable, result)
	fmt.Print(report.String())
	if *logPath != "" {
		if err := os.WriteFile(*logPath, report.Bytes(), 0o644); err != nil {
			exitf("write log: %v", err)
		}
		fmt.Fprintf(os.Stderr, "wrote report to %s\n", *logPath)
	}
	if !result.OK {
		os.Exit(1)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

type paramFlags []string

func (p *paramFlags) String() string {
	return strings.Join(*p, ",")
}

func (p *paramFlags) Set(value string) error {
	if !strings.Contains(value, "=") {
		return fmt.Errorf("parameter must be NAME=value")
	}
	*p = append(*p, value)
	return nil
}

func (p *paramFlags) values() map[string]string {
	out := map[string]string{}
	for _, raw := range *p {
		parts := strings.SplitN(raw, "=", 2)
		out[parts[0]] = parts[1]
	}
	return out
}
