package datakeeper

import "fmt"

// OpCode identifies one stack-machine instruction.
type OpCode string

const (
	OpPushConst    OpCode = "PUSH_CONST"
	OpPushRef      OpCode = "PUSH_REF"
	OpLoadRef      OpCode = "LOAD_REF"
	OpAssign       OpCode = "ASSIGN"
	OpReplace      OpCode = "REPLACE_SUBSTR"
	OpRunSQL       OpCode = "RUN_SQL"
	OpAddObject    OpCode = "ADD_OBJECT"
	OpRemoveObject OpCode = "REMOVE_OBJECT"
	OpRunObjectJob OpCode = "RUN_OBJECTS_JOB"
)

// Instruction is one compiled stack-machine instruction.
type Instruction struct {
	Op  OpCode
	Arg string
}

func (i Instruction) String() string {
	if i.Arg == "" {
		return string(i.Op)
	}
	return fmt.Sprintf("%s %q", i.Op, i.Arg)
}

// Executable is the stack-machine program produced from a Script AST.
type Executable struct {
	Parameters   []string
	Variables    []string
	Instructions []Instruction
}
