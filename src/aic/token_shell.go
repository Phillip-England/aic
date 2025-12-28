package aic

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type ShellHandler struct {
	noSpecial
}

func (ShellHandler) Name() string { return "shell" }

func (ShellHandler) Validate(args []string, d *AiDir) error {
	_ = d
	if len(args) != 1 {
		return fmt.Errorf("$sh/$shell takes exactly 1 string arg")
	}
	if strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("$sh/$shell command cannot be empty")
	}
	return nil
}

func (ShellHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	_ = r
	_ = index

	cmdStr := args[0]
	cmd := exec.Command("sh", "-lc", cmdStr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // we include stderr even on failure

	var sb strings.Builder
	sb.WriteString("SHELL: ")
	sb.WriteString(cmdStr)
	sb.WriteString("\n")

	out := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
	errOut := strings.ReplaceAll(stderr.String(), "\r\n", "\n")

	if strings.TrimSpace(out) != "" {
		sb.WriteString(out)
		if !strings.HasSuffix(out, "\n") {
			sb.WriteString("\n")
		}
	}
	if strings.TrimSpace(errOut) != "" {
		sb.WriteString("STDERR:\n")
		sb.WriteString(errOut)
		if !strings.HasSuffix(errOut, "\n") {
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
