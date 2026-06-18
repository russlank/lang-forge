//go:build langforge_generated

package datakeeper

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestCompileAndRunSample(t *testing.T) {
	source := readSample(t)
	ast, executable, err := Compile(source)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ast.Parameters, []string{"InstanceGuid", "ParentGuid", "ObjectName", "JobsTag"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("parameters = %#v, want %#v", got, want)
	}
	if len(executable.Instructions) == 0 {
		t.Fatal("expected instructions")
	}
	first := executable.Instructions[0]
	if first.Op != OpPushRef || first.Arg != "scriptText" {
		t.Fatalf("first instruction = %#v", first)
	}
	result := executable.Run(map[string]string{
		"InstanceGuid": "sql-instance-01",
		"ParentGuid":   "farm-root",
		"ObjectName":   "daily-backup",
		"JobsTag":      "maintenance",
	})
	if !result.OK {
		t.Fatalf("run failed: %s", result.Error)
	}
	if result.Adapter == nil || len(result.Adapter.Calls) != 4 {
		t.Fatalf("adapter calls = %#v", result.Adapter)
	}
	if result.Adapter.Calls[0].Operation != "RunSQL" {
		t.Fatalf("first adapter call = %#v", result.Adapter.Calls[0])
	}
	if !strings.Contains(result.Adapter.Calls[0].Args[1], "daily-backup") {
		t.Fatalf("sql script did not receive replaced object name: %#v", result.Adapter.Calls[0].Args)
	}
	var report bytes.Buffer
	WriteReport(&report, "sample.dks", ast, executable, result)
	if !strings.Contains(report.String(), "Intermediate stack code") || !strings.Contains(report.String(), "Mock adapter calls") {
		t.Fatalf("report missing expected sections:\n%s", report.String())
	}
}

func TestParserHandlesCommentsTrailingSemicolonAndQuotedStrings(t *testing.T) {
	source := `parameters Name;
begin
  /* old Irony grammar supported block comments */
  greeting = "hello";
  replace(greeting, "hello", Name); // trailing semicolon is accepted
end`
	_, executable, err := Compile(source)
	if err != nil {
		t.Fatal(err)
	}
	result := executable.Run(map[string]string{"Name": "world"})
	if !result.OK {
		t.Fatalf("run failed: %s", result.Error)
	}
	if got := variableText(result, "greeting"); got != "world" {
		t.Fatalf("greeting = %q, want world", got)
	}
}

func TestRunReportsMissingParameters(t *testing.T) {
	_, executable, err := Compile(`parameters Name;
begin
  value = Name
end`)
	if err != nil {
		t.Fatal(err)
	}
	result := executable.Run(nil)
	if result.OK {
		t.Fatal("expected missing parameter failure")
	}
	if result.Adapter == nil || len(result.Adapter.Logs) != 1 || !strings.Contains(result.Adapter.Logs[0].Message, "Name") {
		t.Fatalf("logs = %#v", result.Adapter)
	}
}

func TestParseRejectsMalformedStatement(t *testing.T) {
	_, err := Parse(`begin
  replace("not-a-reference", "a", "b")
end`)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestRejectsUnterminatedBlockComment(t *testing.T) {
	_, err := Parse(`begin
  /* no end
  value = "x"
end`)
	if err == nil {
		t.Fatal("expected lexer error")
	}
}

func TestRejectsDuplicateParameters(t *testing.T) {
	_, _, err := Compile(`parameters Name, Name;
begin
  value = "x"
end`)
	if err == nil {
		t.Fatal("expected duplicate parameter error")
	}
}

func TestRejectsKeywordAsValueReference(t *testing.T) {
	_, err := Parse(`begin
  value = begin
end`)
	if err == nil {
		t.Fatal("expected keyword value error")
	}
}

func readSample(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("sample.dks")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func variableText(result *RunResult, name string) string {
	for _, variable := range result.Variables {
		if variable.Name == name && variable.Set {
			return variable.Data.Text
		}
	}
	return ""
}
