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

func (c *CLI) Run(args []string) error {
	if c.Out == nil {
		c.Out = os.Stdout
	}
	if c.Err == nil {
		c.Err = os.Stderr
	}

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
	aiDir, err := NewAiDir(false)
	if err != nil {
		if !strings.Contains(err.Error(), "ai dir already exists") {
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

	// Debug: print tokens (after validation/downgrade)
	for i, tok := range reader.Tokens {
		fmt.Fprintf(c.Out, "%d: %s\n", i, tok.String())
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
  Running with no command prints the current prompt (./ai/prompt.md) as tokens
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
