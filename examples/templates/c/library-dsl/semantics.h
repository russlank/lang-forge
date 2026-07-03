#ifndef LIBRARY_DSL_SEMANTICS_H
#define LIBRARY_DSL_SEMANTICS_H

#include "ast.h"
#include "generated/parser.h"
#include "generated/parser_typed.h"

typedef struct dsl_semantic_context {
    /* Reserved for domain services or diagnostics shared by reducer handlers. */
    char message[256];
} dsl_semantic_context;

/* Builds a complete generated typed reducer table for grammar.lf. */
library_dsl_typed_reducer dsl_make_typed_reducer(dsl_semantic_context *context);

#endif
