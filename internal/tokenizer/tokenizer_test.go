package tokenizer

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:     "Empty String",
			input:    "",
			expected: nil, // or empty slice
		},
		{
			name:  "Only Raw Text",
			input: "Hello world this is text",
			expected: []Token{
				textToken{content: "Hello world this is text"},
			},
		},
		{
			name:  "Single File Path",
			input: "@path/to/file.txt",
			expected: []Token{
				fileToken{raw: "@path/to/file.txt"},
			},
		},
		{
			name:  "Start with File Path then Text",
			input: "@config.json is the file",
			expected: []Token{
				fileToken{raw: "@config.json"},
				textToken{content: " is the file"},
			},
		},
		{
			name:  "Text then File Path",
			input: "Check @/var/log/syslog",
			expected: []Token{
				textToken{content: "Check "},
				fileToken{raw: "@/var/log/syslog"},
			},
		},
		{
			name:  "Text, File Path, Text",
			input: "Load @image.png into memory",
			expected: []Token{
				textToken{content: "Load "},
				fileToken{raw: "@image.png"},
				textToken{content: " into memory"},
			},
		},
		{
			name:  "Multiple File Paths",
			input: "@file1.txt @file2.txt",
			expected: []Token{
				fileToken{raw: "@file1.txt"},
				textToken{content: " "},
				fileToken{raw: "@file2.txt"},
			},
		},
		{
			name:  "Adjacent File Paths (No Space)",
			input: "@file1@file2",
			expected: []Token{
				// Depending on logic, @ stops at space.
				// Since there is no space, the logic grabs until space or end.
				// However, if '@' is encountered inside the inner loop?
				// The current implementation logic: "until next space"
				// So "@file1@file2" is technically one single token string until a space hits.
				fileToken{raw: "@file1@file2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.input)

			// Simple length check first
			if len(got) != len(tt.expected) {
				t.Errorf("Length mismatch. Got %d, expected %d", len(got), len(tt.expected))
				return
			}

			for i, token := range got {
				expectedToken := tt.expected[i]

				// 1. Check Type
				if token.Type() != expectedToken.Type() {
					t.Errorf("Token[%d] Type mismatch. Got %v, expected %v", i, token.Type(), expectedToken.Type())
				}

				// 2. Check Literal (Exact string match)
				if token.Literal() != expectedToken.Literal() {
					t.Errorf("Token[%d] Literal mismatch.\nGot:      %q\nExpected: %q", i, token.Literal(), expectedToken.Literal())
				}

				// 3. Check Value
				// Note: textToken value is same as literal.
				// fileToken value depends on OS (handled by filepath.FromSlash).
				// We must ensure expectedToken.Value() is calculated correctly for this test runner's OS.
				expectedValue := expectedToken.Value()

				// Manually construct what the value *should* be for file tokens
				// to verify the logic actually ran filepath.FromSlash
				if fToken, ok := expectedToken.(fileToken); ok {
					// We simulate the expected OS transformation here in the test
					rawNoAt := fToken.raw[1:]
					expectedValue = filepath.FromSlash(rawNoAt)
				}

				if token.Value() != expectedValue {
					t.Errorf("Token[%d] Value mismatch.\nGot:      %q\nExpected: %q", i, token.Value(), expectedValue)
				}
			}
		})
	}
}

// TestStringOutput verifies the fancy printing works as expected
func TestStringOutput(t *testing.T) {
	ft := fileToken{raw: "@my/path"}

	// We just want to ensure it doesn't panic and contains key info
	str := ft.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Check if it mentions the type
	if reflect.TypeOf(ft).Name() == "fileToken" && len(str) < 5 {
		t.Error("String() representation seems too short")
	}

	t.Logf("Debug Output Example: %s", str)
}
