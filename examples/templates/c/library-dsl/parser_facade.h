#ifndef LIBRARY_DSL_PARSER_FACADE_H
#define LIBRARY_DSL_PARSER_FACADE_H

#include "ast.h"

#include <stddef.h>

typedef struct dsl_parse_result {
    /*
     * Owned by the caller only when accepted is non-zero. Release it through
     * dsl_parse_result_free, which calls dsl_document_free.
     */
    dsl_document *document;
    /*
     * Application-facing scanner, syntax, or reducer error text. The buffer is
     * embedded in the result, so callers never free it separately.
     */
    char message[512];
    int accepted;
} dsl_parse_result;

/* Initializes a stack- or heap-allocated parse result before use. */
void dsl_parse_result_init(dsl_parse_result *result);
/*
 * Frees a successful document and resets the result. It is safe to call on
 * initialized failed results and safe to call more than once.
 */
void dsl_parse_result_free(dsl_parse_result *result);
/*
 * Parses source through the generated scanner token source and typed reducer.
 *
 * Ownership rules:
 * - source remains caller-owned and must outlive the call;
 * - generated diagnostics are freed inside this facade before return;
 * - reducer-created partial AST values are freed automatically on failure;
 * - on success, result->document owns all AST nodes and copied token text;
 * - callers release successful results with dsl_parse_result_free.
 */
int dsl_parse_source(const char *source, dsl_parse_result *result);

#endif
