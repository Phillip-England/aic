package aic

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
)

const Version = "0.0.1"

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
	case "listen":
		return c.cmdListen(sub)
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

	out, mx, my, _, err := c.renderPromptToClipboard(aiDir)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		fmt.Fprintln(c.Err, "[skipped: empty prompt section]")
		return nil
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

	stopSeq, err := c.startSequenceSystem()
	if err != nil {
		return err
	}
	defer stopSeq()

	fmt.Fprintf(c.Err, "Watching: %s\n", promptPath)
	fmt.Fprintln(c.Err, "Press Ctrl+C to stop.")
	fmt.Fprintln(c.Err, "Sequences: type SPACE ' ; then key (example: ' ;1 prints mouse coords).")

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

				out, mx, my, modified, rerr := c.renderPromptToClipboard(aiDir)
				if rerr != nil {
					fmt.Fprintf(c.Err, "render error: %v\n", rerr)
					continue
				}
				if strings.TrimSpace(out) == "" {
					fmt.Fprintln(c.Err, "skipped (empty prompt section)")
					continue
				}
				if modified {
					if m, s, err := fileModSize(promptPath); err == nil {
						lastMod = m
						lastSize = s
					}
				}
				fmt.Fprintf(c.Err, "updated clipboard [%d chars] mouse=(%d,%d)\n", len(out), mx, my)
			}
		}
	}
}

func (c *CLI) cmdListen(args []string) error {
	fs := flag.NewFlagSet("listen", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	if err := fs.Parse(args); err != nil {
		return err
	}

	stopSeq, err := c.startSequenceSystem()
	if err != nil {
		return err
	}
	defer stopSeq()

	fmt.Fprintln(c.Err, "Listening for sequences globally.")
	fmt.Fprintln(c.Err, "Leader is SPACE ' ; then a key. Example: ' ;1 prints mouse coords.")
	fmt.Fprintln(c.Err, "Press Ctrl+C to stop.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	<-stop
	fmt.Fprintln(c.Err, "")
	fmt.Fprintln(c.Err, "Stopped.")
	return nil
}

func fileModSize(path string) (time.Time, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, 0, err
	}
	return info.ModTime(), info.Size(), nil
}

func (c *CLI) renderPromptToClipboard(aiDir *AiDir) (string, int, int, bool, error) {
	startX, startY := robotgo.GetMousePos()

	rawText, err := aiDir.PromptText()
	if err != nil {
		return "", 0, 0, false, err
	}
	if !promptSectionHasContent(rawText) {
		mx, my := robotgo.GetMousePos()
		return "", mx, my, false, nil
	}

	processed := PreProcess(rawText)
	reader := NewPromptReader(processed)

	if vars, verr := LoadVars(aiDir); verr == nil {
		for k, v := range vars {
			reader.SetVar(k, v)
		}
	}

	if aiDir != nil {
		reader.SetVar("AIC_PROJECT_ROOT", aiDir.WorkingDir)
		reader.SetVar("AIC_AI_DIR", aiDir.Root)
		reader.SetVar("AIC_RULES_DIR", aiDir.Rules)
		reader.SetVar("AIC_VARS_DIR", aiDir.Vars)
	}

	reader.SetVar("AIC_X_START", strconv.Itoa(startX))
	reader.SetVar("AIC_Y_START", strconv.Itoa(startY))

	reader.ValidateOrDowngrade(aiDir)
	reader.BindTokens()

	hasNoRules := false
	for _, tok := range reader.Tokens {
		if dt, ok := tok.(*DollarToken); ok {
			if dt.name == "norules" {
				hasNoRules = true
				break
			}
		}
	}

	if !hasNoRules && aiDir != nil {
		rulesText, err := LoadRules(aiDir)
		if err == nil && rulesText != "" {
			reader.Tokens = append(reader.Tokens, NewRawToken("\n"+rulesText))
		}
	}

	out, err := reader.Render(aiDir)
	if err != nil {
		return "", 0, 0, false, err
	}

	out = applyLabels(out)
	out = PreProcess(out)

	mod1, err := executePostActions(reader.PostActions, PostActionBefore, actionDelay, reader.Vars, aiDir)
	if err != nil {
		return "", 0, 0, false, err
	}

	if err := clipboard.WriteAll(out); err != nil {
		return "", 0, 0, false, err
	}

	mod2, err := executePostActions(reader.PostActions, PostActionAfter, actionDelay, reader.Vars, aiDir)
	if err != nil {
		fmt.Fprintln(c.Err, "post-action error:", err)
	}

	mod3 := false
	if aiDir != nil {
		if err := aiDir.StashRawPrompt(rawText); err != nil {
			fmt.Fprintln(c.Err, "prompt stash error:", err)
		}
		if err := clearPromptPreserveContext(aiDir); err != nil {
			fmt.Fprintln(c.Err, "prompt clear error:", err)
		} else {
			mod3 = true
		}
	}

	mx, my := robotgo.GetMousePos()
	return out, mx, my, mod1 || mod2 || mod3, nil
}

func executePostActions(actions []PostAction, phase PostActionPhase, delay time.Duration, vars map[string]string, aiDir *AiDir) (bool, error) {
	modified := false

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
				return modified, err
			}
			y, err := resolveInt(a.Y, a.YExpr)
			if err != nil {
				return modified, err
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
				return modified, err
			}

		case PostActionSleep:
			if a.Sleep > 0 {
				time.Sleep(a.Sleep)
			} else if a.Sleep == 0 {
				// no-op
			}

		case PostActionPress:
			pressKey(a.Key)

		case PostActionClear:
			// no-op (reserved)
		}

		if delay > 0 {
			time.Sleep(delay)
		}
	}

	return modified, nil
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

