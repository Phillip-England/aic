package aic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AiDir struct {
	Root       string
	WorkingDir string

	Prompts string
	Skills  string

	Ignore *GitIgnore
}

const promptHeader = `# LLM MODEL THIS IS MY PROMPT:
`

func NewAiDir(force bool) (*AiDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	workingAbs := wd
	rootAbs := filepath.Join(workingAbs, "ai")
	promptFile := filepath.Join(rootAbs, "prompt.md")

	promptsAbs := filepath.Join(rootAbs, "prompts")
	skillsAbs := filepath.Join(rootAbs, "skills")

	// Handle existing directory
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

	// Create ai root + subdirs
	if err := os.MkdirAll(rootAbs, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", rootAbs, err)
	}
	if err := os.MkdirAll(promptsAbs, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", promptsAbs, err)
	}
	if err := os.MkdirAll(skillsAbs, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", skillsAbs, err)
	}

	// Write prompt.md
	if err := os.WriteFile(promptFile, []byte(promptHeader), 0o644); err != nil {
		return nil, fmt.Errorf("write prompt.md: %w", err)
	}

	ign, _ := LoadGitIgnore(workingAbs) // ignore missing .gitignore

	return &AiDir{
		Root:       rootAbs,
		WorkingDir: workingAbs,
		Prompts:    promptsAbs,
		Skills:     skillsAbs,
		Ignore:     ign,
	}, nil
}

func OpenAiDir() (*AiDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	workingAbs := wd
	rootAbs := filepath.Join(workingAbs, "ai")

	info, statErr := os.Lstat(rootAbs)
	if statErr != nil {
		return nil, fmt.Errorf("ai dir not found: %s", rootAbs)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("ai path exists but is not a directory: %s", rootAbs)
	}

	promptsAbs := filepath.Join(rootAbs, "prompts")
	skillsAbs := filepath.Join(rootAbs, "skills")

	// Ensure subdirs exist (non-destructive)
	_ = os.MkdirAll(promptsAbs, 0o755)
	_ = os.MkdirAll(skillsAbs, 0o755)

	ign, _ := LoadGitIgnore(workingAbs)

	return &AiDir{
		Root:       rootAbs,
		WorkingDir: workingAbs,
		Prompts:    promptsAbs,
		Skills:     skillsAbs,
		Ignore:     ign,
	}, nil
}

func (d *AiDir) PromptPath() string {
	return filepath.Join(d.Root, "prompt.md")
}

func (d *AiDir) PromptText() (string, error) {
	b, err := os.ReadFile(d.PromptPath())
	if err != nil {
		if os.IsNotExist(err) {
			return promptHeader, nil
		}
		return "", fmt.Errorf("read prompt.md: %w", err)
	}
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s, nil
}
