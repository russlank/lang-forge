# Scanner Encoding Architecture

Document id: `lang-forge-encoding-v1`

Status: `active`

Last updated: `2026-07-01`

Owner: `Project maintainers`

Scope: `Scanner source encoding and Unicode support model`

LangForge's first scanner implementation was byte-oriented. The active scanner
architecture is now encoding-aware, with UTF-8 as the first generated source
encoding for the in-process engine and generated Go, C#, C, and C++ backends.

## Goals

- Treat UTF-8 source text as the normal path for generated scanners.
- Build regex, NFA, DFA, and minimization logic over Unicode scalar values.
- Keep generated scanner APIs deterministic and target-neutral across Go, C#,
  C, and C++.
- Preserve byte offsets for slicing and tooling while also reporting line and
  column positions for diagnostics.
- Keep sparse range-transition tables so Unicode support does not imply giant
  dense tables.

## Encoding Boundary

The core scanner should not operate directly on target-specific string
internals. Instead, each generated runtime should decode its source input into
scanner symbols:

```text
source bytes/string
  -> decoder
  -> Unicode scalar + byte offset + line/column
  -> DFA transition
  -> token span
```

UTF-8 is first because it is the natural source encoding for Go, C, modern
tools, and `.lf` files. Other encodings should be added as decoder adapters,
not by changing DFA semantics.

## Regex Domain

In UTF-8 mode:

- literals may contain non-ASCII characters;
- character classes may contain Unicode scalar ranges;
- negated classes are evaluated over valid Unicode scalar values;
- invalid code points, including surrogate code points, are rejected;
- malformed UTF-8 input is an error by default.

Implemented regex support includes:

- `\uXXXX`;
- `\UXXXXXXXX` and `\u{...}`;
- Unicode properties such as `\p{L}` and `\P{Number}`;
- diagnostics for invalid ranges, surrogate escapes, oversized code points,
  and unsupported properties.

## Generated Runtime Expectations

Go runtime:

- decode UTF-8 with checked rune decoding;
- transition on `rune` ranges;
- keep byte offsets for `Text` slicing.

C# runtime:

- decode .NET strings with `System.Text.Rune`;
- reject malformed surrogate sequences;
- keep UTF-16 string offsets plus scalar line/column positions.

C runtime:

- include a small checked UTF-8 decoder;
- preserve byte offsets and scalar line/column positions on lexemes;
- report malformed input as a scanner error.

C++ runtime:

- include a small checked UTF-8 decoder over caller-owned `std::string_view`
  input;
- preserve byte offsets and scalar line/column positions on lexemes;
- report malformed input as a scanner error without advancing indefinitely.

## Implementation Status

- `.lf` specs can declare `%scanner utf8` or structured scanner settings such
  as `%scanner encoding=utf8 invalid=error newline=lf`.
- Table JSON and manifests record scanner encoding and sparse domain metadata.
- Regex, NFA, DFA, minimization, in-process matching, and generated
  Go/C#/C/C++ matching operate on Unicode scalar values.
- Generated Go lexemes include byte offsets plus scalar line/column positions.
- Generated C# lexemes include UTF-16 string offsets plus scalar line/column
  positions.
- Generated C lexemes include byte offsets plus scalar line/column positions.
- Generated C++ lexemes include byte offsets plus scalar line/column positions.
- Malformed UTF-8 input in Go/C/C++ and malformed UTF-16 surrogate input in C#
  are reported as scanner errors.

Remaining work:

- broaden Unicode property aliases as real users need them;
- add future decoder adapters for non-UTF-8 encodings without changing DFA
  semantics.
