#include "../common/demo.h"
#include "generated/parser.h"
#include "generated/parser_typed.h"

#include <stdio.h>
#include <string.h>

typedef struct vehicle_demo {
    demo_arena arena;
    demo_text report;
    int features;
    int repairs;
    int saw_model;
    int saw_license;
    int saw_distance;
} vehicle_demo;

typedef enum vehicle_reducer_mode {
    VEHICLE_REDUCER_TYPED,
    VEHICLE_REDUCER_BOXED
} vehicle_reducer_mode;

static const vehicle_report_lexeme *vehicle_lexeme(vehicle_report_value value) {
    return (const vehicle_report_lexeme *)value;
}

static char *vehicle_copy_lexeme(vehicle_demo *demo, const vehicle_report_lexeme *lexeme) {
    if (lexeme == NULL) {
        return NULL;
    }
    return demo_arena_copy(&demo->arena, lexeme->text, lexeme->length);
}

static vehicle_report_value vehicle_arg(const vehicle_report_reduction *ctx, size_t index, const char *name, vehicle_report_error *error) {
    /*
     * This is the boxed C reducer boundary. Callers provide a grammar-oriented
     * name such as "feature value" so errors explain the semantic role, not
     * only the numeric parser-stack position.
     */
    if (index >= ctx->rhs_count || ctx->values[index] == NULL) {
        snprintf(error->message, sizeof(error->message), "rule %d missing %s at argument %zu", ctx->rule, name, index + 1);
        return NULL;
    }
    return ctx->values[index];
}

static const vehicle_report_lexeme *vehicle_lexeme_arg(const vehicle_report_reduction *ctx, size_t index, const char *name, vehicle_report_error *error) {
    vehicle_report_value value = vehicle_arg(ctx, index, name, error);
    return value == NULL ? NULL : vehicle_lexeme(value);
}

static char *vehicle_unquote(vehicle_demo *demo, const vehicle_report_lexeme *lexeme) {
    if (lexeme == NULL) {
        return NULL;
    }
    const char *text = lexeme->text;
    size_t length = lexeme->length;
    if (length >= 2 && text[0] == '"' && text[length - 1] == '"') {
        return demo_arena_copy(&demo->arena, text + 1, length - 2);
    }
    return vehicle_copy_lexeme(demo, lexeme);
}

static vehicle_report_value vehicle_default_reduce(const vehicle_report_reduction *ctx) {
    if (ctx->rhs_count == 1) {
        return ctx->values[0];
    }
    return NULL;
}

static vehicle_report_value vehicle_reduce(const vehicle_report_reduction *ctx, void *user, vehicle_report_error *error) {
    vehicle_demo *demo = (vehicle_demo *)user;
    /*
     * Generated action IDs come from {c: ...} labels in vehicle.lf. The switch
     * below turns recognized fields into a report; generated parser code never
     * knows about report formatting.
     */
    switch (ctx->action_id) {
    case VEHICLE_REPORT_ACTION_FIELD_MODEL: {
        char *model = vehicle_unquote(demo, vehicle_lexeme_arg(ctx, 2, "model literal", error));
        demo->saw_model = 1;
        demo_text_appendf(&demo->report, error->message, sizeof(error->message), "model: %s\n", model);
        return NULL;
    }
    case VEHICLE_REPORT_ACTION_FIELD_LICENSE: {
        char *license = vehicle_unquote(demo, vehicle_lexeme_arg(ctx, 2, "license literal", error));
        demo->saw_license = 1;
        demo_text_appendf(&demo->report, error->message, sizeof(error->message), "license: %s\n", license);
        return NULL;
    }
    case VEHICLE_REPORT_ACTION_FIELD_DISTANCE: {
        char *distance = vehicle_copy_lexeme(demo, vehicle_lexeme_arg(ctx, 2, "distance literal", error));
        demo->saw_distance = 1;
        demo_text_appendf(&demo->report, error->message, sizeof(error->message), "distance: %s\n", distance);
        return NULL;
    }
    case VEHICLE_REPORT_ACTION_FIELD_FEATURES:
        return NULL;
    case VEHICLE_REPORT_ACTION_FEATURE: {
        char *name = vehicle_copy_lexeme(demo, vehicle_lexeme_arg(ctx, 0, "feature name", error));
        char *value = vehicle_unquote(demo, vehicle_lexeme_arg(ctx, 2, "feature value", error));
        if (demo->features == 0) {
            demo_text_append(&demo->report, "features:\n", error->message, sizeof(error->message));
        }
        demo->features++;
        demo_text_appendf(&demo->report, error->message, sizeof(error->message), "  - %s = %s\n", name, value);
        return NULL;
    }
    case VEHICLE_REPORT_ACTION_FIELD_REPAIRS:
        return NULL;
    case VEHICLE_REPORT_ACTION_REPAIR: {
        char *date = vehicle_unquote(demo, vehicle_lexeme_arg(ctx, 3, "repair date", error));
        char *description = vehicle_unquote(demo, vehicle_lexeme_arg(ctx, 7, "repair description", error));
        if (demo->repairs == 0) {
            demo_text_append(&demo->report, "repairs:\n", error->message, sizeof(error->message));
        }
        demo->repairs++;
        demo_text_appendf(&demo->report, error->message, sizeof(error->message), "  - %s: %s\n", date, description);
        return NULL;
    }
    case VEHICLE_REPORT_ACTION_NONE:
    default:
        return vehicle_default_reduce(ctx);
    }
}

