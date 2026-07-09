#include "../common/demo.h"
#include "generated/parser.h"
#include "generated/parser_typed.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef struct calc_demo
{
    demo_arena arena;
} calc_demo;

typedef double (*calc_binary_op)(double left, double right);

typedef enum calc_reducer_mode
{
    /* Recommended path for new C code: generated contexts plus native handlers. */
    CALC_REDUCER_TYPED,
    /* Migration path: generated contexts validate before delegating to boxed code. */
    CALC_REDUCER_BOXED_TO_TYPED,
    /* Compatibility/debug path: the historical boxed reducer ABI. */
    CALC_REDUCER_BOXED
} calc_reducer_mode;

static calc_value calc_number(calc_demo *demo, calc_error *error, double value)
{
    double *slot = (double *)demo_arena_alloc(&demo->arena, sizeof(double));
    if (slot == NULL)
    {
        snprintf(error->message, sizeof(error->message), "out of memory storing semantic value");
        return NULL;
    }
    *slot = value;
    return slot;
}

static double calc_value_as_number(calc_value value)
{
    return *((double *)value);
}

static const calc_lexeme *calc_value_as_lexeme(calc_value value)
{
    return (const calc_lexeme *)value;
}

static int calc_parse_number_lexeme(calc_demo *demo, const calc_lexeme *lexeme, double *out, calc_error *error)
{
    char *text = NULL;
    char *end = NULL;
    if (lexeme == NULL)
    {
        snprintf(error->message, sizeof(error->message), "number reduction missing lexeme");
        return 0;
    }
    text = demo_arena_copy(&demo->arena, lexeme->text, lexeme->length);
    if (text == NULL)
    {
        snprintf(error->message, sizeof(error->message), "out of memory parsing number");
        return 0;
    }
    *out = strtod(text, &end);
    if (end == text || *end != '\0')
    {
        snprintf(error->message, sizeof(error->message), "invalid number lexeme '%s'", text);
        return 0;
    }
    return 1;
}

static int calc_check_arg(const calc_reduction *ctx, size_t index, const char *name, calc_error *error)
{
    /*
     * Keep boxed C semantic-value checks in one helper. Reducer branches pass
     * names like "left operand" so failures describe the grammar role instead
     * of only the numeric RHS position.
     */
    if (index >= ctx->rhs_count || ctx->values[index] == NULL)
    {
        snprintf(error->message, sizeof(error->message), "rule %d missing %s at argument %zu", ctx->rule, name, index + 1);
        return 0;
    }
    return 1;
}

static int calc_number_arg(const calc_reduction *ctx, size_t index, const char *name, double *out, calc_error *error)
{
    if (!calc_check_arg(ctx, index, name, error))
    {
        return 0;
    }
    *out = calc_value_as_number(ctx->values[index]);
    return 1;
}

static const calc_lexeme *calc_lexeme_arg(const calc_reduction *ctx, size_t index, const char *name, calc_error *error)
{
    if (!calc_check_arg(ctx, index, name, error))
    {
        return NULL;
    }
    return calc_value_as_lexeme(ctx->values[index]);
}

static double calc_add_numbers(double left, double right)
{
    return left + right;
}

static double calc_subtract_numbers(double left, double right)
{
    return left - right;
}

static double calc_multiply_numbers(double left, double right)
{
    return left * right;
}

static calc_value calc_typed_value(void *user, calc_error *error, double value)
{
    /*
     * C typed contexts expose native fields such as ctx->left and ctx->value,
     * but generated C parsers still store semantic results as calc_value
     * pointers. This example owns every returned number in demo->arena; the
     * caller releases all semantic values with demo_arena_free.
     */
    return calc_number((calc_demo *)user, error, value);
}

static calc_value calc_typed_start(const calc_start_reduction *ctx, void *user, calc_error *error)
{
    return calc_typed_value(user, error, ctx->value);
}

static calc_value calc_typed_pass(const calc_pass_reduction *ctx, void *user, calc_error *error)
{
    return calc_typed_value(user, error, ctx->value);
}

static calc_value calc_typed_group(const calc_group_reduction *ctx, void *user, calc_error *error)
{
    return calc_typed_value(user, error, ctx->value);
}

static calc_value calc_typed_number(const calc_number_reduction *ctx, void *user, calc_error *error)
{
    double value = 0.0;
    calc_demo *demo = (calc_demo *)user;
    if (!calc_parse_number_lexeme(demo, ctx->token, &value, error))
    {
        return NULL;
    }
    return calc_number(demo, error, value);
}

static calc_value calc_typed_negate(const calc_negate_reduction *ctx, void *user, calc_error *error)
{
    return calc_typed_value(user, error, -ctx->value);
}

static calc_value calc_typed_binary(void *user, calc_error *error, double left, double right, calc_binary_op op)
{
    return calc_typed_value(user, error, op(left, right));
}

static calc_value calc_typed_add(const calc_add_reduction *ctx, void *user, calc_error *error)
{
    return calc_typed_binary(user, error, ctx->left, ctx->right, calc_add_numbers);
}

