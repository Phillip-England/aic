package llmactions

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HandleReplace overwrites lines in a file starting at startLine with the provided newLines.
// args example: " ./some_file.py 22;"
func HandleReplace(baseDir, args string, newLines []string) error {
	// 1. Parse arguments (Path and Line Number)
	cleanArgs := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(args), ";"))
	parts := strings.Fields(cleanArgs)
	if len(parts) < 2 {
		return fmt.Errorf("invalid args for REPLACE_START. Expected: <path> <line_num>")
	}

	relPath := parts[0]
	startLineStr := parts[1]

	startLine, err := strconv.Atoi(startLineStr)
	if err != nil {
		return fmt.Errorf("invalid line number '%s': %w", startLineStr, err)
	}

	// 2. Read the target file
	path := filepath.Join(baseDir, relPath)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}

	// Normalize line endings and split
	fileContent := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
	fileLines := strings.Split(fileContent, "\n")

	// 3. Validation
	if startLine < 1 || startLine > len(fileLines)+1 {
		return fmt.Errorf("start line %d out of bounds (file has %d lines)", startLine, len(fileLines))
	}

	// 4. Perform Replacement
	// We are replacing N lines, where N is len(newLines), starting at startLine.
	// Index is 0-based, so startLine-1.
	idx := startLine - 1

	var result []string

	// Append lines BEFORE the replacement zone
	result = append(result, fileLines[:idx]...)

	// Append the NEW lines
	result = append(result, newLines...)

	// Append lines AFTER the replacement zone
	// We skip the existing lines that were overwritten.
	linesToOverwrite := len(newLines)
	resumeIdx := idx + linesToOverwrite

	if resumeIdx < len(fileLines) {
		result = append(result, fileLines[resumeIdx:]...)
	}

	// 5. Write back to disk
	finalContent := strings.Join(result, "\n")
	if err := os.WriteFile(path, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}

	fmt.Printf("[ACTION] Replaced %d lines in %s starting at line %d\n", len(newLines), relPath, startLine)
	return nil
}