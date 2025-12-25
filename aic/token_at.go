package aic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AtToken struct {
	TokenCtx
	literal string // includes leading "@"

	// resolved absolute target after Validate()
	targetAbs string
}

func NewAtToken(lit string) PromptToken {
	return &AtToken{literal: lit}
}

func (t *AtToken) Type() PromptTokenType { return PromptTokenAt }
func (t *AtToken) Literal() string       { return t.literal }

func (t *AtToken) Value() string {
	return strings.TrimPrefix(t.literal, "@")
}

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

func (t *AtToken) Validate(d *AiDir) error {
	if d == nil || d.WorkingDir == "" {
		return fmt.Errorf("missing working directory")
	}

	val := strings.TrimSpace(t.Value())
	if val == "" {
		return fmt.Errorf("empty @ token")
	}

	// @. means project root (WorkingDir)
	if val == "." {
		if err := ensureUnderWorkingDir(d.WorkingDir, d.WorkingDir); err != nil {
			return err
		}
		t.targetAbs = d.WorkingDir
		return nil
	}

	// Disallow absolute paths; resolve relative to WorkingDir.
	if filepath.IsAbs(val) {
		return fmt.Errorf("absolute paths not allowed: %s", val)
	}

	// Clean and ensure it cannot escape working dir.
	cleanRel := filepath.Clean(val)
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path escapes project root: %s", val)
	}

	abs := filepath.Join(d.WorkingDir, cleanRel)
	abs = filepath.Clean(abs)

	// Must exist
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("target not found: %s", abs)
	}

	// Critical rule: resolved target must be under project root (dir containing ai/)
	if err := ensureUnderWorkingDir(abs, d.WorkingDir); err != nil {
		return err
	}

	t.targetAbs = abs
	return nil
}

func (t *AtToken) AfterValidate(r *PromptReader, index int) error {
	t.bind(r, index)
	return nil
}

func (t *AtToken) Render(d *AiDir) (string, error) {
	if t.targetAbs == "" {
		return t.literal, nil
	}

	files, err := CollectReadableFiles(t.targetAbs, d)
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

func (t *AtToken) String() string {
	return fmt.Sprintf("<%s: %q>", t.Type().String(), t.literal)
}
