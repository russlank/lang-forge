#include "semantics.h"

#include <errno.h>
#include <limits.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static char *lexeme_text(const library_dsl_lexeme *lexeme) {
    char *text = NULL;
    if (lexeme == NULL) {
        return NULL;
    }
    text = (char *)calloc(lexeme->length + 1, 1);
    if (text != NULL) {
        memcpy(text, lexeme->text, lexeme->length);
    }
    return text;
}

static char *unquote_lexeme(const library_dsl_lexeme *lexeme) {
    char *quoted = lexeme_text(lexeme);
    char *out = NULL;
    size_t i = 0;
    size_t used = 0;
    size_t length = 0;
    if (quoted == NULL) {
        return NULL;
    }
    length = strlen(quoted);
    if (length < 2 || quoted[0] != '"' || quoted[length - 1] != '"') {
        free(quoted);
        return NULL;
    }
    out = (char *)calloc(length - 1, 1);
    if (out == NULL) {
        free(quoted);
        return NULL;
    }
    for (i = 1; i + 1 < length; i++) {
        if (quoted[i] == '\\') {
            i++;
            if (i + 1 >= length) {
                free(quoted);
                free(out);
                return NULL;
            }
        }
        out[used++] = quoted[i];
    }
    free(quoted);
    return out;
}

static library_dsl_value fail_alloc(library_dsl_error *error, const char *action) {
    if (error != NULL) {
        snprintf(error->message, sizeof(error->message), "action %s could not allocate semantic value", action);
    }
    return NULL;
}

static library_dsl_value reduce_document(const library_dsl_document_reduction *ctx, void *user, library_dsl_error *error) {
    /* Document : entries=Entries {c: document} */
    dsl_document *document = NULL;
    (void)user;
    document = dsl_document_create(ctx->entries);
    if (document == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return document;
}

static library_dsl_value reduce_entries(const library_dsl_entries_reduction *ctx, void *user, library_dsl_error *error) {
    /* Entries : head=Entry tail=EntriesTail {c: entries} */
    dsl_entry_list *list = NULL;
    (void)user;
    list = dsl_entry_list_prepend(ctx->head, ctx->tail);
    if (list == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return list;
}

static library_dsl_value reduce_entries_empty(const library_dsl_entries_empty_reduction *ctx, void *user, library_dsl_error *error) {
    /* Entries : %empty {c: entries.empty} */
    dsl_entry_list *list = NULL;
    (void)ctx;
    (void)user;
    list = dsl_entry_list_empty();
    if (list == NULL) {
        return fail_alloc(error, "entries.empty");
    }
    return list;
}

static library_dsl_value reduce_entries_tail_more(const library_dsl_entries_tail_more_reduction *ctx, void *user, library_dsl_error *error) {
    /* EntriesTail : head=Entry tail=EntriesTail {c: entries.tail.more} */
    dsl_entry_list *list = NULL;
    (void)user;
    list = dsl_entry_list_prepend(ctx->head, ctx->tail);
    if (list == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return list;
}

static library_dsl_value reduce_entries_tail_empty(const library_dsl_entries_tail_empty_reduction *ctx, void *user, library_dsl_error *error) {
    /* EntriesTail : %empty {c: entries.tail.empty} */
    dsl_entry_list *list = NULL;
    (void)ctx;
    (void)user;
    list = dsl_entry_list_empty();
    if (list == NULL) {
        return fail_alloc(error, "entries.tail.empty");
    }
    return list;
}

static library_dsl_value reduce_entry_set(const library_dsl_entry_set_reduction *ctx, void *user, library_dsl_error *error) {
    /* Entry : Set name=Ident Assign value=Value Semi {c: entry.set} */
    char *name = lexeme_text(ctx->name);
    dsl_entry *entry = NULL;
    (void)user;
    if (name == NULL) {
        snprintf(error->message, sizeof(error->message), "rule %d action %s label name is not available", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    entry = dsl_entry_set(name, ctx->value);
    free(name);
    if (entry == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return entry;
}

static library_dsl_value reduce_entry_enable(const library_dsl_entry_enable_reduction *ctx, void *user, library_dsl_error *error) {
    /* Entry : Enable name=Ident Semi {c: entry.enable} */
    char *name = lexeme_text(ctx->name);
    dsl_entry *entry = NULL;
    (void)user;
    if (name == NULL) {
        snprintf(error->message, sizeof(error->message), "rule %d action %s label name is not available", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    entry = dsl_entry_enable(name, dsl_value_bool(1));
    free(name);
    if (entry == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return entry;
}

static library_dsl_value reduce_value_number(const library_dsl_value_number_reduction *ctx, void *user, library_dsl_error *error) {
    /* Value : token=Number {c: value.number} */
    char *text = lexeme_text(ctx->token);
    char *end = NULL;
    long value = 0;
    (void)user;
    if (text == NULL) {
        snprintf(error->message, sizeof(error->message), "rule %d action %s label token is not available", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    errno = 0;
    value = strtol(text, &end, 10);
    if (errno == ERANGE || end == text || *end != '\0' || value < INT_MIN || value > INT_MAX) {
        snprintf(error->message, sizeof(error->message), "rule %d action %s label token value %s is not a valid int", ctx->reduction->rule, ctx->reduction->action, text);
        free(text);
        return NULL;
    }
    free(text);
    {
        dsl_value *out = dsl_value_number((int)value);
        if (out == NULL) {
            return fail_alloc(error, ctx->reduction->action);
        }
        return out;
    }
}

static library_dsl_value reduce_value_string(const library_dsl_value_string_reduction *ctx, void *user, library_dsl_error *error) {
    /* Value : token=String {c: value.string} */
    char *text = unquote_lexeme(ctx->token);
    dsl_value *value = NULL;
    (void)user;
    if (text == NULL) {
        snprintf(error->message, sizeof(error->message), "rule %d action %s label token is not a valid quoted string", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    value = dsl_value_string(text);
    free(text);
    if (value == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return value;
}

static library_dsl_value reduce_value_ident(const library_dsl_value_ident_reduction *ctx, void *user, library_dsl_error *error) {
    /* Value : token=Ident {c: value.ident} */
    char *text = lexeme_text(ctx->token);
    dsl_value *value = NULL;
    (void)user;
    if (text == NULL) {
        snprintf(error->message, sizeof(error->message), "rule %d action %s label token is not available", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    value = dsl_value_ident(text);
    free(text);
    if (value == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return value;
}

library_dsl_typed_reducer dsl_make_typed_reducer(dsl_semantic_context *context) {
    library_dsl_typed_reducer reducer;
    reducer.user = context;
    reducer.document = reduce_document;
    reducer.entries = reduce_entries;
    reducer.entries_empty = reduce_entries_empty;
    reducer.entries_tail_more = reduce_entries_tail_more;
    reducer.entries_tail_empty = reduce_entries_tail_empty;
    reducer.entry_set = reduce_entry_set;
    reducer.entry_enable = reduce_entry_enable;
    reducer.value_number = reduce_value_number;
    reducer.value_string = reduce_value_string;
    reducer.value_ident = reduce_value_ident;
    return reducer;
}
