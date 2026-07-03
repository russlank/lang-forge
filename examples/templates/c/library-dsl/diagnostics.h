#ifndef LIBRARY_DSL_DIAGNOSTICS_H
#define LIBRARY_DSL_DIAGNOSTICS_H

#include "generated/parser.h"

#include <stddef.h>

/* Formats generated parser diagnostics into one application-facing message. */
void dsl_format_parse_diagnostics(const library_dsl_parse_result *result, char *buffer, size_t size);

#endif
