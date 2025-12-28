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
	Rules      string // Renamed from Skills
	Vars       string
	Prompts    string // NEW: prompt history folder
	Ignore     *GitIgnore
}

const promptHeader = `---
$path(".")
---
`

func findAiWorkingDir(start string) (string, error) {
	start = filepath.Clean(start)
	if es, err := filepath.EvalSymlinks(start); err == nil {
		start = es
	}
	dir := start
	for {
		aiPath := filepath.Join(dir, "ai")
		if info, err := os.Lstat(aiPath); err == nil && info.IsDir() {
			if es, err := filepath.EvalSymlinks(dir); err == nil {
				dir = es
			}
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("ai dir not found from %s (searched upward)", start)
}

func NewAiDir(force bool) (*AiDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	workingAbs := filepath.Clean(wd)
	if es, err := filepath.EvalSymlinks(workingAbs); err == nil {
		workingAbs = es
	}

	rootAbs := filepath.Join(workingAbs, "ai")
	promptFile := filepath.Join(rootAbs, "prompt.md")
	rulesAbs := filepath.Join(rootAbs, "rules") // Renamed from skills
	varsAbs := filepath.Join(rootAbs, "vars")
	promptsAbs := filepath.Join(rootAbs, "prompts") // NEW

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
	if err := os.MkdirAll(rulesAbs, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", rulesAbs, err)
	}
	if err := os.MkdirAll(varsAbs, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", varsAbs, err)
	}
	if err := os.MkdirAll(promptsAbs, 0o755); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", promptsAbs, err)
	}

	if err := os.WriteFile(promptFile, []byte(promptHeader), 0o644); err != nil {
		return nil, fmt.Errorf("write prompt.md: %w", err)
	}

	ign, _ := LoadGitIgnore(workingAbs)
	return &AiDir{
		Root:       rootAbs,
		WorkingDir: workingAbs,
		Rules:      rulesAbs,
		Vars:       varsAbs,
		Prompts:    promptsAbs,
		Ignore:     ign,
	}, nil
}

func OpenAiDir() (*AiDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	workingAbs, err := findAiWorkingDir(wd)
	if err != nil {
		return nil, err
	}

	rootAbs := filepath.Join(workingAbs, "ai")
	info, statErr := os.Lstat(rootAbs)
	if statErr != nil {
		return nil, fmt.Errorf("ai dir not found: %s", rootAbs)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("ai path exists but is not a directory: %s", rootAbs)
	}

	rulesAbs := filepath.Join(rootAbs, "rules")
	_ = os.MkdirAll(rulesAbs, 0o755)

	varsAbs := filepath.Join(rootAbs, "vars")
	_ = os.MkdirAll(varsAbs, 0o755)

	promptsAbs := filepath.Join(rootAbs, "prompts")
	_ = os.MkdirAll(promptsAbs, 0o755)

	ign, _ := LoadGitIgnore(workingAbs)
	return &AiDir{
		Root:       rootAbs,
		WorkingDir: workingAbs,
		Rules:      rulesAbs,
		Vars:       varsAbs,
		Prompts:    promptsAbs,
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

// ClearPrompt clears prompt.md back to just the header/context (preserving YAML header or "=== PROMPT ===" marker).
func (d *AiDir) ClearPrompt() error {
	return clearPromptPreserveContext(d)
}
