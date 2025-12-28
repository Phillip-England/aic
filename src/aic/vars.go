package aic

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func LoadVars(d *AiDir) (map[string]string, error) {
	if d == nil || d.Vars == "" {
		return map[string]string{}, nil
	}
	return LoadVarsFromDir(d.Vars)
}

// Loads vars from ai/vars/* files.
// File format (per line): KEY=VALUE
// - ignores blank lines and lines starting with # or //
// - supports: export KEY=VALUE
// - if VALUE is quoted ("...") it will be unquoted via strconv.Unquote
func LoadVarsFromDir(dir string) (map[string]string, error) {
	out := make(map[string]string)

	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("read vars dir %s: %w", dir, err)
	}

	// Deterministic order.
	sort.Slice(ents, func(i, j int) bool { return ents[i].Name() < ents[j].Name() })

	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			continue
		}
		s := strings.ReplaceAll(string(b), "\r\n", "\n")
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
				continue
			}
			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}
			eq := strings.IndexByte(line, '=')
			if eq <= 0 {
				continue
			}
			key := strings.TrimSpace(line[:eq])
			val := strings.TrimSpace(line[eq+1:])
			if key == "" {
				continue
			}
			// Optional unquote for "..." or '...' (best-effort).
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
					if uq, uerr := strconv.Unquote(val); uerr == nil {
						val = uq
					} else if val[0] == '\'' && val[len(val)-1] == '\'' {
						// strconv.Unquote doesn't accept single quotes for multi-char strings; do simple strip.
						val = val[1 : len(val)-1]
					}
				}
			}
			out[key] = val
		}
	}

	return out, nil
}
