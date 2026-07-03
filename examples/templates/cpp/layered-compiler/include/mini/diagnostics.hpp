#pragma once

#include <optional>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

namespace mini {

/// Minimal expected-like result for C++17, where std::expected is unavailable.
template <typename T>
class Result {
public:
    static Result success(T value) {
        Result out;
        out.value_.emplace(std::move(value));
        return out;
    }

    static Result failure(std::vector<std::string> diagnostics) {
        Result out;
        out.diagnostics_ = std::move(diagnostics);
        return out;
    }

    bool ok() const noexcept {
        return value_.has_value();
    }

    explicit operator bool() const noexcept {
        return ok();
    }

    T& value() {
        if (!value_) {
            throw std::logic_error("Result has no value");
        }
        return *value_;
    }

    const T& value() const {
        if (!value_) {
            throw std::logic_error("Result has no value");
        }
        return *value_;
    }

    T&& take_value() {
        if (!value_) {
            throw std::logic_error("Result has no value");
        }
        return std::move(*value_);
    }

    const std::vector<std::string>& diagnostics() const noexcept {
        return diagnostics_;
    }

private:
    std::optional<T> value_;
    std::vector<std::string> diagnostics_;
};

std::vector<std::string> diagnostic_from_exception(const std::exception& ex);

} // namespace mini
