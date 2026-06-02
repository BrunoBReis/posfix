# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`posfix` (a pun on POSIX) is a single-command CLI that renames files to `snake_case`
ASCII: transliterate accents, lowercase, split on any rune outside `[a-z0-9]`, join with
`_`, preserving known composite extensions like `.tar.gz`. It is a learning + portfolio
project, so the bar is idiomatic, well-tested, well-documented Go in the Unix
"do one thing well" spirit. When making a non-obvious idiomatic choice, briefly explain it
(in the commit message or a short comment) rather than leaving it implicit.

Code, comments, commit messages, and documentation are written in English (open-source
portfolio convention). Commits follow Conventional Commits, kept small and logical.

## Commands

```sh
go build ./...                         # build all packages
go run ./cmd/posfix [flags] [targets]  # run without installing
go test ./...                          # run all tests
go test -race ./...                    # what CI runs
go test ./internal/normalize/ -run TestNormalize_Snake   # one package / one test
go test ./internal/normalize/ -run TestNormalize_Snake/dotfile_is_unchanged  # one table row
gofmt -l .                             # list unformatted files (CI fails if non-empty)
go vet ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./...  # lint
```

Subtest names come from each table row's `name` field with spaces replaced by `_`.

## Architecture

Three layers with a strict rule: **the pure core never touches the filesystem.** This is
what makes the rule set exhaustively unit-testable and the I/O layer thin.

- **`internal/normalize/`** — pure `string -> string`, zero I/O. All real complexity lives
  here. `Normalize(name, style)` runs a fixed pipeline: peel a leading-dot hidden prefix →
  `splitExt` (composite-aware) → `transliterate` (NFKD + strip combining marks via
  `golang.org/x/text`) → `tokenize` (lowercase, split on non-`[a-z0-9]`) → `Style.Apply` →
  reattach prefix + lowercased extension → `unnamed` fallback if no tokens survive.
- **`internal/renamer/`** — the only package that touches disk. `Run(targets, opts, stdout,
  stderr) int` discovers files, asks `normalize` for each new name, and renames, returning a
  count of hard errors. Output streams are injected `io.Writer`s so tests capture them into
  buffers.
- **`cmd/posfix/`** — wiring only. Flag parsing (`spf13/pflag`) lives in a testable
  `run(args, stdout, stderr) int`; `main` just calls `os.Exit(run(...))`.

### Load-bearing design decisions

- **`Style` is a strategy interface** (`Apply(tokens []string) string`). The pipeline emits
  `[]string` tokens; the `Style` assembles the final base name. Tokens arrive already
  lowercased, so `Snake` only joins — but **casing belongs in the Style, not the pipeline**,
  so a future `camelCase`/`PascalCase` can re-case tokens without changing the pipeline.
  Adding a naming convention = a new `Style` implementation + rows in `style_test.go`.
  `--style` is intentionally not exposed yet (a one-option flag would be noise; it's roadmap).
- **Composite extensions** (`.tar.gz`, …) live in the `compositeExtensions` slice in
  `normalize.go` and are matched case-insensitively but emitted lowercase. They take
  precedence over a plain last-dot split, which is why `foo.bar.txt` keeps only `.txt` but
  `backup.tar.gz` keeps `.tar.gz`. Extend support by adding to that slice.
- **Safety is in `renamer`, not `normalize`.** Collision handling is **per-directory**: files
  are processed in sorted order with a `claimed` set, and a file is skipped with a warning
  whenever its destination already exists on disk or was claimed earlier in the same run.
  posfix never overwrites. Directories are never renamed (files only, MVP).
- **Hidden files** are skipped during directory scans unless `--all`, but a hidden file named
  explicitly as an argument is always processed.

### Output contract (relied on by tests)

Renames go to **stdout** as `old  ->  new`; warnings (skips, collisions, errors) go to
**stderr** prefixed `posfix:`. In dry-run mode each stdout line is prefixed `[dry-run] `.
Exit code is 1 only on hard errors (failed rename, unreadable path); skips are not errors.

## Conventions specific to this repo

- **Tests are table-driven**, structured so extension means adding rows, not scaffolding.
  Filesystem tests use `t.TempDir()`.
- `go.mod` pins the **real minimum** Go version (currently 1.25, forced by
  `golang.org/x/text`), not the exact local toolchain patch — keep it at the true floor.
- The Go module path is lowercase `github.com/brunobreis/posfix`; the GitHub remote is
  `github.com/BrunoBReis/posfix` (GitHub resolves case-insensitively — this is deliberate).
- Transliteration only handles characters with a Unicode decomposition; `ø`, `ł`, etc. pass
  through and get dropped by the tokenizer. A unidecode-style library is roadmap, not a bug.
