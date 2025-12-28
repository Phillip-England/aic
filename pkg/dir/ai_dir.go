package dir

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const PromptHeader = `---
---
`

const MaxPromptHistory = 100

type AiDir struct {
	Root       string
	WorkingDir string
	Rules      string
	Prompts    string
	Ignore     *GitIgnore
}

func NewAiDir(force bool) (*AiDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	workingAbs := cleanPath(wd)
	rootAbs := filepath.Join(workingAbs, "ai")
	rulesAbs := filepath.Join(rootAbs, "rules")
	promptsAbs := filepath.Join(rootAbs, "prompts")
	promptFile := filepath.Join(rootAbs, "prompt.md")

	if info, err := os.Lstat(rootAbs); err == nil {
		if !info.IsDir() {
			return nil, fmt.Errorf("ai path exists but is not a directory: %s", rootAbs)
		}
		if !force {
			return nil, fmt.Errorf("ai dir already exists: %s", rootAbs)
		}
		os.RemoveAll(rootAbs)
	}

	dirs := []string{rootAbs, rulesAbs, promptsAbs}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", d, err)
		}
	}

	if err := os.WriteFile(promptFile, []byte(PromptHeader), 0o644); err != nil {
		return nil, fmt.Errorf("write prompt.md: %w", err)
	}

	ign, _ := LoadGitIgnore(workingAbs)

	return &AiDir{
		Root:       rootAbs,
		WorkingDir: workingAbs,
		Rules:      rulesAbs,
		Prompts:    promptsAbs,
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

	rootAbs := filepath.Join(workingAbs, "ai")
	ign, _ := LoadGitIgnore(workingAbs)

	return &AiDir{
		Root:       rootAbs,
		WorkingDir: workingAbs,
		Rules:      filepath.Join(rootAbs, "rules"),
		Prompts:    filepath.Join(rootAbs, "prompts"),
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
	newContent := fmt.Sprintf("---\n%s\n---\n\n\n\n\n\n\n\n\n", pre)
	return os.WriteFile(path, []byte(newContent), 0o644)
}

func (d *AiDir) StashPrompt(raw string) error {
	dir := d.Prompts
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ts := time.Now().Format("20060102_150405")
	path := filepath.Join(dir, ts+".md")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		return err
	}
	return d.pruneHistory()
}

func (d *AiDir) pruneHistory() error {
	ents, err := os.ReadDir(d.Prompts)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, filepath.Join(d.Prompts, e.Name()))
		}
	}
	if len(files) <= MaxPromptHistory {
		return nil
	}
	sort.Strings(files) 
	toDelete := len(files) - MaxPromptHistory
	for i := 0; i < toDelete; i++ {
		os.Remove(files[i])
	}
	return nil
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
		if info, err := os.Lstat(filepath.Join(dir, "ai")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("ai dir not found searching up from %s", start)
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