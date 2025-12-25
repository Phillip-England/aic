package aic

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

type DollarToken struct {
	TokenCtx
	literal string // includes leading "$"

	// parsed during Validate()
	isSh      bool
	shCmd     string // command to execute (decoded string literal)
	shCmdDisp string // for printing (the quoted literal as provided)
	isClr     bool
}

func NewDollarToken(lit string) PromptToken {
	return &DollarToken{literal: lit}
}

func (t *DollarToken) Type() PromptTokenType { return PromptTokenDollar }
func (t *DollarToken) Literal() string       { return t.literal }

func (t *DollarToken) Value() string {
	return strings.TrimPrefix(t.literal, "$")
}

// parseDollarCall parses `name(args)` from a `$...` token value (value excludes leading `$`).
// Returns (name, argsInsideParens, okCallForm).
func parseDollarCall(val string) (string, string, bool) {
	s := strings.TrimSpace(val)

	open := strings.IndexByte(s, '(')
	if open < 0 {
		return "", "", false
	}
	if !strings.HasSuffix(s, ")") {
		return "", "", false
	}

	name := strings.TrimSpace(s[:open])
	args := s[open+1 : len(s)-1] // inside parentheses
	return name, args, true
}

// parseSingleDoubleQuotedStringArg expects args containing exactly one double-quoted string,
// allowing whitespace around it, and returns (decoded, display, error).
func parseSingleDoubleQuotedStringArg(args string) (string, string, error) {
	s := strings.TrimSpace(args)
	if s == "" {
		return "", "", fmt.Errorf("$sh: expected 1 string argument")
	}
	if len(s) < 2 || s[0] != '"' {
		return "", "", fmt.Errorf(`$sh: argument must be a double-quoted string, e.g. $sh("ls")`)
	}

	var out strings.Builder
	i := 1 // after opening quote
	escaped := false
	for i < len(s) {
		ch := s[i]

		if escaped {
			switch ch {
			case '"':
				out.WriteByte('"')
			case '\\':
				out.WriteByte('\\')
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case 'r':
				out.WriteByte('\r')
			default:
				// keep unknown escapes as literal char (simple + predictable)
				out.WriteByte(ch)
			}
			escaped = false
			i++
			continue
		}

		if ch == '\\' {
			escaped = true
			i++
			continue
		}

		if ch == '"' {
			// closing quote
			i++
			break
		}

		out.WriteByte(ch)
		i++
	}

	if escaped {
		return "", "", fmt.Errorf("$sh: trailing backslash in string")
	}

	// must have ended on a closing quote
	if i <= 1 || i > len(s) || s[i-1] != '"' {
		return "", "", fmt.Errorf("$sh: unterminated string")
	}

	// only whitespace allowed after closing quote
	rest := strings.TrimSpace(s[i:])
	if rest != "" {
		return "", "", fmt.Errorf("$sh: expected exactly one string argument")
	}

	decoded := out.String()
	display := `"` + decoded + `"` // normalized display (decoded)
	if decoded == "" {
		display = `""`
	}
	return decoded, display, nil
}

func (t *DollarToken) Validate(d *AiDir) error {
	t.isSh = false
	t.shCmd = ""
	t.shCmdDisp = ""
	t.isClr = false

	val := t.Value()
	name, args, ok := parseDollarCall(val)

	// All dollar commands must be call-form now.
	if !ok {
		// If it's some unknown literal like "$foo" we still allow it to render literally,
		// but for known commands we enforce call form via checks below.
		// We'll decide based on prefix matches:
		trim := strings.TrimSpace(val)
		if trim == "clr" || trim == "sh" || strings.HasPrefix(trim, "clr") || strings.HasPrefix(trim, "sh") {
			return fmt.Errorf("$: commands must be called with (), e.g. $clr() or $sh(\"ls\")")
		}
		return nil
	}

	// known commands
	switch name {
	case "clr":
		if strings.TrimSpace(args) != "" {
			return fmt.Errorf("$clr: takes no arguments")
		}
		t.isClr = true
		return nil

	case "sh":
		cmd, disp, err := parseSingleDoubleQuotedStringArg(args)
		if err != nil {
			return err
		}

		// basic hardening
		if strings.ContainsRune(cmd, '\x00') {
			return fmt.Errorf("$sh: command contains NUL")
		}
		if len(cmd) > 4096 {
			return fmt.Errorf("$sh: command too long")
		}

		t.isSh = true
		t.shCmd = cmd
		t.shCmdDisp = disp
		return nil

	default:
		// Unknown $name(...) forms are allowed (render literally)
		return nil
	}
}

func (t *DollarToken) AfterValidate(r *PromptReader, index int) error {
	t.bind(r, index)
	return nil
}

func renderShOutput(cmdDisplay string, out []byte) string {
	if bytes.IndexByte(out, 0) >= 0 || !utf8.Valid(out) {
		enc := base64.StdEncoding.EncodeToString(out)
		var sb strings.Builder
		sb.WriteString("sh OUTPUT (base64)\n")
		sb.WriteString("CMD: ")
		sb.WriteString(cmdDisplay)
		sb.WriteString("\n")
		sb.WriteString(enc)
		if !strings.HasSuffix(enc, "\n") {
			sb.WriteString("\n")
		}
		return sb.String()
	}

	s := string(out)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

func (t *DollarToken) Render(d *AiDir) (string, error) {
	// $clr()
	if t.isClr {
		if d == nil {
			return "", fmt.Errorf("$clr: missing ai dir")
		}
		path := d.PromptPath()
		if path == "" {
			return "", fmt.Errorf("$clr: missing prompt path")
		}

		if err := os.WriteFile(path, []byte(promptHeader), 0o644); err != nil {
			return "", fmt.Errorf("$clr: write prompt.md: %w", err)
		}
		return "", nil
	}

	// $sh("...")
	if t.isSh {
		wd := ""
		if d != nil {
			wd = d.WorkingDir
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		c := exec.CommandContext(ctx, "sh", "-lc", t.shCmd)
		if wd != "" {
			c.Dir = wd
		}

		out, err := c.CombinedOutput()

		const maxOut = 256 * 1024
		if len(out) > maxOut {
			out = out[:maxOut]
			out = append(out, []byte("\n...[truncated]\n")...)
		}

		cmdDisplay := t.shCmdDisp
		if cmdDisplay == "" {
			cmdDisplay = `"` + t.shCmd + `"`
		}

		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("sh ERROR: timeout after 2s\nCMD: %s\n", cmdDisplay), nil
		}
		if err != nil {
			var sb strings.Builder
			sb.WriteString("sh ERROR: ")
			sb.WriteString(err.Error())
			sb.WriteString("\nCMD: ")
			sb.WriteString(cmdDisplay)
			sb.WriteString("\n")
			if len(out) > 0 {
				sb.WriteString(renderShOutput(cmdDisplay, out))
				if !strings.HasSuffix(sb.String(), "\n") {
					sb.WriteString("\n")
				}
			}
			return sb.String(), nil
		}

		return renderShOutput(cmdDisplay, out), nil
	}

	// default: render literally
	return t.literal, nil
}

func (t *DollarToken) String() string {
	return fmt.Sprintf("<%s: %q>", t.Type().String(), t.literal)
}
