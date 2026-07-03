# Handwritten Integration Guide

Document id: `lang-forge-handwritten-integration-guide-v1`

Status: `active`

Last updated: `2026-07-01`

Owner: `Project maintainers`

Scope: `Beginner-friendly guide for the code a LangForge user writes beside .lf grammars and generated recognizers`

LangForge reads a grammar and generates scanner/parser machinery. That is a
large part of a compiler front end, but it is not the whole application.

The grammar answers:

- which tokens exist;
- which token sequences are valid;
- which grammar reduction labels should be raised.

Your handwritten code answers:

- what each reduction means;
- which AST, model, command, or value is produced;
- how diagnostics are shown;
- how parsing is exposed to the rest of an application;
- how generated code is built, tested, and packaged.

This guide focuses on that handwritten boundary. It is written for readers who
are new to compiler tools or new to one of the target languages.

## Mental Model

Keep this picture in mind:

```text
grammar.lf
  -> lang-forge generate
  -> generated scanner/parser tables and runtime
  -> handwritten reducer or semantic adapter
  -> handwritten application, library, compiler, renderer, or service
```

LangForge produces automata and small target-native runtimes. Your code should
wrap those generated APIs behind a useful domain API.

For example, a calculator grammar can say:

```lf
Expr : left=Expr Plus right=Term {go: add}
     | left=Expr Minus right=Term {go: subtract}
     ;
```

The generated parser knows that a reduction named `add` happened. It does not
know that addition means `left + right`. The handwritten reducer supplies that
behavior.

## What You Own

A practical LangForge project usually has these handwritten pieces:

| Piece | Purpose |
|---|---|
| `.lf` grammar | Tokens, parser rules, action labels, target package/namespace, semantic type declarations |
| Domain model | AST nodes, values, commands, report objects, or IR instructions |
| Semantic reducer | Code that maps generated action IDs such as `Add` to domain behavior |
| Parser facade | A small stable API such as `ParseProgram(source)` that hides generated details |
| Diagnostics policy | Error formatting, recovery reporting, semantic validation messages |
| Runtime/compiler layer | Interpreter, code generator, renderer, stack-machine lowering, or business logic |
| Tests | Valid inputs, invalid scanner/parser inputs, semantic failures, reducer coverage, concurrency |
| Build files | Makefile, project file, CMake/MSBuild/go test glue, generated-output hygiene |

Generated files should normally stay in `generated/` or `Generated/` and be
ignored by Git. Treat the `.lf` file and handwritten source as the source of
truth.

## The `.lf` Contract

The grammar is the contract between generated recognizers and handwritten
semantics. A useful reducer-mode grammar normally includes:

```lf
%target go
%package calc
%semantic go mode reducer
%semantic go type Expr float64
%semantic go type Term float64
%semantic go type Factor float64

%start Input
%token Number Plus Minus Star Slash LParen RParen

%% lexer
DIGIT = [0-9];
NUMBER = DIGIT+ ("." DIGIT+)?;

NUMBER => token(Number);
"+"    => token(Plus);
"-"    => token(Minus);
"*"    => token(Star);
"/"    => token(Slash);
"("    => token(LParen);
")"    => token(RParen);
[1-32]+ => skip;

%% parser
Input  : value=Expr {go: start} ;
Expr   : left=Expr Plus right=Term {go: add}
       | left=Expr Minus right=Term {go: subtract}
       | value=Term {go: pass}
       ;
Term   : left=Term Star right=Factor {go: multiply}
       | left=Term Slash right=Factor {go: divide}
       | value=Factor {go: pass}
       ;
Factor : token=Number {go: number}
       | LParen value=Expr RParen {go: group}
       ;
```

The important parts are:

- `%target` selects the generated backend.
- `%package` selects the package, namespace, or C symbol prefix.
- `%semantic <target> mode reducer` says grammar actions are labels for
  handwritten reducer code.
- `%semantic <target> type Nonterminal TypeName` tells the generator which
  semantic values are expected for nonterminals.
- `left=Expr` and `right=Term` are named RHS labels. Typed reducer contexts use
  them as field names.
