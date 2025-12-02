package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/phillip-england/aic/internal/context"
	"github.com/phillip-england/aic/internal/scanner"
	"github.com/phillip-england/aic/internal/tokenizer"
	"github.com/phillip-england/aic/internal/whip"
)

func main() {
	// 1. Setup Environment & Dependencies
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	ctxManager := context.New(cwd)
	fileScanner := scanner.New(cwd)

	// 2. Define Command Factories
	collectFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &CollectCmd{Scanner: fileScanner, Context: ctxManager}, nil
	}

	promptFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &PromptCmd{}, nil
	}

	pathsFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &PathsCmd{Scanner: fileScanner, Context: ctxManager}, nil
	}

	initFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &InitCmd{Context: ctxManager}, nil
	}

	addFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &AddCmd{Context: ctxManager}, nil
	}

	listFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &ListCmd{Context: ctxManager}, nil
	}

	deleteFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &DeleteCmd{Context: ctxManager}, nil
	}

	helpFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &HelpCmd{}, nil
	}

	// 3. Initialize Whip CLI
	cli, err := whip.New(helpFactory)
	if err != nil {
		log.Fatalf("Failed to initialize CLI: %v", err)
	}

	// 4. Register Commands
	cli.At("collect", collectFactory)
	cli.At("prompt", promptFactory) // Registered here
	cli.At("paths", pathsFactory)
	cli.At("init", initFactory)
	cli.At("add", addFactory)
	cli.At("list", listFactory)
	cli.At("delete", deleteFactory)
	cli.At("rm", deleteFactory)
	cli.At("help", helpFactory)

	// 5. Run
	if err := cli.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// --- COMMAND IMPLEMENTATIONS ---

// 1. CollectCmd
type CollectCmd struct {
	Scanner *scanner.Scanner
	Context *context.Manager
}

func (c *CollectCmd) Execute(cli *whip.Cli) error {
	if !c.Context.Exists() {
		return fmt.Errorf("Context file '%s' not found. Run 'aic init' first.", context.FileName)
	}

	prompt := ""
	if cli.FlagExists("-p") {
		flagArg := cli.Flags["-p"]
		if val, ok := cli.ArgGetByPosition(flagArg.Position + 1); ok {
			prompt = val
		}
	}

	runCollect(c.Scanner, c.Context, prompt)
	return nil
}

// 2. PromptCmd (New)
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

// 3. PathsCmd
type PathsCmd struct {
	Scanner *scanner.Scanner
	Context *context.Manager
}

func (c *PathsCmd) Execute(cli *whip.Cli) error {
	if !c.Context.Exists() {
		return fmt.Errorf("Context file '%s' not found. Run 'aic init' first.", context.FileName)
	}
	runPaths(c.Scanner)
	return nil
}

// 4. InitCmd
type InitCmd struct {
	Context *context.Manager
}

func (c *InitCmd) Execute(cli *whip.Cli) error {
	if err := c.Context.Init(); err != nil {
		return err
	}
	fmt.Printf("Initialized %s successfully.\n", context.FileName)
	return nil
}

// 5. AddCmd
type AddCmd struct {
	Context *context.Manager
}

func (c *AddCmd) Execute(cli *whip.Cli) error {
	var parts []string
	i := 2
	for {
		val, ok := cli.ArgGetByPosition(i)
		if !ok {
			break
		}
		parts = append(parts, val)
		i++
	}

	if len(parts) == 0 {
		return fmt.Errorf("Usage: ctx add <text>")
	}

	count, entry, err := c.Context.Add(parts)
	if err != nil {
		return err
	}
	fmt.Printf("Added context entry [%d]: \"%s\"\n", count, entry)
	return nil
}

// 6. ListCmd
type ListCmd struct {
	Context *context.Manager
}

func (c *ListCmd) Execute(cli *whip.Cli) error {
	content, err := c.Context.List()
	if err != nil {
		return err
	}
	fmt.Println("--- Current Application Context ---")
	fmt.Println(content)
	fmt.Println("-----------------------------------")
	return nil
}

// 7. DeleteCmd
type DeleteCmd struct {
	Context *context.Manager
}

func (c *DeleteCmd) Execute(cli *whip.Cli) error {
	idStr, ok := cli.ArgGetByPosition(2)
	if !ok {
		return fmt.Errorf("Usage: ctx delete <index number>")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return fmt.Errorf("Invalid ID: %s", idStr)
	}

	removed, err := c.Context.Delete(id)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted [%d]: \"%s\"\n", id, removed)
	fmt.Println("Context file re-indexed.")
	return nil
}

