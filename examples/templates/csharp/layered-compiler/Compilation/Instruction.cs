namespace LangForge.Examples.Templates.LayeredCompiler.Compilation;

/// <summary>Operation codes for the tiny stack machine used by the demo compiler.</summary>
public enum OpCode
{
    /// <summary>Pushes <see cref="Instruction.Argument" /> onto the stack.</summary>
    Push,

    /// <summary>Pops two values and pushes their sum.</summary>
    Add,

    /// <summary>Pops one value and appends it to the program output.</summary>
    Print,
}

/// <summary>One stack-machine instruction emitted by the compiler layer.</summary>
/// <param name="Op">Operation to execute.</param>
/// <param name="Argument">Integer argument used by <see cref="OpCode.Push" />.</param>
public sealed record Instruction(OpCode Op, int Argument = 0);