- `{go: add}` is an action label. It is not embedded arithmetic code.

Use the equivalent target tag for each language:

```lf
{go: add}
{csharp: add}
{c: add}
{cpp: add}
```

## Integration Styles

Choose the smallest wrapper that matches your use case.

| Style | Good for | Handwritten shape |
|---|---|---|
| Demo or command | Examples, scripts, learning | `main` calls a generated scanner/source parser API with a reducer map, then prints output |
| Reusable library | DSL embedded in a larger app | Public facade hides generated package and returns domain types |
| Compiler pipeline | Real compiler/interpreter | Facade returns AST, then semantic validation, IR lowering, and execution happen in separate modules |
| Service with DI | C# applications, hosted services, tests | Interfaces for parser and semantics; generated code remains an implementation detail |
| Multiple parsers | Apps with several DSLs | Separate generated directories, namespaces/prefixes, and parser facades |

For anything beyond a tiny demo, prefer a parser facade. It keeps generated API
changes local and makes tests easier to read.

## Recommended Layouts

### Go

```text
mydsl/
  mydsl.lf
  generated/          # ignored; produced by LangForge
  model/              # AST and data-only semantic types
  semantics/          # reducer map
  parser/             # public facade, optional
  cmd/mydsl-demo/     # application entrypoint, optional
  Makefile
```

Put AST/model types in a package that does not import `generated`. Generated
code may import the model package when `%semantic go type` refers to those
types, so this avoids import cycles.

### C#

```text
MyDsl/
  Grammar/mydsl.lf
  Generated/          # ignored; contains *.g.cs files
  Ast/
  Semantics/
  Parsing/
    IMyDslParser.cs
    MyDslParser.cs
    ServiceCollectionExtensions.cs
  MyDsl.csproj
```

Generated C# files use `.g.cs` names. Keep handwritten code outside
`Generated/`. For library-style projects, expose interfaces from `Parsing/` and
return domain types from `Ast/`, not generated parser values.

### C

```text
mydsl/
  mydsl.lf
  generated/          # ignored; tokens.h, scanner.h/.c, parser.h/.c
  ast.h
  ast.c
  semantics.h
  semantics.c
  parser_adapter.h
  parser_adapter.c
  main.c              # optional
  Makefile
```

C generated names are prefixed from `%package`. Include generated headers with
paths such as `generated/parser.h`. That keeps generated headers as the single
source of truth while allowing IDEs to resolve types after generation.

### C++

```text
mydsl/
  mydsl.lf
  generated/          # ignored; tokens.hpp, scanner.hpp/.cpp, parser.hpp/.cpp
  include/mydsl/
    ast.hpp
    parser_facade.hpp
    semantics.hpp
  src/
    parser_facade.cpp
    semantics.cpp
    main.cpp          # optional
  Makefile or CMakeLists.txt
```

Use namespaces to separate generated code from handwritten domain code. The
examples use aliases such as:

```cpp
namespace lfcalc = LangForge::Examples::Calc::Generated;
```

## Go: Handwritten Reducer And Facade

Generated Go code exposes semantic action constants, typed reduction contexts,
typed adapter helpers, and `ReducerMap`.

A small reducer package can look like this:

```go
//go:build langforge_generated

package semantics

import (
	"fmt"
	"strconv"

	calc "example.com/mydsl/generated"
)

var reducers = calc.ReducerMap{
	// Shortened for the guide. A real reducer map must cover every generated
	// semantic action, or coverage validation will fail before parsing.
	// Grammar: Input : value=Expr {go: start}
	calc.SemanticActionStart: calc.TypedStart(func(ctx calc.StartReduction) (float64, error) {
		return ctx.Value, nil
	}),
	// Grammar: Expr : left=Expr Plus right=Term {go: add}
	calc.SemanticActionAdd: calc.TypedAdd(func(ctx calc.AddReduction) (float64, error) {
		return ctx.Left + ctx.Right, nil
	}),
	// Grammar: Term : left=Term Slash right=Factor {go: divide}
	calc.SemanticActionDivide: calc.TypedDivide(func(ctx calc.DivideReduction) (float64, error) {
		if ctx.Right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return ctx.Left / ctx.Right, nil
	}),
	// Grammar: Factor : token=Number {go: number}
	calc.SemanticActionNumber: calc.TypedNumber(func(ctx calc.NumberReduction) (float64, error) {
		return strconv.ParseFloat(ctx.Token.Text, 64)
	}),
}

// Reduce is passed to the generated parser.
func Reduce(ctx calc.Reduction) (calc.Value, error) {
	return reducers.Reduce(ctx)
}
```

