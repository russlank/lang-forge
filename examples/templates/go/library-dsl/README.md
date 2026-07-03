# Go Library DSL Template

This template shows how to hide generated LangForge details behind a stable Go
package boundary. The generated scanner/parser live in `generated/`, while
application code imports only `model` and `parser`.

Run it from this directory:

```sh
make test
make run
```

The DSL accepts configuration-like statements:

```text
set retries = 3;
set title = "nightly";
enable audit;
```

The parser facade uses `ParseWithReducerFromSource(new Scanner(...), reducers)`
so production code consumes tokens lazily from the generated scanner. A
collection-based `ParseTokens` method is kept for debugging and compatibility.
