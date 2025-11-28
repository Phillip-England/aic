package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanResult holds the path and the content (if loaded)
type ScanResult struct {
	AbsolutePath string
	RelativePath string
	Content      string
}

// Scanner handles the filesystem walking
type Scanner struct {
	Root string
}

func New(root string) *Scanner {
	return &Scanner{Root: root}
}

// CollectPaths returns absolute paths of all valid files
func (s *Scanner) CollectPaths() ([]string, error) {
	var results []string
	ignorePatterns := s.readGitIgnore()
	ignorePatterns = append(ignorePatterns, ".git")

	err := filepath.WalkDir(s.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()
		if s.isIgnored(name, ignorePatterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			// Ensure absolute path
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			results = append(results, abs)
		}

		return nil
	})

	return results, err
}

// CollectContent reads all valid files and returns a formatted string builder
// ready for the clipboard.
func (s *Scanner) CollectContent(preamble string, contextContent string) (string, int, error) {
	paths, err := s.CollectPaths()
	if err != nil {
		return "", 0, err
	}

	var sb strings.Builder

	// 1. Add Preamble (Prompt)
	if preamble != "" {
		sb.WriteString(preamble)
		sb.WriteString("\n\n")
	}

	// 2. Add Current Working Directory
	// This helps the LLM understand where these files are located in the project structure
	fmt.Fprintf(&sb, "Directory: %s\n\n", s.Root)

	sb.WriteString("-----------------------------------\n\n")

	// 3. Add Context
	if contextContent != "" {
		sb.WriteString("!!! CONTEXT FILE LOADED (.ai) !!!\n")
		sb.WriteString(contextContent)
		sb.WriteString("\n-----------------------------------\n\n")
	}

	fileCount := 0
	for _, path := range paths {
		// Skip the context file itself if it appears in the list
		if strings.HasSuffix(path, ".ai") {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", path, err)
			continue
		}

		// Use Absolute Path for output as requested
		fmt.Fprintf(&sb, "PATH: %s\n\n", path)
		sb.Write(content)
		sb.WriteString("\n\n---\n\n")

		fileCount++

		// Log relative path for cleaner console output
		rel, _ := filepath.Rel(s.Root, path)
		fmt.Printf("Collected: %s\n", rel)
	}

	return sb.String(), fileCount, nil
}

func (s *Scanner) readGitIgnore() []string {
	var patterns []string
	ignorePath := filepath.Join(s.Root, ".gitignore")

	file, err := os.Open(ignorePath)
	if err != nil {
		return patterns
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cleanLine := strings.TrimSuffix(line, "/")
		patterns = append(patterns, cleanLine)
	}
	return patterns
}

func (s *Scanner) isIgnored(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if name == pattern {
			return true
		}
		matched, _ := filepath.Match(pattern, name)
		if matched {
			return true
		}
	}
	return false
}