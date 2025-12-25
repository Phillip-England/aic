// --- START FILE: aic/cli.go ---
package aic

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const Version = "0.0.1"

type CLI struct {
	Out io.Writer
	Err io.Writer
}

func NewCLI() *CLI {
	return &CLI{
		Out: os.Stdout,
		Err: os.Stderr,
	}
}

// Run routes args to subcommands.
// Expect os.Args[1:] (i.e., without program name).
func (c *CLI) Run(args []string) error {
	if c.Out == nil {
		c.Out = os.Stdout
	}
	if c.Err == nil {
		c.Err = os.Stderr
	}

	// NEW: no args => run default route (print prompt)
	if len(args) == 0 {
		return c.cmdDefault()
	}

	cmd := strings.TrimSpace(args[0])
	sub := args[1:]

	switch cmd {
	case "help", "-h", "--help":
		topic := ""
		if len(sub) > 0 {
			topic = strings.TrimSpace(sub[0])
		}
		c.printHelp(topic)
		return nil

	case "version", "-v", "--version":
		fmt.Fprintln(c.Out, Version)
		return nil

	case "init":
		return c.cmdInit(sub)

	default:
		fmt.Fprintf(c.Err, "Unknown command: %s\n\n", cmd)
		c.printHelp("")
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func (c *CLI) cmdDefault() error {
	// Ensure ./ai exists (without forcing deletion).
	aiDir, err := NewAiDir(false)
	if err != nil {
		// If it already exists, NewAiDir(false) errors; that's fine—we just need to read.
		// So: if it's the "already exists" case, open it by constructing AiDir from cwd.
		if !strings.Contains(err.Error(), "ai dir already exists") {
			// Might be "exists but not directory" etc.
			return err
		}

		wd, werr := os.Getwd()
		if werr != nil {
			return werr
		}
		aiDir = &AiDir{
			Root:    filepath.Join(wd, "ai"),
			Tmp:     filepath.Join(wd, "ai", "tmp"),
			Prompts: filepath.Join(wd, "ai", "prompts"),
			Skills:  filepath.Join(wd, "ai", "skills"),
		}
	}

	text, err := aiDir.PromptText()
	if err != nil {
		return err
	}

	reader := NewPromptReader(text)

	// Print prompt to stdout (use c.Out if it is stdout; otherwise write explicitly).
	// Keep it simple: just write to c.Out.
	if _, werr := io.WriteString(c.Out, reader.String()); werr != nil {
		return werr
	}

	// Ensure newline at end for nicer terminal output.
	if !strings.HasSuffix(reader.String(), "\n") {
		fmt.Fprintln(c.Out)
	}

	return nil
}

func (c *CLI) printHelp(topic string) {
	switch topic {
	case "init":
		fmt.Fprintln(c.Out, `Usage:
  aic init [--force]

Creates ./ai and writes ./ai/prompt.md (only).

Options:
  --force   Remove existing ./ai before creating it.
`)
		return
	case "help":
		fmt.Fprintln(c.Out, `Usage:
  aic help [command]

Shows help for a command (or general help).
`)
		return
	case "version":
		fmt.Fprintln(c.Out, `Usage:
  aic version

Prints the CLI version.
`)
		return
	case "":
		// fallthrough
	default:
		fmt.Fprintf(c.Out, "No detailed help for %q.\n\n", topic)
	}

	fmt.Fprintln(c.Out, `aic - minimal CLI

Usage:
  aic <command> [args]

Commands:
  init       Create ./ai with prompt.md only
  help       Show help (optionally for a command)
  version    Print version

Default:
  Running with no command prints the current prompt (./ai/prompt.md)

Examples:
  aic
  aic init
  aic init --force
  aic help init
`)
}

func (c *CLI) cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(c.Err)

	force := fs.Bool("force", false, "remove existing ./ai dir before creating it")

	if err := fs.Parse(args); err != nil {
		return err
	}

	aiDir, err := NewAiDir(*force)
	if err != nil {
		return err
	}

	relRoot := aiDir.Root
	if wd, werr := os.Getwd(); werr == nil {
		if rel, rerr := filepath.Rel(wd, aiDir.Root); rerr == nil {
			relRoot = "." + string(os.PathSeparator) + rel
		}
	}

	fmt.Fprintln(c.Out, "Initialized:", relRoot)
	fmt.Fprintln(c.Out, "  prompt:", filepath.Join(aiDir.Root, "prompt.md"))
	return nil
}

// --- END FILE: aic/cli.go ---
