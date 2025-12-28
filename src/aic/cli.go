package aic

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
)

const Version = "0.0.1"

// Delay applied AFTER each post-action (jump/click/type/clear).
const actionDelay = 50 * time.Millisecond

const defaultTypeDelay = 20 * time.Millisecond

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

	out, mx, my, err := c.renderPromptToClipboard(aiDir)
	if err != nil {
		return err
	}

	fmt.Fprint(c.Out, out)
	if !strings.HasSuffix(out, "\n") {
		fmt.Fprintln(c.Out)
	}
	fmt.Fprintf(c.Err, "[copied output to clipboard] mouse=(%d,%d)\n", mx, my)
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

	if out, mx, my, err := c.renderPromptToClipboard(aiDir); err != nil {
		fmt.Fprintf(c.Err, "initial render error: %v\n", err)
	} else {
		fmt.Fprintf(c.Err, "initial copy [%d chars] mouse=(%d,%d)\n", len(out), mx, my)
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
				out, mx, my, rerr := c.renderPromptToClipboard(aiDir)
				if rerr != nil {
					fmt.Fprintf(c.Err, "render error: %v\n", rerr)
					continue
				}
				fmt.Fprintf(c.Err, "updated clipboard [%d chars] mouse=(%d,%d)\n", len(out), mx, my)
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

func (c *CLI) renderPromptToClipboard(aiDir *AiDir) (string, int, int, error) {
	startX, startY := robotgo.GetMousePos()

	text, err := aiDir.PromptText()
	if err != nil {
		return "", 0, 0, err
	}

	processed := PreProcess(text)
	reader := NewPromptReader(processed)

	if vars, verr := LoadVars(aiDir); verr == nil {
		for k, v := range vars {
			reader.SetVar(k, v)
		}
	}

	if aiDir != nil {
		reader.SetVar("AIC_PROJECT_ROOT", aiDir.WorkingDir)
		reader.SetVar("AIC_AI_DIR", aiDir.Root)
		reader.SetVar("AIC_SKILLS_DIR", aiDir.Skills)
		reader.SetVar("AIC_VARS_DIR", aiDir.Vars)
	}

	reader.SetVar("AIC_X_START", strconv.Itoa(startX))
	reader.SetVar("AIC_Y_START", strconv.Itoa(startY))

	reader.ValidateOrDowngrade(aiDir)
	reader.BindTokens()

	out, err := reader.Render(aiDir)
	if err != nil {
		return "", 0, 0, err
	}

	out = applyLabels(out)
	out = PreProcess(out)

	if err := executePostActions(reader.PostActions, PostActionBefore, actionDelay, reader.Vars, aiDir); err != nil {
		return "", 0, 0, err
	}

	if err := clipboard.WriteAll(out); err != nil {
		return "", 0, 0, err
	}

	if err := executePostActions(reader.PostActions, PostActionAfter, actionDelay, reader.Vars, aiDir); err != nil {
		fmt.Fprintln(c.Err, "post-action error:", err)
	}

	mx, my := robotgo.GetMousePos()
	return out, mx, my, nil
}

func executePostActions(actions []PostAction, phase PostActionPhase, delay time.Duration, vars map[string]string, aiDir *AiDir) error {
	resolveInt := func(v int, expr string) (int, error) {
		if strings.TrimSpace(expr) == "" {
			return v, nil
		}
		if vars == nil {
			return 0, fmt.Errorf("missing vars map (needed %q)", expr)
		}
		s, ok := vars[expr]
		if !ok {
			return 0, fmt.Errorf("unknown var %q", expr)
		}
		i, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return 0, fmt.Errorf("var %q is not an int: %q", expr, s)
		}
		return i, nil
	}

	for _, a := range actions {
		if a.Phase != phase {
			continue
		}

		switch a.Kind {
		case PostActionJump:
			x, err := resolveInt(a.X, a.XExpr)
			if err != nil {
				return err
			}
			y, err := resolveInt(a.Y, a.YExpr)
			if err != nil {
				return err
			}
			mouseJump(x, y)

		case PostActionClick:
			mouseClick(a.Button)

		case PostActionType:
			perKey := defaultTypeDelay
			if a.DelayMs > 0 {
				perKey = time.Duration(a.DelayMs) * time.Millisecond
			}
			if err := typeWithModifiers(a.Text, a.Mods, perKey); err != nil {
				return err
			}

		case PostActionClear:
			if aiDir == nil {
				return fmt.Errorf("clearAfter requires AiDir")
			}
			if err := os.WriteFile(aiDir.PromptPath(), []byte(promptHeader), 0o644); err != nil {
				return fmt.Errorf("clear prompt.md: %w", err)
			}
		}

		// Delay is applied AFTER the action.
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	return nil
}

func mouseJump(x, y int) {
	robotgo.Move(x, y)
}

func mouseClick(btn string) {
	btn = strings.ToLower(strings.TrimSpace(btn))
	if btn == "" || btn == "left" {
		robotgo.Click()
		return
	}
	robotgo.Click(btn) // e.g. "right"
}

func normalizeModifier(mod string) string {
	m := strings.TrimSpace(mod)
	if m == "" {
		return ""
	}
	u := strings.ToUpper(m)
	switch u {
	case "CTRL":
		u = "CONTROL"
	case "CMD":
		u = "COMMAND"
	case "WIN":
		u = "WINDOWS"
	}

	base := func(key string) string {
		switch key {
		case "SHIFT":
			return "shift"
		case "CONTROL":
			return "ctrl"
		case "ALT":
			return "alt"
		case "OPTION":
			return "alt"
		case "COMMAND":
			return "cmd"
		case "WINDOWS":
			return "cmd"
		case "SUPER":
			return "super"
		case "META":
			return "cmd"
		default:
			return ""
		}
	}

	if strings.HasPrefix(u, "LEFT_") {
		b := base(strings.TrimPrefix(u, "LEFT_"))
		if b == "" {
			return "l" + strings.ToLower(strings.TrimPrefix(u, "LEFT_"))
		}
		return "l" + b
	}

	if strings.HasPrefix(u, "RIGHT_") {
		b := base(strings.TrimPrefix(u, "RIGHT_"))
		if b == "" {
			return "r" + strings.ToLower(strings.TrimPrefix(u, "RIGHT_"))
		}
		return "r" + b
	}

	if b := base(u); b != "" {
		return b
	}

	return strings.ToLower(m)
}

func typeWithModifiers(text string, mods []string, perKeyDelay time.Duration) error {
	if text == "" {
		return nil
	}

	var norm []string
	for _, m := range mods {
		n := normalizeModifier(m)
		if n != "" {
			norm = append(norm, n)
		}
	}

	for _, m := range norm {
		robotgo.KeyDown(m)
	}

	delayMs := int(perKeyDelay / time.Millisecond)
	if delayMs <= 0 {
		robotgo.TypeStr(text)
	} else {
		robotgo.TypeStrDelay(text, delayMs)
	}

	for i := len(norm) - 1; i >= 0; i-- {
		robotgo.KeyUp(norm[i])
	}

	return nil
}

func applyLabels(in string) string {
	s := strings.TrimSpace(in)
	if !strings.HasPrefix(s, "---") {
		return s
	}
	s = s[3:]
	const separator = "\n---\n"
	splitIdx := strings.Index(s, separator)
	if splitIdx == -1 {
		return "===" + s
	}

	contextContent := strings.TrimSpace(s[:splitIdx])
	promptStart := splitIdx + len(separator)
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
prompt.md starts with:
  ---
  $path(".")
  ---
Options:
  --force   Remove existing ./ai before creating it.
`)
		return

	case "watch":
		fmt.Fprint(c.Out, `Usage:
  aic watch [--poll DURATION] [--debounce DURATION]
Watches ./ai/prompt.md for changes. On save (debounced), tokenizes and copies output to clipboard.
Options:
  --poll        Poll interval (default: 200ms)
  --debounce    Stable window to consider file "saved" (default: 350ms)
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
  init          Create ./ai with prompt.md only
  watch         Watch ./ai/prompt.md and copy expanded output to clipboard on save
  help          Show help (optionally for a command)
  version       Print version
Default:
  Running with no command prints the expanded prompt (./ai/prompt.md) and copies output to clipboard.
Tokens:
  $path("...")                 include files under project root
  $shell("...")                run a shell command (alias: $sh)
  $clear()                     clear prompt.md back to header
  $clearAfter()                clear prompt.md back to header AFTER the current prompt is copied
  $skill("name")               include a skill markdown file
  $jump(x,y)                   mouse move after copy (supports vars: $jump(AIC_X_START,AIC_Y_START))
  $click() / $click("right")   mouse click after copy
  $type("text", ["MODS"], ms)  type text while holding modifiers after copy (ms optional per-key delay)
Vars:
  Put KEY=VALUE files in ./ai/vars/ and reference KEY in $jump(...).
Built-ins:
  AIC_X_START, AIC_Y_START (mouse position at start of render)
  AIC_PROJECT_ROOT, AIC_AI_DIR, AIC_SKILLS_DIR, AIC_VARS_DIR
Examples:
  aic
  aic watch
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
	fmt.Fprintln(c.Out, "  vars:", aiDir.Vars)
	return nil
}
