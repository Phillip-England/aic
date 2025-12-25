package aic

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func CollectReadableFiles(targetAbs string, d *AiDir) ([]string, error) {
	info, err := os.Stat(targetAbs)
	if err != nil {
		return nil, fmt.Errorf("stat target %s: %w", targetAbs, err)
	}

	// If a file, just return it (unless ignored)
	if !info.IsDir() {
		if shouldIgnoreAbs(targetAbs, d) {
			return []string{}, nil
		}
		return []string{filepath.Clean(targetAbs)}, nil
	}

	// Directory walk
	var files []string
	err = filepath.WalkDir(targetAbs, func(path string, de os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		name := de.Name()

		// Always skip .git
		if de.IsDir() && name == ".git" {
			return filepath.SkipDir
		}

		// Skip ignored directories early
		if de.IsDir() {
			if shouldIgnoreAbs(path, d) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip ignored files
		if shouldIgnoreAbs(path, d) {
			return nil
		}

		files = append(files, filepath.Clean(path))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", targetAbs, err)
	}

	sort.Strings(files)
	return files, nil
}

func shouldIgnoreAbs(abs string, d *AiDir) bool {
	if d == nil || d.WorkingDir == "" {
		return false
	}

	rel, err := filepath.Rel(d.WorkingDir, abs)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)

	// If it escapes working dir, ignore (safety)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return true
	}

	// Never include ./ai itself when expanding @. unless user explicitly points there
	// (still allow reading it if targetAbs is inside ai, but default ignore patterns may cover)
	// We'll not hard-block it here.

	if d.Ignore == nil {
		return false
	}
	return d.Ignore.Match(rel)
}
