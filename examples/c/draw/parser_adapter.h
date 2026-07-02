#ifndef LANGFORGE_EXAMPLES_C_DRAW_PARSER_ADAPTER_H
#define LANGFORGE_EXAMPLES_C_DRAW_PARSER_ADAPTER_H

#include "ast.h"

/** Selects which generated semantic reducer ABI the parser adapter uses. */
typedef enum draw_reducer_mode {
    DRAW_REDUCER_TYPED,
    DRAW_REDUCER_BOXED
} draw_reducer_mode;

/** Compiles DRAW source text into a typed AST.
 *
 * @param ctx caller-owned allocation context for the resulting AST
 * @param source UTF-8/ASCII source text accepted by the generated scanner
 * @param out receives the parsed program on success
 * @param message receives a human-readable scanner/parser/reducer error
 * @param message_size size of the message buffer
 * @return non-zero on success, zero on failure
 */
int draw_compile_source(draw_context *ctx, const char *source, draw_program **out, char *message, size_t message_size);

/** Scans and parses DRAW source text with an explicit reducer mode. */
int draw_compile_source_with_mode(draw_context *ctx, const char *source, draw_reducer_mode mode, draw_program **out, char *message, size_t message_size);

#endif
