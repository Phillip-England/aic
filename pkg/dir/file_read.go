package dir

import (
	"os"
	"path/filepath"
	"strings"
)

func (d *AiDir) ReadAnyFile(relPath string) (string, error) {
	// Try relative to .aic dir first, then relative to working dir
	path := filepath.Join(d.WorkingDir, relPath)
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n"), nil
}
