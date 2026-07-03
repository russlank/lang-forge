#ifndef LIBRARY_DSL_PARSER_FACADE_H
#define LIBRARY_DSL_PARSER_FACADE_H

#include "ast.h"

#include <stddef.h>

typedef struct dsl_parse_result {
    dsl_document *document;
    char message[512];
    int accepted;
} dsl_parse_result;

/* Initializes a parse result before use. */
void dsl_parse_result_init(dsl_parse_result *result);
/* Frees a successful document and resets the result. */
void dsl_parse_result_free(dsl_parse_result *result);
/* Parses source through the generated scanner token source and typed reducer. */
int dsl_parse_source(const char *source, dsl_parse_result *result);

#endif
