#include "semantics.hpp"

#include <stdexcept>
#include <string>
#include <utility>

namespace lfgen = LangForge::Examples::Templates::LibraryDsl::Generated;

namespace library_dsl {

static std::vector<Entry> prepend(Entry head, std::vector<Entry> tail) {
    std::vector<Entry> out;
    out.reserve(tail.size() + 1);
    out.push_back(std::move(head));
    out.insert(out.end(), tail.begin(), tail.end());
    return out;
}

static std::string unquote(std::string text) {
    if (text.size() < 2 || text.front() != '"' || text.back() != '"') {
        throw std::runtime_error("string literal is not quoted: " + text);
    }
    std::string out;
    for (std::size_t i = 1; i + 1 < text.size(); ++i) {
        if (text[i] == '\\') {
            ++i;
            if (i + 1 >= text.size()) {
                throw std::runtime_error("string literal ends with an escape: " + text);
            }
        }
        out.push_back(text[i]);
    }
    return out;
}

const lfgen::ReducerMap& make_reducers() {
    static const lfgen::ReducerMap reducers{
        // Document : entries=Entries {cpp: document}
        {lfgen::SemanticAction::Document, lfgen::typed_document([](const lfgen::DocumentReduction& ctx) -> Document {
            return Document{ctx.entries};
        })},

        // Entries : head=Entry tail=EntriesTail {cpp: entries}
        {lfgen::SemanticAction::Entries, lfgen::typed_entries([](const lfgen::EntriesReduction& ctx) -> std::vector<Entry> {
            return prepend(ctx.head, ctx.tail);
        })},

        // Entries : %empty {cpp: entries.empty}
        {lfgen::SemanticAction::EntriesEmpty, lfgen::typed_entries_empty([](const lfgen::EntriesEmptyReduction&) -> std::vector<Entry> {
            return {};
        })},

        // EntriesTail : head=Entry tail=EntriesTail {cpp: entries.tail.more}
        {lfgen::SemanticAction::EntriesTailMore, lfgen::typed_entries_tail_more([](const lfgen::EntriesTailMoreReduction& ctx) -> std::vector<Entry> {
            return prepend(ctx.head, ctx.tail);
        })},

        // EntriesTail : %empty {cpp: entries.tail.empty}
        {lfgen::SemanticAction::EntriesTailEmpty, lfgen::typed_entries_tail_empty([](const lfgen::EntriesTailEmptyReduction&) -> std::vector<Entry> {
            return {};
        })},

        // Entry : Set name=Ident Assign value=Value Semi {cpp: entry.set}
        {lfgen::SemanticAction::EntrySet, lfgen::typed_entry_set([](const lfgen::EntrySetReduction& ctx) -> Entry {
            return Entry{EntryKind::Set, std::string(ctx.name.text), ctx.value};
        })},

        // Entry : Enable name=Ident Semi {cpp: entry.enable}
        {lfgen::SemanticAction::EntryEnable, lfgen::typed_entry_enable([](const lfgen::EntryEnableReduction& ctx) -> Entry {
            return Entry{EntryKind::Enable, std::string(ctx.name.text), Value::bool_value(true)};
        })},

        // Value : token=Number {cpp: value.number}
        {lfgen::SemanticAction::ValueNumber, lfgen::typed_value_number([](const lfgen::ValueNumberReduction& ctx) -> Value {
            const std::string text(ctx.token.text);
            try {
                std::size_t consumed = 0;
                const int value = std::stoi(text, &consumed);
                if (consumed != text.size()) {
                    throw std::invalid_argument("trailing characters");
                }
                return Value::number_value(value);
            } catch (const std::exception& ex) {
                throw std::runtime_error("rule " + std::to_string(ctx.reduction.rule) +
                                         " action " + std::string(ctx.reduction.action) +
                                         " label token value " + text +
                                         " is not a valid int: " + ex.what());
            }
        })},

        // Value : token=String {cpp: value.string}
        {lfgen::SemanticAction::ValueString, lfgen::typed_value_string([](const lfgen::ValueStringReduction& ctx) -> Value {
            return Value::string_value(unquote(std::string(ctx.token.text)));
        })},

        // Value : token=Ident {cpp: value.ident}
        {lfgen::SemanticAction::ValueIdent, lfgen::typed_value_ident([](const lfgen::ValueIdentReduction& ctx) -> Value {
            return Value::identifier_value(std::string(ctx.token.text));
        })},
    };
    return reducers;
}

} // namespace library_dsl
