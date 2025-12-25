package aic

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

func ReadTextFile(absPath string) (content string, ok bool, stats ReadStats, err error) {
	b, err := os.ReadFile(absPath)
	if err != nil {
		return "", false, ReadStats{}, fmt.Errorf("read file %s: %w", absPath, err)
	}

	// Skip binary-ish content quickly.
	if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) {
		return "", false, ReadStats{}, nil
	}

	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")

	lines := 0
	if s != "" {
		// Count '\n' lines; if no trailing newline, still a line.
		lines = strings.Count(s, "\n")
		if !strings.HasSuffix(s, "\n") {
			lines++
		}
	}

	return s, true, ReadStats{Lines: lines, Chars: len(s)}, nil
}
