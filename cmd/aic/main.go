package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/atotto/clipboard"
	"github.com/phillip-england/aic/internal/context"
	"github.com/phillip-england/aic/internal/scanner"
)

func main() {
	// 1. Setup Environment
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Setup Dependencies
	ctxManager := context.New(cwd)
	fileScanner := scanner.New(cwd)

	// 3. Parse Flags & Commands
	promptPtr := flag.String("p", "", "Prepend a prompt to the clipboard output")
	flag.Usage = printUsage
	flag.Parse()

	args := flag.Args()
	command := ""
	if len(args) > 0 {
		command = args[0]
	}

	// 4. Router
	switch command {
	case "", "collect":
		// === REQUIRED CHECK FOR COLLECT/DEFAULT ===
		if !ctxManager.Exists() {
			fmt.Printf("❌ Error: Context file '%s' not found in %s.\n", context.FileName, cwd)
			fmt.Printf("👉 Please run 'aic init' to initialize the project context.\n")
			os.Exit(1)
		}
		runCollect(fileScanner, ctxManager, *promptPtr)
	case "paths":
		// === REQUIRED CHECK FOR PATHS ===
		if !ctxManager.Exists() {
			fmt.Printf("❌ Error: Context file '%s' not found in %s.\n", context.FileName, cwd)
			fmt.Printf("👉 Please run 'aic init' to initialize the project context.\n")
			os.Exit(1)
		}
		runPaths(fileScanner)
	case "init":
		if err := ctxManager.Init(); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Initialized %s successfully.\n", context.FileName)
	case "add":
		if len(args) < 2 {
			fmt.Println("Usage: ctx add <text>")
			os.Exit(1)
		}
		// NOTE: ctxManager.Add handles its own existence check and error message.
		count, entry, err := ctxManager.Add(args[1:])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Added context entry [%d]: \"%s\"\n", count, entry)
	case "list":
		content, err := ctxManager.List()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("--- Current Application Context ---")
		fmt.Println(content)
		fmt.Println("-----------------------------------")
	case "delete", "rm":
		if len(args) < 2 {
			fmt.Println("Usage: ctx delete <index number>")
			os.Exit(1)
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("Invalid ID: %s", args[1])
		}
		removed, err := ctxManager.Delete(id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Deleted [%d]: \"%s\"\n", id, removed)
		fmt.Println("Context file re-indexed.")
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// --- COMMAND IMPLEMENTATIONS ---

func runCollect(s *scanner.Scanner, c *context.Manager, prompt string) {
	// The existence of the file is guaranteed by the check in main().
	contextContent := ""
	raw, err := c.List()
	if err == nil {
		contextContent = raw
	}
	// Note: If List() returns "No context entries found." the content will still be collected, which is acceptable.

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

	output := ""
	for _, p := range paths {
		output += p + "\n"
	}

	fmt.Print(output)

	if err := clipboard.WriteAll(output); err != nil {
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
	fmt.Println("  (no command) -> Collects all file contents to clipboard")
	fmt.Println("  paths        -> Collects absolute file paths to clipboard")
	fmt.Println("  init         -> Initialize the .ai context file")
	fmt.Println("  add <text>   -> Add a context entry")
	fmt.Println("  list         -> List all context entries")
	fmt.Println("  delete <id>  -> Remove a context entry by ID")
}