static calc_value calc_typed_subtract(const calc_subtract_reduction *ctx, void *user, calc_error *error)
{
    return calc_typed_binary(user, error, ctx->left, ctx->right, calc_subtract_numbers);
}

static calc_value calc_typed_multiply(const calc_multiply_reduction *ctx, void *user, calc_error *error)
{
    return calc_typed_binary(user, error, ctx->left, ctx->right, calc_multiply_numbers);
}

static calc_value calc_typed_divide(const calc_divide_reduction *ctx, void *user, calc_error *error)
{
    if (ctx->right == 0.0)
    {
        snprintf(error->message, sizeof(error->message), "division by zero");
        return NULL;
    }
    return calc_typed_value(user, error, ctx->left / ctx->right);
}

static calc_typed_reducer calc_make_direct_typed_reducer(calc_demo *demo)
{
    static const calc_typed_reducer typed_template = {
        .start = calc_typed_start,
        .add = calc_typed_add,
        .subtract = calc_typed_subtract,
        .pass = calc_typed_pass,
        .multiply = calc_typed_multiply,
        .divide = calc_typed_divide,
        .number = calc_typed_number,
        .group = calc_typed_group,
        .negate = calc_typed_negate,
    };
    calc_typed_reducer typed = typed_template;
    typed.user = demo;
    return typed;
}

static calc_value calc_reduce_binary(const calc_reduction *ctx, calc_demo *demo, calc_error *error, calc_binary_op op)
{
    double left = 0.0;
    double right = 0.0;
    if (!calc_number_arg(ctx, 0, "left operand", &left, error) ||
        !calc_number_arg(ctx, 2, "right operand", &right, error))
    {
        return NULL;
    }
    return calc_number(demo, error, op(left, right));
}

static calc_value calc_reduce(const calc_reduction *ctx, void *user, calc_error *error)
{
    calc_demo *demo = (calc_demo *)user;
    /*
     * action_id values are generated from {c: ...} labels in calc.lf. The
     * handwritten reducer supplies the arithmetic; generated code only decides
     * which grammar rule reduced.
     */
    switch (ctx->action_id)
    {
    case CALC_ACTION_START:
    case CALC_ACTION_PASS:
        return ctx->values[0];
    case CALC_ACTION_GROUP:
        return ctx->values[1];
    case CALC_ACTION_NUMBER:
    {
        const calc_lexeme *lexeme = calc_lexeme_arg(ctx, 0, "number lexeme", error);
        double value = 0.0;
        if (lexeme == NULL)
        {
            return NULL;
        }
        if (!calc_parse_number_lexeme(demo, lexeme, &value, error))
        {
            return NULL;
        }
        return calc_number(demo, error, value);
    }
    case CALC_ACTION_NEGATE:
    {
        double operand = 0.0;
        if (!calc_number_arg(ctx, 1, "operand", &operand, error))
        {
            return NULL;
        }
        return calc_number(demo, error, -operand);
    }
    case CALC_ACTION_ADD:
        return calc_reduce_binary(ctx, demo, error, calc_add_numbers);
    case CALC_ACTION_SUBTRACT:
        return calc_reduce_binary(ctx, demo, error, calc_subtract_numbers);
    case CALC_ACTION_MULTIPLY:
        return calc_reduce_binary(ctx, demo, error, calc_multiply_numbers);
    case CALC_ACTION_DIVIDE:
    {
        double left = 0.0;
        double right = 0.0;
        if (!calc_number_arg(ctx, 0, "left operand", &left, error) ||
            !calc_number_arg(ctx, 2, "right operand", &right, error))
        {
            return NULL;
        }
        if (right == 0.0)
        {
            snprintf(error->message, sizeof(error->message), "division by zero");
            return NULL;
        }
        return calc_number(demo, error, left / right);
    }
    case CALC_ACTION_NONE:
    default:
        return ctx->rhs_count == 1 ? ctx->values[0] : NULL;
    }
}

static int calc_eval(calc_demo *demo, const char *source, calc_reducer_mode mode, double *out, char *message, size_t message_size)
{
    calc_error error;
    calc_scanner scanner;
    calc_lexeme_source token_source;
    calc_value value = NULL;
    error.message[0] = '\0';
    calc_scanner_init(&scanner, source);
    token_source.user = &scanner;
    token_source.next = calc_scanner_source_next;
    if (mode == CALC_REDUCER_TYPED)
    {
        /*
         * Recommended direct typed path: generated code validates named RHS
         * labels and provides fields such as ctx->left and ctx->right to
         * handwritten handlers. No boxed ctx->values indexing is needed here.
         */
        calc_typed_reducer typed = calc_make_direct_typed_reducer(demo);
        if (!calc_parse_value_source_typed(&token_source, &typed, &value, &error))
        {
            return demo_set_error(message, message_size, "parse failed: %s", error.message);
        }
    }
    else if (mode == CALC_REDUCER_BOXED_TO_TYPED)
    {
        /*
         * Migration path: keep an older boxed reducer while letting generated
         * typed contexts validate labels and semantic value types first.
         */
        calc_boxed_typed_reducer boxed = {0};
        calc_typed_reducer typed = calc_typed_reducer_from_boxed(&boxed, calc_reduce, demo);
        if (!calc_parse_value_source_typed(&token_source, &typed, &value, &error))
        {
            return demo_set_error(message, message_size, "parse failed: %s", error.message);
        }
    }
    else if (!calc_parse_value_source(&token_source, calc_reduce, demo, &value, &error))
    {
        return demo_set_error(message, message_size, "parse failed: %s", error.message);
    }
    *out = calc_value_as_number(value);
    return 1;
}

