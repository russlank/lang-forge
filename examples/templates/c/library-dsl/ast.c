#include "ast.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef struct dsl_allocation {
    void *ptr;
    struct dsl_allocation *next;
} dsl_allocation;

struct dsl_allocator {
    dsl_allocation *items;
};

dsl_allocator *dsl_allocator_create(void) {
    return (dsl_allocator *)calloc(1, sizeof(dsl_allocator));
}

void dsl_allocator_destroy(dsl_allocator *memory) {
    dsl_allocation *item = NULL;
    if (memory == NULL) {
        return;
    }
    item = memory->items;
    while (item != NULL) {
        dsl_allocation *next = item->next;
        free(item->ptr);
        free(item);
        item = next;
    }
    free(memory);
}

void *dsl_allocator_alloc(dsl_allocator *memory, size_t size) {
    dsl_allocation *item = NULL;
    void *ptr = NULL;
    if (memory == NULL || size == 0) {
        return NULL;
    }
    ptr = calloc(1, size);
    item = (dsl_allocation *)calloc(1, sizeof(dsl_allocation));
    if (ptr == NULL || item == NULL) {
        free(ptr);
        free(item);
        return NULL;
    }
    item->ptr = ptr;
    item->next = memory->items;
    memory->items = item;
    return ptr;
}

char *dsl_allocator_copy_span(dsl_allocator *memory, const char *text, size_t length) {
    char *copy = (char *)dsl_allocator_alloc(memory, length + 1);
    if (copy != NULL && text != NULL) {
        memcpy(copy, text, length);
    }
    return copy;
}

char *dsl_allocator_copy(dsl_allocator *memory, const char *text) {
    return dsl_allocator_copy_span(memory, text, text == NULL ? 0 : strlen(text));
}

static dsl_value *dsl_value_create(dsl_allocator *memory, dsl_value_kind kind) {
    dsl_value *value = (dsl_value *)dsl_allocator_alloc(memory, sizeof(dsl_value));
    if (value != NULL) {
        value->kind = kind;
    }
    return value;
}

dsl_value *dsl_value_number(dsl_allocator *memory, int number) {
    dsl_value *value = dsl_value_create(memory, DSL_VALUE_NUMBER);
    if (value != NULL) {
        value->number = number;
    }
    return value;
}

dsl_value *dsl_value_string(dsl_allocator *memory, const char *text) {
    dsl_value *value = dsl_value_create(memory, DSL_VALUE_STRING);
    if (value != NULL) {
        value->text = dsl_allocator_copy(memory, text);
        if (value->text == NULL) {
            return NULL;
        }
    }
    return value;
}

dsl_value *dsl_value_ident(dsl_allocator *memory, const char *text) {
    dsl_value *value = dsl_value_create(memory, DSL_VALUE_IDENT);
    if (value != NULL) {
        value->text = dsl_allocator_copy(memory, text);
        if (value->text == NULL) {
            return NULL;
        }
    }
    return value;
}

dsl_value *dsl_value_bool(dsl_allocator *memory, int value) {
    dsl_value *out = dsl_value_create(memory, DSL_VALUE_BOOL);
    if (out != NULL) {
        out->boolean = value ? 1 : 0;
    }
    return out;
}

static dsl_entry *dsl_entry_create(dsl_allocator *memory, dsl_entry_kind kind, const char *name, dsl_value *value) {
    dsl_entry *entry = (dsl_entry *)dsl_allocator_alloc(memory, sizeof(dsl_entry));
    if (entry == NULL || value == NULL) {
        return NULL;
    }
    entry->kind = kind;
    entry->name = dsl_allocator_copy(memory, name);
    entry->value = value;
    if (entry->name == NULL) {
        return NULL;
    }
    return entry;
}

dsl_entry *dsl_entry_set(dsl_allocator *memory, const char *name, dsl_value *value) {
    return dsl_entry_create(memory, DSL_ENTRY_SET, name, value);
}

dsl_entry *dsl_entry_enable(dsl_allocator *memory, const char *name, dsl_value *value) {
    return dsl_entry_create(memory, DSL_ENTRY_ENABLE, name, value);
}

dsl_entry_list *dsl_entry_list_empty(dsl_allocator *memory) {
    return (dsl_entry_list *)dsl_allocator_alloc(memory, sizeof(dsl_entry_list));
}

dsl_entry_list *dsl_entry_list_prepend(dsl_allocator *memory, dsl_entry *head, dsl_entry_list *tail) {
    if (head == NULL) {
        return NULL;
    }
    if (tail == NULL) {
        tail = dsl_entry_list_empty(memory);
        if (tail == NULL) {
            return NULL;
        }
    }
    head->next = tail->head;
    tail->head = head;
    if (tail->tail == NULL) {
        tail->tail = head;
    }
    return tail;
}

dsl_document *dsl_document_create(dsl_allocator *memory, dsl_entry_list *entries) {
    dsl_document *document = (dsl_document *)dsl_allocator_alloc(memory, sizeof(dsl_document));
    if (document == NULL) {
        return NULL;
    }
    document->memory = memory;
    if (entries != NULL) {
        document->entries = entries->head;
    }
    return document;
}

void dsl_document_free(dsl_document *document) {
    dsl_allocator *memory = NULL;
    if (document == NULL) {
        return;
    }
    memory = document->memory;
    document->memory = NULL;
    dsl_allocator_destroy(memory);
}

const char *dsl_entry_kind_name(dsl_entry_kind kind) {
    return kind == DSL_ENTRY_ENABLE ? "enable" : "set";
}

void dsl_value_format(const dsl_value *value, char *buffer, size_t size) {
    if (value == NULL) {
        snprintf(buffer, size, "<null>");
        return;
    }
    switch (value->kind) {
    case DSL_VALUE_NUMBER:
        snprintf(buffer, size, "%d", value->number);
        break;
    case DSL_VALUE_STRING:
        snprintf(buffer, size, "\"%s\"", value->text == NULL ? "" : value->text);
        break;
    case DSL_VALUE_IDENT:
        snprintf(buffer, size, "%s", value->text == NULL ? "" : value->text);
        break;
    case DSL_VALUE_BOOL:
        snprintf(buffer, size, "%s", value->boolean ? "true" : "false");
        break;
    default:
        snprintf(buffer, size, "<unknown>");
        break;
    }
}