func pressKey(key string) {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return
	}
	robotgo.KeyTap(k)
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

func looksLikeHotkeyKey(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return false
	}
	if utf8.RuneCountInString(t) == 1 {
		r, _ := utf8.DecodeRuneInString(t)
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return true
		}
		if strings.ContainsRune("[]\\;',./`-=", r) {
			return true
		}
		return false
	}
	k := strings.ToLower(t)
	switch k {
	case "enter", "return", "tab", "space", "esc", "escape",
		"backspace", "delete", "del",
		"left", "right", "up", "down",
		"home", "end", "pageup", "pagedown":
		return true
	}
	if len(k) >= 2 && k[0] == 'f' {
		n := k[1:]
		if n == "" {
			return false
		}
		for i := 0; i < len(n); i++ {
			if n[i] < '0' || n[i] > '9' {
				return false
			}
		}
		return true
	}
	return false
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

	if len(norm) > 0 && looksLikeHotkeyKey(text) {
		key := strings.ToLower(strings.TrimSpace(text))
		if runtime.GOOS == "darwin" {
			for i := range norm {
				if norm[i] == "lcmd" || norm[i] == "rcmd" {
					norm[i] = "cmd"
				}
			}
		}
		tapArgs := make([]interface{}, 0, len(norm))
		for _, m := range norm {
			tapArgs = append(tapArgs, m)
		}
		robotgo.KeyTap(key, tapArgs...)
		return nil
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
	s = s[3:] // skip first ---
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
Also creates ./ai/rules/, ./ai/vars/, and ./ai/prompts/.
prompt.md starts with:
  ---
  $path(".")
  === PROMPT ===
Options:
  --force    Remove existing ./ai before creating it.
`)
		return
	case "watch":
		fmt.Fprint(c.Out, `Usage:
  aic watch [--poll DURATION] [--debounce DURATION]
Watches ./ai/prompt.md for changes. On save (debounced), tokenizes and copies output to clipboard.
Automatically includes all files in ./ai/rules/ unless $norules() is used.
After each successful copy, the RAW prompt.md is saved to ./ai/prompts/ (keeps last 100)
and prompt.md is cleared back to its header/context.
Also runs the global Sequence listener while watching:
  leader is SPACE ' ; then a key (example: ' ;1 prints mouse coords)
Options:
  --poll          Poll interval (default: 200ms)
  --debounce      Stable window to consider file "saved" (default: 350ms)
`)
		return
	case "listen":
		fmt.Fprint(c.Out, `Usage:
  aic listen
