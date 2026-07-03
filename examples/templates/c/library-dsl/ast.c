#include "ast.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static char *dsl_strdup(const char *text) {
    size_t length = text == NULL ? 0 : strlen(text);
    char *copy = (char *)calloc(length + 1, 1);
    if (copy != NULL && text != NULL) {
        memcpy(copy, text, length);
    }
    return copy;
}

static dsl_value *dsl_value_create(dsl_value_kind kind) {
    dsl_value *value = (dsl_value *)calloc(1, sizeof(dsl_value));
    if (value != NULL) {
        value->kind = kind;
    }
    return value;
}

dsl_value *dsl_value_number(int number) {
    dsl_value *value = dsl_value_create(DSL_VALUE_NUMBER);
    if (value != NULL) {
        value->number = number;
    }
    return value;
}

dsl_value *dsl_value_string(const char *text) {
    dsl_value *value = dsl_value_create(DSL_VALUE_STRING);
    if (value != NULL) {
        value->text = dsl_strdup(text);
        if (value->text == NULL) {
            dsl_value_free(value);
            return NULL;
        }
    }
    return value;
}

dsl_value *dsl_value_ident(const char *text) {
    dsl_value *value = dsl_value_create(DSL_VALUE_IDENT);
    if (value != NULL) {
        value->text = dsl_strdup(text);
        if (value->text == NULL) {
            dsl_value_free(value);
            return NULL;
        }
    }
    return value;
}

dsl_value *dsl_value_bool(int value) {
    dsl_value *out = dsl_value_create(DSL_VALUE_BOOL);
    if (out != NULL) {
        out->boolean = value ? 1 : 0;
    }
    return out;
}

void dsl_value_free(dsl_value *value) {
    if (value == NULL) {
        return;
    }
    free(value->text);
    free(value);
}

static dsl_entry *dsl_entry_create(dsl_entry_kind kind, const char *name, dsl_value *value) {
    dsl_entry *entry = (dsl_entry *)calloc(1, sizeof(dsl_entry));
    if (entry == NULL) {
        dsl_value_free(value);
        return NULL;
    }
    entry->kind = kind;
    entry->name = dsl_strdup(name);
    entry->value = value;
    if (entry->name == NULL || entry->value == NULL) {
        dsl_entry_free_all(entry);
        return NULL;
    }
    return entry;
}

dsl_entry *dsl_entry_set(const char *name, dsl_value *value) {
    return dsl_entry_create(DSL_ENTRY_SET, name, value);
}

dsl_entry *dsl_entry_enable(const char *name, dsl_value *value) {
    return dsl_entry_create(DSL_ENTRY_ENABLE, name, value);
}

void dsl_entry_free_all(dsl_entry *entry) {
    while (entry != NULL) {
        dsl_entry *next = entry->next;
        free(entry->name);
        dsl_value_free(entry->value);
        free(entry);
        entry = next;
    }
}

dsl_entry_list *dsl_entry_list_empty(void) {
    return (dsl_entry_list *)calloc(1, sizeof(dsl_entry_list));
}

dsl_entry_list *dsl_entry_list_prepend(dsl_entry *head, dsl_entry_list *tail) {
    if (head == NULL) {
        dsl_entry_list_free_shallow(tail);
        return NULL;
    }
    if (tail == NULL) {
        tail = dsl_entry_list_empty();
        if (tail == NULL) {
            dsl_entry_free_all(head);
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

void dsl_entry_list_free_shallow(dsl_entry_list *list) {
    free(list);
}

dsl_document *dsl_document_create(dsl_entry_list *entries) {
    dsl_document *document = (dsl_document *)calloc(1, sizeof(dsl_document));
    if (document == NULL) {
        if (entries != NULL) {
            dsl_entry_free_all(entries->head);
            dsl_entry_list_free_shallow(entries);
        }
        return NULL;
    }
    if (entries != NULL) {
        document->entries = entries->head;
        dsl_entry_list_free_shallow(entries);
    }
    return document;
}

void dsl_document_free(dsl_document *document) {
    if (document == NULL) {
        return;
    }
    dsl_entry_free_all(document->entries);
    free(document);
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
