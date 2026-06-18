package datakeeper

import (
	"fmt"
	"sort"
)

// CompileScript compiles an already parsed script to stack-machine code.
func CompileScript(script *Script) (*Executable, error) {
	c := compiler{variables: map[string]bool{}}
	seenParameters := map[string]bool{}
	for _, parameter := range script.Parameters {
		if seenParameters[parameter] {
			return nil, fmt.Errorf("duplicate parameter %q", parameter)
		}
		seenParameters[parameter] = true
		c.addVariable(parameter)
	}
	for _, stmt := range script.Statements {
		if err := c.compileStatement(stmt); err != nil {
			return nil, err
		}
	}
	return &Executable{
		Parameters:   append([]string(nil), script.Parameters...),
		Variables:    c.sortedVariables(),
		Instructions: append([]Instruction(nil), c.instructions...),
	}, nil
}

type compiler struct {
	instructions []Instruction
	variables    map[string]bool
}

func (c *compiler) compileStatement(stmt Statement) error {
	switch s := stmt.(type) {
	case AssignStatement:
		c.compileReference(s.Name)
		c.compileValue(s.Value)
		c.emit(OpAssign, "")
	case ReplaceStatement:
		c.compileReference(s.Target)
		c.compileValue(s.Old)
		c.compileValue(s.New)
		c.emit(OpReplace, "")
	case RunSQLStatement:
		c.compileValue(s.Instance)
		c.compileValue(s.Script)
		c.emit(OpRunSQL, "")
	case AddObjectStatement:
		c.compileValue(s.Parent)
		c.compileValue(s.XML)
		c.emit(OpAddObject, "")
	case RemoveObjectStatement:
		c.compileValue(s.Parent)
		c.compileValue(s.Name)
		c.emit(OpRemoveObject, "")
	case RunObjectsJobStatement:
		c.compileValue(s.Parent)
		c.compileValue(s.Name)
		c.compileValue(s.JobsTag)
		c.emit(OpRunObjectJob, "")
	default:
		return fmt.Errorf("unsupported statement %T", stmt)
	}
	return nil
}

func (c *compiler) compileValue(value ValueExpr) {
	switch v := value.(type) {
	case StringLiteral:
		c.emit(OpPushConst, v.Value)
	case NumberLiteral:
		// The original C# IntConstNode pushes StringValue, so keep numbers as
		// strings in this compatibility VM.
		c.emit(OpPushConst, v.Value)
	case ReferenceExpr:
		c.compileReference(v.Name)
		c.emit(OpLoadRef, "")
	}
}

func (c *compiler) compileReference(name string) {
	c.addVariable(name)
	c.emit(OpPushRef, name)
}

func (c *compiler) emit(op OpCode, arg string) {
	c.instructions = append(c.instructions, Instruction{Op: op, Arg: arg})
}

func (c *compiler) addVariable(name string) {
	c.variables[name] = true
}

func (c *compiler) sortedVariables() []string {
	out := make([]string, 0, len(c.variables))
	for name := range c.variables {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
