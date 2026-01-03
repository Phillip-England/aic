package dir

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const AiDirName = ".aic"
const PromptHeader = `---
---

`

type AiDir struct {
	Root       string
	WorkingDir string
	Rules      string
	Ignore     *GitIgnore
}

func NewAiDir(force bool) (*AiDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	workingAbs := cleanPath(wd)
	rootAbs := filepath.Join(workingAbs, AiDirName)
	rulesAbs := filepath.Join(rootAbs, "rules")
	promptFile := filepath.Join(rootAbs, "prompt.md")

	if info, err := os.Lstat(rootAbs); err == nil {
		if !info.IsDir() {
			return nil, fmt.Errorf("%s path exists but is not a directory: %s", AiDirName, rootAbs)
		}
		if !force {
			// If not forcing, we leave the existing dir, but if it's from an old version,
			// we might want to migrate it. For now, just leave it.
		} else {
			os.RemoveAll(rootAbs)
		}
	}

	dirs := []string{rootAbs, rulesAbs}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", d, err)
		}
	}

	if err := os.WriteFile(promptFile, []byte(PromptHeader), 0o644); err != nil {
		return nil, fmt.Errorf("write prompt.md: %w", err)
	}

	if err := ensureGitIgnoreHasEntry(workingAbs, AiDirName); err != nil {
		return nil, fmt.Errorf("update .gitignore: %w", err)
	}

	ign, _ := LoadGitIgnore(workingAbs)

	return &AiDir{
		Root:       rootAbs,
		WorkingDir: workingAbs,
		Rules:      rulesAbs,
		Ignore:     ign,
	}, nil
}

func OpenAiDir() (*AiDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	workingAbs, err := findAiWorkingDir(wd)
	if err != nil {
		return nil, err
	}

	rootAbs := filepath.Join(workingAbs, AiDirName)
	ign, _ := LoadGitIgnore(workingAbs)

	return &AiDir{
		Root:       rootAbs,
		WorkingDir: workingAbs,
		Rules:      filepath.Join(rootAbs, "rules"),
		Ignore:     ign,
	}, nil
}

func (d *AiDir) PromptPath() string {
	return filepath.Join(d.Root, "prompt.md")
}

func (d *AiDir) ReadPrompt() (string, error) {
	b, err := os.ReadFile(d.PromptPath())
	if err != nil {
		if os.IsNotExist(err) {
			return PromptHeader, nil
		}
		return "", err
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n"), nil
}

func (d *AiDir) ClearPrompt() error {
	path := d.PromptPath()
	content, err := d.ReadPrompt()
	if err != nil {
		return os.WriteFile(path, []byte(PromptHeader), 0o644)
	}
	parts := strings.Split(content, "---")
	if len(parts) < 3 {
		return os.WriteFile(path, []byte(PromptHeader), 0o644)
	}
	pre := strings.TrimSpace(parts[1])
	newContent := fmt.Sprintf("---\n%s\n---\n\n", pre)
	return os.WriteFile(path, []byte(newContent), 0o644)
}

func (d *AiDir) StashPrompt(raw string) error {
	return d.AppendToHistory(raw)
}

func cleanPath(p string) string {
	p = filepath.Clean(p)
	if es, err := filepath.EvalSymlinks(p); err == nil {
		p = es
	}
	return p
}


func findAiWorkingDir(start string) (string, error) {
	dir := cleanPath(start)
	for {
	if info, err := os.Lstat(filepath.Join(dir, AiDirName)); err == nil && info.IsDir() {
		return dir, nil
	}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("%s dir not found searching up from %s", AiDirName, start)
}

type GitIgnore struct {
	patterns []string
}

func LoadGitIgnore(wd string) (*GitIgnore, error) {
	f, err := os.Open(filepath.Join(wd, ".gitignore"))
	if err != nil {
		return &GitIgnore{}, nil
	}
	defer f.Close()

	var pats []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			pats = append(pats, line)
		}
	}
	return &GitIgnore{patterns: pats}, nil
}

func ensureGitIgnoreHasEntry(wd, entry string) error {
	path := filepath.Join(wd, ".gitignore")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte(entry+"\n"), 0o644)
		}
		return err
	}

	normalized := strings.ReplaceAll(string(b), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == entry || trimmed == entry+"/" {
			return nil
		}
	}

	if !strings.HasSuffix(normalized, "\n") {
		normalized += "\n"
	}
	normalized += entry + "\n"
	return os.WriteFile(path, []byte(normalized), 0o644)
}

func (g *GitIgnore) Match(relPath string) bool {
	for _, p := range g.patterns {
		if strings.Contains(relPath, p) {
			return true
		}
	}
	return false
}

func (d *AiDir) CollectRules() (string, error) {
	files, err := d.collectFiles(d.Rules)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("\n=== RULES ===\n")
	for _, f := range files {
		content, _ := os.ReadFile(f)
		sb.WriteString("FILE: " + f + "\n")
		sb.Write(content)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func (d *AiDir) collectFiles(root string) ([]string, error) {
	var results []string
	err := filepath.WalkDir(root, func(path string, de os.DirEntry, err error) error {
		if err != nil || de.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(d.WorkingDir, path)
		if d.Ignore.Match(rel) {
			return nil
		}
		results = append(results, path)
		return nil
	})
	return results, err
}
