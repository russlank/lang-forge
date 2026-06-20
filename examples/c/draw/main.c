#include "../common/demo.h"
#include "generated/parser.h"
#include "parser_adapter.h"
#include "renderer.h"
#include "report.h"

#include <stdio.h>
#include <string.h>

/* High-level demo pipeline:
 * source text -> generated scanner/parser -> typed DRAW AST -> renderer -> PNG
 * and deterministic text report. The detailed parser and renderer mechanics
 * live in separate modules so this file stays focused on CLI orchestration. */
static int draw_render_source(draw_context *ctx, const char *source, const char *input_path, const char *output_path, draw_renderer *renderer, demo_text *report, char *message, size_t message_size) {
    draw_program *program = NULL;
    if (!draw_parse_source(ctx, source, &program, message, message_size) ||
        !draw_render(program, renderer, message, message_size) ||
        !demo_write_png(output_path, &renderer->image, message, message_size) ||
        !draw_build_report(renderer, input_path, output_path, report, message, message_size)) {
        return 0;
    }
    return 1;
}

/* Assertions exercise the real generated scanner/parser and handwritten
 * semantic pipeline instead of checking a mocked result. That makes the example
 * useful as a compact integration test for future backend changes. */
static int draw_run_assertions(const char *source, const char *output_path, char *message, size_t message_size) {
    draw_context ctx;
    draw_renderer renderer;
    demo_text report = {0};
    draw_error error;
    draw_lexeme *tokens = NULL;
    size_t count = 0;
    draw_context_init(&ctx);
    error.message[0] = '\0';
    if (!draw_render_source(&ctx, source, "sample.draw", output_path, &renderer, &report, message, message_size)) {
        demo_text_free(&report);
        draw_context_free(&ctx);
        return 0;
    }
    if (renderer.image.width != 960 || renderer.image.height != 640 || renderer.counts.line != 90 || renderer.counts.circle != 196 || renderer.counts.box != 2) {
        draw_renderer_free(&renderer);
        demo_text_free(&report);
        draw_context_free(&ctx);
        return demo_set_error(message, message_size, "unexpected draw render summary");
    }
    draw_renderer_free(&renderer);
    demo_text_free(&report);
    draw_context_free(&ctx);
    if (draw_tokenize("canvas 1, @", &tokens, &count, &error)) {
        draw_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected scanner failure");
    }
    if (!draw_tokenize("draw ;", &tokens, &count, &error)) {
        return demo_set_error(message, message_size, "unexpected scanner failure: %s", error.message);
    }
    if (draw_parse(tokens, count, &error)) {
        draw_free_lexemes(tokens);
        return demo_set_error(message, message_size, "expected parser failure");
    }
    draw_free_lexemes(tokens);
    return 1;
}

static const char *draw_read_option(int *argc, char **argv, const char *name, const char *fallback) {
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

static int draw_take_flag(int *argc, char **argv, const char *name) {
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
    demo_text report = {0};
    draw_context ctx;
    draw_renderer renderer;
    int assert_mode = draw_take_flag(&argc, argv, "--assert");
    const char *output_path = draw_read_option(&argc, argv, "--output", "dist/sample-c.png");
    const char *log_path = draw_read_option(&argc, argv, "--log", "dist/draw-c-demo.log");
    const char *input_path = argc > 1 ? argv[1] : "sample.draw";
    draw_context_init(&ctx);
    if (!demo_read_file(input_path, &source, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        draw_context_free(&ctx);
        return 1;
    }
    if (assert_mode && !draw_run_assertions(source.data, output_path, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        draw_context_free(&ctx);
        return 1;
    }
    if (!draw_render_source(&ctx, source.data, input_path, output_path, &renderer, &report, message, sizeof(message)) ||
        !demo_write_text(log_path, report.data, message, sizeof(message))) {
        fprintf(stderr, "%s\n", message);
        demo_free_buffer(&source);
        demo_text_free(&report);
        draw_context_free(&ctx);
        return 1;
    }
    printf("%s", report.data);
    draw_renderer_free(&renderer);
    demo_free_buffer(&source);
    demo_text_free(&report);
    draw_context_free(&ctx);
    return 0;
}
