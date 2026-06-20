#include "report.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static int draw_compare_names(const void *left, const void *right) {
    const char *const *a = (const char *const *)left;
    const char *const *b = (const char *const *)right;
    return strcmp(*a, *b);
}

static char *draw_color_hex(draw_color color, char *buffer, size_t size) {
    snprintf(buffer, size, "#%02X%02X%02X", color.r, color.g, color.b);
    return buffer;
}

static int draw_append_figures(draw_renderer *renderer, demo_text *report, char *message, size_t message_size) {
    size_t count = draw_figure_count(renderer);
    const char **names = NULL;
    size_t index = 0;
    draw_named_figure *figure = renderer->figures;
    if (count == 0) {
        return demo_text_append(report, "[]", message, message_size);
    }
    names = (const char **)calloc(count, sizeof(char *));
    if (names == NULL) {
        return demo_set_error(message, message_size, "out of memory sorting figure names");
    }
    while (figure != NULL) {
        names[index++] = figure->name;
        figure = figure->next;
    }
    qsort(names, count, sizeof(char *), draw_compare_names);
    if (!demo_text_append(report, "[", message, message_size)) {
        free(names);
        return 0;
    }
    for (index = 0; index < count; index++) {
        if (!demo_text_appendf(report, message, message_size, "%s%s", index == 0 ? "" : ", ", names[index])) {
            free(names);
            return 0;
        }
    }
    free(names);
    return demo_text_append(report, "]", message, message_size);
}

static int draw_append_count(demo_text *report, char *message, size_t message_size, const char *name, int count) {
    if (count <= 0) {
        return 1;
    }
    return demo_text_appendf(report, message, message_size, "  %s: %d\n", name, count);
}

static int draw_append_define_counts(draw_renderer *renderer, demo_text *report, char *message, size_t message_size) {
    size_t count = draw_figure_count(renderer);
    const char **names = NULL;
    size_t index = 0;
    draw_named_figure *figure = renderer->figures;
    if (count == 0) {
        return 1;
    }
    names = (const char **)calloc(count, sizeof(char *));
    if (names == NULL) {
        return demo_set_error(message, message_size, "out of memory sorting figure names");
    }
    while (figure != NULL) {
        names[index++] = figure->name;
        figure = figure->next;
    }
    qsort(names, count, sizeof(char *), draw_compare_names);
    for (index = 0; index < count; index++) {
        if (!demo_text_appendf(report, message, message_size, "  define %s: 1\n", names[index])) {
            free(names);
            return 0;
        }
    }
    free(names);
    return 1;
}

int draw_build_report(draw_renderer *renderer, const char *input_path, const char *output_path, demo_text *report, char *message, size_t message_size) {
    char background[16];
    char background_label[32];
    char canvas_label[32];
    snprintf(background_label, sizeof(background_label), "background %s", draw_color_hex(renderer->background, background, sizeof(background)));
    snprintf(canvas_label, sizeof(canvas_label), "canvas %d,%d", renderer->image.width, renderer->image.height);
    if (!demo_text_appendf(report, message, message_size,
            "DRAW C render report\nSource: %s\nOutput: %s\nCanvas: %dx%d\nFigures: ",
            input_path,
            output_path,
            renderer->image.width,
            renderer->image.height) ||
        !draw_append_figures(renderer, report, message, message_size) ||
        !demo_text_appendf(report, message, message_size, "\nVariables: %lu\n\nOperation summary:\n", (unsigned long)draw_var_count(renderer)) ||
        !draw_append_count(report, message, message_size, background_label, renderer->counts.background) ||
        !draw_append_count(report, message, message_size, "box 4 args", renderer->counts.box) ||
        !draw_append_count(report, message, message_size, canvas_label, renderer->counts.canvas) ||
        !draw_append_count(report, message, message_size, "circle 3 args", renderer->counts.circle) ||
        !draw_append_define_counts(renderer, report, message, message_size) ||
        !draw_append_count(report, message, message_size, "line 4 args", renderer->counts.line) ||
        !draw_append_count(report, message, message_size, "point 2 args", renderer->counts.point)) {
        return 0;
    }
    return 1;
}
