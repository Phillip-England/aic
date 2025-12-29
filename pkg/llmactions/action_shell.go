package llmactions

import (
	"fmt"
	"os/exec"
	"strings"
)

const PrefixShell = "AIC: SHELL"

func HandleShell(input string) error {
	cmdStr := strings.TrimSpace(input)
	if cmdStr == "" {
		return fmt.Errorf("empty shell command")
	}

	// Semicolon check removed. 'sh -c' handles commands without them fine.
	// This makes the parser less brittle.

	fmt.Printf("[ACTION] Executing Shell: %s\n", cmdStr)
	cmd := exec.Command("sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	
	if len(out) > 0 {
		fmt.Printf("--- Output ---\n%s\n------------\n", string(out))
	}
	
	if err != nil {
		return fmt.Errorf("shell execution failed: %w", err)
	}
	return nil
}