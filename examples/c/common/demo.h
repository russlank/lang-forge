#ifndef LANGFORGE_EXAMPLES_C_DEMO_H
#define LANGFORGE_EXAMPLES_C_DEMO_H

#include <stddef.h>

/*
 * Small support library shared by the C examples.
 *
 * The generated LangForge scanner and parser stay dependency-free. These
 * helpers live outside generated code and provide only demo concerns: reading
 * sample input files, collecting reducer-owned values, writing reports, and
 * writing a tiny RGB PNG for the DRAW example.
 */

typedef struct demo_buffer {
    char *data;
    size_t length;
} demo_buffer;

typedef struct demo_arena_node {
    void *ptr;
    struct demo_arena_node *next;
} demo_arena_node;

typedef struct demo_arena {
    demo_arena_node *head;
} demo_arena;

typedef struct demo_text {
    char *data;
    size_t length;
    size_t capacity;
} demo_text;

typedef struct demo_image {
    int width;
    int height;
    unsigned char *rgb;
} demo_image;

int demo_set_error(char *error, size_t error_size, const char *format, ...);

int demo_read_file(const char *path, demo_buffer *out, char *error, size_t error_size);
void demo_free_buffer(demo_buffer *buffer);

int demo_write_text(const char *path, const char *text, char *error, size_t error_size);
int demo_ensure_parent_dir(const char *path, char *error, size_t error_size);

void *demo_arena_alloc(demo_arena *arena, size_t size);
char *demo_arena_copy(demo_arena *arena, const char *data, size_t length);
char *demo_arena_copy_cstr(demo_arena *arena, const char *value);
void demo_arena_free(demo_arena *arena);

int demo_text_append(demo_text *text, const char *value, char *error, size_t error_size);
int demo_text_appendf(demo_text *text, char *error, size_t error_size, const char *format, ...);
void demo_text_free(demo_text *text);

int demo_image_init(demo_image *image, int width, int height, char *error, size_t error_size);
void demo_image_free(demo_image *image);
void demo_image_fill(demo_image *image, unsigned char r, unsigned char g, unsigned char b);
void demo_image_set_pixel(demo_image *image, int x, int y, unsigned char r, unsigned char g, unsigned char b);
void demo_image_line(demo_image *image, int x0, int y0, int x1, int y1, unsigned char r, unsigned char g, unsigned char b);
void demo_image_rect(demo_image *image, int x, int y, int width, int height, unsigned char r, unsigned char g, unsigned char b);
void demo_image_circle(demo_image *image, int cx, int cy, int radius, unsigned char r, unsigned char g, unsigned char b);
int demo_write_png(const char *path, const demo_image *image, char *error, size_t error_size);

#endif
