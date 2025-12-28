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
	// 1. Parse the file sections
	parts := strings.Split(rawContent, "---")
	
	// Robustness check: Ensure we have at least the expected sections. 
	// [0]empty [1]Pre [2]Body [3]Post [4]empty/trailing
	if len(parts) < 4 {
		return fmt.Errorf("invalid prompt.md format: missing '---' sections")
	}

	preCommands := parts[1]
	promptBody := strings.TrimSpace(parts[2])
	postCommands := parts[3]

	// If prompt body is empty, do nothing
	if promptBody == "" {
		return nil
	}

	// 2. Execute Pre-Commands
	i.executeShell(preCommands)

	// 3. Build the Payload (Clipboard Content + Rules + Prompt)
	currentClip, err := clipboard.ReadAll()
	if err != nil {
		fmt.Printf("[Interpreter] Warning: Could not read clipboard: %v\n", err)
		currentClip = ""
	}

	rules, err := i.Dir.CollectRules()
	if err != nil {
		fmt.Printf("[Interpreter] Warning: Could not collect rules: %v\n", err)
	}

	// Construct final output
	// Format: STUFF CURRENTLY ON CLIPBOARD -> RULES -> PROMPT
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

	// 4. Write to Clipboard
	if err := clipboard.WriteAll(finalOutput.String()); err != nil {
		return fmt.Errorf("clipboard write error: %w", err)
	}

	// 5. Stash the prompt (History)
	if err := i.Dir.StashPrompt(rawContent); err != nil {
		fmt.Printf("[Interpreter] Warning: Failed to stash prompt: %v\n", err)
	}

	// 6. Clear the prompt file (keep pre/post, clear body)
	if err := i.Dir.ClearPrompt(); err != nil {
		return fmt.Errorf("failed to clear prompt: %w", err)
	}

	// 7. Execute Post-Commands
	i.executeShell(postCommands)

	return nil
}

func (i *Interpreter) executeShell(script string) {
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Use 'sh -c' to execute so we can use pipes/env vars if needed
		cmd := exec.Command("sh", "-c", line)
		if err := cmd.Run(); err != nil {
			fmt.Printf("[Interpreter] Command Failed: '%s' -> %v\n", line, err)
		} else {
			// Optional: fmt.Printf("[Interpreter] Ran: %s\n", line)
		}

		// 50ms delay between commands as requested
		time.Sleep(50 * time.Millisecond)
	}
}