#include "parser_facade.h"

#include <stdio.h>
#include <string.h>

static int fail(char *message, size_t size, const char *text) {
    snprintf(message, size, "%s", text);
    return 0;
}

static int test_successful_parse(char *message, size_t size) {
    const char *source = "set retries = 3;\nset title = \"nightly\";\nenable audit;\n";
    dsl_parse_result result;
    const dsl_entry *first = NULL;
    const dsl_entry *second = NULL;
    const dsl_entry *third = NULL;
    dsl_parse_result_init(&result);

    if (!dsl_parse_lexeme_source(source, &result)) {
        snprintf(message, size, "expected valid source, got: %.460s", result.message);
        dsl_parse_result_free(&result);
        return 0;
    }
    if (!result.accepted || result.document == NULL) {
        dsl_parse_result_free(&result);
        return fail(message, size, "expected accepted document");
    }

    first = result.document->entries;
    second = first == NULL ? NULL : first->next;
    third = second == NULL ? NULL : second->next;
    if (first == NULL || first->kind != DSL_ENTRY_SET || first->value == NULL || first->value->number != 3) {
        dsl_parse_result_free(&result);
        return fail(message, size, "unexpected first setting");
    }
    if (second == NULL || second->kind != DSL_ENTRY_SET || second->value == NULL || strcmp(second->value->text, "nightly") != 0) {
        dsl_parse_result_free(&result);
        return fail(message, size, "unexpected second setting");
    }
    if (third == NULL || third->kind != DSL_ENTRY_ENABLE || third->value == NULL || !third->value->boolean) {
        dsl_parse_result_free(&result);
        return fail(message, size, "unexpected enable entry");
    }

    dsl_parse_result_free(&result);
    if (result.document != NULL || result.accepted != 0 || result.message[0] != '\0') {
        return fail(message, size, "parse result cleanup did not reset fields");
    }
    return 1;
}

static int test_syntax_error(char *message, size_t size) {
    dsl_parse_result result;
    dsl_parse_result_init(&result);

    if (dsl_parse_lexeme_source("set retries = ;", &result)) {
        dsl_parse_result_free(&result);
        return fail(message, size, "expected syntax error");
    }
    if (result.accepted || result.document != NULL || strstr(result.message, "unexpected") == NULL) {
        snprintf(message, size, "unexpected syntax diagnostic: %.460s", result.message);
        dsl_parse_result_free(&result);
        return 0;
    }

    dsl_parse_result_free(&result);
    return 1;
}

static int test_reducer_error(char *message, size_t size) {
    dsl_parse_result result;
    dsl_parse_result_init(&result);

    if (dsl_parse_lexeme_source("set retries = 999999999999999999999999;", &result)) {
        dsl_parse_result_free(&result);
        return fail(message, size, "expected reducer error");
    }
    if (result.accepted || result.document != NULL ||
        (strstr(result.message, "value.number") == NULL && strstr(result.message, "valid int") == NULL)) {
        snprintf(message, size, "unexpected reducer diagnostic: %.460s", result.message);
        dsl_parse_result_free(&result);
        return 0;
    }

    dsl_parse_result_free(&result);
    return 1;
}

static int test_cleanup_is_idempotent(char *message, size_t size) {
    dsl_parse_result result;
    dsl_parse_result_init(&result);

    if (!dsl_parse_lexeme_source("enable audit;", &result)) {
        snprintf(message, size, "expected cleanup fixture to parse: %.460s", result.message);
        dsl_parse_result_free(&result);
        return 0;
    }

    dsl_parse_result_free(&result);
    dsl_parse_result_free(&result);
    if (result.document != NULL || result.accepted != 0 || result.message[0] != '\0') {
        return fail(message, size, "second cleanup did not leave result reset");
    }
    return 1;
}

int main(void) {
    char message[512] = {0};

    if (!test_successful_parse(message, sizeof(message)) ||
        !test_syntax_error(message, sizeof(message)) ||
        !test_reducer_error(message, sizeof(message)) ||
        !test_cleanup_is_idempotent(message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        return 1;
    }

    puts("C library DSL template tests passed");
    return 0;
}
