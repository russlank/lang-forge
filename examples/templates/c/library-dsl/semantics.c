#include "semantics.h"

#include <errno.h>
#include <limits.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static void set_error(library_dsl_error *error, const char *format, ...) {
    va_list args;
    if (error == NULL) {
        return;
    }
    va_start(args, format);
    vsnprintf(error->message, sizeof(error->message), format, args);
    va_end(args);
}

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

static dsl_semantic_context *semantic_context(void *user, library_dsl_error *error, const char *action) {
    dsl_semantic_context *context = (dsl_semantic_context *)user;
    if (context == NULL || context->memory == NULL) {
        set_error(error, "action %s has no active semantic context", action == NULL ? "<unknown>" : action);
        return NULL;
    }
    return context;
}

static library_dsl_value fail_alloc(library_dsl_error *error, const char *action) {
    set_error(error, "action %s could not allocate semantic value", action == NULL ? "<unknown>" : action);
    return NULL;
}

int dsl_semantic_context_init(dsl_semantic_context *context) {
    if (context == NULL) {
        return 0;
    }
    context->memory = dsl_allocator_create();
    context->message[0] = '\0';
    return context->memory != NULL;
}

void dsl_semantic_context_dispose(dsl_semantic_context *context) {
    if (context == NULL) {
        return;
    }
    dsl_allocator_destroy(context->memory);
    context->memory = NULL;
    context->message[0] = '\0';
}

void dsl_semantic_context_release_document(dsl_semantic_context *context) {
    if (context == NULL) {
        return;
    }
    context->memory = NULL;
}

