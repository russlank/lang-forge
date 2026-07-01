# Parser Algorithm Fixtures

This directory contains small LangForge specs used by
[../../doc/parser-algorithms.md](../../doc/parser-algorithms.md).

The `lr1-not-slr-*` fixtures use the classic LR(1)-but-not-SLR example:

```text
S : L Eq R | R ;
L : Star R | ID ;
R : L ;
```

The `mysterious-conflict-*` fixtures use a grammar that is LR(1), but where
plain LALR merging combines lookaheads that should remain separate:

```text
Def : ParamSpec ReturnSpec Comma ;
ParamSpec : Type | NameList Colon Type ;
ReturnSpec : Type | Name Colon Type ;
Type : ID ;
Name : ID ;
NameList : Name | Name Comma NameList ;
```

Run:

```sh
make test
```

Expected behavior:

- `lr1-not-slr-lalr.lf` validates.
- `lr1-not-slr-ielr.lf` validates and keeps the same compact state shape as LALR.
- `lr1-not-slr-canonical.lf` validates.
- `lr1-not-slr-slr.lf` reports the expected SLR conflict.
- `mysterious-conflict-ielr.lf` validates.
- `mysterious-conflict-canonical.lf` validates.
- `mysterious-conflict-lalr.lf` reports the expected LALR merge conflict.

To see why IELR split or kept a core merge, inspect the IELR fixture:

```sh
go run ../../cmd/lang-forge inspect --spec mysterious-conflict-ielr.lf --format text
```

The report includes LALR/IELR/canonical state counts and the accepted/rejected
merge decisions for the LR(0) cores that differ from plain LALR.
