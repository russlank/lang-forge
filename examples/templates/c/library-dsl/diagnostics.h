#ifndef LIBRARY_DSL_DIAGNOSTICS_H
#define LIBRARY_DSL_DIAGNOSTICS_H

#include "generated/parser.h"

#include <stddef.h>

/*
 * Formats generated parser diagnostics into one application-facing message.
 * The generated diagnostic array remains owned by library_dsl_parse_result and
 * is released by library_dsl_parse_result_free in parser_facade.c.
 */
void dsl_format_parse_diagnostics(const library_dsl_parse_result *result, char *buffer, size_t size);

#endif
