#ifndef LANGFORGE_EXAMPLES_C_DRAW_REPORT_H
#define LANGFORGE_EXAMPLES_C_DRAW_REPORT_H

#include "renderer.h"

/** Builds the deterministic text report printed by the demo and written to the
 * log file. The report is intentionally stable so tests can compare summaries
 * without depending on binary PNG bytes. */
int draw_build_report(draw_renderer *renderer, const char *input_path, const char *output_path, demo_text *report, char *message, size_t message_size);

#endif
