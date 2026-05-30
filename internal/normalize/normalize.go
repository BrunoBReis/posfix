// Package normalize converts arbitrary file names into normalized, ASCII-only
// names (for example "Café FINAL!!! .TXT" -> "cafe_final.txt").
//
// The package is pure: nothing here touches the filesystem. That keeps the
// rules easy to reason about and lets the entire rule set be covered by
// table-driven tests. The renaming side effects live in the renamer package.
//
// Normalization is split into a shared pipeline (extension handling,
// transliteration, tokenization) and a pluggable [Style] that assembles the
// final base name from tokens, so new conventions can be added without
// touching the pipeline.
package normalize

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// fallback is the base name used when normalization leaves no usable tokens,
// e.g. a name made entirely of punctuation.
const fallback = "unnamed"

// compositeExtensions are multi-part suffixes that must be treated as a single
// extension instead of splitting on the last dot. They are matched
// case-insensitively; add entries here to support more archive formats.
var compositeExtensions = []string{
	".tar.gz",
	".tar.bz2",
	".tar.xz",
	".tar.zst",
	".tar.lz",
	".tar.lzma",
	".tar.z",
}

// transliterator decomposes runes (NFKD), drops the resulting combining marks
// (Unicode category Mn), then recomposes (NFC). This strips Latin accents
// ("ç" -> "c", "ã" -> "a") and folds compatibility forms ("ª" -> "a"). Runes
// without a decomposition (e.g. "ø", "ł") are left for the tokenizer to drop,
// which keeps the output strictly ASCII.
var transliterator = transform.Chain(
	norm.NFKD,
	runes.Remove(runes.In(unicode.Mn)),
	norm.NFC,
)

// Normalize converts a file name into a normalized name using the given Style.
//
// The pipeline, in order: peel a leading-dot (hidden) prefix, split off the
// extension (composite-aware), transliterate the base to ASCII, tokenize on any
// non-[a-z0-9] rune, hand the tokens to the Style, then reattach the prefix and
// the lowercased extension. If no tokens survive, the base falls back to
// "unnamed".
func Normalize(name string, style Style) string {
	hidden, rest := splitHiddenPrefix(name)
	base, ext := splitExt(rest)

	tokens := tokenize(transliterate(base))
	out := style.Apply(tokens)
	if out == "" {
		out = fallback
	}

	return hidden + out + strings.ToLower(ext)
}

// splitHiddenPrefix peels a single leading dot off a name so that dotfiles such
// as ".bashrc" keep their dot instead of having it tokenized away.
func splitHiddenPrefix(name string) (prefix, rest string) {
	if strings.HasPrefix(name, ".") {
		return ".", name[1:]
	}
	return "", name
}

// splitExt separates a base name from its extension. A known composite suffix
// (".tar.gz", ...) takes precedence over a plain last-dot split; a name with no
// dot has no extension. The returned extension keeps its original casing and is
// lowercased by the caller.
func splitExt(name string) (base, ext string) {
	lower := strings.ToLower(name)
	for _, ce := range compositeExtensions {
		if strings.HasSuffix(lower, ce) {
			cut := len(name) - len(ce)
			return name[:cut], name[cut:]
		}
	}
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[:i], name[i:]
	}
	return name, ""
}

// transliterate maps accented and compatibility characters to ASCII wherever a
// Unicode decomposition exists. It never fails for valid input; on the rare
// transform error it returns the original string unchanged.
func transliterate(s string) string {
	out, _, err := transform.String(transliterator, s)
	if err != nil {
		return s
	}
	return out
}

// tokenize lowercases the input and splits it into runs of [a-z0-9], treating
// every other rune (spaces, punctuation, leftover non-ASCII) as a separator.
// strings.FieldsFunc already drops empty fields, so the result is a clean slice
// of word tokens.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
}
