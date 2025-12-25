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

func (t *AtToken) Validate(d *AiDir) error {
	if d == nil || d.WorkingDir == "" {
		return fmt.Errorf("missing working directory")
	}

	val := strings.TrimSpace(t.Value())
	if val == "" {
		return fmt.Errorf("empty @ token")
	}

	// @. means project root (working dir)
	if val == "." {
		t.targetAbs = d.WorkingDir
		return nil
	}

	// Disallow absolute paths; resolve relative to working dir.
	if filepath.IsAbs(val) {
		return fmt.Errorf("absolute paths not allowed: %s", val)
	}

	// Clean and ensure it cannot escape working dir.
	cleanRel := filepath.Clean(val)
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path escapes working dir: %s", val)
	}

	abs := filepath.Join(d.WorkingDir, cleanRel)
	abs = filepath.Clean(abs)

	// Ensure still under working dir after clean.
	relToWd, err := filepath.Rel(d.WorkingDir, abs)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	if relToWd == ".." || strings.HasPrefix(relToWd, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path escapes working dir: %s", val)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("target not found: %s", abs)
	}
	_ = info // can be file or directory

	t.targetAbs = abs
	return nil
}

func (t *AtToken) AfterValidate(r *PromptReader, index int) error {
	t.bind(r, index)
	return nil
}

func (t *AtToken) Render(d *AiDir) (string, error) {
	// If Validate wasn't called, fallback safely.
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
			// hard fail; you can soften this later if you want partial output
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
