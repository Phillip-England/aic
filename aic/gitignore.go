package aic

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type GitIgnore struct {
	workingDir string
	patterns   []string
}

func LoadGitIgnore(workingDir string) (*GitIgnore, error) {
	path := filepath.Join(workingDir, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		// Missing .gitignore is fine
		if os.IsNotExist(err) {
			return &GitIgnore{workingDir: workingDir}, nil
		}
		return nil, err
	}
	defer f.Close()

	var pats []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// NOTE: We intentionally do not implement negation (!) yet.
		// You can add it later if you want.
		pats = append(pats, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	return &GitIgnore{
		workingDir: workingDir,
		patterns:   pats,
	}, nil
}

// Match checks whether a project-relative POSIX path (e.g. "src/main.go") is ignored.
func (g *GitIgnore) Match(relSlash string) bool {
	if g == nil {
		return false
	}
	p := strings.TrimPrefix(relSlash, "./")
	p = strings.TrimPrefix(p, "/")

	// Always ignore .git regardless of patterns
	if p == ".git" || strings.HasPrefix(p, ".git/") {
		return true
	}

	for _, pat := range g.patterns {
		if pat == "" {
			continue
		}

		// Normalize pattern to slash
		pp := strings.ReplaceAll(strings.TrimSpace(pat), "\\", "/")

		// Directory pattern "build/" matches anything under it
		if strings.HasSuffix(pp, "/") {
			base := strings.TrimSuffix(pp, "/")
			if p == base || strings.HasPrefix(p, base+"/") {
				return true
			}
			continue
		}

		// If pattern contains no slash, match against any segment basename
		if !strings.Contains(pp, "/") {
			// Exact basename match
			if filepath.Base(p) == pp {
				return true
			}
			// Glob basename match
			if ok, _ := filepath.Match(pp, filepath.Base(p)); ok {
				return true
			}
			continue
		}

		// Pattern has slashes: match against whole relative path (slash form)
		// Support simple globbing.
		if ok, _ := filepath.Match(pp, p); ok {
			return true
		}
		// Also allow patterns like "src/**" by treating trailing "/**" as prefix
		if strings.HasSuffix(pp, "/**") {
			base := strings.TrimSuffix(pp, "/**")
			if p == base || strings.HasPrefix(p, base+"/") {
				return true
			}
		}
	}

	return false
}
