#pragma once

#include <cstddef>
#include <limits>
#include <memory>
#include <stdexcept>
#include <variant>
#include <vector>

namespace mini::ast {

/// Copyable semantic no-op returned by the final grammar action.
struct Unit {};

/// Stable handle to an expression node owned by Program.
struct ExprId {
    std::size_t index = invalid();

    static constexpr std::size_t invalid() noexcept {
        return std::numeric_limits<std::size_t>::max();
    }

    bool valid() const noexcept {
        return index != invalid();
    }
};

/// Expr node for: Term : token=Number {cpp: number}.
struct NumberExpr {
    int value = 0;
};

/// Expr node for: Expr : left=Expr Plus right=Term {cpp: add}.
struct AddExpr {
    ExprId left;
    ExprId right;
};

/// Variant keeps the AST closed and explicit without a base-class hierarchy.
using ExprNode = std::variant<NumberExpr, AddExpr>;

/// Owned expression node. Program stores these as std::unique_ptr<Expr>.
struct Expr {
    explicit Expr(ExprNode value);

    ExprNode node;
};

/// Statement node for: Statement : Print expr=Expr Semi {cpp: print}.
struct PrintStatement {
    ExprId expr;
};

using Statement = std::variant<PrintStatement>;

/// Move-only AST root returned by the parser facade.
struct Program {
    std::vector<std::unique_ptr<Expr>> expressions;
    std::vector<Statement> statements;

    Program() = default;
    Program(std::vector<std::unique_ptr<Expr>> expression_nodes, std::vector<Statement> statement_nodes);
    Program(Program&&) noexcept = default;
    Program& operator=(Program&&) noexcept = default;
    Program(const Program&) = delete;
    Program& operator=(const Program&) = delete;

    const Expr& expression(ExprId id) const;
};

/// Per-parse owner for AST nodes while reductions are still running.
class ProgramBuilder {
public:
    ExprId number(int value);
    ExprId add(ExprId left, ExprId right);
    Statement print(ExprId expr) const;
    Program finish(std::vector<Statement> statements);

private:
    ExprId store(ExprNode node);

    std::vector<std::unique_ptr<Expr>> expressions_;
};

} // namespace mini::ast
