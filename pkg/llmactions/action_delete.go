package llmactions

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HandleDelete removes lines from startLine to endLine (inclusive).
// args example: " ./test.txt 2 4;"
func HandleDelete(baseDir, args string) error {
	// 1. Parse arguments
	cleanArgs := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(args), ";"))
	parts := strings.Fields(cleanArgs)
	if len(parts) < 3 {
		return fmt.Errorf("invalid args for DELETE_LINE. Expected: <path> <start_line> <end_line>")
	}

	relPath := parts[0]
	startLineStr := parts[1]
	endLineStr := parts[2]

	startLine, err := strconv.Atoi(startLineStr)
	if err != nil {
		return fmt.Errorf("invalid start line '%s': %w", startLineStr, err)
	}
	endLine, err := strconv.Atoi(endLineStr)
	if err != nil {
		return fmt.Errorf("invalid end line '%s': %w", endLineStr, err)
	}

	if startLine > endLine {
		return fmt.Errorf("start line %d cannot be greater than end line %d", startLine, endLine)
	}

	// 2. Read File
	path := filepath.Join(baseDir, relPath)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}

	fileContent := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
	fileLines := strings.Split(fileContent, "\n")

	// 3. Validation
	if startLine < 1 || endLine > len(fileLines) {
		return fmt.Errorf("range %d-%d out of bounds (file has %d lines)", startLine, endLine, len(fileLines))
	}

	// 4. Perform Deletion
	// startLine is 1-based. To delete line 2, we remove index 1.
	// We want to keep lines up to startLine-1 (index startLine-2).
	// We want to resume at endLine (index endLine).
	
	startIdx := startLine - 1 
	endIdx := endLine         

	var result []string

	// Keep lines before the deletion range
	if startIdx > 0 {
		result = append(result, fileLines[:startIdx]...)
	}

	// Keep lines after the deletion range
	if endIdx < len(fileLines) {
		result = append(result, fileLines[endIdx:]...)
	}

	// 5. Write back
	finalContent := strings.Join(result, "\n")
	if err := os.WriteFile(path, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}

	fmt.Printf("[ACTION] Deleted lines %d-%d in %s\n", startLine, endLine, relPath)
	return nil
}