Runs the global Sequence listener.
Leader is SPACE ' ; then a key (example: ' ;1 prints mouse coords).
Press Ctrl+C to stop.
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
  init            Create ./ai with prompt.md, rules/, vars/, and prompts/
  watch           Watch ./ai/prompt.md and copy expanded output on save
  listen          Listen globally for leader sequences (SPACE ' ; then key)
  help            Show help (optionally for a command)
  version         Print version
Default:
  Running with no command prints the expanded prompt (./ai/prompt.md) and copies output to clipboard.
  Files in ./ai/rules/ are AUTOMATICALLY included at the end of the prompt unless $norules() is present.
  After each successful copy, prompt.md is automatically cleared and the RAW prompt is stashed into ./ai/prompts/
  (keeping the last 100).
Tokens:
  $path("...")                    include files under project root
  $shell("...")                   run a shell command (alias: $sh)
  $norules()                      do NOT automatically include files from ./ai/rules/
  $jump(x,y)                      mouse move after copy (supports vars: $jump(AIC_X_START,AIC_Y_START))
  $click() / $click("right")      mouse click after copy
  $type("text", ["MODS"], ms)     type text while holding modifiers after copy (ms optional per-key delay)
  $press("ENTER")                 press a single key (ENTER, BACKSPACE, TAB, ESC, arrows, F1..F24, etc.)
  $sleep(250)                     sleep for N milliseconds after copy
  $sleep("750ms")                 sleep for a duration string after copy (e.g. "1.2s")
Sequences (global, when aic watch/listen is running):
  Leader is SPACE ' ; then a command key
  Example: ' ;1 prints current mouse coords
Vars:
  Put KEY=VALUE files in ./ai/vars/ and reference KEY in $jump(...).
  Built-ins:
  AIC_X_START, AIC_Y_START (mouse position at start of render)
  AIC_PROJECT_ROOT, AIC_AI_DIR, AIC_RULES_DIR, AIC_VARS_DIR
Examples:
  aic
  aic watch
  aic listen
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
	fmt.Fprintln(c.Out, "  rules:", aiDir.Rules)
	fmt.Fprintln(c.Out, "  vars:", aiDir.Vars)
	fmt.Fprintln(c.Out, "  prompts:", aiDir.PromptsDir())
	return nil
}

func promptSectionHasContent(raw string) bool {
	s := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(s, "\n")

	promptIdx := -1
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "=== PROMPT ===" {
			promptIdx = i
			break
		}
	}

	if promptIdx != -1 {
		after := []string{}
		if promptIdx+1 < len(lines) {
			after = lines[promptIdx+1:]
		}

		sep := -1
		for i := 0; i < len(after); i++ {
			if strings.TrimSpace(after[i]) == "---" {
				sep = i
				break
			}
		}

		var promptPart string
		if sep != -1 {
			if sep+1 < len(after) {
				promptPart = strings.Join(after[sep+1:], "\n")
			} else {
				promptPart = ""
			}
		} else {
			promptPart = strings.Join(after, "\n")
		}

		return strings.TrimSpace(PreProcess(promptPart)) != ""
	}

	if strings.HasPrefix(strings.TrimSpace(s), "---") {
		ss := strings.TrimSpace(s)[3:]
		const sep = "\n---\n"
		if splitIdx := strings.Index(ss, sep); splitIdx != -1 {
			promptStart := splitIdx + len(sep)
			promptPart := ""
			if promptStart < len(ss) {
				promptPart = ss[promptStart:]
			}
			return strings.TrimSpace(PreProcess(promptPart)) != ""
		}
	}

	return strings.TrimSpace(PreProcess(s)) != ""
}

func (c *CLI) startSequenceSystem() (func(), error) {
	mgr := NewSequenceManager()
	if err := mgr.Register(MouseCoordsSequence{}); err != nil {
		return nil, err
	}

	listener := NewSequenceListener(mgr)

	// Sequence debug logging removed (sequence system still runs).
	stop, err := listener.Start(func(key string) error {
		seq, ok := mgr.Get(key)
		if !ok {
			_, _ = fmt.Fprintf(c.Err, "[seq] unknown command %q\n", key)
			return nil
		}
		if err := seq.Run(SeqContext{Out: c.Out, Err: c.Err}); err != nil {
			_, _ = fmt.Fprintf(c.Err, "[seq %q] error: %v\n", seq.Key(), err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return stop, nil
}
