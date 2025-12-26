package aic

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

type DollarToken struct {
	TokenCtx
	literal string // includes leading "$"

	// Command flags and data
	isSh      bool
	shCmd     string
	shCmdDisp string

	isClear bool

	isSkill   bool
	skillName string

	isAt     bool
	atTarget string // Absolute path to target

	isHttp  bool
	httpUrl string
}

func NewDollarToken(lit string) PromptToken {
	return &DollarToken{literal: lit}
}

func (t *DollarToken) Type() PromptTokenType { return PromptTokenDollar }
func (t *DollarToken) Literal() string       { return t.literal }
func (t *DollarToken) Value() string {
	return strings.TrimPrefix(t.literal, "$")
}

// Helpers from old token_at.go
func ensureUnderWorkingDir(targetAbs string, workingDir string) error {
	if workingDir == "" {
		return fmt.Errorf("missing working directory")
	}
	wd := filepath.Clean(workingDir)
	if es, err := filepath.EvalSymlinks(wd); err == nil {
		wd = es
	}
	abs := filepath.Clean(targetAbs)
	if es, err := filepath.EvalSymlinks(abs); err == nil {
		abs = es
	}
	rel, err := filepath.Rel(wd, abs)
	if err != nil {
		return fmt.Errorf("resolve relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path escapes project root: %s", targetAbs)
	}
	return nil
}

