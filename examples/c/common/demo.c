#include "demo.h"

#include <errno.h>
#include <stdarg.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifdef _WIN32
#include <direct.h>
#define DEMO_MKDIR(path) _mkdir(path)
#else
#include <sys/stat.h>
#include <sys/types.h>
#define DEMO_MKDIR(path) mkdir(path, 0755)
#endif

int demo_set_error(char *error, size_t error_size, const char *format, ...) {
    if (error != NULL && error_size > 0) {
        va_list args;
        va_start(args, format);
        vsnprintf(error, error_size, format, args);
        va_end(args);
    }
    return 0;
}

int demo_read_file(const char *path, demo_buffer *out, char *error, size_t error_size) {
    FILE *file = fopen(path, "rb");
    long length = 0;
    size_t read_count = 0;
    char *data = NULL;
    if (file == NULL) {
        return demo_set_error(error, error_size, "open %s: %s", path, strerror(errno));
    }
    if (fseek(file, 0, SEEK_END) != 0 || (length = ftell(file)) < 0 || fseek(file, 0, SEEK_SET) != 0) {
        fclose(file);
        return demo_set_error(error, error_size, "measure %s: %s", path, strerror(errno));
    }
    data = (char *)malloc((size_t)length + 1);
    if (data == NULL) {
        fclose(file);
        return demo_set_error(error, error_size, "out of memory reading %s", path);
    }
    read_count = fread(data, 1, (size_t)length, file);
    if (read_count != (size_t)length) {
        int read_error = ferror(file);
        int saved_errno = errno;
        free(data);
        fclose(file);
        return demo_set_error(error, error_size, "read %s: %s", path, read_error ? strerror(saved_errno) : "short read");
    }
    fclose(file);
    data[length] = '\0';
    out->data = data;
    out->length = (size_t)length;
    return 1;
}

void demo_free_buffer(demo_buffer *buffer) {
    if (buffer != NULL) {
        free(buffer->data);
        buffer->data = NULL;
        buffer->length = 0;
    }
}

static int demo_is_separator(char ch) {
    return ch == '/' || ch == '\\';
}

int demo_ensure_parent_dir(const char *path, char *error, size_t error_size) {
    size_t length = strlen(path);
    char *copy = (char *)malloc(length + 1);
    size_t i = 0;
    if (copy == NULL) {
        return demo_set_error(error, error_size, "out of memory creating parent directories");
    }
    memcpy(copy, path, length + 1);
    for (i = 1; i < length; i++) {
        if (!demo_is_separator(copy[i])) {
            continue;
        }
        copy[i] = '\0';
        if (copy[0] != '\0' && DEMO_MKDIR(copy) != 0 && errno != EEXIST) {
            int saved = errno;
            free(copy);
            return demo_set_error(error, error_size, "mkdir %s: %s", path, strerror(saved));
        }
        copy[i] = '/';
    }
    free(copy);
    return 1;
}

int demo_write_text(const char *path, const char *text, char *error, size_t error_size) {
    FILE *file = NULL;
    size_t length = strlen(text);
    if (!demo_ensure_parent_dir(path, error, error_size)) {
        return 0;
    }
    file = fopen(path, "wb");
    if (file == NULL) {
        return demo_set_error(error, error_size, "create %s: %s", path, strerror(errno));
    }
    if (fwrite(text, 1, length, file) != length) {
        fclose(file);
        return demo_set_error(error, error_size, "write %s: %s", path, strerror(errno));
    }
    if (fclose(file) != 0) {
        return demo_set_error(error, error_size, "close %s: %s", path, strerror(errno));
    }
    return 1;
}

void *demo_arena_alloc(demo_arena *arena, size_t size) {
    demo_arena_node *node = NULL;
    void *ptr = calloc(1, size == 0 ? 1 : size);
    if (ptr == NULL) {
        return NULL;
    }
    node = (demo_arena_node *)malloc(sizeof(demo_arena_node));
    if (node == NULL) {
        free(ptr);
        return NULL;
    }
    node->ptr = ptr;
    node->next = arena->head;
    arena->head = node;
    return ptr;
}

char *demo_arena_copy(demo_arena *arena, const char *data, size_t length) {
    char *copy = (char *)demo_arena_alloc(arena, length + 1);
    if (copy == NULL) {
        return NULL;
    }
    memcpy(copy, data, length);
    copy[length] = '\0';
    return copy;
}

