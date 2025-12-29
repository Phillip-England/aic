package interpreter

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/phillip-england/aic/pkg/dir"
)

type Interpreter struct {
	Dir *dir.AiDir
}

func New(d *dir.AiDir) *Interpreter {
	return &Interpreter{Dir: d}
}

func (i *Interpreter) Run(rawContent string) error {
	parts := strings.Split(rawContent, "---")
	if len(parts) < 3 {
		return fmt.Errorf("invalid prompt.md format: missing '---' sections")
	}
	preCommands := parts[1]
	promptBody := strings.TrimSpace(parts[2])

	if promptBody != "" {
		if err := clipboard.WriteAll(promptBody); err != nil {
			fmt.Printf("[Interpreter] Warning: Failed to copy prompt to clipboard: %v\n", err)
		} else {
			fmt.Println("[Interpreter] Copied prompt to clipboard.")
		}
	}

	if err := i.Dir.StashPrompt(rawContent); err != nil {
		fmt.Printf("[Interpreter] Warning: Failed to stash prompt: %v\n", err)
	}

	if err := i.Dir.ClearPrompt(); err != nil {
		return fmt.Errorf("failed to clear prompt: %w", err)
	}

	i.executeShell(preCommands)
	return nil
}

func (i *Interpreter) executeShell(script string) {
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cmd := exec.Command("sh", "-c", line)
		if err := cmd.Run(); err != nil {
			fmt.Printf("[Interpreter] Command Failed: '%s' -> %v\n", line, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}