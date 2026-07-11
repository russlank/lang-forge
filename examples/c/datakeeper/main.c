#include "../common/demo.h"
#include "generated/parser.h"
#include "generated/parser_typed.h"

#include <stdio.h>
#include <string.h>

typedef struct dks_demo {
    demo_arena arena;
    demo_text report;
    int parameters;
    int commands;
} dks_demo;

typedef enum dks_reducer_mode {
    DKS_REDUCER_TYPED,
    DKS_REDUCER_BOXED
} dks_reducer_mode;

static char *dks_copy_lexeme(dks_demo *demo, const datakeeper_lexeme *lexeme) {
    return demo_arena_copy(&demo->arena, lexeme->text, lexeme->length);
}

static const datakeeper_lexeme *dks_lexeme(datakeeper_value value) {
    return (const datakeeper_lexeme *)value;
}

static datakeeper_value dks_arg(const datakeeper_reduction *ctx, size_t index, const char *name, datakeeper_error *error) {
    /*
     * C generated reducers expose boxed values as void pointers. Keep all
     * positional access in this helper and pass the grammar role as name so
     * diagnostics remain close to labels such as parent=Value or jobsTag=Value.
     */
    if (index >= ctx->rhs_count || ctx->values[index] == NULL) {
        snprintf(error->message, sizeof(error->message), "rule %d missing %s at argument %zu", ctx->rule, name, index + 1);
        return NULL;
    }
    return ctx->values[index];
}

static const datakeeper_lexeme *dks_lexeme_arg(const datakeeper_reduction *ctx, size_t index, const char *name, datakeeper_error *error) {
    datakeeper_value value = dks_arg(ctx, index, name, error);
    return value == NULL ? NULL : dks_lexeme(value);
}

static const char *dks_string_arg(const datakeeper_reduction *ctx, size_t index, const char *name, datakeeper_error *error) {
    datakeeper_value value = dks_arg(ctx, index, name, error);
    return value == NULL ? NULL : (const char *)value;
}

static char *dks_decode_literal(dks_demo *demo, const datakeeper_lexeme *lexeme) {
    const char *text = lexeme->text;
    size_t length = lexeme->length;
    char *out = NULL;
    size_t i = 0;
    size_t j = 0;
    if (length >= 4 && text[0] == '#' && text[1] == '{' && text[length - 2] == '#' && text[length - 1] == '}') {
        out = (char *)demo_arena_alloc(&demo->arena, length - 2);
        if (out == NULL) {
            return NULL;
        }
        for (i = 2; i + 2 < length; i++) {
            if (text[i] == '#' && i + 1 < length && text[i + 1] == '#') {
                out[j++] = '#';
                i++;
            } else {
                out[j++] = text[i];
            }
        }
        out[j] = '\0';
        return out;
    }
    if (length >= 2 && text[0] == '"' && text[length - 1] == '"') {
        out = (char *)demo_arena_alloc(&demo->arena, length);
        if (out == NULL) {
            return NULL;
        }
        for (i = 1; i + 1 < length; i++) {
            if (text[i] == '\\' && i + 1 < length - 1) {
                i++;
            }
            out[j++] = text[i];
        }
        out[j] = '\0';
        return out;
    }
    return dks_copy_lexeme(demo, lexeme);
}

static char *dks_ident_value(dks_demo *demo, const datakeeper_lexeme *lexeme) {
    char *name = dks_copy_lexeme(demo, lexeme);
    char *value = NULL;
    if (name == NULL) {
        return NULL;
    }
    value = (char *)demo_arena_alloc(&demo->arena, strlen(name) + 2);
    if (value == NULL) {
        return NULL;
    }
    value[0] = '$';
    strcpy(value + 1, name);
    return value;
}

static int dks_append_parameter(dks_demo *demo, const char *name, datakeeper_error *error) {
    demo->parameters++;
    if (!demo_text_appendf(&demo->report, error->message, sizeof(error->message), "  param %-2d %s\n", demo->parameters, name)) {
        return 0;
    }
    return 1;
}

static int dks_append_command(dks_demo *demo, datakeeper_error *error, const char *kind, const char *a, const char *b, const char *c) {
    demo->commands++;
    if (!demo_text_appendf(&demo->report, error->message, sizeof(error->message), "  %02d %-14s", demo->commands, kind)) {
        return 0;
    }
    if (a != NULL && !demo_text_appendf(&demo->report, error->message, sizeof(error->message), " %s", a)) {
        return 0;
    }
    if (b != NULL && !demo_text_appendf(&demo->report, error->message, sizeof(error->message), " | %s", b)) {
        return 0;
    }
    if (c != NULL && !demo_text_appendf(&demo->report, error->message, sizeof(error->message), " | %s", c)) {
        return 0;
    }
    return demo_text_append(&demo->report, "\n", error->message, sizeof(error->message));
}

static datakeeper_value dks_default_reduce(const datakeeper_reduction *ctx) {
    if (ctx->rhs_count == 1) {
        return ctx->values[0];
    }
    return NULL;
}

