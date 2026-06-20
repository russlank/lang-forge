#ifndef LANGFORGE_EXAMPLES_C_DRAW_PARSER_ADAPTER_H
#define LANGFORGE_EXAMPLES_C_DRAW_PARSER_ADAPTER_H

#include "ast.h"

/** Scans and parses DRAW source text into a typed AST.
 *
 * @param ctx caller-owned allocation context for the resulting AST
 * @param source UTF-8/ASCII source text accepted by the generated scanner
 * @param out receives the parsed program on success
 * @param message receives a human-readable scanner/parser/reducer error
 * @param message_size size of the message buffer
 * @return non-zero on success, zero on failure
 */
int draw_parse_source(draw_context *ctx, const char *source, draw_program **out, char *message, size_t message_size);

#endif
