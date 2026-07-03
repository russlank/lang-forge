#include "library_dsl/ast.hpp"

#include <sstream>

namespace library_dsl {

Value Value::number_value(int value) {
    Value out;
    out.kind = ValueKind::Number;
    out.number = value;
    return out;
}

Value Value::string_value(std::string value) {
    Value out;
    out.kind = ValueKind::String;
    out.text = std::move(value);
    return out;
}

Value Value::identifier_value(std::string value) {
    Value out;
    out.kind = ValueKind::Identifier;
    out.text = std::move(value);
    return out;
}

Value Value::bool_value(bool value) {
    Value out;
    out.kind = ValueKind::Boolean;
    out.boolean = value;
    return out;
}

std::string Value::format() const {
    switch (kind) {
    case ValueKind::Number:
        return std::to_string(number);
    case ValueKind::String:
        return "\"" + text + "\"";
    case ValueKind::Identifier:
        return text;
    case ValueKind::Boolean:
        return boolean ? "true" : "false";
    }
    return "<unknown>";
}

std::map<std::string, Value> Document::settings() const {
    std::map<std::string, Value> out;
    for (const auto& entry : entries) {
        out[entry.name] = entry.value;
    }
    return out;
}

const char* entry_kind_name(EntryKind kind) noexcept {
    return kind == EntryKind::Enable ? "enable" : "set";
}

} // namespace library_dsl
