package aic

import "strings"

// Compressor is responsible for taking an output string and compressing it.
// It also carries a compression map for future token/phrase compression.
type Compressor struct {
	Map map[string]string
}

func NewCompressor() *Compressor {
	return &Compressor{
		Map: map[string]string{},
	}
}

// Compress applies cleanup steps to the provided string.
// It removes full-line comments (starting with //) and empty/whitespace-only lines.
func (c *Compressor) Compress(in string) string {
	// First remove comments, then clean up any resulting empty lines
	s := removeComments(in)
	return removeEmptyLines(s)
}

// removeComments removes any line that starts with "//" (ignoring leading whitespace).
// Note: This does not remove "#" lines to preserve Markdown headers.
func removeComments(in string) string {
	if in == "" {
		return ""
	}
	// Normalize newlines
	s := strings.ReplaceAll(in, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		// specific check for double-slash comments
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// removeEmptyLines removes any line that is empty or contains only whitespace.
// It preserves a trailing newline if the input had one and the output is non-empty.
func removeEmptyLines(in string) string {
	if in == "" {
		return ""
	}
	// Normalize newlines just in case.
	s := strings.ReplaceAll(in, "\r\n", "\n")
	hadTrailing := strings.HasSuffix(s, "\n")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return ""
	}
	res := strings.Join(out, "\n")
	if hadTrailing {
		res += "\n"
	}
	return res
}