static int vehicle_parse_lexeme_source(vehicle_demo *demo, const char *source, vehicle_reducer_mode mode, char *message, size_t message_size) {
    vehicle_report_error error;
    vehicle_report_scanner scanner;
    vehicle_report_lexeme_source lexeme_source;
    int parsed = 0;
    error.message[0] = '\0';
    if (!demo_text_append(&demo->report, "Vehicle report C generated-parser demo\n", message, message_size)) {
        return 0;
    }
    vehicle_report_scanner_init(&scanner, source);
    lexeme_source.user = &scanner;
    lexeme_source.next = vehicle_report_scanner_lexeme_source_next;
    if (mode == VEHICLE_REDUCER_TYPED) {
        vehicle_report_boxed_typed_reducer boxed = {0};
        vehicle_report_typed_reducer typed = vehicle_report_typed_reducer_from_boxed(&boxed, vehicle_reduce, demo);
        parsed = vehicle_report_parse_value_lexeme_source_typed(&lexeme_source, &typed, NULL, &error);
    } else {
        parsed = vehicle_report_parse_value_lexeme_source(&lexeme_source, vehicle_reduce, demo, NULL, &error);
    }
    if (!parsed) {
        return demo_set_error(message, message_size, "parse failed: %s", error.message);
    }
    return demo_text_appendf(&demo->report, message, message_size, "summary: %d features, %d repairs\n", demo->features, demo->repairs);
}

static int vehicle_run_assertions(const char *source, char *message, size_t message_size) {
    vehicle_demo demo = {0};
    vehicle_demo boxed_demo = {0};
    vehicle_report_error error;
    vehicle_report_lexeme *tokens = NULL;
    size_t count = 0;
    if (!vehicle_parse_lexeme_source(&demo, source, VEHICLE_REDUCER_TYPED, message, message_size)) {
        demo_text_free(&demo.report);
        demo_arena_free(&demo.arena);
        return 0;
    }
    if (!demo.saw_model || !demo.saw_license || !demo.saw_distance || demo.features != 4 || demo.repairs != 3) {
        demo_text_free(&demo.report);
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "unexpected vehicle summary");
    }
    demo_text_free(&demo.report);
    demo_arena_free(&demo.arena);
    if (!vehicle_parse_lexeme_source(&boxed_demo, source, VEHICLE_REDUCER_BOXED, message, message_size) ||
        !boxed_demo.saw_model || !boxed_demo.saw_license || !boxed_demo.saw_distance ||
        boxed_demo.features != 4 || boxed_demo.repairs != 3) {
        demo_text_free(&boxed_demo.report);
        demo_arena_free(&boxed_demo.arena);
        return demo_set_error(message, message_size, "boxed vehicle summary mismatch");
    }
    demo_text_free(&boxed_demo.report);
    demo_arena_free(&boxed_demo.arena);
    if (vehicle_report_tokenize("car = @", &tokens, &count, &error)) {
        vehicle_report_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected scanner failure");
    }
    if (!vehicle_report_tokenize("car = {}", &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "unexpected scanner failure: %s", error.message);
    }
    if (vehicle_report_parse(tokens, count, &error)) {
        vehicle_report_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected parser failure");
    }
    vehicle_report_free_lexemes(tokens);
    return 1;
}

static const char *vehicle_read_option(int *argc, char **argv, const char *name, const char *fallback) {
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

static int vehicle_take_flag(int *argc, char **argv, const char *name) {
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
    vehicle_demo demo = {0};
    int assert_mode = vehicle_take_flag(&argc, argv, "--assert");
    int boxed_mode = vehicle_take_flag(&argc, argv, "--boxed");
    const char *log_path = vehicle_read_option(&argc, argv, "--log", "dist/vehicle-report-c-demo.log");
    const char *input_path = argc > 1 ? argv[1] : "sample.vehicle";
    if (!demo_read_file(input_path, &source, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        return 1;
    }
    if (assert_mode && !vehicle_run_assertions(source.data, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        return 1;
    }
    if (!vehicle_parse_lexeme_source(&demo, source.data, boxed_mode ? VEHICLE_REDUCER_BOXED : VEHICLE_REDUCER_TYPED, message, sizeof(message)) ||
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
