#include "ast.h"

#include <string.h>

void draw_context_init(draw_context *ctx) {
    memset(ctx, 0, sizeof(*ctx));
}

void draw_context_free(draw_context *ctx) {
    demo_arena_free(&ctx->arena);
}
