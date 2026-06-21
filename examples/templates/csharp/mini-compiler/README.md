# C# Mini-Compiler Template

This template keeps generated C# files under `Generated/` and handwritten
compiler code in `Program.cs`. The reducer builds an AST, the compiler lowers
that AST to stack instructions, and the runtime produces a small execution log.

```sh
make run
make test
```
