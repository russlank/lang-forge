#ifndef LIBRARY_DSL_SEMANTICS_H
#define LIBRARY_DSL_SEMANTICS_H

#include "ast.h"
#include "generated/parser.h"
#include "generated/parser_typed.h"

typedef struct dsl_semantic_context {
    /*
     * The context owns this allocator during parsing. Reducer handlers allocate
     * all semantic return values from it. The parser facade either destroys it
     * on failure or transfers it to the final dsl_document on success.
     */
    dsl_allocator *memory;
    char message[256];
} dsl_semantic_context;

/* Initializes per-parse semantic state. Returns zero if the allocator fails. */
int dsl_semantic_context_init(dsl_semantic_context *context);
/* Releases semantic allocations when parsing fails before a document is owned. */
void dsl_semantic_context_dispose(dsl_semantic_context *context);
/* Marks the allocator as transferred to the successful dsl_document result. */
void dsl_semantic_context_release_document(dsl_semantic_context *context);

/* Builds a complete generated typed reducer table for grammar.lf. */
library_dsl_typed_reducer dsl_make_typed_reducer(dsl_semantic_context *context);

#endif