For a reusable library, wrap generated APIs behind your own parser type:

```go
//go:build langforge_generated

package parser

import (
	calc "example.com/mydsl/generated"
	"example.com/mydsl/semantics"
)

// Parser is safe to share when its reducer dependencies are immutable.
type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) Eval(source string) (float64, error) {
	value, err := calc.ParseWithReducerFromSource(
		calc.NewScanner(source),
		calc.ReducerFunc(semantics.Reduce),
	)
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}
```

Use the `langforge_generated` build tag for Go files that import generated
packages. Add a small non-generated placeholder main for root `go test ./...`
when the generated package does not exist yet.

## C#: Handwritten Reducer, Facade, And DI

Generated C# code exposes:

- `new Scanner(source)` as an `ILexemeSource`;
- `Parser.ParseWithReducerFromSource(source, reducerMap)`;
- `Scanner.Tokenize(source)` and `Parser.ParseWithReducer(tokens, reducerMap)`
  as compatibility/debugging helpers;
- `SemanticAction` enum values;
- `ReducerMap`;
- `SemanticReducerContexts.Typed<ActionName>` adapter methods;
- typed reduction records such as `AddReduction`.

A small handwritten reducer can be:

```csharp
using System.Globalization;
using Example.Calc.Generated;
using static Example.Calc.Generated.SemanticReducerContexts;

public sealed class CalcSemantics
{
    public ReducerMap CreateReducers() => new()
    {
        // Shortened for the guide. A real map must cover every generated
        // semantic action used by the grammar.
        // Grammar: Input : value=Expr {csharp: start}
        [SemanticAction.Start] = TypedStart(Start),
        // Grammar: Expr : left=Expr Plus right=Term {csharp: add}
        [SemanticAction.Add] = TypedAdd(Add),
        // Grammar: Term : left=Term Slash right=Factor {csharp: divide}
        [SemanticAction.Divide] = TypedDivide(Divide),
        // Grammar: Factor : token=Number {csharp: number}
        [SemanticAction.Number] = TypedNumber(Number),
    };

    private static double Start(StartReduction ctx) => ctx.Value;

    private static double Add(AddReduction ctx) => ctx.Left + ctx.Right;

    private static double Divide(DivideReduction ctx)
    {
        if (ctx.Right == 0.0)
        {
            throw new InvalidOperationException("division by zero");
        }
        return ctx.Left / ctx.Right;
    }

    private static double Number(NumberReduction ctx) =>
        double.Parse(ctx.Token.Text, CultureInfo.InvariantCulture);
}
```

A library facade can hide generated details:

```csharp
using Example.Calc.Generated;

public interface ICalcParser
{
    double Evaluate(string source);
}

public sealed class CalcParser : ICalcParser
{
    private readonly CalcSemantics semantics;

    public CalcParser(CalcSemantics semantics)
    {
        this.semantics = semantics;
    }

    public double Evaluate(string source)
    {
        return (double)Parser.ParseWithReducerFromSource(
            new Scanner(source),
            semantics.CreateReducers())!;
    }
}
```

If the host application uses `Microsoft.Extensions.DependencyInjection`, keep
DI in handwritten code. Generated code does not require a container.

```csharp
using Microsoft.Extensions.DependencyInjection;

public static class CalcServices
{
    public static IServiceCollection AddCalcParser(this IServiceCollection services)
    {
        return services
            .AddSingleton<CalcSemantics>()
            .AddSingleton<ICalcParser, CalcParser>();
    }
}
```

