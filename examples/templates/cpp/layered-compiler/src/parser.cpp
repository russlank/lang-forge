#include "mini/parser.hpp"

#include "generated/parser.hpp"
#include "generated/parser_typed.hpp"

#include <cerrno>
#include <climits>
#include <cstdlib>
#include <stdexcept>
#include <string>
#include <utility>

namespace lfgen = LangForge::Examples::Templates::LayeredCompiler::Generated;

namespace mini::ast {

Expr::Expr(ExprNode value) : node(std::move(value)) {}

Program::Program(std::vector<std::unique_ptr<Expr>> expression_nodes, std::vector<Statement> statement_nodes)
    : expressions(std::move(expression_nodes)), statements(std::move(statement_nodes)) {}

const Expr& Program::expression(ExprId id) const {
    if (!id.valid() || id.index >= expressions.size() || expressions[id.index] == nullptr) {
        throw std::out_of_range("invalid expression id");
    }
    return *expressions[id.index];
}

ExprId ProgramBuilder::number(int value) {
    return store(NumberExpr{value});
}

ExprId ProgramBuilder::add(ExprId left, ExprId right) {
    if (!left.valid() || !right.valid()) {
        throw std::runtime_error("add expression received an invalid operand handle");
    }
    return store(AddExpr{left, right});
}

Statement ProgramBuilder::print(ExprId expr) const {
    if (!expr.valid()) {
        throw std::runtime_error("print statement received an invalid expression handle");
    }
    return PrintStatement{expr};
}

Program ProgramBuilder::finish(std::vector<Statement> statements) {
    return Program{std::move(expressions_), std::move(statements)};
}

ExprId ProgramBuilder::store(ExprNode node) {
    const ExprId id{expressions_.size()};
    expressions_.push_back(std::make_unique<Expr>(std::move(node)));
    return id;
}

} // namespace mini::ast

namespace mini {

namespace {

struct ParseSession {
    ast::ProgramBuilder builder;
    std::optional<ast::Program> program;
};

std::vector<ast::Statement> prepend(ast::Statement head, std::vector<ast::Statement> tail) {
    std::vector<ast::Statement> out;
    out.reserve(tail.size() + 1);
    out.push_back(std::move(head));
    out.insert(out.end(), tail.begin(), tail.end());
    return out;
}

int parse_int_token(const lfgen::NumberReduction& ctx) {
    const std::string text(ctx.token.text);
    char* end = nullptr;
    errno = 0;
    const long value = std::strtol(text.c_str(), &end, 10);
    if (errno == ERANGE || end == text.c_str() || *end != '\0' || value < INT_MIN || value > INT_MAX) {
        throw std::runtime_error("rule " + std::to_string(ctx.reduction.rule) +
                                 " action " + std::string(ctx.reduction.action) +
                                 " label token value " + text +
                                 " is not a valid int");
    }
    return static_cast<int>(value);
}

lfgen::ReducerMap make_reducers(ParseSession& session) {
    return lfgen::ReducerMap{
        // Program : statements=Statements {cpp: program}
        {lfgen::SemanticAction::Program, lfgen::typed_program([&session](const lfgen::ProgramReduction& ctx) -> ast::Unit {
            session.program = session.builder.finish(ctx.statements);
            return {};
        })},

        // Statements : head=Statement tail=StatementsTail {cpp: statements}
        {lfgen::SemanticAction::Statements, lfgen::typed_statements([](const lfgen::StatementsReduction& ctx) -> std::vector<ast::Statement> {
            return prepend(ctx.head, ctx.tail);
        })},

        // StatementsTail : head=Statement tail=StatementsTail {cpp: statements.tail.more}
        {lfgen::SemanticAction::StatementsTailMore, lfgen::typed_statements_tail_more([](const lfgen::StatementsTailMoreReduction& ctx) -> std::vector<ast::Statement> {
            return prepend(ctx.head, ctx.tail);
        })},

        // StatementsTail : %empty {cpp: statements.tail.empty}
        {lfgen::SemanticAction::StatementsTailEmpty, lfgen::typed_statements_tail_empty([](const lfgen::StatementsTailEmptyReduction&) -> std::vector<ast::Statement> {
            return {};
        })},

        // Statement : Print expr=Expr Semi {cpp: print}
        {lfgen::SemanticAction::Print, lfgen::typed_print([&session](const lfgen::PrintReduction& ctx) -> ast::Statement {
            return session.builder.print(ctx.expr);
        })},

        // Expr : left=Expr Plus right=Term {cpp: add}
        {lfgen::SemanticAction::Add, lfgen::typed_add([&session](const lfgen::AddReduction& ctx) -> ast::ExprId {
            return session.builder.add(ctx.left, ctx.right);
        })},

        // Expr : value=Term {cpp: pass}
        {lfgen::SemanticAction::Pass, lfgen::typed_pass([](const lfgen::PassReduction& ctx) -> ast::ExprId {
            return ctx.value;
        })},

        // Term : token=Number {cpp: number}
        {lfgen::SemanticAction::Number, lfgen::typed_number([&session](const lfgen::NumberReduction& ctx) -> ast::ExprId {
            return session.builder.number(parse_int_token(ctx));
        })},
    };
}

} // namespace

Result<ast::Program> Parser::parse(std::string_view source) const {
    try {
        ParseSession session;
        lfgen::Scanner scanner(source);
        /*
         * This is the preferred source-based path. The generated parser pulls
         * tokens from scanner as needed; callers never materialize a token
         * vector unless they explicitly choose a debugging API.
         */
        (void)lfgen::parse_value(scanner, make_reducers(session));
        if (!session.program.has_value()) {
            return Result<ast::Program>::failure({"parser accepted without producing a program"});
        }
        return Result<ast::Program>::success(std::move(*session.program));
    } catch (const std::exception& ex) {
        return Result<ast::Program>::failure(diagnostic_from_exception(ex));
    }
}

} // namespace mini
