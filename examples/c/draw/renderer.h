#ifndef LANGFORGE_EXAMPLES_C_DRAW_RENDERER_H
#define LANGFORGE_EXAMPLES_C_DRAW_RENDERER_H

#include "ast.h"

/** Runtime variable stored by the interpreter while rendering. */
typedef struct draw_var {
    char *name;
    double value;
    struct draw_var *next;
} draw_var;

/** Runtime named figure binding. The AST owns the block; the renderer owns
 * only this lookup node. */
typedef struct draw_named_figure {
    char *name;
    draw_figure_block *block;
    struct draw_named_figure *next;
} draw_named_figure;

/** Operation counters used by assertions and deterministic reports. */
typedef struct draw_counts {
    int canvas;
    int background;
    int point;
    int line;
    int box;
    int circle;
    int define;
} draw_counts;

/** Renderer/interpreter state for one DRAW program execution. */
typedef struct draw_renderer {
    demo_image image;
    int has_image;
    draw_var *vars;
    draw_named_figure *figures;
    draw_color stroke;
    draw_color fill;
    draw_color background;
    int fill_on;
    double line_width;
    draw_counts counts;
} draw_renderer;

/** Initializes default style and empty runtime state. */
void draw_renderer_init(draw_renderer *renderer);

/** Releases image memory and renderer-owned lookup nodes. */
void draw_renderer_free(draw_renderer *renderer);

/** Interprets a parsed DRAW program into the renderer image and counters. */
int draw_render(draw_program *program, draw_renderer *renderer, char *message, size_t message_size);

/** Counts currently visible runtime variables. */
size_t draw_var_count(draw_renderer *renderer);

/** Counts currently defined figures. */
size_t draw_figure_count(draw_renderer *renderer);

#endif
