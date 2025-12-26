package aic

import "strings"

// PreProcess cleans the input text by removing comments and empty lines.
// This allows users to "comment out" tokens like // $at(...) or # $at(...) effectively.
func PreProcess(in string) string {
	// Normalize newlines
	s := strings.ReplaceAll(in, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	
	var out []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		
		// Skip Empty Lines
		if trim == "" {
			continue
		}
		
		// Skip Comments (// or #)
		if strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#") {
			continue
		}
		
		// Keep the line (preserving original indentation)
		out = append(out, line)
	}
	
	return strings.Join(out, "\n")
}