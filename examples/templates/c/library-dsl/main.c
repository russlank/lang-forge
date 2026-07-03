#include "parser_facade.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static char *read_file(const char *path) {
    FILE *file = fopen(path, "rb");
    long size = 0;
    char *data = NULL;
    if (file == NULL) {
        return NULL;
    }
    if (fseek(file, 0, SEEK_END) != 0 || (size = ftell(file)) < 0 || fseek(file, 0, SEEK_SET) != 0) {
        fclose(file);
        return NULL;
    }
    data = (char *)calloc((size_t)size + 1, 1);
    if (data == NULL) {
        fclose(file);
        return NULL;
    }
    if (fread(data, 1, (size_t)size, file) != (size_t)size) {
        free(data);
        fclose(file);
        return NULL;
    }
    fclose(file);
    return data;
}

static int write_file(const char *path, const char *text) {
    FILE *file = fopen(path, "wb");
    if (file == NULL) {
        return 0;
    }
    fputs(text, file);
    fclose(file);
    return 1;
}

static const char *read_option(int *argc, char **argv, const char *name, const char *fallback) {
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

static void append_report(char *buffer, size_t size, const char *input_path, const dsl_document *document) {
    size_t used = 0;
    const dsl_entry *entry = NULL;
    used += (size_t)snprintf(buffer + used, size - used, "Library DSL C template: %s\n", input_path);
    for (entry = document->entries; entry != NULL && used + 1 < size; entry = entry->next) {
        char value[128] = {0};
        dsl_value_format(entry->value, value, sizeof(value));
        used += (size_t)snprintf(buffer + used, size - used, "  %s %s = %s\n", dsl_entry_kind_name(entry->kind), entry->name, value);
    }
}

int main(int argc, char **argv) {
    char report[2048] = {0};
    dsl_parse_result parsed;
    const char *log_path = read_option(&argc, argv, "--log", "dist/library-c.log");
    const char *input_path = argc > 1 ? argv[1] : "input.dsl";
    char *source = read_file(input_path);
    dsl_parse_result_init(&parsed);
    if (source == NULL) {
        fprintf(stderr, "cannot read %s\n", input_path);
        return 1;
    }
    if (!dsl_parse_source(source, &parsed)) {
        fprintf(stderr, "%s\n", parsed.message);
        free(source);
        dsl_parse_result_free(&parsed);
        return 1;
    }
    append_report(report, sizeof(report), input_path, parsed.document);
    printf("%s", report);
    if (!write_file(log_path, report)) {
        fprintf(stderr, "cannot write %s\n", log_path);
        free(source);
        dsl_parse_result_free(&parsed);
        return 1;
    }
    free(source);
    dsl_parse_result_free(&parsed);
    return 0;
}
