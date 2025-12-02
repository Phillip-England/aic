package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/phillip-england/aic/internal/tokenizer"
	"github.com/phillip-england/aic/internal/whip"
)

func main() {
	// 1. Define Command Factories
	promptFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &PromptCmd{}, nil
	}

	helpFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &HelpCmd{}, nil
	}

	// 2. Initialize Whip CLI
	cli, err := whip.New(helpFactory)
	if err != nil {
		log.Fatalf("Failed to initialize CLI: %v", err)
	}

	// 3. Register Commands
	cli.At("prompt", promptFactory)
	cli.At("help", helpFactory)

	// 4. Run
	if err := cli.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// --- COMMAND IMPLEMENTATIONS ---

// 1. PromptCmd
type PromptCmd struct{}

func (c *PromptCmd) Execute(cli *whip.Cli) error {
	// The prompt string is expected to be the argument after "prompt"
	input, ok := cli.ArgGetByPosition(2)
	if !ok {
		return fmt.Errorf("Usage: aic prompt \"your prompt with @file/paths\"")
	}

	runPrompt(input)
	return nil
}

// 2. HelpCmd
type HelpCmd struct{}

func (c *HelpCmd) Execute(cli *whip.Cli) error {
	printUsage()
	return nil
}

// --- HELPER LOGIC ---

func runPrompt(input string) {
	tokens := tokenizer.Tokenize(input)

	var finalOutput strings.Builder
	filesFound := 0

	for _, token := range tokens {

		// Case 1: Standard Text (or expanded directories from the tokenizer)
		if token.Type() == tokenizer.RawText {
			finalOutput.WriteString(token.Literal())
			continue
		}

		// Case 2: File Path (@file.go)
		// We replace the token with the actual file content inline
		if token.Type() == tokenizer.FilePath {
			relativePath := token.Value()
			absPath, err := filepath.Abs(relativePath)

			// If we can't resolve/read the file, we just leave the original text (@file)
			// so the user sees the error in the prompt or knows it wasn't expanded.
			if err != nil {
				fmt.Printf("⚠️ Warning: Could not resolve path %s\n", relativePath)
				finalOutput.WriteString(token.Literal())
				continue
			}

			content, err := os.ReadFile(absPath)
			if err != nil {
				fmt.Printf("⚠️ Warning: Could not read file %s: %v\n", relativePath, err)
				finalOutput.WriteString(token.Literal())
				continue
			}

			// Add visual delimiters so the LLM knows where the file starts and ends
			fmt.Fprintf(&finalOutput, "\n\n--- START FILE: %s ---\n", relativePath)
			finalOutput.Write(content)
			fmt.Fprintf(&finalOutput, "\n--- END FILE: %s ---\n\n", relativePath)

			filesFound++
		}
	}

	finalString := finalOutput.String()

	// Output to Console
	fmt.Println("---------------------")
	fmt.Println("--- PROMPT START ---")
	fmt.Print(finalString)
	fmt.Println("\n--- PROMPT END ---")

	// Copy to Clipboard
	if err := clipboard.WriteAll(finalString); err != nil {
		log.Fatalf("Failed to copy to clipboard: %v", err)
	}

	fmt.Println("---------------------")
	fmt.Printf("Success! Copied prompt + %d inlined files to clipboard.\n", filesFound)
}

func printUsage() {
	fmt.Println("Usage: ctx [command]")
	fmt.Println("\nCommands:")
	fmt.Println("  (no command) -> Show this help message")
	fmt.Println("  prompt \"...\" -> Wraps text and expands @file/paths INLINE")
	fmt.Println("  help         -> Show this help message")
}
