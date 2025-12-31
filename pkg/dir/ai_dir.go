package dir

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const PromptHeader = `---
---
`

const MaxPromptHistory = 500
const HistoryFileName = "history.gz"

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
	rootAbs := filepath.Join(workingAbs, "aic")
	rulesAbs := filepath.Join(rootAbs, "rules")
	promptFile := filepath.Join(rootAbs, "prompt.md")

	if info, err := os.Lstat(rootAbs); err == nil {
		if !info.IsDir() {
			return nil, fmt.Errorf("aic path exists but is not a directory: %s", rootAbs)
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

	rootAbs := filepath.Join(workingAbs, "aic")
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
	newContent := fmt.Sprintf("---\n%s\n---\n\n\n\n\n\n\n\n\n", pre)
	return os.WriteFile(path, []byte(newContent), 0o644)
}

func (d *AiDir) GetHistoryFilePath() string {
	return filepath.Join(d.Root, HistoryFileName)
}

func (d *AiDir) historyPath() string {
	return d.GetHistoryFilePath()
}

func (d *AiDir) ReadHistory() (map[string]string, error) {
	history := make(map[string]string)
	path := d.historyPath()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return history, nil // Return empty map if file doesn't exist
		}
		return nil, err
	}
	defer f.Close()

	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer zr.Close()

	jsonDecoder := json.NewDecoder(zr)
	if err := jsonDecoder.Decode(&history); err != nil {
		// If file is empty or corrupt, start fresh
		if err == io.EOF {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to decode history JSON: %w", err)
	}

	return history, nil
}

func (d *AiDir) WriteHistory(history map[string]string) error {
	path := d.historyPath()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	jsonEncoder := json.NewEncoder(zw)
	if err := jsonEncoder.Encode(history); err != nil {
		return fmt.Errorf("failed to encode history JSON: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func (d *AiDir) StashPrompt(raw string) error {
	history, err := d.ReadHistory()
	if err != nil {
		return fmt.Errorf("failed to read prompt history: %w", err)
	}

	// Remove empty lines from raw content
	var cleanedLines []string
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}
	cleanedRaw := strings.Join(cleanedLines, "\n")

	// Find the next key
	nextKey := 0
	for keyStr := range history {
		key, err := strconv.Atoi(keyStr)
		if err == nil && key >= nextKey {
			nextKey = key + 1
		}
	}

	// Add new prompt
	history[strconv.Itoa(nextKey)] = cleanedRaw

	// Prune history if it exceeds the max size
	if len(history) > MaxPromptHistory {
		var keys []int
		for keyStr := range history {
			key, err := strconv.Atoi(keyStr)
			if err == nil {
				keys = append(keys, key)
			}
		}
		sort.Ints(keys)
		toDeleteCount := len(keys) - MaxPromptHistory
		for i := 0; i < toDeleteCount; i++ {
			delete(history, strconv.Itoa(keys[i]))
		}
	}

	return d.WriteHistory(history)
}

func (d *AiDir) GetAllHistory() (map[string]string, error) {
	return d.ReadHistory()
}

func (d *AiDir) GetHistoryEntry(key int) (string, error) {
	history, err := d.ReadHistory()
	if err != nil {
		return "", err
	}
	prompt, ok := history[strconv.Itoa(key)]
	if !ok {
		return "", fmt.Errorf("no history entry found for key: %d", key)
	}
	return prompt, nil
}

func (d *AiDir) GetHistoryRange(start, end int) (map[string]string, error) {
	history, err := d.ReadHistory()
	if err != nil {
		return nil, err
	}
	results := make(map[string]string)
	for i := start; i <= end; i++ {
		key := strconv.Itoa(i)
		if prompt, ok := history[key]; ok {
			results[key] = prompt
		}
	}
	return results, nil
}

func (d *AiDir) DeleteHistoryEntry(key int) error {
	history, err := d.ReadHistory()
	if err != nil {
		return err
	}
	delete(history, strconv.Itoa(key))
	return d.WriteHistory(history)
}

func (d *AiDir) DeleteHistoryRange(start, end int) error {
	history, err := d.ReadHistory()
	if err != nil {
		return err
	}
	for i := start; i <= end; i++ {
		delete(history, strconv.Itoa(i))
	}
	return d.WriteHistory(history)
}

func (d *AiDir) LoadHistory(newHistoryData []byte) error {
	var newHistory map[string]string
	if err := json.Unmarshal(newHistoryData, &newHistory); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Validate that keys are sequential and start from 0
	var keys []int
	for k := range newHistory {
		i, err := strconv.Atoi(k)
		if err != nil {
			return fmt.Errorf("invalid key in new history (must be integer): %s", k)
		}
		keys = append(keys, i)
	}
	sort.Ints(keys)

	if len(keys) > 0 && keys[0] != 0 {
		return fmt.Errorf("new history must start with key 0")
	}

	for i := 0; i < len(keys); i++ {
		if keys[i] != i {
			return fmt.Errorf("new history keys are not sequential (missing key %d)", i)
		}
	}

	return d.WriteHistory(newHistory)
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
		if info, err := os.Lstat(filepath.Join(dir, "aic")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("aic dir not found searching up from %s", start)
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