package aic

import (
	"fmt"
	"path/filepath"
	"strings"
)

type PathHandler struct {
	noSpecial
}

func (PathHandler) Name() string { return "path" }

func (PathHandler) Validate(args []string, d *AiDir) error {
	if len(args) == 0 {
		return fmt.Errorf("$path needs at least 1 arg")
	}
	if d == nil || d.WorkingDir == "" {
		return fmt.Errorf("$path requires a working directory")
	}
	return nil
}

func (PathHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = r
	_ = index

	targetAbs := filepath.Join(append([]string{d.WorkingDir}, args...)...)
	targetAbs = filepath.Clean(targetAbs)

	files, err := CollectReadableFiles(targetAbs, d)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return literal, nil
	}

	var sb strings.Builder
	for _, f := range files {
		content, ok, _, rerr := ReadTextFile(f)
		if rerr != nil {
			return "", rerr
		}
		if !ok {
			continue
		}
		sb.WriteString("FILE: ")
		sb.WriteString(f)
		sb.WriteString("\n")
		sb.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
	}

	out := sb.String()
	out = strings.TrimRight(out, "\n")
	out += "\n"
	return out, nil
}