static library_dsl_value reduce_document(const library_dsl_document_reduction *ctx, void *user, library_dsl_error *error) {
    /* Document : entries=Entries {c: document} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    dsl_document *document = NULL;
    if (state == NULL) {
        return NULL;
    }
    document = dsl_document_create(state->memory, ctx->entries);
    if (document == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return document;
}

static library_dsl_value reduce_entries(const library_dsl_entries_reduction *ctx, void *user, library_dsl_error *error) {
    /* Entries : head=Entry tail=EntriesTail {c: entries} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    dsl_entry_list *list = NULL;
    if (state == NULL) {
        return NULL;
    }
    list = dsl_entry_list_prepend(state->memory, ctx->head, ctx->tail);
    if (list == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return list;
}

static library_dsl_value reduce_entries_empty(const library_dsl_entries_empty_reduction *ctx, void *user, library_dsl_error *error) {
    /* Entries : %empty {c: entries.empty} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    dsl_entry_list *list = NULL;
    if (state == NULL) {
        return NULL;
    }
    list = dsl_entry_list_empty(state->memory);
    if (list == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return list;
}

static library_dsl_value reduce_entries_tail_more(const library_dsl_entries_tail_more_reduction *ctx, void *user, library_dsl_error *error) {
    /* EntriesTail : head=Entry tail=EntriesTail {c: entries.tail.more} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    dsl_entry_list *list = NULL;
    if (state == NULL) {
        return NULL;
    }
    list = dsl_entry_list_prepend(state->memory, ctx->head, ctx->tail);
    if (list == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return list;
}

static library_dsl_value reduce_entries_tail_empty(const library_dsl_entries_tail_empty_reduction *ctx, void *user, library_dsl_error *error) {
    /* EntriesTail : %empty {c: entries.tail.empty} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    dsl_entry_list *list = NULL;
    if (state == NULL) {
        return NULL;
    }
    list = dsl_entry_list_empty(state->memory);
    if (list == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return list;
}

static library_dsl_value reduce_entry_set(const library_dsl_entry_set_reduction *ctx, void *user, library_dsl_error *error) {
    /* Entry : Set name=Ident Assign value=Value Semi {c: entry.set} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    char *name = NULL;
    dsl_entry *entry = NULL;
    if (state == NULL) {
        return NULL;
    }
    name = lexeme_text(ctx->name);
    if (name == NULL) {
        set_error(error, "rule %d action %s label name is not available", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    entry = dsl_entry_set(state->memory, name, ctx->value);
    free(name);
    if (entry == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return entry;
}

static library_dsl_value reduce_entry_enable(const library_dsl_entry_enable_reduction *ctx, void *user, library_dsl_error *error) {
    /* Entry : Enable name=Ident Semi {c: entry.enable} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    char *name = NULL;
    dsl_entry *entry = NULL;
    if (state == NULL) {
        return NULL;
    }
    name = lexeme_text(ctx->name);
    if (name == NULL) {
        set_error(error, "rule %d action %s label name is not available", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    entry = dsl_entry_enable(state->memory, name, dsl_value_bool(state->memory, 1));
    free(name);
    if (entry == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return entry;
}

static library_dsl_value reduce_value_number(const library_dsl_value_number_reduction *ctx, void *user, library_dsl_error *error) {
    /* Value : token=Number {c: value.number} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    char *text = NULL;
    char *end = NULL;
    long value = 0;
    dsl_value *out = NULL;
    if (state == NULL) {
        return NULL;
    }
    text = lexeme_text(ctx->token);
    if (text == NULL) {
        set_error(error, "rule %d action %s label token is not available", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    errno = 0;
    value = strtol(text, &end, 10);
    if (errno == ERANGE || end == text || *end != '\0' || value < INT_MIN || value > INT_MAX) {
        set_error(error, "rule %d action %s label token value %s is not a valid int", ctx->reduction->rule, ctx->reduction->action, text);
        free(text);
        return NULL;
    }
    free(text);
    out = dsl_value_number(state->memory, (int)value);
    if (out == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return out;
}

static library_dsl_value reduce_value_string(const library_dsl_value_string_reduction *ctx, void *user, library_dsl_error *error) {
    /* Value : token=String {c: value.string} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    char *text = NULL;
    dsl_value *value = NULL;
    if (state == NULL) {
        return NULL;
    }
    text = unquote_lexeme(ctx->token);
    if (text == NULL) {
        set_error(error, "rule %d action %s label token is not a valid quoted string", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    value = dsl_value_string(state->memory, text);
    free(text);
    if (value == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return value;
}

static library_dsl_value reduce_value_ident(const library_dsl_value_ident_reduction *ctx, void *user, library_dsl_error *error) {
    /* Value : token=Ident {c: value.ident} */
    dsl_semantic_context *state = semantic_context(user, error, ctx->reduction->action);
    char *text = NULL;
    dsl_value *value = NULL;
    if (state == NULL) {
        return NULL;
    }
    text = lexeme_text(ctx->token);
    if (text == NULL) {
        set_error(error, "rule %d action %s label token is not available", ctx->reduction->rule, ctx->reduction->action);
        return NULL;
    }
    value = dsl_value_ident(state->memory, text);
    free(text);
    if (value == NULL) {
        return fail_alloc(error, ctx->reduction->action);
    }
    return value;
}

library_dsl_typed_reducer dsl_make_typed_reducer(dsl_semantic_context *context) {
    /*
     * Handler function pointers are static reducer wiring. The only per-parse
     * field is `user`, which points to the semantic allocator/context owned by
     * parser_facade.c.
     */
    static const library_dsl_typed_reducer reducer_template = {
        .document = reduce_document,
        .entries = reduce_entries,
        .entries_empty = reduce_entries_empty,
        .entries_tail_more = reduce_entries_tail_more,
        .entries_tail_empty = reduce_entries_tail_empty,
        .entry_set = reduce_entry_set,
        .entry_enable = reduce_entry_enable,
        .value_number = reduce_value_number,
        .value_string = reduce_value_string,
        .value_ident = reduce_value_ident,
    };
    library_dsl_typed_reducer reducer = reducer_template;
    reducer.user = context;
    return reducer;
}
