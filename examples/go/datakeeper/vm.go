package datakeeper

import (
	"fmt"
	"sort"
	"strings"
)

// ValueKind describes the type of one stack value.
type ValueKind string

const (
	ValueString    ValueKind = "string"
	ValueReference ValueKind = "reference"
)

// Value is one VM stack or variable value.
type Value struct {
	Kind ValueKind
	Text string
}

func (v Value) String() string {
	switch v.Kind {
	case ValueReference:
		return "&" + v.Text
	case ValueString:
		return fmt.Sprintf("%q", v.Text)
	default:
		return "<unset>"
	}
}

// Variable is one named VM storage cell.
type Variable struct {
	Name string
	Data Value
	Set  bool
}

// Adapter mocks the real DataKeeper runtime integration boundary.
type Adapter interface {
	RunSQL(instanceGuid, script string)
	AddObject(parentGuid, objectXML string)
	RemoveObject(parentGuid, name string)
	RunObjectsJob(parentGuid, name, jobsTag string)
	Log(kind, message string)
}

// AdapterCall records one mocked external operation.
type AdapterCall struct {
	Operation string
	Args      []string
}

// LogLine records one runtime log line.
type LogLine struct {
	Kind    string
	Message string
}

// MockAdapter records every operation instead of touching a real database.
type MockAdapter struct {
	Calls []AdapterCall
	Logs  []LogLine
}

func (m *MockAdapter) RunSQL(instanceGuid, script string) {
	m.Calls = append(m.Calls, AdapterCall{Operation: "RunSQL", Args: []string{instanceGuid, script}})
}

func (m *MockAdapter) AddObject(parentGuid, objectXML string) {
	m.Calls = append(m.Calls, AdapterCall{Operation: "AddObject", Args: []string{parentGuid, objectXML}})
}

func (m *MockAdapter) RemoveObject(parentGuid, name string) {
	m.Calls = append(m.Calls, AdapterCall{Operation: "RemoveObject", Args: []string{parentGuid, name}})
}

func (m *MockAdapter) RunObjectsJob(parentGuid, name, jobsTag string) {
	m.Calls = append(m.Calls, AdapterCall{Operation: "RunObjectsJob", Args: []string{parentGuid, name, jobsTag}})
}

func (m *MockAdapter) Log(kind, message string) {
	m.Logs = append(m.Logs, LogLine{Kind: kind, Message: message})
}

// TraceStep captures one instruction's before/after stack state.
type TraceStep struct {
	PC          int
	Instruction Instruction
	StackBefore []Value
	StackAfter  []Value
}

// RunResult is the complete mock execution result.
type RunResult struct {
	OK        bool
	Error     string
	Trace     []TraceStep
	Variables []Variable
	Adapter   *MockAdapter
}

// Run executes the compiled program with a MockAdapter.
func (e *Executable) Run(parameters map[string]string) *RunResult {
	adapter := &MockAdapter{}
	return e.RunWithAdapter(parameters, adapter)
}

// RunWithAdapter executes the compiled program with a caller-supplied adapter.
func (e *Executable) RunWithAdapter(parameters map[string]string, adapter Adapter) *RunResult {
	if adapter == nil {
		adapter = &MockAdapter{}
	}
	vm := runtimeVM{
		variables: map[string]Variable{},
		adapter:   adapter,
	}
	for _, name := range e.Variables {
		vm.variables[name] = Variable{Name: name}
	}
	for _, name := range e.Parameters {
		value, ok := parameters[name]
		if ok {
			vm.variables[name] = Variable{Name: name, Data: Value{Kind: ValueString, Text: value}, Set: true}
		}
	}
	if missing := vm.missingParameters(e.Parameters); len(missing) > 0 {
		message := fmt.Sprintf("found %d missing parameter(s): %s", len(missing), strings.Join(missing, ", "))
		adapter.Log("Error", message)
		return vm.result(false, message)
	}
	for pc, instruction := range e.Instructions {
		before := cloneValues(vm.stack)
		if err := vm.execute(instruction); err != nil {
			adapter.Log("Error", fmt.Sprintf("%07d - %s", pc+1, err))
			vm.trace = append(vm.trace, TraceStep{PC: pc, Instruction: instruction, StackBefore: before, StackAfter: cloneValues(vm.stack)})
			return vm.result(false, err.Error())
		}
		vm.trace = append(vm.trace, TraceStep{PC: pc, Instruction: instruction, StackBefore: before, StackAfter: cloneValues(vm.stack)})
	}
	return vm.result(true, "")
}

type runtimeVM struct {
	stack     []Value
	variables map[string]Variable
	adapter   Adapter
	trace     []TraceStep
}

