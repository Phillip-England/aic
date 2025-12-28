package aic

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxPromptHistory = 100

func (d *AiDir) PromptsDir() string {
	if d == nil {
		return ""
	}
	if d.Prompts != "" {
		return d.Prompts
	}
	if d.Root == "" {
		return ""
	}
	return filepath.Join(d.Root, "prompts")
}

func (d *AiDir) ensurePromptsDir() (string, error) {
	dir := d.PromptsDir()
	if dir == "" {
		return "", fmt.Errorf("prompts dir unavailable")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create prompts dir %s: %w", dir, err)
	}
	return dir, nil
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

func (d *AiDir) StashRawPrompt(raw string) error {
	if d == nil {
		return fmt.Errorf("stash prompt requires AiDir")
	}
	dir, err := d.ensurePromptsDir()
	if err != nil {
		return err
	}

	raw = normalizeNewlines(raw)
	raw = strings.TrimRight(raw, "\n") + "\n"

	// Filename sorts oldest->newest lexicographically.
	ts := time.Now().Format("20060102_150405.000000000")
	ts = strings.ReplaceAll(ts, ".", "_")
	base := ts + ".md"
	path := filepath.Join(dir, base)

	// Avoid extremely rare collisions.
	for i := 1; ; i++ {
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			break
		}
		path = filepath.Join(dir, fmt.Sprintf("%s_%03d.md", ts, i))
		if i >= 999 {
			return fmt.Errorf("failed to choose unique prompt snapshot filename")
		}
	}

	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		return fmt.Errorf("write prompt snapshot: %w", err)
	}

	return d.PrunePromptHistory(maxPromptHistory)
}

func (d *AiDir) PrunePromptHistory(max int) error {
	if d == nil {
		return nil
	}
	if max <= 0 {
		max = maxPromptHistory
	}
	dir := d.PromptsDir()
	if dir == "" {
		return nil
	}

	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read prompts dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}

	if len(files) <= max {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return filepath.Base(files[i]) < filepath.Base(files[j])
	})

	toDelete := len(files) - max
	for i := 0; i < toDelete; i++ {
		_ = os.Remove(files[i])
	}
	return nil
}
