// --- START FILE: aic/ai_dir.go ---
package aic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AiDir struct {
	Root string

	// Kept for potential future use (not created anymore)
	Tmp     string
	Prompts string
	Skills  string
}

const promptHeader = `# LLM MODEL THIS IS MY PROMPT:
`

func NewAiDir(force bool) (*AiDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	rootAbs := filepath.Join(wd, "ai")
	promptFile := filepath.Join(rootAbs, "prompt.md")

	tmpAbs := filepath.Join(rootAbs, "tmp")
	promptsAbs := filepath.Join(rootAbs, "prompts")
	skillsAbs := filepath.Join(rootAbs, "skills")

	if info, statErr := os.Lstat(rootAbs); statErr == nil {
		if !info.IsDir() {
			return nil, fmt.Errorf("ai path exists but is not a directory: %s", rootAbs)
		}
		if !force {
			return nil, fmt.Errorf("ai dir already exists: %s", rootAbs)
		}
		if err := os.RemoveAll(rootAbs); err != nil {
			return nil, fmt.Errorf("remove existing ai dir: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("stat ai dir: %w", statErr)
	}

	if err := os.MkdirAll(rootAbs, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", rootAbs, err)
	}

	if err := os.WriteFile(promptFile, []byte(promptHeader), 0o644); err != nil {
		return nil, fmt.Errorf("write prompt.md: %w", err)
	}

	return &AiDir{
		Root:    rootAbs,
		Tmp:     tmpAbs,
		Prompts: promptsAbs,
		Skills:  skillsAbs,
	}, nil
}

func (d *AiDir) PromptPath() string {
	return filepath.Join(d.Root, "prompt.md")
}

// PromptText reads ./ai/prompt.md and returns the contents as a string.
// It tolerates missing file by returning the default header content (so first run still works).
func (d *AiDir) PromptText() (string, error) {
	b, err := os.ReadFile(d.PromptPath())
	if err != nil {
		if os.IsNotExist(err) {
			return promptHeader, nil
		}
		return "", fmt.Errorf("read prompt.md: %w", err)
	}

	// Normalize line endings to \n for consistent downstream behavior.
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s, nil
}

// --- END FILE: aic/ai_dir.go ---