func (vm *runtimeVM) execute(instruction Instruction) error {
	switch instruction.Op {
	case OpPushConst:
		vm.push(Value{Kind: ValueString, Text: instruction.Arg})
	case OpPushRef:
		vm.ensureVariable(instruction.Arg)
		vm.push(Value{Kind: ValueReference, Text: instruction.Arg})
	case OpLoadRef:
		ref, err := vm.popRef()
		if err != nil {
			return err
		}
		variable := vm.variables[ref]
		if !variable.Set {
			return fmt.Errorf("variable %q is unset", ref)
		}
		vm.push(variable.Data)
	case OpAssign:
		value, err := vm.popStringLike()
		if err != nil {
			return err
		}
		ref, err := vm.popRef()
		if err != nil {
			return err
		}
		vm.variables[ref] = Variable{Name: ref, Data: value, Set: true}
	case OpReplace:
		newValue, err := vm.popString()
		if err != nil {
			return err
		}
		oldValue, err := vm.popString()
		if err != nil {
			return err
		}
		ref, err := vm.popRef()
		if err != nil {
			return err
		}
		variable := vm.variables[ref]
		if !variable.Set {
			return fmt.Errorf("variable %q is unset", ref)
		}
		variable.Data = Value{Kind: ValueString, Text: strings.ReplaceAll(variable.Data.Text, oldValue, newValue)}
		variable.Set = true
		vm.variables[ref] = variable
	case OpRunSQL:
		script, err := vm.popString()
		if err != nil {
			return err
		}
		instance, err := vm.popString()
		if err != nil {
			return err
		}
		vm.adapter.RunSQL(instance, script)
	case OpAddObject:
		objectXML, err := vm.popString()
		if err != nil {
			return err
		}
		parent, err := vm.popString()
		if err != nil {
			return err
		}
		vm.adapter.AddObject(parent, objectXML)
	case OpRemoveObject:
		name, err := vm.popString()
		if err != nil {
			return err
		}
		parent, err := vm.popString()
		if err != nil {
			return err
		}
		vm.adapter.RemoveObject(parent, name)
	case OpRunObjectJob:
		jobsTag, err := vm.popString()
		if err != nil {
			return err
		}
		name, err := vm.popString()
		if err != nil {
			return err
		}
		parent, err := vm.popString()
		if err != nil {
			return err
		}
		vm.adapter.RunObjectsJob(parent, name, jobsTag)
	default:
		return fmt.Errorf("unknown opcode %q", instruction.Op)
	}
	return nil
}

func (vm *runtimeVM) missingParameters(parameters []string) []string {
	var missing []string
	for _, name := range parameters {
		if variable, ok := vm.variables[name]; !ok || !variable.Set {
			missing = append(missing, name)
		}
	}
	return missing
}

func (vm *runtimeVM) result(ok bool, err string) *RunResult {
	var adapter *MockAdapter
	if mock, isMock := vm.adapter.(*MockAdapter); isMock {
		adapter = mock
	}
	var variables []Variable
	for _, variable := range vm.variables {
		variables = append(variables, variable)
	}
	sort.Slice(variables, func(i, j int) bool { return variables[i].Name < variables[j].Name })
	return &RunResult{OK: ok, Error: err, Trace: vm.trace, Variables: variables, Adapter: adapter}
}

func (vm *runtimeVM) ensureVariable(name string) {
	if _, ok := vm.variables[name]; !ok {
		vm.variables[name] = Variable{Name: name}
	}
}

func (vm *runtimeVM) push(value Value) {
	vm.stack = append(vm.stack, value)
}

func (vm *runtimeVM) pop() (Value, error) {
	if len(vm.stack) == 0 {
		return Value{}, fmt.Errorf("stack underflow")
	}
	idx := len(vm.stack) - 1
	value := vm.stack[idx]
	vm.stack = vm.stack[:idx]
	return value, nil
}

func (vm *runtimeVM) popRef() (string, error) {
	value, err := vm.pop()
	if err != nil {
		return "", err
	}
	if value.Kind != ValueReference {
		return "", fmt.Errorf("expected reference, got %s", value)
	}
	return value.Text, nil
}

func (vm *runtimeVM) popString() (string, error) {
	value, err := vm.popStringLike()
	if err != nil {
		return "", err
	}
	return value.Text, nil
}

func (vm *runtimeVM) popStringLike() (Value, error) {
	value, err := vm.pop()
	if err != nil {
		return Value{}, err
	}
	if value.Kind != ValueString {
		return Value{}, fmt.Errorf("expected string, got %s", value)
	}
	return value, nil
}

func cloneValues(in []Value) []Value {
	return append([]Value(nil), in...)
}
