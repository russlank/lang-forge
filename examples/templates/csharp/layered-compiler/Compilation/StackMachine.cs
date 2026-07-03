namespace LangForge.Examples.Templates.LayeredCompiler.Compilation;

/// <summary>Executes the mock stack-machine code emitted by <see cref="MiniCompiler" />.</summary>
public static class StackMachine
{
    /// <summary>Runs instructions and returns the values printed by the program.</summary>
    public static IReadOnlyList<int> Execute(IReadOnlyList<Instruction> code)
    {
        var stack = new Stack<int>();
        var output = new List<int>();
        for (var pc = 0; pc < code.Count; pc++)
        {
            var instruction = code[pc];
            switch (instruction.Op)
            {
                case OpCode.Push:
                    stack.Push(instruction.Argument);
                    break;
                case OpCode.Add:
                    if (stack.Count < 2)
                    {
                        throw new InvalidOperationException($"pc {pc}: add needs two stack values");
                    }
                    var right = stack.Pop();
                    var left = stack.Pop();
                    stack.Push(left + right);
                    break;
                case OpCode.Print:
                    if (stack.Count < 1)
                    {
                        throw new InvalidOperationException($"pc {pc}: print needs one stack value");
                    }
                    output.Add(stack.Pop());
                    break;
                default:
                    throw new InvalidOperationException($"pc {pc}: unknown opcode {instruction.Op}");
            }
        }
        return output;
    }
}
