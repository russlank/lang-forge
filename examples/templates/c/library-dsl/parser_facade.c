#include "parser_facade.h"

#include "diagnostics.h"
#include "semantics.h"

#include <stdio.h>
#include <string.h>

void dsl_parse_result_init(dsl_parse_result *result) {
    if (result == NULL) {
        return;
    }
    result->document = NULL;
    result->message[0] = '\0';
    result->accepted = 0;
}

void dsl_parse_result_free(dsl_parse_result *result) {
    if (result == NULL) {
        return;
    }
    dsl_document_free(result->document);
    dsl_parse_result_init(result);
}

int dsl_parse_source(const char *source, dsl_parse_result *result) {
    library_dsl_error error;
    library_dsl_scanner scanner;
    library_dsl_lexeme_source token_source;
    library_dsl_parse_result generated_result;
    dsl_semantic_context semantics;
    library_dsl_typed_reducer reducer;
    dsl_parse_result_init(result);
    error.message[0] = '\0';
    semantics.message[0] = '\0';
    library_dsl_parse_result_init(&generated_result);
    library_dsl_scanner_init(&scanner, source == NULL ? "" : source);
    token_source.user = &scanner;
    token_source.next = library_dsl_scanner_source_next;
    reducer = dsl_make_typed_reducer(&semantics);

    if (!library_dsl_parse_value_recovering_source_typed(&token_source, &reducer, &generated_result, &error)) {
        snprintf(result->message, sizeof(result->message), "%s", error.message[0] == '\0' ? "parse failed" : error.message);
        library_dsl_parse_result_free(&generated_result);
        return 0;
    }
    if (!generated_result.accepted || generated_result.diagnostic_count != 0) {
        dsl_format_parse_diagnostics(&generated_result, result->message, sizeof(result->message));
        library_dsl_parse_result_free(&generated_result);
        return 0;
    }
    result->document = (dsl_document *)generated_result.value;
    result->accepted = 1;
    generated_result.value = NULL;
    library_dsl_parse_result_free(&generated_result);
    return 1;
}
