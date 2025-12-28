package aic

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HttpHandler struct {
	noSpecial
}

func (HttpHandler) Name() string { return "http" }

func (HttpHandler) Validate(args []string, d *AiDir) error {
	_ = d
	if len(args) != 1 {
		return fmt.Errorf("$http takes exactly 1 arg")
	}
	if strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("$http url cannot be empty")
	}
	return nil
}

func (HttpHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	_ = r
	_ = index

	url := args[0]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return literal, nil
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	body := strings.ReplaceAll(string(b), "\r\n", "\n")

	var sb strings.Builder
	sb.WriteString("HTTP: ")
	sb.WriteString(url)
	sb.WriteString("\nSTATUS: ")
	sb.WriteString(resp.Status)
	sb.WriteString("\n")
	sb.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
