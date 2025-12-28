package aic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SkillHandler struct {
	noSpecial
}

func (SkillHandler) Name() string { return "skill" }

func (SkillHandler) Validate(args []string, d *AiDir) error {
	if len(args) != 1 {
		return fmt.Errorf("$skill takes exactly 1 arg (skill name)")
	}
	if d == nil || d.Skills == "" {
		return fmt.Errorf("$skill requires d.Skills to be set")
	}
	return nil
}

func (SkillHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = r
	_ = index

	name := args[0]
	path := filepath.Join(d.Skills, name+".md")
	b, err := os.ReadFile(path)
	if err != nil {
		// keep visible if missing
		return literal, nil
	}
	s := strings.ReplaceAll(string(b), "\r\n", "\n")

	var sb strings.Builder
	sb.WriteString("FILE: ")
	sb.WriteString(path)
	sb.WriteString("\n")
	sb.WriteString(s)
	if !strings.HasSuffix(s, "\n") {
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
