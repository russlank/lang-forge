#pragma once

#include <map>
#include <string>
#include <vector>

namespace library_dsl {

/// Identifies which grammar alternative produced an Entry.
enum class EntryKind {
    /// Entry : Set name=Ident Assign value=Value Semi.
    Set,
    /// Entry : Enable name=Ident Semi.
    Enable,
};

/// Identifies which Value grammar alternative was reduced.
enum class ValueKind {
    /// Value : token=Number.
    Number,
    /// Value : token=String.
    String,
    /// Value : token=Ident.
    Identifier,
    /// Implicit value used by enable statements.
    Boolean,
};

/// Domain value carried by assignment and enable entries.
struct Value {
    ValueKind kind = ValueKind::Identifier;
    std::string text;
    int number = 0;
    bool boolean = false;

    static Value number_value(int value);
    static Value string_value(std::string value);
    static Value identifier_value(std::string value);
    static Value bool_value(bool value);
    std::string format() const;
};

/// One top-level DSL statement.
struct Entry {
    EntryKind kind = EntryKind::Set;
    std::string name;
    Value value;
};

/// Stable AST root returned by the parser facade.
struct Document {
    std::vector<Entry> entries;
    std::map<std::string, Value> settings() const;
};

const char* entry_kind_name(EntryKind kind) noexcept;

} // namespace library_dsl
