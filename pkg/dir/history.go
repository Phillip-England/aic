package dir

import (
	"bufio"
	"os"
	"path/filepath"
)

const (
	HistoryFileName = "history"
	MaxHistory      = 10
)

// AppendToHistory appends a new prompt to the history file, ensuring it only contains the last MaxHistory prompts.
func (d *AiDir) AppendToHistory(prompt string) error {
	historyPath := filepath.Join(d.Root, HistoryFileName)

	// Read existing history
	lines, err := readLines(historyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// If the file doesn't exist, lines will be an empty slice
	}

	// Append new prompt
	lines = append(lines, prompt)

	// Trim history if it exceeds the max size
	if len(lines) > MaxHistory {
		lines = lines[len(lines)-MaxHistory:]
	}

	// Write the updated history back to the file
	return writeLines(historyPath, lines)
}

// readLines reads a file and returns its lines as a slice of strings.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}
