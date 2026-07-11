#include "../common/demo.h"
#include "generated/parser.h"

#include <stdio.h>
#include <string.h>

static const char *recovery_read_option(int *argc, char **argv, const char *name, const char *fallback)
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

static int recovery_take_flag(int *argc, char **argv, const char *name)
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

static int recovery_append_expected(demo_text *report, const recovery_parse_diagnostic *diagnostic, char *message, size_t message_size)
{
    size_t index = 0;
    if (diagnostic->expected_count == 0)
    {
        return demo_text_appendf(report, message, message_size, "<none>");
    }
    for (index = 0; index < diagnostic->expected_count; index++)
    {
        if (index > 0 && !demo_text_appendf(report, message, message_size, ", "))
        {
            return 0;
        }
        if (!demo_text_appendf(report, message, message_size, "%s", diagnostic->expected[index].display))
        {
            return 0;
        }
    }
    return 1;
}

static int recovery_build_report(const recovery_parse_result *result, demo_text *report, char *message, size_t message_size)
{
    size_t index = 0;
    if (!demo_text_appendf(report, message, message_size, "accepted: %s\n", result->accepted ? "true" : "false"))
    {
        return 0;
    }
    for (index = 0; index < result->diagnostic_count; index++)
    {
        const recovery_parse_diagnostic *diagnostic = &result->diagnostics[index];
        if (!demo_text_appendf(
                report,
                message,
                message_size,
                "%zu. %d:%d unexpected %s; expected ",
                index + 1,
                diagnostic->start_line,
                diagnostic->start_column,
                diagnostic->unexpected_display))
        {
            return 0;
        }
        if (!recovery_append_expected(report, diagnostic, message, message_size))
        {
            return 0;
        }
        if (!demo_text_appendf(
                report,
                message,
                message_size,
                "; recovery=%s discarded=%zu\n",
                diagnostic->recovery,
                diagnostic->discarded))
        {
            return 0;
        }
    }
    return 1;
}

static int recovery_result_has_discard(const recovery_parse_result *result)
{
    size_t index = 0;
    for (index = 0; index < result->diagnostic_count; index++)
    {
        if (result->diagnostics[index].discarded > 0)
        {
            return 1;
        }
    }
    return 0;
}

static int recovery_result_expects_number(const recovery_parse_result *result)
{
    size_t diagnostic_index = 0;
    for (diagnostic_index = 0; diagnostic_index < result->diagnostic_count; diagnostic_index++)
    {
        size_t expected_index = 0;
        int found = 0;
        const recovery_parse_diagnostic *diagnostic = &result->diagnostics[diagnostic_index];
        for (expected_index = 0; expected_index < diagnostic->expected_count; expected_index++)
        {
            if (strcmp(diagnostic->expected[expected_index].display, "number literal") == 0)
            {
                found = 1;
                break;
            }
        }
        if (!found)
        {
            return 0;
        }
    }
    return 1;
}

static int recovery_parse_demo_source(const char *source, recovery_parse_result *result, char *message, size_t message_size)
{
    recovery_error error;
    recovery_scanner scanner;
    recovery_lexeme_source lexeme_source;
    error.message[0] = '\0';
    recovery_scanner_init(&scanner, source);
    lexeme_source.user = &scanner;
    lexeme_source.next = recovery_scanner_lexeme_source_next;
    if (!recovery_parse_recovering_lexeme_source(&lexeme_source, result, &error))
    {
        return demo_set_error(message, message_size, "parse failed: %s", error.message);
    }
    return 1;
}

int main(int argc, char **argv)
{
    char message[512] = {0};
    demo_buffer source = {0};
    demo_text report = {0};
    recovery_parse_result result;
    int assert_mode = recovery_take_flag(&argc, argv, "--assert");
    const char *log_path = recovery_read_option(&argc, argv, "--log", "dist/parser-recovery-c-demo.log");
    const char *input_path = argc > 1 ? argv[1] : "input.recovery";
    int ok = 0;

    recovery_parse_result_init(&result);
    if (!demo_read_file(input_path, &source, message, sizeof(message)))
    {
        fprintf(stderr, "%s\n", message);
        return 1;
    }
    if (!recovery_parse_demo_source(source.data, &result, message, sizeof(message)) ||
        !recovery_build_report(&result, &report, message, sizeof(message)) ||
        !demo_write_text(log_path, report.data, message, sizeof(message)))
    {
        fprintf(stderr, "%s\n", message);
        goto cleanup;
    }
    printf("%s", report.data);
    ok = 1;

    if (assert_mode)
    {
        /*
         * Assertions map directly to the grammar fixture:
         * - line 1 recovers by discarding "y" before the semicolon;
         * - line 3 shifts error and immediately synchronizes at Semi;
         * - both diagnostics expect Number via the "number literal" alias.
         */
        if (!result.accepted || result.diagnostic_count != 2 || !recovery_result_has_discard(&result) || !recovery_result_expects_number(&result))
        {
            fprintf(stderr, "fixture result = accepted:%d diagnostics:%zu, want accepted:1 diagnostics:2 with number-literal recovery\n", result.accepted, result.diagnostic_count);
            ok = 0;
        }
    }

cleanup:
    recovery_parse_result_free(&result);
    demo_text_free(&report);
    demo_free_buffer(&source);
    return ok ? 0 : 1;
}
