package normalize

import "testing"

// TestStyle_Apply exercises each Style in isolation: given already-tokenized
// input, the Style must produce the expected base string. The table is keyed by
// Style, so supporting a new convention later (kebab, camel, ...) means adding
// rows here rather than new test scaffolding.
func TestStyle_Apply(t *testing.T) {
	tests := []struct {
		name   string
		style  Style
		tokens []string
		want   string
	}{
		{"snake joins with underscore", Snake{}, []string{"hello", "world"}, "hello_world"},
		{"snake single token", Snake{}, []string{"readme"}, "readme"},
		{"snake numeric tokens", Snake{}, []string{"vol", "1", "2a"}, "vol_1_2a"},
		{"snake empty tokens", Snake{}, []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.style.Apply(tt.tokens); got != tt.want {
				t.Errorf("%T.Apply(%v) = %q, want %q", tt.style, tt.tokens, got, tt.want)
			}
		})
	}
}