static int calc_close_enough(double actual, double expected)
{
    double delta = actual - expected;
    if (delta < 0)
    {
        delta = -delta;
    }
    return delta < 0.000001;
}

static int calc_run_assertions(char *message, size_t message_size)
{
    calc_demo demo = {0};
    double value = 0.0;
    calc_error error;
    calc_lexeme *tokens = NULL;
    size_t count = 0;
    error.message[0] = '\0';
    if (!calc_eval(&demo, "1 + 2 * (3 - 4.5)", CALC_REDUCER_TYPED, &value, message, message_size) || !calc_close_enough(value, -2.0))
    {
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "calculator assertion failed, got %.6f", value);
    }
    if (!calc_eval(&demo, "7.5/2.5", CALC_REDUCER_BOXED, &value, message, message_size) || !calc_close_enough(value, 3.0))
    {
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "boxed decimal division assertion failed, got %.6f", value);
    }
    if (!calc_eval(&demo, "3 + 4", CALC_REDUCER_BOXED_TO_TYPED, &value, message, message_size) || !calc_close_enough(value, 7.0))
    {
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "boxed-to-typed migration assertion failed, got %.6f", value);
    }
    if (calc_eval(&demo, "1/0", CALC_REDUCER_TYPED, &value, message, message_size))
    {
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "expected division-by-zero failure");
    }
    if (calc_eval(&demo, "1@", CALC_REDUCER_TYPED, &value, message, message_size))
    {
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "expected source scanner failure");
    }
    if (calc_eval(&demo, "1+", CALC_REDUCER_TYPED, &value, message, message_size))
    {
        demo_arena_free(&demo.arena);
        return demo_set_error(message, message_size, "expected source parser failure");
    }
    demo_arena_free(&demo.arena);
    if (calc_tokenize("1@", &tokens, &count, &error))
    {
        calc_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected scanner failure for unmatched input");
    }
    if (!calc_tokenize("1+", &tokens, &count, &error))
    {
        return demo_set_error(message, message_size, "unexpected scanner failure: %s", error.message);
    }
    if (calc_parse(tokens, count, &error))
    {
        calc_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected parser failure for incomplete expression");
    }
    calc_free_lexemes(tokens);
    return 1;
}

static const char *read_option(int *argc, char **argv, const char *name, const char *fallback)
{
    int i = 1;
    for (i = 1; i + 1 < *argc; i++)
    {
        if (strcmp(argv[i], name) == 0)
        {
            const char *value = argv[i + 1];
            int j = i;
            for (j = i; j + 2 < *argc; j++)
            {
                argv[j] = argv[j + 2];
            }
            *argc -= 2;
            return value;
        }
    }
    return fallback;
}

static int take_flag(int *argc, char **argv, const char *name)
{
    int i = 1;
    for (i = 1; i < *argc; i++)
    {
        if (strcmp(argv[i], name) == 0)
        {
            int j = i;
            for (j = i; j + 1 < *argc; j++)
            {
                argv[j] = argv[j + 1];
            }
            *argc -= 1;
            return 1;
        }
    }
    return 0;
}

int main(int argc, char **argv)
{
    char message[512] = {0};
    demo_buffer source = {0};
    demo_text report = {0};
    calc_demo demo = {0};
    double result = 0.0;
    int assert_mode = take_flag(&argc, argv, "--assert");
    int boxed_mode = take_flag(&argc, argv, "--boxed");
    int boxed_typed_mode = take_flag(&argc, argv, "--boxed-typed");
    calc_reducer_mode mode = boxed_typed_mode ? CALC_REDUCER_BOXED_TO_TYPED : (boxed_mode ? CALC_REDUCER_BOXED : CALC_REDUCER_TYPED);
    const char *log_path = read_option(&argc, argv, "--log", "dist/calc-c-demo.log");
    const char *input_path = argc > 1 ? argv[1] : "input.calc";
    if (assert_mode && !calc_run_assertions(message, sizeof(message)))
    {
        fprintf(stderr, "%s\n", message);
        return 1;
    }
    if (!demo_read_file(input_path, &source, message, sizeof(message)) ||
        !calc_eval(&demo, source.data, mode, &result, message, sizeof(message)) ||
        !demo_text_appendf(&report, message, sizeof(message), "LangForge C calculator demo\nsource: %s\nresult: %.10g\n", source.data, result) ||
        !demo_write_text(log_path, report.data, message, sizeof(message)))
    {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        demo_text_free(&report);
        demo_arena_free(&demo.arena);
        return 1;
    }
    printf("%s", report.data);
    demo_free_buffer(&source);
    demo_text_free(&report);
    demo_arena_free(&demo.arena);
    return 0;
}
