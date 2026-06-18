# UCDT Fixture Corpus

These text fixtures are curated from the Pascal
[UCDT](https://github.com/russlank/UCDT) reference implementation used during
LangForge design and migration testing.

Included fixture groups:

- `calc`: original calculator Lex/Yacc sample.
- `draw`: original DRAW Lex/Yacc sample and input text.
- `metas`: original meta grammars for the Lex/Yacc tools plus small conflict
  fixtures.

Only source-style text fixtures are kept here. Generated Pascal outputs,
executables, graphics drivers, and other binary/runtime artifacts are excluded.

The fixtures are used to validate LangForge's legacy split `.l`/`.y` parser and
to document compatibility gaps while the modern combined `.lf` format evolves.
