package context

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FileName = ".ai"

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

func (m *Manager) Init() error {
	if m.Exists() {
		return fmt.Errorf("%s already exists in %s", FileName, m.Root)
	}
	return m.Write([]string{})
}

func (m *Manager) Add(args []string) (int, string, error) {
	// path := m.GetPath()
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
	// Remove and preserve order
	entries = append(entries[:id-1], entries[id:]...)

	if err := m.Write(entries); err != nil {
		return "", err
	}
	return removed, nil
}

// Read parses lines starting with "[N]" or "- "
func (m *Manager) Read() ([]string, error) {
	file, err := os.Open(m.GetPath())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle [1] format
		if strings.HasPrefix(line, "[") {
			closeBracket := strings.Index(line, "]")
			if closeBracket != -1 && len(line) > closeBracket+1 {
				entries = append(entries, strings.TrimSpace(line[closeBracket+1:]))
				continue
			}
		}
		// Handle legacy - format
		if strings.HasPrefix(line, "- ") {
			entries = append(entries, strings.TrimPrefix(line, "- "))
			continue
		}
		// Fallback
		entries = append(entries, line)
	}
	return entries, scanner.Err()
}

func (m *Manager) Write(entries []string) error {
	f, err := os.Create(m.GetPath())
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("# Application Context\n\n"); err != nil {
		return err
	}

	for i, entry := range entries {
		if _, err := f.WriteString(fmt.Sprintf("[%d] %s\n", i+1, entry)); err != nil {
			return err
		}
	}
	return nil
}