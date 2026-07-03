#ifndef LIBRARY_DSL_AST_H
#define LIBRARY_DSL_AST_H

#include <stddef.h>

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
    dsl_entry *entries;
} dsl_document;

/* Creates a DSL value from: Value : token=Number. */
dsl_value *dsl_value_number(int number);
/* Creates a DSL value from: Value : token=String. The text is copied. */
dsl_value *dsl_value_string(const char *text);
/* Creates a DSL value from: Value : token=Ident. The text is copied. */
dsl_value *dsl_value_ident(const char *text);
/* Creates the implicit value used by: Entry : Enable name=Ident Semi. */
dsl_value *dsl_value_bool(int value);
/* Releases a value allocated by this module. */
void dsl_value_free(dsl_value *value);

/* Creates an assignment entry from: Entry : Set name=Ident Assign value=Value Semi. */
dsl_entry *dsl_entry_set(const char *name, dsl_value *value);
/* Creates a flag entry from: Entry : Enable name=Ident Semi. */
dsl_entry *dsl_entry_enable(const char *name, dsl_value *value);
/* Releases a linked list of entries and their owned values. */
void dsl_entry_free_all(dsl_entry *entry);

/* Creates an empty list for %empty list reductions. */
dsl_entry_list *dsl_entry_list_empty(void);
/* Prepends one entry to a generated list tail. */
dsl_entry_list *dsl_entry_list_prepend(dsl_entry *head, dsl_entry_list *tail);
/* Releases only the list wrapper after ownership moved to a document. */
void dsl_entry_list_free_shallow(dsl_entry_list *list);

/* Creates the root AST from: Document : entries=Entries. */
dsl_document *dsl_document_create(dsl_entry_list *entries);
/* Releases a document returned by dsl_parse_source. */
void dsl_document_free(dsl_document *document);

/* Returns a stable lower-case name for reports. */
const char *dsl_entry_kind_name(dsl_entry_kind kind);
/* Formats a value for a demo report. */
void dsl_value_format(const dsl_value *value, char *buffer, size_t size);

#endif
