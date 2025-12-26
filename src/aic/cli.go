package aic

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
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
	case "watch":
		return c.cmdWatch(sub)
	default:
		fmt.Fprintf(c.Err, "Unknown command: %s\n\n", cmd)
		c.printHelp("")
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func (c *CLI) cmdDefault() error {
	aiDir, err := OpenAiDir()
	if err != nil {
		aiDir, err = NewAiDir(false)
		if err != nil {
			return err
		}
	}

	out, err := c.renderPromptToClipboard(aiDir)
	if err != nil {
		return err
	}

	fmt.Fprint(c.Out, out)
	if !strings.HasSuffix(out, "\n") {
		fmt.Fprintln(c.Out)
	}
	fmt.Fprintln(c.Err, "[copied output to clipboard]")
	return nil
}

func (c *CLI) cmdWatch(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	poll := fs.Duration("poll", 200*time.Millisecond, "poll interval for file changes")
	debounce := fs.Duration("debounce", 350*time.Millisecond, "debounce window to treat changes as a single save")

	if err := fs.Parse(args); err != nil {
		return err
	}

	aiDir, err := OpenAiDir()
	if err != nil {
		aiDir, err = NewAiDir(false)
		if err != nil {
			return err
		}
	}

	promptPath := aiDir.PromptPath()
	if _, err := os.Stat(promptPath); err != nil {
		return fmt.Errorf("prompt.md not found: %s", promptPath)
	}

	fmt.Fprintf(c.Err, "Watching: %s\n", promptPath)
	fmt.Fprintln(c.Err, "Press Ctrl+C to stop.")

	// Perform initial render so clipboard is ready immediately
	if out, err := c.renderPromptToClipboard(aiDir); err != nil {
		fmt.Fprintf(c.Err, "initial render error: %v\n", err)
	} else {
		fmt.Fprintf(c.Err, "initial copy [%d chars]\n", len(out))
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	lastMod, lastSize, err := fileModSize(promptPath)
	if err != nil {
		return err
	}

	var pending bool
	var pendingSince time.Time
	ticker := time.NewTicker(*poll)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			fmt.Fprintln(c.Err, "")
			fmt.Fprintln(c.Err, "Stopped.")
			return nil
		case <-ticker.C:
			mod, size, statErr := fileModSize(promptPath)
			if statErr != nil {
				continue
			}
			changed := mod.After(lastMod) || size != lastSize
			if changed {
				lastMod = mod
				lastSize = size
				pending = true
				pendingSince = time.Now()
				continue
			}

			if pending && time.Since(pendingSince) >= *debounce {
				pending = false
				out, rerr := c.renderPromptToClipboard(aiDir)
				if rerr != nil {
					fmt.Fprintf(c.Err, "render error: %v\n", rerr)
					continue
				}
				fmt.Fprintf(c.Err, "updated clipboard [%d chars]\n", len(out))
			}
		}
	}
}

func fileModSize(path string) (time.Time, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, 0, err
	}
	return info.ModTime(), info.Size(), nil
}

func (c *CLI) renderPromptToClipboard(aiDir *AiDir) (string, error) {
	text, err := aiDir.PromptText()
	if err != nil {
		return "", err
	}

	// Step 1: Pre-process (Remove comments and empty lines)
	// This ensures that commented-out tokens like // $clear() are removed
	// before the tokenizer even sees them.
	processed := PreProcess(text)

	// Step 2: Tokenize and Parse
	reader := NewPromptReader(processed)
	reader.ValidateOrDowngrade(aiDir)
	reader.BindTokens()

	// Step 3: Render (Execute tokens)
	out, err := reader.Render(aiDir)
	if err != nil {
		return "", err
	}

	// Step 4: Apply Labels (Context/Prompt separation)
	out = applyLabels(out)

	if err := clipboard.WriteAll(out); err != nil {
		return "", fmt.Errorf("copy to clipboard: %w", err)
	}

	return out, nil
}

func applyLabels(in string) string {
	s := strings.TrimSpace(in)
	if !strings.HasPrefix(s, "---") {
		return s
	}
	s = s[3:]
	splitIdx := strings.Index(s, "\n---")
	if splitIdx == -1 {
		return "---" + s
	}

	contextContent := strings.TrimSpace(s[:splitIdx])
	promptStart := splitIdx + 4
	promptContent := ""
	if promptStart < len(s) {
		promptContent = strings.TrimSpace(s[promptStart:])
	}

	var sb strings.Builder
	if contextContent != "" {
		sb.WriteString("=== CONTEXT ===\n")
		sb.WriteString(contextContent)
		sb.WriteString("\n\n")
	}
	sb.WriteString("=== PROMPT ===\n")
	sb.WriteString(promptContent)
	return sb.String()
}

func (c *CLI) printHelp(topic string) {
	switch topic {
	case "init":
		fmt.Fprint(c.Out, `Usage:
  aic init [--force]

Creates ./ai and writes ./ai/prompt.md (only).

Options:
  --force   Remove existing ./ai before creating it.
`)
		return
	case "watch":
		fmt.Fprint(c.Out, `Usage:
  aic watch [--poll DURATION] [--debounce DURATION]

Watches ./ai/prompt.md for changes. On save (debounced), tokenizes and copies output to clipboard.

Options:
  --poll       Poll interval (default: 200ms)
  --debounce   Stable window to consider file "saved" (default: 350ms)
`)
		return
	case "help":
		fmt.Fprint(c.Out, `Usage:
  aic help [command]

Shows help for a command (or general help).
`)
		return
	case "version":
		fmt.Fprint(c.Out, `Usage:
  aic version

Prints the CLI version.
`)
		return
	case "":
	default:
		fmt.Fprintf(c.Out, "No detailed help for %q.\n\n", topic)
	}

	fmt.Fprint(c.Out, `aic - minimal CLI

Usage:
  aic <command> [args]

Commands:
  init         Create ./ai with prompt.md only
  watch        Watch ./ai/prompt.md and copy expanded output to clipboard on save
  help         Show help (optionally for a command)
  version      Print version

Default:
  Running with no command prints the expanded prompt (./ai/prompt.md) and copies output to clipboard.

Examples:
  aic
  aic watch
  aic watch --debounce 500ms
  aic init --force
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
	fmt.Fprintln(c.Out, "  skills:", aiDir.Skills)
	return nil
}