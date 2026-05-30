# posfix

> A small, Unix-philosophy CLI that normalizes filenames to `snake_case` ASCII.

`posfix` — a pun on **POSIX** — does one thing well: it renames files so their names
are predictable, portable, and shell-friendly. It transliterates accents to ASCII,
lowercases everything, collapses runs of punctuation and whitespace into single
underscores, and preserves file extensions (including composite ones like `.tar.gz`).

```text
Introduction to Modern Cryptography, Third Edition.pdf
        ->  introduction_to_modern_cryptography_third_edition.pdf

Programação - Vol 1 (2ª ed).pdf   ->  programacao_vol_1_2a_ed.pdf
Relatório  FINAL!!! .docx         ->  relatorio_final.docx
Backup FINAL.TAR.GZ               ->  backup_final.tar.gz
```

## Why

Files with spaces, accents, mixed case, and stray punctuation are awkward to type,
fragile in shell pipelines, and inconsistent across systems. `posfix` makes them
uniform `snake_case` ASCII in one pass, with safety rails so it never clobbers data.

## Install

```sh
go install github.com/brunobreis/posfix/cmd/posfix@latest
```

This puts the `posfix` binary in your `$GOBIN` (usually `~/go/bin`; make sure it's on
your `PATH`). An AUR package is planned — see the roadmap.

## Usage

```text
posfix [flags] [targets...]

Targets:
  (none)        normalize files in the current directory
  <file>        normalize that file
  <dir>         normalize the files inside that directory
  multiple      process each target in turn

Flags:
  -n, --dry-run     show what would happen, without renaming
  -r, --recursive   descend into subdirectories
  -a, --all         include hidden files (those starting with '.')
  -h, --help        show help
      --version     show version
```

Renames are printed to `stdout` as `old  ->  new`. Warnings (skips, collisions) go to
`stderr`. With `--dry-run`, each line is prefixed with `[dry-run]`.

```sh
# Preview changes in the current directory, touching nothing:
posfix --dry-run

# Normalize one file:
posfix "Café com Leite.txt"        # -> cafe_com_leite.txt

# Normalize a whole tree, including hidden files:
posfix -r -a ~/Downloads
```

### Examples

| Input | Output |
|---|---|
| `Introduction to Modern Cryptography, Third Edition.pdf` | `introduction_to_modern_cryptography_third_edition.pdf` |
| `Programação - Vol 1 (2ª ed).pdf` | `programacao_vol_1_2a_ed.pdf` |
| `Relatório  FINAL!!! .docx` | `relatorio_final.docx` |
| `UPPER CASE FILE.JPG` | `upper_case_file.jpg` |
| `café com leite.txt` | `cafe_com_leite.txt` |
| `arquivo@com#simbolos&estranhos.md` | `arquivo_com_simbolos_estranhos.md` |
| `meu backup.tar.gz` | `meu_backup.tar.gz` |
| `Backup FINAL.TAR.GZ` | `backup_final.tar.gz` |
| `foo.bar.txt` | `foo_bar.txt` |
| `already_normalized.txt` | `already_normalized.txt` (unchanged) |
| `.bashrc` | `.bashrc` (unchanged) |

## Behavior & current decisions

- **Snake case, strict.** Hyphens (`-`) become underscores. (A `--keep-hyphens` flag is
  on the roadmap.)
- **Extensions are preserved and lowercased.** Known composite extensions
  (`.tar.gz`, `.tar.bz2`, `.tar.xz`, `.tar.zst`, `.tar.lz`, …) are detected
  case-insensitively and kept whole; otherwise the extension is whatever follows the
  last `.`. So `foo.bar.txt` keeps only `.txt`.
- **Dotfiles and extensionless files are respected.** `.bashrc` stays `.bashrc`.
- **Hidden files are skipped by default** — include them with `--all`, or by naming one
  explicitly as an argument.
- **Files only.** Directories are never renamed in this version (it's on the roadmap).
- **Never overwrites.** If the target name already exists — on disk or as the result of
  another file in the same run — the conflicting file is skipped with a warning.
- **No-ops are silent.** Already-normalized names produce no output and no rename.
- **Transliteration** uses Unicode NFKD decomposition and strips combining marks, which
  covers Latin/Portuguese accents (`ã→a`, `ç→c`, `é→e`). Characters with no
  decomposition (e.g. `ø`, `ł`) pass through unchanged; richer transliteration is on the
  roadmap.

## Architecture

`posfix` separates pure logic from side effects across three layers:

```
cmd/posfix/          entry point: flag parsing, orchestration, output
internal/normalize/  PURE core — string -> string, zero I/O
internal/renamer/    filesystem layer — discovery, collision checks, renames
```

- **`internal/normalize`** holds all the real complexity (extension splitting,
  transliteration, tokenization) and is tested in isolation with table-driven tests.
  The normalization style is a strategy behind a `Style` interface; the shared pipeline
  turns a name into a list of tokens, and the `Style` turns tokens into the final
  string. Today there's only `Snake`, but adding kebab-case, camelCase, etc. is just a
  new `Style` implementation — the pipeline doesn't change.
- **`internal/renamer`** is the only package that touches the disk.
- **`cmd/posfix`** wires the two together.

## Roadmap

- Additional styles via the `Style` interface (`kebab-case`, `camelCase`, `PascalCase`)
  exposed through a `--style` flag.
- `--keep-hyphens`.
- Renaming directories (processed deepest-first so paths don't break mid-run).
- Undo / rollback of a run.
- Configuration file and an interactive (confirm-each) mode.
- `git mv` integration inside repositories.
- Extended transliteration via a unidecode-style library (`ø`, `ł`, …).
- AUR packaging (`PKGBUILD`).

## License

[MIT](LICENSE) © 2026 Bruno Bragança dos Reis