static datakeeper_value dks_reduce(const datakeeper_reduction *ctx, void *user, datakeeper_error *error) {
    dks_demo *demo = (dks_demo *)user;
    /*
     * action_id values are generated from {c: ...} labels in datakeeper.lf.
     * This switch is handwritten semantics: it records mock instructions and
     * returns only the intermediate values later reductions need.
     */
    switch (ctx->action_id) {
    case DATAKEEPER_ACTION_PARAMETERS_DECL: {
        const datakeeper_lexeme *parameter_name = dks_lexeme_arg(ctx, 0, "parameter name", error);
        char *name = parameter_name == NULL ? NULL : dks_copy_lexeme(demo, parameter_name);
        if (name == NULL || !dks_append_parameter(demo, name, error)) {
            snprintf(error->message, sizeof(error->message), "out of memory collecting parameter");
        }
        return NULL;
    }
    case DATAKEEPER_ACTION_PARAMETERS_TAIL_MORE: {
        const datakeeper_lexeme *parameter_name = dks_lexeme_arg(ctx, 1, "parameter name", error);
        char *name = parameter_name == NULL ? NULL : dks_copy_lexeme(demo, parameter_name);
        if (name == NULL || !dks_append_parameter(demo, name, error)) {
            snprintf(error->message, sizeof(error->message), "out of memory collecting parameter");
        }
        return NULL;
    }
    case DATAKEEPER_ACTION_VALUE_STRING:
    {
        const datakeeper_lexeme *literal = dks_lexeme_arg(ctx, 0, "string literal", error);
        char *value = literal == NULL ? NULL : dks_decode_literal(demo, literal);
        if (value == NULL) {
            snprintf(error->message, sizeof(error->message), "out of memory decoding string literal");
        }
        return value;
    }
    case DATAKEEPER_ACTION_VALUE_NUMBER:
    {
        const datakeeper_lexeme *literal = dks_lexeme_arg(ctx, 0, "number literal", error);
        char *value = literal == NULL ? NULL : dks_copy_lexeme(demo, literal);
        if (value == NULL) {
            snprintf(error->message, sizeof(error->message), "out of memory copying number literal");
        }
        return value;
    }
    case DATAKEEPER_ACTION_VALUE_IDENT:
    {
        const datakeeper_lexeme *identifier = dks_lexeme_arg(ctx, 0, "identifier value", error);
        char *value = identifier == NULL ? NULL : dks_ident_value(demo, identifier);
        if (value == NULL) {
            snprintf(error->message, sizeof(error->message), "out of memory copying identifier value");
        }
        return value;
    }
    case DATAKEEPER_ACTION_ASSIGN: {
        const datakeeper_lexeme *assignment_name = dks_lexeme_arg(ctx, 0, "assignment name", error);
        const char *assignment_value = dks_string_arg(ctx, 2, "assignment value", error);
        char *name = assignment_name == NULL ? NULL : dks_copy_lexeme(demo, assignment_name);
        if (name == NULL || assignment_value == NULL || !dks_append_command(demo, error, "assign", name, assignment_value, NULL)) {
            snprintf(error->message, sizeof(error->message), "out of memory collecting assignment");
        }
        return NULL;
    }
    case DATAKEEPER_ACTION_REPLACE: {
        const datakeeper_lexeme *target = dks_lexeme_arg(ctx, 2, "replace target", error);
        dks_append_command(demo, error, "replace",
                           target == NULL ? NULL : dks_copy_lexeme(demo, target),
                           dks_string_arg(ctx, 4, "old value", error),
                           dks_string_arg(ctx, 6, "new value", error));
        return NULL;
    }
    case DATAKEEPER_ACTION_SQLRUN:
        dks_append_command(demo, error, "sqlrun",
                           dks_string_arg(ctx, 2, "instance", error),
                           dks_string_arg(ctx, 4, "script", error),
                           NULL);
        return NULL;
    case DATAKEEPER_ACTION_ADD_OBJECT:
        dks_append_command(demo, error, "addobject",
                           dks_string_arg(ctx, 2, "parent", error),
                           dks_string_arg(ctx, 4, "xml", error),
                           NULL);
        return NULL;
    case DATAKEEPER_ACTION_REMOVE_OBJECT:
        dks_append_command(demo, error, "removeobject",
                           dks_string_arg(ctx, 2, "parent", error),
                           dks_string_arg(ctx, 4, "name", error),
                           NULL);
        return NULL;
    case DATAKEEPER_ACTION_RUN_OBJECTS_JOB:
        dks_append_command(demo, error, "runobjectsjob",
                           dks_string_arg(ctx, 2, "parent", error),
                           dks_string_arg(ctx, 4, "name", error),
                           dks_string_arg(ctx, 6, "jobs tag", error));
        return NULL;
    case DATAKEEPER_ACTION_NONE:
    default:
        return dks_default_reduce(ctx);
    }
}