// 8. HelpCmd
type HelpCmd struct{}

func (c *HelpCmd) Execute(cli *whip.Cli) error {
	printUsage()
	return nil
}

// --- HELPER LOGIC ---

func runPrompt(input string) {
	tokens := tokenizer.Tokenize(input)

	var promptBuilder strings.Builder
	var contextBuilder strings.Builder

	filesFound := 0

	for _, token := range tokens {
		// Reconstruct the original prompt text
		promptBuilder.WriteString(token.Literal())

		// If it's a file path, load the content
		if token.Type() == tokenizer.FilePath {
			relativePath := token.Value()
			absPath, err := filepath.Abs(relativePath)
			if err != nil {
				fmt.Printf("⚠️ Warning: Could not resolve path %s\n", relativePath)
				continue
			}

			content, err := os.ReadFile(absPath)
			if err != nil {
				fmt.Printf("⚠️ Warning: Could not read file %s: %v\n", relativePath, err)
				continue
			}

			// Add to the context section (bottom)
			fmt.Fprintf(&contextBuilder, "PATH: %s\n\n", absPath)
			contextBuilder.Write(content)
			contextBuilder.WriteString("\n\n---\n\n")

			filesFound++
		}
	}

	// Combine Prompt + Separator + Context
	var finalOutput strings.Builder
	finalOutput.WriteString(promptBuilder.String())

	if filesFound > 0 {
		finalOutput.WriteString("\n\n")
		finalOutput.WriteString("-----------------------------------\n")
		finalOutput.WriteString("!!! REFERENCED FILES LOADED !!!\n")
		finalOutput.WriteString("-----------------------------------\n\n")
		finalOutput.WriteString(contextBuilder.String())
	}

	finalString := finalOutput.String()

	// Output to Console
	fmt.Println("---------------------")
	fmt.Println("--- PROMPT START ---")
	fmt.Print(finalString)
	fmt.Println("--- PROMPT END ---")

	// Copy to Clipboard
	if err := clipboard.WriteAll(finalString); err != nil {
		log.Fatalf("Failed to copy to clipboard: %v", err)
	}

	fmt.Println("---------------------")
	fmt.Printf("Success! Copied prompt + %d referenced files to clipboard.\n", filesFound)
}

func runCollect(s *scanner.Scanner, c *context.Manager, prompt string) {
	contextContent := ""
	raw, err := c.List()
	if err == nil {
		contextContent = raw
	}

	finalString, count, err := s.CollectContent(prompt, contextContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("---------------------")
	fmt.Println("--- CONTENT START ---")
	fmt.Print(finalString)
	fmt.Println("--- CONTENT END ---")

	if err := clipboard.WriteAll(finalString); err != nil {
		log.Fatalf("Failed to copy to clipboard: %v", err)
	}

	fmt.Println("---------------------")
	fmt.Printf("Success! Copied %d files to clipboard.\n", count)
}

func runPaths(s *scanner.Scanner) {
	paths, err := s.CollectPaths()
	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(paths)

	var output strings.Builder
	for _, p := range paths {
		output.WriteString(p + "\n")
	}

	fmt.Print(output.String())

	if err := clipboard.WriteAll(output.String()); err != nil {
		log.Fatalf("Failed to copy to clipboard: %v", err)
	}
	fmt.Println("---------------------")
	fmt.Printf("Success! Copied %d absolute file paths to clipboard.\n", len(paths))
}

func printUsage() {
	fmt.Println("Usage: ctx [flags] [command]")
	fmt.Println("Flags:")
	fmt.Println("  -p \"text\"    -> Prepend a custom prompt to the clipboard output (collect command only)")
	fmt.Println("\nCommands:")
	fmt.Println("  (no command) -> Show this help message")
	fmt.Println("  prompt \"...\" -> Wraps text and expands @file/paths to content at bottom")
	fmt.Println("  collect      -> Collects all file contents to clipboard")
	fmt.Println("  paths        -> Collects absolute file paths to clipboard")
	fmt.Println("  init         -> Initialize the .ai context file")
	fmt.Println("  add <text>   -> Add a context entry")
	fmt.Println("  list         -> List all context entries")
	fmt.Println("  delete <id>  -> Remove a context entry by ID")
}
