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

// Compress applies compression steps to the provided string.
// For now, it only removes empty/whitespace-only lines.
func (c *Compressor) Compress(in string) string {
	return removeEmptyLines(in)
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
