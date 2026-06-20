#ifndef LANGFORGE_EXAMPLES_C_DRAW_AST_H
#define LANGFORGE_EXAMPLES_C_DRAW_AST_H

#include "../common/demo.h"

#include <stddef.h>

/** Maximum accepted `repdraw` count. Keeps malformed scripts from creating an
 * accidental unbounded render loop during demos or CI runs. */
#define DRAW_MAX_REPDRAW_ITERATIONS 20000

/** RGB color value used by parsed style statements and the renderer. */
typedef struct draw_color {
    unsigned char r;
    unsigned char g;
    unsigned char b;
} draw_color;

/** Kinds of numeric expressions supported by DRAW. */
typedef enum draw_expr_kind {
    DRAW_EXPR_NUMBER,
    DRAW_EXPR_VARIABLE,
    DRAW_EXPR_UNARY,
    DRAW_EXPR_BINARY,
    DRAW_EXPR_CALL
} draw_expr_kind;

/** Numeric expression node built by generated parser reductions. */
typedef struct draw_expr {
    draw_expr_kind kind;
    double number;
    char *name;
    char op;
    struct draw_expr *left;
    struct draw_expr *right;
    struct draw_expr *arg;
} draw_expr;

/** Kinds of executable DRAW statements. */
typedef enum draw_statement_kind {
    DRAW_STMT_CANVAS,
    DRAW_STMT_BACKGROUND,
    DRAW_STMT_STROKE,
    DRAW_STMT_FILL,
    DRAW_STMT_WIDTH,
    DRAW_STMT_ASSIGN,
    DRAW_STMT_DEFINE_FIGURE,
    DRAW_STMT_DRAW,
    DRAW_STMT_REPDRAW,
    DRAW_STMT_PRIMITIVE
} draw_statement_kind;

typedef struct draw_statement draw_statement;
typedef struct draw_statement_node draw_statement_node;

/** Ordered statement list. Parser reductions prepend nodes while preserving
 * the grammar's left-to-right execution order. */
typedef struct draw_statement_list {
    draw_statement_node *head;
    draw_statement_node *tail;
    size_t count;
} draw_statement_list;

/** Linked-list node for statement sequences. */
struct draw_statement_node {
    draw_statement *statement;
    draw_statement_node *next;
};

/** Reusable block of figure-local statements. */
typedef struct draw_figure_block {
    draw_statement_list *statements;
} draw_figure_block;

/** Kinds of figure references accepted by `draw` and `repdraw`. */
typedef enum draw_figure_ref_kind {
    DRAW_FIGURE_NAMED,
    DRAW_FIGURE_INLINE
} draw_figure_ref_kind;

/** Reference to either a previously defined figure or an inline block. */
typedef struct draw_figure_ref {
    draw_figure_ref_kind kind;
    char *name;
    draw_figure_block *block;
} draw_figure_ref;

/** Executable statement node. Fields are shared across statement variants so
 * the C example stays compact without introducing a large tagged union. */
struct draw_statement {
    draw_statement_kind kind;
    char *name;
    char primitive[8];
    draw_color color;
    int enabled;
    draw_expr *exprs[4];
    size_t expr_count;
    draw_figure_block *figure;
    draw_figure_ref *target;
};

/** Root node for one parsed DRAW source file. */
typedef struct draw_program {
    draw_statement_list *statements;
} draw_program;

/** Per-parse allocation context.
 *
 * The generated parser is reentrant; this arena keeps all AST allocations
 * owned by the caller rather than hidden in global state.
 */
typedef struct draw_context {
    demo_arena arena;
} draw_context;

/** Initializes a DRAW allocation context before parsing. */
void draw_context_init(draw_context *ctx);

/** Releases all AST allocations owned by a context. */
void draw_context_free(draw_context *ctx);

#endif
