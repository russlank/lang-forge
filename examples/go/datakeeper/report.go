package datakeeper

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// WriteReport writes a human-readable end-to-end compiler and execution report.
func WriteReport(w io.Writer, sourceName string, script *Script, executable *Executable, result *RunResult) {
	fmt.Fprintf(w, "DataKeeper script demo: %s\n", sourceName)
	fmt.Fprintln(w, strings.Repeat("=", 72))
	fmt.Fprintln(w)
	writeParameters(w, script)
	writeInstructions(w, executable)
	writeTrace(w, result)
	writeAdapterCalls(w, result)
	writeVariables(w, result)
	if result.OK {
		fmt.Fprintln(w, "Result: ok")
	} else {
		fmt.Fprintf(w, "Result: failed: %s\n", result.Error)
	}
}

func writeParameters(w io.Writer, script *Script) {
	fmt.Fprintln(w, "Required parameters")
	if len(script.Parameters) == 0 {
		fmt.Fprintln(w, "  none")
		fmt.Fprintln(w)
		return
	}
	for _, parameter := range script.Parameters {
		fmt.Fprintf(w, "  - %s\n", parameter)
	}
	fmt.Fprintln(w)
}

func writeInstructions(w io.Writer, executable *Executable) {
	fmt.Fprintln(w, "Intermediate stack code")
	for i, instruction := range executable.Instructions {
		fmt.Fprintf(w, "  %03d  %s\n", i, instruction)
	}
	fmt.Fprintln(w)
}

func writeTrace(w io.Writer, result *RunResult) {
	fmt.Fprintln(w, "Execution trace")
	if len(result.Trace) == 0 {
		fmt.Fprintln(w, "  no instructions executed")
		fmt.Fprintln(w)
		return
	}
	for _, step := range result.Trace {
		fmt.Fprintf(w, "  %03d  %-32s  stack: %-36s -> %s\n", step.PC, step.Instruction, stackString(step.StackBefore), stackString(step.StackAfter))
	}
	fmt.Fprintln(w)
}

func writeAdapterCalls(w io.Writer, result *RunResult) {
	fmt.Fprintln(w, "Mock adapter calls")
	if result.Adapter == nil || len(result.Adapter.Calls) == 0 {
		fmt.Fprintln(w, "  none")
	} else {
		for _, call := range result.Adapter.Calls {
			fmt.Fprintf(w, "  - %s(%s)\n", call.Operation, quotedArgs(call.Args))
		}
	}
	if result.Adapter != nil && len(result.Adapter.Logs) > 0 {
		fmt.Fprintln(w, "Runtime logs")
		for _, line := range result.Adapter.Logs {
			fmt.Fprintf(w, "  - %s: %s\n", line.Kind, line.Message)
		}
	}
	fmt.Fprintln(w)
}

func writeVariables(w io.Writer, result *RunResult) {
	fmt.Fprintln(w, "Final variables")
	var variables []Variable
	variables = append(variables, result.Variables...)
	sort.Slice(variables, func(i, j int) bool { return variables[i].Name < variables[j].Name })
	for _, variable := range variables {
		if variable.Set {
			fmt.Fprintf(w, "  - %s = %s\n", variable.Name, variable.Data)
		} else {
			fmt.Fprintf(w, "  - %s = <unset>\n", variable.Name)
		}
	}
	fmt.Fprintln(w)
}

func stackString(values []Value) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, value.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func quotedArgs(args []string) string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		out = append(out, fmt.Sprintf("%q", arg))
	}
	return strings.Join(out, ", ")
}