char *demo_arena_copy_cstr(demo_arena *arena, const char *value) {
    return demo_arena_copy(arena, value, strlen(value));
}

void demo_arena_free(demo_arena *arena) {
    demo_arena_node *node = arena->head;
    while (node != NULL) {
        demo_arena_node *next = node->next;
        free(node->ptr);
        free(node);
        node = next;
    }
    arena->head = NULL;
}

int demo_text_append(demo_text *text, const char *value, char *error, size_t error_size) {
    size_t value_length = strlen(value);
    size_t required = text->length + value_length + 1;
    if (required > text->capacity) {
        size_t next_capacity = text->capacity == 0 ? 256 : text->capacity;
        char *next = NULL;
        while (next_capacity < required) {
            next_capacity *= 2;
        }
        next = (char *)realloc(text->data, next_capacity);
        if (next == NULL) {
            return demo_set_error(error, error_size, "out of memory growing report");
        }
        text->data = next;
        text->capacity = next_capacity;
    }
    memcpy(text->data + text->length, value, value_length + 1);
    text->length += value_length;
    return 1;
}

int demo_text_appendf(demo_text *text, char *error, size_t error_size, const char *format, ...) {
    va_list args;
    va_list copy;
    int needed = 0;
    char *buffer = NULL;
    int ok = 0;
    va_start(args, format);
    va_copy(copy, args);
    needed = vsnprintf(NULL, 0, format, copy);
    va_end(copy);
    if (needed < 0) {
        va_end(args);
        return demo_set_error(error, error_size, "format report text failed");
    }
    buffer = (char *)malloc((size_t)needed + 1);
    if (buffer == NULL) {
        va_end(args);
        return demo_set_error(error, error_size, "out of memory formatting report");
    }
    vsnprintf(buffer, (size_t)needed + 1, format, args);
    va_end(args);
    ok = demo_text_append(text, buffer, error, error_size);
    free(buffer);
    return ok;
}

void demo_text_free(demo_text *text) {
    if (text != NULL) {
        free(text->data);
        text->data = NULL;
        text->length = 0;
        text->capacity = 0;
    }
}

int demo_image_init(demo_image *image, int width, int height, char *error, size_t error_size) {
    size_t pixels = 0;
    if (width <= 0 || height <= 0 || width > 4096 || height > 4096) {
        return demo_set_error(error, error_size, "invalid image size %dx%d", width, height);
    }
    pixels = (size_t)width * (size_t)height;
    if (pixels > ((size_t)-1) / 3) {
        return demo_set_error(error, error_size, "image is too large");
    }
    image->rgb = (unsigned char *)calloc(pixels * 3, 1);
    if (image->rgb == NULL) {
        return demo_set_error(error, error_size, "out of memory allocating image");
    }
    image->width = width;
    image->height = height;
    return 1;
}

void demo_image_free(demo_image *image) {
    if (image != NULL) {
        free(image->rgb);
        image->rgb = NULL;
        image->width = 0;
        image->height = 0;
    }
}

void demo_image_fill(demo_image *image, unsigned char r, unsigned char g, unsigned char b) {
    int x = 0;
    int y = 0;
    for (y = 0; y < image->height; y++) {
        for (x = 0; x < image->width; x++) {
            demo_image_set_pixel(image, x, y, r, g, b);
        }
    }
}

void demo_image_set_pixel(demo_image *image, int x, int y, unsigned char r, unsigned char g, unsigned char b) {
    size_t offset = 0;
    if (x < 0 || y < 0 || x >= image->width || y >= image->height || image->rgb == NULL) {
        return;
    }
    offset = ((size_t)y * (size_t)image->width + (size_t)x) * 3;
    image->rgb[offset + 0] = r;
    image->rgb[offset + 1] = g;
    image->rgb[offset + 2] = b;
}

static int demo_abs_int(int value) {
    return value < 0 ? -value : value;
}

