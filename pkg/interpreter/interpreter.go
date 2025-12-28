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
	// We expect [empty, pre-script, prompt-body]
	if len(parts) < 3 {
		return fmt.Errorf("invalid prompt.md format: missing '---' sections")
	}

	preCommands := parts[1]
	promptBody := strings.TrimSpace(parts[2])

	if promptBody == "" {
		return nil
	}

	// 1. Prepare Content (Clipboard + Rules + Prompt)
	currentClip, err := clipboard.ReadAll()
	if err != nil {
		fmt.Printf("[Interpreter] Warning: Could not read clipboard: %v\n", err)
		currentClip = ""
	}

	rules, err := i.Dir.CollectRules()
	if err != nil {
		fmt.Printf("[Interpreter] Warning: Could not collect rules: %v\n", err)
	}

	var finalOutput strings.Builder
	if currentClip != "" {
		finalOutput.WriteString(currentClip)
		finalOutput.WriteString("\n\n")
	}
	if rules != "" {
		finalOutput.WriteString(rules)
		finalOutput.WriteString("\n\n")
	}
	finalOutput.WriteString("=== PROMPT ===\n")
	finalOutput.WriteString(promptBody)

	// 2. Copy to Clipboard
	if err := clipboard.WriteAll(finalOutput.String()); err != nil {
		return fmt.Errorf("clipboard write error: %w", err)
	}

	// 3. Stash the prompt for history
	if err := i.Dir.StashPrompt(rawContent); err != nil {
		fmt.Printf("[Interpreter] Warning: Failed to stash prompt: %v\n", err)
	}

	// 4. Clear the prompt file immediately
	if err := i.Dir.ClearPrompt(); err != nil {
		return fmt.Errorf("failed to clear prompt: %w", err)
	}

	// 5. Execute Pre-Commands (Script at top of file) LAST
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