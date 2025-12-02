package tokenizer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillip-england/aic/internal/scanner"
	// !!! UPDATE THIS IMPORT PATH TO MATCH YOUR PROJECT STRUCTURE !!!
)

// TokenType represents the category of the token
type TokenType int

const (
	RawText TokenType = iota
	FilePath
)

// String returns the readable name of the token type.
func (t TokenType) String() string {
	switch t {
	case RawText:
		return "RawText"
	case FilePath:
		return "FilePath"
	default:
		return "Unknown"
	}
}

// Token defines the behavior required for all token types.
type Token interface {
	Literal() string
	Value() string
	Type() TokenType
	String() string
}

// textToken represents standard string content.
type textToken struct {
	content string
}

func (t textToken) Literal() string { return t.content }
func (t textToken) Value() string   { return t.content }
func (t textToken) Type() TokenType { return RawText }

func (t textToken) String() string {
	// Truncate for display if too long
	display := t.content
	if len(display) > 20 {
		display = display[:20] + "..."
	}
	return fmt.Sprintf("<Text: %q>", display)
}

// fileToken represents a file path prefixed by '@'.
type fileToken struct {
	raw string
}

func (f fileToken) Literal() string { return f.raw }

func (f fileToken) Value() string {
	if len(f.raw) == 0 {
		return ""
	}
	cleanPath := f.raw[1:]
	return filepath.FromSlash(cleanPath)
}

func (f fileToken) Type() TokenType { return FilePath }

func (f fileToken) String() string {
	return fmt.Sprintf("<File: %q>", f.raw)
}

// Tokenize parses the input string.
// If an '@' token points to a directory, it uses the scanner to collect all text
// and returns it as a RawText token (expanding the directory).
// If it points to a file, it returns a FilePath token.
func Tokenize(input string) []Token {
	var tokens []Token
	var buffer strings.Builder

	for i := 0; i < len(input); i++ {
		char := input[i]

		if char == '@' {
			// Flush existing text buffer
			if buffer.Len() > 0 {
				tokens = append(tokens, textToken{content: buffer.String()})
				buffer.Reset()
			}

			start := i
			// Read until space or end of string
			for i < len(input) && input[i] != ' ' {
				i++
			}

			fileStr := input[start:i]
			cleanPath := filepath.FromSlash(fileStr[1:]) // Remove '@'

			// Check if this path is a directory
			info, err := os.Stat(cleanPath)
			if err == nil && info.IsDir() {
				// It is a directory: Use scanner to collect all content
				sc := scanner.New(cleanPath)
				// Passing empty strings for preamble/context as we just want the raw file dump
				content, _, scanErr := sc.CollectContent("", "")

				if scanErr != nil {
					// On error, fallback to treating it as a standard file token
					// so the error can be debugged later by the consumer
					tokens = append(tokens, fileToken{raw: fileStr})
				} else {
					// Success: The directory reference becomes the collected text content
					tokens = append(tokens, textToken{content: content})
				}
			} else {
				// It is a regular file (or doesn't exist), keep as FilePath token
				tokens = append(tokens, fileToken{raw: fileStr})
			}

			i-- // Step back because the loop increments i
		} else {
			buffer.WriteByte(char)
		}
	}

	if buffer.Len() > 0 {
		tokens = append(tokens, textToken{content: buffer.String()})
	}

	return tokens
}
