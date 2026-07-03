namespace LangForge.Examples.Templates.LayeredCompiler.Ast;

/// <summary>Root AST node returned by the public parser facade.</summary>
/// <param name="Statements">Top-level statements in source order.</param>
public sealed record ProgramNode(IReadOnlyList<StatementNode> Statements);

/// <summary>Base type for statement nodes in the mini language.</summary>
public abstract record StatementNode;

/// <summary>Statement node for grammar rule: Statement : Print expr=Expr Semi {csharp: print}.</summary>
/// <param name="Expression">Expression whose value is printed at runtime.</param>
public sealed record PrintStatementNode(ExprNode Expression) : StatementNode;

/// <summary>Base type for expression nodes in the mini language.</summary>
public abstract record ExprNode;

/// <summary>Expression node for grammar rule: Term : token=Number {csharp: number}.</summary>
/// <param name="Value">Parsed integer literal value.</param>
public sealed record NumberExprNode(int Value) : ExprNode;

/// <summary>Expression node for grammar rule: Expr : left=Expr Plus right=Term {csharp: add}.</summary>
/// <param name="Left">Left operand expression.</param>
/// <param name="Right">Right operand expression.</param>
public sealed record AddExprNode(ExprNode Left, ExprNode Right) : ExprNode;
