package llmactions

import (
	"fmt"
	"strings"
)

const (
	ModeNone = iota
	ModeReplace
	ModeInsert
)

func ProcessClipboard(content string, baseDir string) {
	cleanContent := strings.TrimSpace(content)
	lines := strings.Split(cleanContent, "\n")
	if len(lines) < 2 {
		return
	}

	firstLine := strings.TrimSpace(lines[0])
	lastLine := strings.TrimSpace(lines[len(lines)-1])

	if !IsStart(firstLine) || !IsEnd(lastLine) {
		return
	}

	fmt.Println("[AIC] Valid Command Block Detected. Processing...")

	commandLines := lines[1 : len(lines)-1]

	var currentMode int = ModeNone
	var actionArgs string
	var actionBuffer []string

	for _, line := range commandLines {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)

		// --- DETECT BLOCK START ---
		if strings.HasPrefix(upper, "AIC: REPLACE_START") || strings.HasPrefix(upper, "AIC:REPLACE_START") {
			if currentMode != ModeNone {
				fmt.Println("[ERROR] Nested blocks not supported.")
				return
			}
			currentMode = ModeReplace
			_, args, _ := strings.Cut(trimmed, "START")
			actionArgs = args
			actionBuffer = []string{}
			continue
		}

		if strings.HasPrefix(upper, "AIC: INSERT_START") || strings.HasPrefix(upper, "AIC:INSERT_START") {
			if currentMode != ModeNone {
				fmt.Println("[ERROR] Nested blocks not supported.")
				return
			}
			currentMode = ModeInsert
			_, args, _ := strings.Cut(trimmed, "START")
			actionArgs = args
			actionBuffer = []string{}
			continue
		}

		// --- DETECT BLOCK END ---
		if strings.HasPrefix(upper, "AIC: REPLACE_END;") || strings.HasPrefix(upper, "AIC:REPLACE_END;") {
			if currentMode != ModeReplace {
				fmt.Println("[ERROR] Orphaned REPLACE_END.")
				return
			}
			if err := HandleReplace(baseDir, actionArgs, actionBuffer); err != nil {
				fmt.Printf("[ERROR] Replace: %v\n", err)
			}
			currentMode = ModeNone
			continue
		}

		if strings.HasPrefix(upper, "AIC: INSERT_END;") || strings.HasPrefix(upper, "AIC:INSERT_END;") {
			if currentMode != ModeInsert {
				fmt.Println("[ERROR] Orphaned INSERT_END.")
				return
			}
			if err := HandleInsert(baseDir, actionArgs, actionBuffer); err != nil {
				fmt.Printf("[ERROR] Insert: %v\n", err)
			}
			currentMode = ModeNone
			continue
		}

		// --- ACCUMULATE ---
		if currentMode != ModeNone {
			actionBuffer = append(actionBuffer, line)
			continue
		}

		// --- SINGLE COMMANDS ---
		processLine(baseDir, trimmed)
	}
}

func processLine(baseDir, line string) {
	if line == "" {
		return
	}
	upperLine := strings.ToUpper(line)
	switch {
	case strings.HasPrefix(upperLine, "AIC: SHELL") || strings.HasPrefix(upperLine, "AIC:SHELL"):
		_, payload, found := strings.Cut(line, "SHELL")
		if found {
			if err := HandleShell(payload); err != nil {
				fmt.Printf("[ERROR] Shell Action: %v\n", err)
			}
		}
	
	case strings.HasPrefix(upperLine, "AIC: DELETE_LINE") || strings.HasPrefix(upperLine, "AIC:DELETE_LINE"):
		// Extract args after DELETE_LINE
		// This handles both "AIC: DELETE_LINE" and "AIC:DELETE_LINE" due to strings.Cut behavior on prefix check
		var payload string
		if strings.HasPrefix(upperLine, "AIC: DELETE_LINE") {
			_, payload, _ = strings.Cut(line, "DELETE_LINE")
		} else {
			_, payload, _ = strings.Cut(line, "AIC:DELETE_LINE")
		}
		
		if err := HandleDelete(baseDir, payload); err != nil {
			fmt.Printf("[ERROR] Delete Action: %v\n", err)
		}

	default:
	}
}