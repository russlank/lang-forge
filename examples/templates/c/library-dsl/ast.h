#ifndef LIBRARY_DSL_AST_H
#define LIBRARY_DSL_AST_H

#include <stddef.h>

/*
 * The C template uses one allocator per parse. Reducers allocate every AST
 * object and copied token string from that allocator. On parse failure, the
 * parser facade releases the allocator. On parse success, ownership of the
 * allocator moves to the returned dsl_document and dsl_document_free releases
 * the complete tree.
 */
typedef struct dsl_allocator dsl_allocator;

typedef enum dsl_entry_kind {
    DSL_ENTRY_SET,
    DSL_ENTRY_ENABLE
} dsl_entry_kind;

typedef enum dsl_value_kind {
    DSL_VALUE_NUMBER,
    DSL_VALUE_STRING,
    DSL_VALUE_IDENT,
    DSL_VALUE_BOOL
} dsl_value_kind;

typedef struct dsl_value {
    dsl_value_kind kind;
    char *text;
    int number;
    int boolean;
} dsl_value;

typedef struct dsl_entry {
    dsl_entry_kind kind;
    char *name;
    dsl_value *value;
    struct dsl_entry *next;
} dsl_entry;

typedef struct dsl_entry_list {
    dsl_entry *head;
    dsl_entry *tail;
} dsl_entry_list;

typedef struct dsl_document {
    dsl_allocator *memory;
    dsl_entry *entries;
} dsl_document;

/* Creates the per-parse allocator used by reducers and documents. */
dsl_allocator *dsl_allocator_create(void);
/* Releases all objects and strings allocated from the allocator. */
void dsl_allocator_destroy(dsl_allocator *memory);
/* Allocates zeroed memory owned by the allocator. */
void *dsl_allocator_alloc(dsl_allocator *memory, size_t size);
/* Copies text into memory owned by the allocator. */
char *dsl_allocator_copy(dsl_allocator *memory, const char *text);
/* Copies a byte span and appends a terminating NUL byte. */
char *dsl_allocator_copy_span(dsl_allocator *memory, const char *text, size_t length);

/* Creates a DSL value from: Value : token=Number. */
dsl_value *dsl_value_number(dsl_allocator *memory, int number);
/* Creates a DSL value from: Value : token=String. The text is copied. */
dsl_value *dsl_value_string(dsl_allocator *memory, const char *text);
/* Creates a DSL value from: Value : token=Ident. The text is copied. */
dsl_value *dsl_value_ident(dsl_allocator *memory, const char *text);
/* Creates the implicit value used by: Entry : Enable name=Ident Semi. */
dsl_value *dsl_value_bool(dsl_allocator *memory, int value);

/* Creates an assignment entry from: Entry : Set name=Ident Assign value=Value Semi. */
dsl_entry *dsl_entry_set(dsl_allocator *memory, const char *name, dsl_value *value);
/* Creates a flag entry from: Entry : Enable name=Ident Semi. */
dsl_entry *dsl_entry_enable(dsl_allocator *memory, const char *name, dsl_value *value);

/* Creates an empty list for %empty list reductions. */
dsl_entry_list *dsl_entry_list_empty(dsl_allocator *memory);
/* Prepends one entry to a generated list tail. */
dsl_entry_list *dsl_entry_list_prepend(dsl_allocator *memory, dsl_entry *head, dsl_entry_list *tail);

/*
 * Creates the root AST from: Document : entries=Entries. The returned document
 * owns memory and must be released with dsl_document_free by the caller.
 */
dsl_document *dsl_document_create(dsl_allocator *memory, dsl_entry_list *entries);
/* Releases a document returned by dsl_parse_source, including all child nodes. */
void dsl_document_free(dsl_document *document);

/* Returns a stable lower-case name for reports. */
const char *dsl_entry_kind_name(dsl_entry_kind kind);
/* Formats a value for a demo report. */
void dsl_value_format(const dsl_value *value, char *buffer, size_t size);

#endif
