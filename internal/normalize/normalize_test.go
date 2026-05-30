package normalize

import "testing"

// TestNormalize_Snake drives the full pipeline through Snake, the default
// style. Every row from the project's spec table is here, plus edge cases that
// exercise the fallback, composite extensions, and hidden files.
func TestNormalize_Snake(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// Spec examples.
		{"sentence with comma", "Introduction to Modern Cryptography, Third Edition.pdf", "introduction_to_modern_cryptography_third_edition.pdf"},
		{"portuguese ordinal and parens", "Programação - Vol 1 (2ª ed).pdf", "programacao_vol_1_2a_ed.pdf"},
		{"collapses spaces and punctuation", "Relatório  FINAL!!! .docx", "relatorio_final.docx"},
		{"uppercase name and extension", "UPPER CASE FILE.JPG", "upper_case_file.jpg"},
		{"latin accents", "café com leite.txt", "cafe_com_leite.txt"},
		{"symbols are separators", "arquivo@com#simbolos&estranhos.md", "arquivo_com_simbolos_estranhos.md"},
		{"composite extension", "meu backup.tar.gz", "meu_backup.tar.gz"},
		{"composite extension uppercase", "Backup FINAL.TAR.GZ", "backup_final.tar.gz"},
		{"unknown double extension keeps last only", "foo.bar.txt", "foo_bar.txt"},
		{"already normalized is unchanged", "already_normalized.txt", "already_normalized.txt"},
		{"dotfile is unchanged", ".bashrc", ".bashrc"},

		// Additional edge cases.
		{"only symbols falls back to unnamed", "@#$.txt", "unnamed.txt"},
		{"empty falls back to unnamed", "", "unnamed"},
		{"only spaces falls back to unnamed", "   .txt", "unnamed.txt"},
		{"multiple unknown dots", "a.b.c.txt", "a_b_c.txt"},
		{"composite extension mixed case", "ARCHIVE.TAR.BZ2", "archive.tar.bz2"},
		{"no extension", "Hello World", "hello_world"},
		{"hidden file with extension", ".My Config.json", ".my_config.json"},
		{"leading and trailing junk", "--- draft ---.md", "draft.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in, Snake{}); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestSplitExt isolates extension handling, including the composite-suffix rule
// that distinguishes "backup.tar.gz" from "foo.bar.txt".
func TestSplitExt(t *testing.T) {
	tests := []struct {
		in       string
		wantBase string
		wantExt  string
	}{
		{"backup.tar.gz", "backup", ".tar.gz"},
		{"Backup.TAR.GZ", "Backup", ".TAR.GZ"},
		{"foo.bar.txt", "foo.bar", ".txt"},
		{"report.pdf", "report", ".pdf"},
		{"noext", "noext", ""},
		{"archive.tar.zst", "archive", ".tar.zst"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			base, ext := splitExt(tt.in)
			if base != tt.wantBase || ext != tt.wantExt {
				t.Errorf("splitExt(%q) = (%q, %q), want (%q, %q)", tt.in, base, ext, tt.wantBase, tt.wantExt)
			}
		})
	}
}
