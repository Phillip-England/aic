package context

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FileName = ".ai"

// Section Headers
const ContextHeader = "# Application Context"
const IgnoreHeader = "## Ignore Patterns"

// ContextData holds the parsed content of the .ai file, separated by function
type ContextData struct {
	Entries []string
	Ignores []string
	RawContent string
}

// Manager handles the interaction with the context file
type Manager struct {
	Root string
}

func New(root string) *Manager {
	return &Manager{Root: root}
}

func (m *Manager) GetPath() string {
	return filepath.Join(m.Root, FileName)
}

func (m *Manager) Exists() bool {
	_, err := os.Stat(m.GetPath())
	return err == nil
}

// Init creates the file with both required headers
func (m *Manager) Init() error {
	if m.Exists() {
		return fmt.Errorf("%s already exists in %s", FileName, m.Root)
	}
	f, err := os.Create(m.GetPath())
	if err != nil {
		return err
	}
	defer f.Close()

	// Initialize with Ignore Patterns section first
	if _, err := f.WriteString(fmt.Sprintf("%s\n\n", IgnoreHeader)); err != nil {
		return err
	}
	// Then the Context section
	if _, err := f.WriteString(fmt.Sprintf("%s\n\n", ContextHeader)); err != nil {
		return err
	}
	return nil
}

// ReadContext reads and parses the entire file, splitting patterns and entries
func (m *Manager) ReadContext() (*ContextData, error) {
	if !m.Exists() {
		return nil, fmt.Errorf("%s not found", FileName)
	}

	file, err := os.Open(m.GetPath())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data ContextData
	var rawLines []string
	var entries []string
	var ignores []string

	// State machine to track which section we are in
	section := "" 
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		rawLines = append(rawLines, line)
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == ContextHeader {
			section = "context"
			continue
		}
		if trimmedLine == IgnoreHeader {
			section = "ignore"
			continue
		}
		
		if strings.HasPrefix(trimmedLine, "#") || trimmedLine == "" {
			continue // Skip comments and empty lines
		}

		if section == "ignore" {
			// Lines in the ignore section are patterns
			ignores = append(ignores, trimmedLine)
		} else if section == "context" {
			// Lines in the context section are entries (must parse ID)
			if strings.HasPrefix(trimmedLine, "[") {
				closeBracket := strings.Index(trimmedLine, "]")
				if closeBracket != -1 && len(trimmedLine) > closeBracket+1 {
					entries = append(entries, strings.TrimSpace(trimmedLine[closeBracket+1:]))
				}
			} else if strings.HasPrefix(trimmedLine, "- ") {
				entries = append(entries, strings.TrimPrefix(trimmedLine, "- "))
			} else {
				// Fallback: If user manually added a line without ID
				entries = append(entries, trimmedLine)
			}
		}
	}

	data.Ignores = ignores
	data.Entries = entries
	data.RawContent = strings.Join(rawLines, "\n")
	
	return &data, scanner.Err()
}

// Read is the old context reader, now relying on ReadContext for entries
// This is used by Add/Delete to maintain compatibility.
func (m *Manager) Read() ([]string, error) {
	data, err := m.ReadContext()
	if err != nil {
		return nil, err
	}
	return data.Entries, nil
}

// List fetches the raw file content to display to the user
func (m *Manager) List() (string, error) {
	if !m.Exists() {
		return "", fmt.Errorf("%s not found", FileName)
	}
	content, err := os.ReadFile(m.GetPath())
	if err != nil {
		return "", err
	}
	if len(content) == 0 {
		return "No context entries found.", nil
	}
	return string(content), nil
}

// Add and Delete logic remains the same, as they rely on Write() which maintains integrity.

func (m *Manager) Add(args []string) (int, string, error) {
	if !m.Exists() {
		return 0, "", fmt.Errorf("%s not found. Run 'ctx init' first", FileName)
	}

	entries, err := m.Read()
	if err != nil {
		return 0, "", err
	}

	newEntry := strings.Join(args, " ")
	entries = append(entries, newEntry)

	if err := m.Write(entries); err != nil {
		return 0, "", err
	}
	return len(entries), newEntry, nil
}

func (m *Manager) Delete(id int) (string, error) {
	if !m.Exists() {
		return "", fmt.Errorf("%s not found", FileName)
	}

	entries, err := m.Read()
	if err != nil {
		return "", err
	}

	if id < 1 || id > len(entries) {
		return "", fmt.Errorf("ID %d is out of range", id)
	}

	removed := entries[id-1]
	entries = append(entries[:id-1], entries[id:]...)

	if err := m.Write(entries); err != nil {
		return "", err
	}
	return removed, nil
}

// Write now preserves the Ignored Patterns section when context entries change.
func (m *Manager) Write(newEntries []string) error {
	// 1. Read existing ignore patterns
	data, err := m.ReadContext()
	if err != nil && !os.IsNotExist(err) {
		// Log but don't fail hard if we can't read the context file structure, 
		// but since we checked Exists in callers, this is usually an IO error.
		return err 
	}
	
	f, err := os.Create(m.GetPath())
	if err != nil {
		return err
	}
	defer f.Close()

	// 2. Write Ignore Header and Patterns
	if _, err := f.WriteString(fmt.Sprintf("%s\n", IgnoreHeader)); err != nil {
		return err
	}
	for _, pattern := range data.Ignores {
		if _, err := f.WriteString(fmt.Sprintf("%s\n", pattern)); err != nil {
			return err
		}
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}

	// 3. Write Context Header and Entries
	if _, err := f.WriteString(fmt.Sprintf("%s\n\n", ContextHeader)); err != nil {
		return err
	}
	for i, entry := range newEntries {
		if _, err := f.WriteString(fmt.Sprintf("[%d] %s\n", i+1, entry)); err != nil {
			return err
		}
	}
	return nil
}