This makes tests straightforward:

```csharp
var services = new ServiceCollection()
    .AddCalcParser()
    .BuildServiceProvider();

var parser = services.GetRequiredService<ICalcParser>();
Assert.Equal(3.0, parser.Evaluate("1+2"));
```

For current LangForge, this DI layer is handwritten. A future scaffold/template
feature should be able to generate the starting point.

## C: Handwritten Reducer And Adapter

Generated C output exposes prefixed functions and types. With `%package calc`,
the generated API includes names such as:

- `calc_tokenize`;
- `calc_parse_value`;
- `calc_parse_value_source`;
- `calc_parse_value_typed`;
- `calc_parse_value_source_typed`;
- `calc_lexeme_source`;
- `calc_scanner_source_next`;
- `calc_reduction`;
- `calc_typed_reducer`;
- `calc_typed_reducer_from_boxed`;
- `CALC_ACTION_ADD`.

C semantic values are pointers (`calc_value`). The application owns the memory
behind those pointers. The examples use a small arena per parse.

```c
#include "generated/parser.h"
#include "generated/parser_typed.h"

#include <stdio.h>
#include <stdlib.h>

typedef struct app_arena app_arena;
app_arena *app_arena_create(void);
void *app_arena_alloc(app_arena *arena, size_t size);
void app_arena_destroy(app_arena *arena);

typedef struct calc_context {
    app_arena *arena; /* Your project allocator, arena, or memory pool. */
} calc_context;

static calc_value make_number(calc_context *ctx, calc_error *error, double value) {
    double *slot = app_arena_alloc(ctx->arena, sizeof(double));
    if (slot == NULL) {
        snprintf(error->message, sizeof(error->message), "out of memory");
        return NULL;
    }
    *slot = value;
    return slot;
}

static double as_number(calc_value value) {
    return *((double *)value);
}

static calc_value calc_reduce(const calc_reduction *ctx, void *user, calc_error *error) {
    calc_context *app = (calc_context *)user;

    /* Shortened for the guide. A real reducer covers every generated action. */
    switch (ctx->action_id) {
    /* Grammar: Expr : left=Expr Plus right=Term {c: add} */
    case CALC_ACTION_ADD:
        return make_number(app, error, as_number(ctx->values[0]) + as_number(ctx->values[2]));
    /* Grammar: Term : left=Term Slash right=Factor {c: divide} */
    case CALC_ACTION_DIVIDE: {
        double right = as_number(ctx->values[2]);
        if (right == 0.0) {
            snprintf(error->message, sizeof(error->message), "division by zero");
            return NULL;
        }
        return make_number(app, error, as_number(ctx->values[0]) / right);
    }
    default:
        return ctx->rhs_count == 1 ? ctx->values[0] : NULL;
    }
}
```

Then write a small adapter function:

```c
int calc_eval_text(const char *source, double *out, char *message, size_t message_size) {
    calc_context ctx = {app_arena_create()};
    calc_error error = {0};
    calc_scanner scanner;
    calc_lexeme_source source_reader;
    calc_value value = NULL;

    if (ctx.arena == NULL) {
        snprintf(message, message_size, "out of memory creating parser context");
        return 0;
    }

    calc_scanner_init(&scanner, source);
    source_reader.user = &scanner;
    source_reader.next = calc_scanner_source_next;

    calc_boxed_typed_reducer boxed = {0};
    calc_typed_reducer typed = calc_typed_reducer_from_boxed(&boxed, calc_reduce, &ctx);
    if (!calc_parse_value_source_typed(&source_reader, &typed, &value, &error)) {
        snprintf(message, message_size, "parse failed: %s", error.message);
        app_arena_destroy(ctx.arena);
        return 0;
    }

    *out = as_number(value);
    app_arena_destroy(ctx.arena);
    return 1;
}
```

Keep the typed reducer bridge and its boxed storage alive for the duration of
the parse call. Avoid global mutable semantic state; pass per-parse state
through the generated `void *user` argument.

## C++: Handwritten Reducer, Facade, And Abstractions