func fetchUrlContent(urlStr string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "aic/0.0.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Parsing logic
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

func parseMultiStringArgs(args string) ([]string, error) {
	var results []string
	rest := strings.TrimSpace(args)
	
	// If empty args, return empty slice
	if rest == "" {
		return []string{}, nil
	}

	for len(rest) > 0 {
		if rest[0] != '"' {
			return nil, fmt.Errorf("expected '\"' at start of argument, got %q", rest[0])
		}

		var buf strings.Builder
		escaped := false
		idx := 1
		foundEnd := false

		for idx < len(rest) {
			ch := rest[idx]
			if escaped {
				switch ch {
				case '"':
					buf.WriteByte('"')
				case '\\':
					buf.WriteByte('\\')
				case 'n':
					buf.WriteByte('\n')
				case 't':
					buf.WriteByte('\t')
				case 'r':
					buf.WriteByte('\r')
				default:
					buf.WriteByte(ch)
				}
				escaped = false
				idx++
				continue
			}
			if ch == '\\' {
				escaped = true
				idx++
				continue
			}
			if ch == '"' {
				foundEnd = true
				idx++
				break
			}
			buf.WriteByte(ch)
			idx++
		}

		if !foundEnd {
			return nil, fmt.Errorf("unterminated string")
		}
		results = append(results, buf.String())

		// Consume comma
		rest = strings.TrimSpace(rest[idx:])
		if len(rest) > 0 {
			if rest[0] != ',' {
				return nil, fmt.Errorf("expected comma between arguments")
			}
			rest = strings.TrimSpace(rest[1:])
		}
	}
	return results, nil
}

func (t *DollarToken) Validate(d *AiDir) error {
	// Reset all flags
	t.isSh = false
	t.shCmd = ""
	t.shCmdDisp = ""
	t.isClear = false
	t.isSkill = false
	t.skillName = ""
	t.isAt = false
	t.atTarget = ""
	t.isHttp = false
	t.httpUrl = ""

	val := t.Value()
	name, args, ok := parseDollarCall(val)

	// Enforce function syntax for specific keywords
	if !ok {
		trim := strings.TrimSpace(val)
		keywords := []string{"clear", "sh", "skill", "at", "http", "clr", "sk"}
		for _, k := range keywords {
			if trim == k || strings.HasPrefix(trim, k) {
				return fmt.Errorf("$: command '%s' must be called with (), e.g. $%s(...)", k, k)
			}
		}
		return nil // Not a known command, treat as simple dollar text (unlikely given tokenizer logic but safe)
	}

	argList, err := parseMultiStringArgs(args)
	if err != nil {
		return fmt.Errorf("$%s: %w", name, err)
	}

	switch name {
	case "clear":
		if len(argList) > 0 {
			return fmt.Errorf("$clear: takes no arguments")
		}
		t.isClear = true
		return nil

	case "skill":
		if len(argList) != 1 {
			return fmt.Errorf("$skill: expected exactly 1 argument (skill name)")
		}
		sName := argList[0]
		if d == nil || d.Skills == "" {
			return fmt.Errorf("$skill: skills directory not configured")
		}

		// Wildcard handling
		if sName == "*" {
			if _, err := os.Stat(d.Skills); err != nil {
				return fmt.Errorf("$skill: skills directory not found: %s", d.Skills)
			}
			t.isSkill = true
			t.skillName = "*"
			return nil
		}

		// Single skill handling
		target := filepath.Join(d.Skills, sName+".md")
		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("$skill: skill not found: %s (checked %s)", sName, target)
			}
			return fmt.Errorf("$skill: error checking skill: %w", err)
		}
		t.isSkill = true
		t.skillName = sName
		return nil

	case "at":
		if len(argList) == 0 {
			return fmt.Errorf("$at: expected at least 1 path argument")
		}
		if d == nil || d.WorkingDir == "" {
			return fmt.Errorf("$at: missing working directory")
		}
		
		// Build path from parts
		relPath := filepath.Join(argList...)
		
		// Handle "." or "*" as "Current Directory"
		if relPath == "." || relPath == "*" {
			if err := ensureUnderWorkingDir(d.WorkingDir, d.WorkingDir); err != nil {
				return err
			}
			t.isAt = true
			t.atTarget = d.WorkingDir
			return nil
		}

		if filepath.IsAbs(relPath) {
			return fmt.Errorf("$at: absolute paths not allowed: %s", relPath)
		}

		abs := filepath.Join(d.WorkingDir, relPath)
		abs = filepath.Clean(abs)
		
		if _, err := os.Stat(abs); err != nil {
			return fmt.Errorf("$at: target not found: %s", abs)
		}
		if err := ensureUnderWorkingDir(abs, d.WorkingDir); err != nil {
			return err
		}
		t.isAt = true
		t.atTarget = abs
		return nil

	case "http":
		if len(argList) != 1 {
			return fmt.Errorf("$http: expected exactly 1 argument (url)")
		}
		rawUrl := argList[0]
		if rawUrl == "" {
			return fmt.Errorf("$http: empty url")
		}
		// Auto-prepend protocol if missing
		if !strings.HasPrefix(rawUrl, "http://") && !strings.HasPrefix(rawUrl, "https://") {
			rawUrl = "https://" + rawUrl
		}
		t.isHttp = true
		t.httpUrl = rawUrl
		return nil

	case "sh":
		if len(argList) != 1 {
			return fmt.Errorf("$sh: expected exactly 1 argument (command)")
		}
		cmd := argList[0]
		if strings.ContainsRune(cmd, '\x00') {
			return fmt.Errorf("$sh: command contains NUL")
		}
		if len(cmd) > 4096 {
			return fmt.Errorf("$sh: command too long")
		}
		t.isSh = true
		t.shCmd = cmd
		// Reconstruct display from arg (simplified)
		t.shCmdDisp = fmt.Sprintf("%q", cmd)
		return nil

	default:
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
	if t.isClear {
		if d == nil {
			return "", fmt.Errorf("$clear: missing ai dir")
		}
		path := d.PromptPath()
		if path == "" {
			return "", fmt.Errorf("$clear: missing prompt path")
		}
		var content string
		if b, err := os.ReadFile(path); err == nil {
			content = string(b)
			content = strings.ReplaceAll(content, "\r\n", "\n")
		}
		newContent := promptHeader
		const separator = "\n---\n"
		if idx := strings.Index(content, separator); idx >= 0 {
			cut := idx + len(separator)
			newContent = content[:cut]
		}

		// Fix: remove the clear command itself from the preserved content to prevent infinite recursion
		newContent = strings.ReplaceAll(newContent, t.literal, "")

		if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
			return "", fmt.Errorf("$clear: write prompt.md: %w", err)
		}
		return "", nil
	}

	if t.isSkill {
		// Wildcard Render
		if t.skillName == "*" {
			entries, err := os.ReadDir(d.Skills)
			if err != nil {
				return "", fmt.Errorf("$skill(*): read dir: %w", err)
			}
			var sb strings.Builder
			count := 0
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".md")
				path := filepath.Join(d.Skills, e.Name())
				content, ok, _, err := ReadTextFile(path)
				if err != nil {
					return "", err
				}
				if !ok {
					continue
				}

				sb.WriteString(fmt.Sprintf("=== SKILL: %s ===\n", name))
				sb.WriteString(content)
				if !strings.HasSuffix(content, "\n") {
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
				count++
			}
			return sb.String(), nil
		}

		// Specific Render
		path := filepath.Join(d.Skills, t.skillName+".md")
		content, ok, _, err := ReadTextFile(path)
		if err != nil {
			return "", fmt.Errorf("$skill: read failed: %w", err)
		}
		if !ok {
			return "", fmt.Errorf("$skill: file is binary or invalid utf8: %s", t.skillName)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("=== SKILL: %s ===\n", t.skillName))
		sb.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
		return sb.String(), nil
	}

	if t.isAt {
		files, err := CollectReadableFiles(t.atTarget, d)
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		stats := ReadStats{}
		for _, abs := range files {
			content, ok, rstats, rerr := ReadTextFile(abs)
			if rerr != nil {
				return "", rerr
			}
			if !ok {
				continue
			}
			sb.WriteString("FILE: ")
			sb.WriteString(abs)
			sb.WriteString("\n")
			sb.WriteString(content)
			if !strings.HasSuffix(content, "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
			stats.Files++
			stats.Lines += rstats.Lines
			stats.Chars += rstats.Chars
		}
		sb.WriteString(fmt.Sprintf("read [%d files] [%d lines] [%d characters]", stats.Files, stats.Lines, stats.Chars))
		sb.WriteString("\n")
		return sb.String(), nil
	}

	if t.isHttp {
		content, err := fetchUrlContent(t.httpUrl)
		if err != nil {
			return "", fmt.Errorf("fetch url %s: %w", t.httpUrl, err)
		}
		var sb strings.Builder
		sb.WriteString("URL: ")
		sb.WriteString(t.httpUrl)
		sb.WriteString("\n")
		sb.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		return sb.String(), nil
	}

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

	return t.literal, nil
}

func (t *DollarToken) String() string {
	return fmt.Sprintf("<%s: %q>", t.Type().String(), t.literal)
}