void demo_image_line(demo_image *image, int x0, int y0, int x1, int y1, unsigned char r, unsigned char g, unsigned char b) {
    int dx = demo_abs_int(x1 - x0);
    int sx = x0 < x1 ? 1 : -1;
    int dy = -demo_abs_int(y1 - y0);
    int sy = y0 < y1 ? 1 : -1;
    int err = dx + dy;
    while (1) {
        int e2 = 0;
        demo_image_set_pixel(image, x0, y0, r, g, b);
        if (x0 == x1 && y0 == y1) {
            break;
        }
        e2 = 2 * err;
        if (e2 >= dy) {
            err += dy;
            x0 += sx;
        }
        if (e2 <= dx) {
            err += dx;
            y0 += sy;
        }
    }
}

void demo_image_rect(demo_image *image, int x, int y, int width, int height, unsigned char r, unsigned char g, unsigned char b) {
    demo_image_line(image, x, y, x + width, y, r, g, b);
    demo_image_line(image, x, y + height, x + width, y + height, r, g, b);
    demo_image_line(image, x, y, x, y + height, r, g, b);
    demo_image_line(image, x + width, y, x + width, y + height, r, g, b);
}

void demo_image_circle(demo_image *image, int cx, int cy, int radius, unsigned char r, unsigned char g, unsigned char b) {
    int x = radius;
    int y = 0;
    int err = 0;
    while (x >= y) {
        demo_image_set_pixel(image, cx + x, cy + y, r, g, b);
        demo_image_set_pixel(image, cx + y, cy + x, r, g, b);
        demo_image_set_pixel(image, cx - y, cy + x, r, g, b);
        demo_image_set_pixel(image, cx - x, cy + y, r, g, b);
        demo_image_set_pixel(image, cx - x, cy - y, r, g, b);
        demo_image_set_pixel(image, cx - y, cy - x, r, g, b);
        demo_image_set_pixel(image, cx + y, cy - x, r, g, b);
        demo_image_set_pixel(image, cx + x, cy - y, r, g, b);
        y++;
        if (err <= 0) {
            err += 2 * y + 1;
        }
        if (err > 0) {
            x--;
            err -= 2 * x + 1;
        }
    }
}

static void demo_write_be32(FILE *file, uint32_t value) {
    fputc((int)((value >> 24) & 0xff), file);
    fputc((int)((value >> 16) & 0xff), file);
    fputc((int)((value >> 8) & 0xff), file);
    fputc((int)(value & 0xff), file);
}

static uint32_t demo_crc32_step(uint32_t crc, const unsigned char *data, size_t length) {
    size_t i = 0;
    for (i = 0; i < length; i++) {
        int bit = 0;
        crc ^= data[i];
        for (bit = 0; bit < 8; bit++) {
            uint32_t mask = (uint32_t)(-(int)(crc & 1u));
            crc = (crc >> 1) ^ (0xedb88320u & mask);
        }
    }
    return crc;
}

static uint32_t demo_adler32(const unsigned char *data, size_t length) {
    uint32_t a = 1;
    uint32_t b = 0;
    size_t i = 0;
    for (i = 0; i < length; i++) {
        a = (a + data[i]) % 65521u;
        b = (b + a) % 65521u;
    }
    return (b << 16) | a;
}

static int demo_write_chunk(FILE *file, const char type[4], const unsigned char *data, size_t length) {
    uint32_t crc = 0xffffffffu;
    demo_write_be32(file, (uint32_t)length);
    fwrite(type, 1, 4, file);
    if (length > 0) {
        fwrite(data, 1, length, file);
    }
    crc = demo_crc32_step(crc, (const unsigned char *)type, 4);
    if (length > 0) {
        crc = demo_crc32_step(crc, data, length);
    }
    demo_write_be32(file, crc ^ 0xffffffffu);
    return ferror(file) == 0;
}