Generated C++ output exposes:

- `Scanner` as a `LexemeSource`;
- `parse(source)` and `parse_value(source, reducerMap)`;
- `tokenize(source)`, `parse(tokens)`, and `parse_value(tokens, reducerMap)`
  as compatibility/debugging helpers;
- `SemanticAction` enum class values;
- `ReducerMap`;
- `typed_reducer_map_from_boxed`;
- typed contexts in `parser_typed.hpp`.

A compact reducer map can be:

```cpp
#include "generated/parser.hpp"
#include "generated/parser_typed.hpp"

#include <any>
#include <stdexcept>

namespace gen = Example::Calc::Generated;

static double number_arg(const gen::Reduction& ctx, std::size_t index) {
    return std::any_cast<double>(ctx.values.at(index));
}

gen::ReducerMap make_reducers() {
    return gen::typed_reducer_map_from_boxed(gen::ReducerMap{
        // Shortened for the guide. A real map must cover every generated
        // semantic action used by the grammar.
        // Grammar: Expr : left=Expr Plus right=Term {cpp: add}
        {gen::SemanticAction::Add, [](const gen::Reduction& ctx) -> gen::Value {
            return number_arg(ctx, 0) + number_arg(ctx, 2);
        }},
        // Grammar: Term : left=Term Slash right=Factor {cpp: divide}
        {gen::SemanticAction::Divide, [](const gen::Reduction& ctx) -> gen::Value {
            double right = number_arg(ctx, 2);
            if (right == 0.0) {
                throw std::runtime_error("division by zero");
            }
            return number_arg(ctx, 0) / right;
        }},
    });
}
```

For a reusable C++ library, prefer a facade class:

```cpp
#include "generated/parser.hpp"

#include <any>
#include <string_view>
#include <utility>

namespace gen = Example::Calc::Generated;

class CalcParser {
public:
    explicit CalcParser(gen::ReducerMap reducers)
        : reducers_(std::move(reducers)) {}

    double evaluate(std::string_view source) const {
        gen::Scanner scanner(source);
        auto value = gen::parse_value(scanner, reducers_);
        return std::any_cast<double>(value);
    }

private:
    gen::ReducerMap reducers_;
};
```

If the semantic layer needs services, inject an interface or policy object into
your handwritten facade rather than teaching generated code about a framework:

```cpp
class ICalcSemantics {
public:
    virtual ~ICalcSemantics() = default;
    virtual gen::ReducerMap reducers() const = 0;
};

class CalcParser {
public:
    explicit CalcParser(const ICalcSemantics& semantics)
        : reducers_(semantics.reducers()) {}

    double evaluate(std::string_view source) const;

private:
    gen::ReducerMap reducers_;
};
```

The C++ examples favor reducer maps and typed adapters instead of long
reduction switches. That keeps action lookup explicit and lets the generated
map validate reducer coverage.

## Main Program Or Reusable Library?

For a command-line tool, it is acceptable for `main` to call generated APIs
directly:

```text
read file -> Scanner.Next -> Parser.ParseWithReducerFromSource -> print report
```

For a library, keep generated types behind a stable boundary:

```text
public ParseProgram(string source) -> ProgramAst
public Compile(string source) -> Instruction[]
public Render(string source) -> Image
```

Recommended library rules:

- expose domain types, not generated `Reduction` objects;
- wrap scanner/parser exceptions or errors in your own diagnostics type;
- keep reducer maps immutable after construction;
- keep per-parse state in an instance/context, not global variables;
- make generated output private to the package/project when the language
  allows it;
- document which generated directory can be deleted and recreated.

## Multiple Parsers In One Application

Multiple generated parsers can live in one application when each grammar has a
distinct package, namespace, or prefix.

### Go

