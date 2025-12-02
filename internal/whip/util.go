package whip

import (
	"os"
	"path/filepath"
	"strings"
)

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func IsFile(path string) bool {
	p := filepath.Clean(path)
	if strings.HasSuffix(p, string(filepath.Separator)) {
		return false
	}
	ext := filepath.Ext(p)
	return ext != ""
}

func IsDir(path string) bool {
	p := filepath.Clean(path)
	if strings.HasSuffix(path, string(filepath.Separator)) {
		return true
	}
	ext := filepath.Ext(p)
	return ext == ""
}
