#include "diagnostics.h"

#include <stdio.h>
#include <string.h>

void dsl_format_parse_diagnostics(const library_dsl_parse_result *result, char *buffer, size_t size) {
    size_t used = 0;
    size_t i = 0;
    if (size == 0) {
        return;
    }
    buffer[0] = '\0';
    if (result == NULL || result->diagnostic_count == 0) {
        snprintf(buffer, size, "parse failed without diagnostics");
        return;
    }
    for (i = 0; i < result->diagnostic_count && used + 1 < size; i++) {
        const library_dsl_parse_diagnostic *diagnostic = &result->diagnostics[i];
        const char *expected = diagnostic->expected_count == 0 ? "no known continuation" : diagnostic->expected[0].display;
        int written = snprintf(buffer + used, size - used, "%d:%d: unexpected %s; expected %s%s",
            diagnostic->start_line,
            diagnostic->start_column,
            diagnostic->unexpected_display,
            expected,
            i + 1 == result->diagnostic_count ? "" : "\n");
        if (written < 0) {
            return;
        }
        used += (size_t)written;
    }
}