static int demo_build_zlib_store(const unsigned char *raw, size_t raw_length, unsigned char **out, size_t *out_length) {
    size_t blocks = raw_length == 0 ? 1 : (raw_length + 65534u) / 65535u;
    size_t capacity = 2 + raw_length + blocks * 5 + 4;
    unsigned char *data = (unsigned char *)malloc(capacity);
    size_t pos = 0;
    size_t input = 0;
    if (data == NULL) {
        return 0;
    }
    data[pos++] = 0x78;
    data[pos++] = 0x01;
    while (input < raw_length || (raw_length == 0 && input == 0)) {
        size_t remaining = raw_length - input;
        uint16_t block_length = (uint16_t)(remaining > 65535u ? 65535u : remaining);
        uint16_t block_length_complement = (uint16_t)~block_length;
        int final = input + block_length >= raw_length;
        data[pos++] = (unsigned char)(final ? 0x01 : 0x00);
        data[pos++] = (unsigned char)(block_length & 0xff);
        data[pos++] = (unsigned char)((block_length >> 8) & 0xff);
        data[pos++] = (unsigned char)(block_length_complement & 0xff);
        data[pos++] = (unsigned char)((block_length_complement >> 8) & 0xff);
        if (block_length > 0) {
            memcpy(data + pos, raw + input, block_length);
            pos += block_length;
            input += block_length;
        } else {
            break;
        }
    }
    {
        uint32_t adler = demo_adler32(raw, raw_length);
        data[pos++] = (unsigned char)((adler >> 24) & 0xff);
        data[pos++] = (unsigned char)((adler >> 16) & 0xff);
        data[pos++] = (unsigned char)((adler >> 8) & 0xff);
        data[pos++] = (unsigned char)(adler & 0xff);
    }
    *out = data;
    *out_length = pos;
    return 1;
}

int demo_write_png(const char *path, const demo_image *image, char *error, size_t error_size) {
    static const unsigned char png_signature[8] = {137, 80, 78, 71, 13, 10, 26, 10};
    FILE *file = NULL;
    unsigned char ihdr[13];
    unsigned char *raw = NULL;
    unsigned char *zlib = NULL;
    size_t raw_stride = 0;
    size_t raw_length = 0;
    size_t zlib_length = 0;
    int y = 0;
    if (image == NULL || image->rgb == NULL || image->width <= 0 || image->height <= 0) {
        return demo_set_error(error, error_size, "invalid PNG image");
    }
    raw_stride = (size_t)image->width * 3u + 1u;
    raw_length = raw_stride * (size_t)image->height;
    raw = (unsigned char *)malloc(raw_length);
    if (raw == NULL) {
        return demo_set_error(error, error_size, "out of memory preparing PNG");
    }
    for (y = 0; y < image->height; y++) {
        size_t raw_offset = (size_t)y * raw_stride;
        size_t image_offset = (size_t)y * (size_t)image->width * 3u;
        raw[raw_offset] = 0;
        memcpy(raw + raw_offset + 1, image->rgb + image_offset, (size_t)image->width * 3u);
    }
    if (!demo_build_zlib_store(raw, raw_length, &zlib, &zlib_length)) {
        free(raw);
        return demo_set_error(error, error_size, "out of memory compressing PNG");
    }
    if (!demo_ensure_parent_dir(path, error, error_size)) {
        free(raw);
        free(zlib);
        return 0;
    }
    file = fopen(path, "wb");
    if (file == NULL) {
        free(raw);
        free(zlib);
        return demo_set_error(error, error_size, "create %s: %s", path, strerror(errno));
    }
    fwrite(png_signature, 1, sizeof(png_signature), file);
    ihdr[0] = (unsigned char)((image->width >> 24) & 0xff);
    ihdr[1] = (unsigned char)((image->width >> 16) & 0xff);
    ihdr[2] = (unsigned char)((image->width >> 8) & 0xff);
    ihdr[3] = (unsigned char)(image->width & 0xff);
    ihdr[4] = (unsigned char)((image->height >> 24) & 0xff);
    ihdr[5] = (unsigned char)((image->height >> 16) & 0xff);
    ihdr[6] = (unsigned char)((image->height >> 8) & 0xff);
    ihdr[7] = (unsigned char)(image->height & 0xff);
    ihdr[8] = 8;
    ihdr[9] = 2;
    ihdr[10] = 0;
    ihdr[11] = 0;
    ihdr[12] = 0;
    {
        int write_ok = demo_write_chunk(file, "IHDR", ihdr, sizeof(ihdr)) &&
            demo_write_chunk(file, "IDAT", zlib, zlib_length) &&
            demo_write_chunk(file, "IEND", NULL, 0);
        int close_ok = fclose(file) == 0;
        if (!write_ok || !close_ok) {
            free(raw);
            free(zlib);
            return demo_set_error(error, error_size, "write %s failed", path);
        }
    }
    free(raw);
    free(zlib);
    return 1;
}
