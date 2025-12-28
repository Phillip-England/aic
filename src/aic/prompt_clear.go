package aic

import (
	"fmt"
	"os"
	"strings"
)

func clearPromptPreserveContext(d *AiDir) error {
	if d == nil {
		return fmt.Errorf("clear requires AiDir")
	}
	path := d.PromptPath()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte(promptHeader), 0o644)
		}
		return fmt.Errorf("read prompt.md: %w", err)
	}
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	lines := strings.Split(s, "\n")

	keepThrough := func(pred func(string) bool) (string, bool) {
		for i := 0; i < len(lines); i++ {
			if pred(lines[i]) {
				out := strings.Join(lines[:i+1], "\n")
				if !strings.HasSuffix(out, "\n") {
					out += "\n"
				}
				return out, true
			}
		}
		return "", false
	}

	// Prefer "=== PROMPT ===" style
	if out, ok := keepThrough(func(line string) bool {
		return strings.TrimSpace(line) == "=== PROMPT ==="
	}); ok {
		return os.WriteFile(path, []byte(out), 0o644)
	}

	// Fallback to YAML header: keep from first '---' through second '---'
	firstDash := -1
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			firstDash = i
			break
		}
	}
	if firstDash != -1 {
		for j := firstDash + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "---" {
				out := strings.Join(lines[:j+1], "\n")
				if !strings.HasSuffix(out, "\n") {
					out += "\n"
				}
				return os.WriteFile(path, []byte(out), 0o644)
			}
		}
	}

	// Last resort: rewrite default header
	return os.WriteFile(path, []byte(promptHeader), 0o644)
}
