package llmactions

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HandleInsert injects newLines into the file at startLine, shifting existing content down.
// args example: " ./some_file.py 22;"
func HandleInsert(baseDir, args string, newLines []string) error {
	// 1. Parse arguments (Path and Line Number)
	cleanArgs := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(args), ";"))
	parts := strings.Fields(cleanArgs)
	if len(parts) < 2 {
		return fmt.Errorf("invalid args for INSERT_START. Expected: <path> <line_num>")
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
	// We allow inserting at len(fileLines) + 1 (appending to the end)
	if startLine < 1 || startLine > len(fileLines)+1 {
		return fmt.Errorf("insert line %d out of bounds (file has %d lines)", startLine, len(fileLines))
	}

	// 4. Perform Insertion
	// Index is 0-based, so startLine-1.
	idx := startLine - 1

	var result []string

	// Append lines BEFORE the insertion point
	if idx < len(fileLines) {
		result = append(result, fileLines[:idx]...)
	} else {
		// If appending to the very end
		result = append(result, fileLines...)
	}

	// Append the NEW lines
	result = append(result, newLines...)

	// Append the REST of the original file (shifted down)
	if idx < len(fileLines) {
		result = append(result, fileLines[idx:]...)
	}

	// 5. Write back to disk
	finalContent := strings.Join(result, "\n")
	if err := os.WriteFile(path, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}

	fmt.Printf("[ACTION] Inserted %d lines in %s at line %d\n", len(newLines), relPath, startLine)
	return nil
}