```go
import (
	querygen "example.com/app/query/generated"
	policygen "example.com/app/policy/generated"
)

type QueryParser struct{}
type PolicyParser struct{}

func (QueryParser) Parse(source string) (*query.Model, error) {
	value, err := querygen.ParseWithReducerFromSource(
		querygen.NewScanner(source),
		querygen.ReducerFunc(querysem.Reduce),
	)
	if err != nil {
		return nil, err
	}
	return value.(*query.Model), nil
}

func (PolicyParser) Parse(source string) (*policy.Model, error) {
	value, err := policygen.ParseWithReducerFromSource(
		policygen.NewScanner(source),
		policygen.ReducerFunc(policysem.Reduce),
	)
	if err != nil {
		return nil, err
	}
	return value.(*policy.Model), nil
}
```

### C#

```csharp
using QueryGen = Example.Query.Generated;
using PolicyGen = Example.Policy.Generated;

services.AddSingleton<IQueryParser, QueryParser>();
services.AddSingleton<IPolicyParser, PolicyParser>();
```

Keep each grammar in its own generated namespace:

```lf
%package Example.Query.Generated
%package Example.Policy.Generated
```

### C

Use distinct prefixes:

```lf
%package query
%package policy
```

Then generated names stay separate:

```c
query_scanner_source_next(...);
query_parse_value_source(...);

policy_scanner_source_next(...);
policy_parse_value_source(...);
```

### C++

Use distinct namespaces:

```cpp
namespace querygen = Example::Query::Generated;
namespace policygen = Example::Policy::Generated;
```

Then write separate facade classes:

```cpp
class QueryParser { /* uses querygen */ };
class PolicyParser { /* uses policygen */ };
```

Avoid sharing mutable reducer state between parsers. Shared read-only services
are fine when they are thread-safe.

## Diagnostics And Recovery

Generated scanners and parsers report syntax-level failures. Handwritten
reducers report semantic failures.

Examples:

- scanner failure: `@` does not match any lexical rule;
- parser failure: `1 +` ends before an expression appears;
- recovery diagnostic: a statement is skipped until a synchronization token;
- semantic failure: division by zero, undefined variable, duplicate symbol.

Keep these categories visible in your facade:

```text
ParseResult
  Value or AST
  Syntax diagnostics
  Semantic diagnostics
  WasRecovered flag, if recovery is enabled
```

Do not hide semantic failures as generic parse errors. Users can fix syntax
and semantics faster when the message says which layer failed.

## Thread Safety

Current generated Go, C#, C, and C++ examples are designed around local scanner
and parser state. That makes concurrent parsing practical, but your handwritten
semantic code must follow the same rule.

Checklist:

- no process-wide mutable current token or current AST;
- no shared arena without synchronization;
- no shared mutable reducer map after startup;
- no captured mutable C# or C++ service state unless it is thread-safe;
- per-parse context for symbol tables, diagnostics, and temporary memory.

In C, pass context through `void *user`. In C# and C++, inject immutable or
thread-safe services into the parser facade. In Go, prefer immutable reducer
maps and per-call local state.

## Testing Checklist

Every serious grammar should have tests for:

- one smallest valid input;
- representative valid input;
- invalid scanner input;
- invalid parser input;
- semantic failure, such as undefined variable or division by zero;
- reducer coverage or required-handler validation;
- generated output can be deleted and regenerated;
- multiple parser instances can run independently;
- optional boxed reducer compatibility, if you keep it;
- error recovery cases, if the grammar uses `error` productions.

A useful golden test shape is:

```text
input source
  -> parse
  -> AST/IR/report
  -> compare stable text output
```

The examples under `examples/go`, `examples/csharp`, `examples/c`, and
`examples/cpp` follow this pattern in different sizes.

## What LangForge Should Scaffold Later

Today, users can hand-write all of the patterns in this guide. The next
usability step is to scaffold them from maintained templates:

- `lang-forge init` starter projects per target;
- optional parser facade files;
- C# service-registration and DI starter code;
- C++ facade and semantic policy templates;
- a multi-parser starter project showing two grammars in one application;
- generated or template-backed tests for reducer coverage and regeneration
  hygiene.

Until those features exist, use the `library-dsl` templates for reusable
parser-facade architecture, the `mini-compiler` templates for compact compiler
pipelines, and the larger examples as copyable starting points.
