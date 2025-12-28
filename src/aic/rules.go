package aic

import (
	"fmt"
	"strings"
)

// LoadRules walks the ai/rules directory and returns a formatted string
// containing the content of all valid files found.
func LoadRules(d *AiDir) (string, error) {
	if d == nil || d.Rules == "" {
		return "", nil
	}

	files, err := CollectReadableFiles(d.Rules, d)
	if err != nil {
		return "", fmt.Errorf("collect rules: %w", err)
	}

	if len(files) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("\n=== RULES ===\n")

	for _, f := range files {
		content, ok, _, rerr := ReadTextFile(f)
		if rerr != nil {
			return "", rerr
		}
		if !ok {
			continue
		}

		sb.WriteString("FILE: ")
		sb.WriteString(f)
		sb.WriteString("\n")
		sb.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}