static int dks_parse(dks_demo *demo, const char *source, dks_reducer_mode mode, char *message, size_t message_size) {
    datakeeper_error error;
    datakeeper_scanner scanner;
    datakeeper_lexeme_source lexeme_source;
    int parsed = 0;
    error.message[0] = '\0';
    datakeeper_scanner_init(&scanner, source);
    lexeme_source.user = &scanner;
    lexeme_source.next = datakeeper_scanner_lexeme_source_next;
    if (!demo_text_append(&demo->report, "DataKeeper C mock compiler\nparameters:\n", message, message_size)) {
        return 0;
    }
    if (mode == DKS_REDUCER_TYPED) {
        datakeeper_boxed_typed_reducer boxed = {0};
        datakeeper_typed_reducer typed = datakeeper_typed_reducer_from_boxed(&boxed, dks_reduce, demo);
        parsed = datakeeper_parse_value_lexeme_source_typed(&lexeme_source, &typed, NULL, &error);
    } else {
        parsed = datakeeper_parse_value_lexeme_source(&lexeme_source, dks_reduce, demo, NULL, &error);
    }
    if (!parsed) {
        return demo_set_error(message, message_size, "parse failed: %s", error.message);
    }
    if (!demo_text_appendf(&demo->report, message, message_size, "summary: %d parameters, %d mock stack instructions\n", demo->parameters, demo->commands)) {
        return 0;
    }
    return 1;
}

static int dks_run_assertions(const char *source, char *message, size_t message_size) {
    dks_demo demo = {0};
    dks_demo boxed_demo = {0};
    datakeeper_error error;
    datakeeper_lexeme *tokens = NULL;
    size_t count = 0;
    if (!dks_parse(&demo, source, DKS_REDUCER_TYPED, message, message_size)) {
        demo_text_free(&demo.report);
        demo_arena_free(&demo.arena);
        return 0;
    }
    if (demo.parameters != 4 || demo.commands != 8) {
        demo_text_free(&demo.report);
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "unexpected summary: %d parameters, %d commands", demo.parameters, demo.commands);
    }
    demo_text_free(&demo.report);
    demo_arena_free(&demo.arena);
    if (!dks_parse(&boxed_demo, source, DKS_REDUCER_BOXED, message, message_size) || boxed_demo.parameters != 4 || boxed_demo.commands != 8) {
        demo_text_free(&boxed_demo.report);
        demo_arena_free(&boxed_demo.arena);
        return demo_set_error(message, message_size, "boxed reducer summary mismatch");
    }
    demo_text_free(&boxed_demo.report);
    demo_arena_free(&boxed_demo.arena);
    if (datakeeper_tokenize("begin @ end", &tokens, &count, &error)) {
        datakeeper_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected scanner failure");
    }
    if (!datakeeper_tokenize("begin end", &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "unexpected scanner failure: %s", error.message);
    }
    if (datakeeper_parse(tokens, count, &error)) {
        datakeeper_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected parser failure");
    }
    datakeeper_free_lexemes(tokens);
    return 1;
}

static const char *dks_read_option(int *argc, char **argv, const char *name, const char *fallback) {
    int i = 1;
    for (i = 1; i + 1 < *argc; i++) {
        if (strcmp(argv[i], name) == 0) {
            const char *value = argv[i + 1];
            int j = i;
            for (j = i; j + 2 < *argc; j++) {
                argv[j] = argv[j + 2];
            }
            *argc -= 2;
            return value;
        }
    }
    return fallback;
}

static int dks_take_flag(int *argc, char **argv, const char *name) {
    int i = 1;
    for (i = 1; i < *argc; i++) {
        if (strcmp(argv[i], name) == 0) {
            int j = i;
            for (j = i; j + 1 < *argc; j++) {
                argv[j] = argv[j + 1];
            }
            *argc -= 1;
            return 1;
        }
    }
    return 0;
}

int main(int argc, char **argv) {
    char message[512] = {0};
    demo_buffer source = {0};
    dks_demo demo = {0};
    int assert_mode = dks_take_flag(&argc, argv, "--assert");
    int boxed_mode = dks_take_flag(&argc, argv, "--boxed");
    const char *log_path = dks_read_option(&argc, argv, "--log", "dist/datakeeper-c-demo.log");
    const char *input_path = argc > 1 ? argv[1] : "sample.dks";
    if (!demo_read_file(input_path, &source, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        return 1;
    }
    if (assert_mode && !dks_run_assertions(source.data, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        return 1;
    }
    if (!dks_parse(&demo, source.data, boxed_mode ? DKS_REDUCER_BOXED : DKS_REDUCER_TYPED, message, sizeof(message)) ||
        !demo_write_text(log_path, demo.report.data, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        demo_text_free(&demo.report);
        demo_arena_free(&demo.arena);
        return 1;
    }
    printf("%s", demo.report.data);
    demo_free_buffer(&source);
    demo_text_free(&demo.report);
    demo_arena_free(&demo.arena);
    return 0;
}
