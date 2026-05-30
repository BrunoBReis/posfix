package normalize

import "strings"

// Style assembles the final base name from the normalized, lowercase tokens
// produced by the pipeline. Separating the pipeline (which tokenizes) from the
// Style (which assembles) is the strategy pattern: a new naming convention is
// just a new Style implementation, with no change to Normalize.
type Style interface {
	// Apply joins tokens into the final base name. Tokens are already
	// lowercase and contain only [a-z0-9].
	Apply(tokens []string) string
}

// Snake produces snake_case by joining tokens with underscores. Because tokens
// arrive lowercase, Snake only has to join them.
type Snake struct{}

// Apply implements [Style].
func (Snake) Apply(tokens []string) string {
	return strings.Join(tokens, "_")
}
