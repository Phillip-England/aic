package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/phillip-england/aic/internal/tokenizer"
	"github.com/phillip-england/aic/internal/whip"
)

func main() {
	// 1. Define Command Factories
	promptFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &PromptCmd{}, nil
	}

	helpFactory := func(cli *whip.Cli) (whip.Cmd, error) {
		return &HelpCmd{}, nil
	}

	// 2. Initialize Whip CLI
	// Make "prompt" the default command.
	cli, err := whip.New(promptFactory)
	if err != nil {
		log.Fatalf("Failed to initialize CLI: %v", err)
	}

	// 3. Register Commands
	cli.At("prompt", promptFactory)
	cli.At("help", helpFactory)

	// 4. Run
	if err := cli.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// --- COMMAND IMPLEMENTATIONS ---

type PromptCmd struct{}

func (c *PromptCmd) Execute(cli *whip.Cli) error {
	// Support BOTH:
	//   aic "your prompt"
	//   aic prompt "your prompt"
	startPos := 1
	if first, ok := cli.ArgGetByPosition(1); ok && first == "prompt" {
		startPos = 2
	}

	// Collect everything from startPos onward (supports prompts without quotes too)
	var parts []string
	for pos := startPos; ; pos++ {
		val, ok := cli.ArgGetByPosition(pos)
		if !ok {
			break
		}
		parts = append(parts, val)
	}

	if len(parts) == 0 {
		return fmt.Errorf("Usage: aic \"your prompt with @file/paths\"  OR  aic prompt \"your prompt with @file/paths\"")
	}

	runPrompt(strings.Join(parts, " "))
	return nil
}

type HelpCmd struct{}

func (c *HelpCmd) Execute(cli *whip.Cli) error {
	printUsage()
	return nil
}

// --- HELPER LOGIC ---

type inlinedFileStat struct {
	Path  string
	Chars int
	Kind  string // "file" | "scan"
}

func stripEmptyAndCommentLines(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))

	inBlock := false

	for _, ln := range lines {
		t := strings.TrimSpace(ln)

		// If we are inside a block comment, keep discarding until we see */
		if inBlock {
			if strings.Contains(t, "*/") {
				// end block comment (can be on same line it started, or later)
				inBlock = false
			}
			continue
		}

		// Remove empty lines
		if t == "" {
			continue
		}

		// Remove single-line comments that start with //
		if strings.HasPrefix(t, "//") {
			continue
		}

		// Remove block comments that start with /*
		if strings.HasPrefix(t, "/*") {
			// If it ends on the same line, just drop this line and continue
			if !strings.Contains(t, "*/") {
				inBlock = true
			}
			continue
		}

		// Keep the original (untrimmed) line so code formatting is preserved
		out = append(out, ln)
	}

	return strings.Join(out, "\n")
}

// parseCapturedFilesFromOutput finds BOTH:
//  1. direct file inlines:  --- START FILE: X --- ... --- END FILE: X ---
//  2. directory scan blocks: PATH: /abs/path ... \n---\n  (scanner output)
func parseCapturedFilesFromOutput(raw string) []inlinedFileStat {
	stats := make([]inlinedFileStat, 0, 32)
	seen := make(map[string]inlinedFileStat)

	// --- 1) START FILE blocks (direct @file)
	{
		const startPrefix = "--- START FILE:"
		const endPrefix = "--- END FILE:"

		i := 0
		for {
			start := strings.Index(raw[i:], startPrefix)
			if start == -1 {
				break
			}
			start += i

			lineEnd := strings.IndexByte(raw[start:], '\n')
			if lineEnd == -1 {
				break
			}
			lineEnd += start
			line := strings.TrimSpace(raw[start:lineEnd])

			pathPart := strings.TrimSpace(strings.TrimPrefix(line, startPrefix))
			pathPart = strings.TrimSpace(strings.TrimSuffix(pathPart, "---"))

			contentStart := lineEnd + 1

			endMarker := endPrefix + " " + pathPart
			endPos := strings.Index(raw[contentStart:], endMarker)
			if endPos == -1 {
				endPos = strings.Index(raw[contentStart:], endPrefix)
				if endPos == -1 {
					break
				}
			}
			endPos += contentStart

			content := raw[contentStart:endPos]
			content = strings.Trim(content, "\n")

			st := inlinedFileStat{Path: pathPart, Chars: len(content), Kind: "file"}
			if prev, ok := seen[st.Path]; !ok || st.Chars > prev.Chars {
				seen[st.Path] = st
			}

			i = endPos
		}
	}

	// --- 2) PATH blocks (directory scans)
	{
		const pathPrefix = "PATH: "
		const delim = "\n---"

		i := 0
		for {
			p := strings.Index(raw[i:], pathPrefix)
			if p == -1 {
				break
			}
			p += i

			lineEnd := strings.IndexByte(raw[p:], '\n')
			if lineEnd == -1 {
				break
			}
			lineEnd += p
			pathLine := strings.TrimSpace(raw[p:lineEnd])
			absPath := strings.TrimSpace(strings.TrimPrefix(pathLine, pathPrefix))

			contentStart := lineEnd + 1

			d := strings.Index(raw[contentStart:], delim)
			if d == -1 {
				d = len(raw) - contentStart
			}
			contentEnd := contentStart + d

			content := raw[contentStart:contentEnd]
			content = strings.Trim(content, "\n")

			st := inlinedFileStat{Path: absPath, Chars: len(content), Kind: "scan"}
			if prev, ok := seen[st.Path]; !ok || st.Chars > prev.Chars {
				seen[st.Path] = st
			}

			i = contentEnd
		}
	}

	for _, v := range seen {
		stats = append(stats, v)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Path < stats[j].Path
	})

	return stats
}

func runPrompt(input string) {
	tokens := tokenizer.Tokenize(input)

	var finalOutput strings.Builder

	for _, token := range tokens {
		if token.Type() == tokenizer.RawText {
			finalOutput.WriteString(token.Literal())
			continue
		}

		if token.Type() == tokenizer.FilePath {
			relativePath := token.Value()
			absPath, err := filepath.Abs(relativePath)
			if err != nil {
				fmt.Printf("⚠️ Warning: Could not resolve path %s\n", relativePath)
				finalOutput.WriteString(token.Literal())
				continue
			}

			content, err := os.ReadFile(absPath)
			if err != nil {
				fmt.Printf("⚠️ Warning: Could not read file %s: %v\n", relativePath, err)
				finalOutput.WriteString(token.Literal())
				continue
			}

			fmt.Fprintf(&finalOutput, "\n\n--- START FILE: %s ---\n", relativePath)
			finalOutput.Write(content)
			fmt.Fprintf(&finalOutput, "\n--- END FILE: %s ---\n\n", relativePath)
		}
	}

	rawOut := finalOutput.String()

	// Compute captured file stats from the raw output (covers @file AND @dir expansions)
	stats := parseCapturedFilesFromOutput(rawOut)

	// Remove empty lines and comment lines BEFORE copying
	finalString := stripEmptyAndCommentLines(rawOut)

	// Copy to Clipboard
	if err := clipboard.WriteAll(finalString); err != nil {
		log.Fatalf("Failed to copy to clipboard: %v", err)
	}

	// Console output: DO NOT print the finalString.
	fmt.Println("---------------------")

	if len(stats) == 0 {
		fmt.Println("Success! Copied prompt (no captured files).")
		fmt.Printf("Totals: files=0, chars=%d\n", len(finalString))
		fmt.Println("---------------------")
		return
	}

	fmt.Println("Success! Copied prompt with captured files:")
	for _, st := range stats {
		fmt.Printf("  - %s (%d chars) [%s]\n", st.Path, st.Chars, st.Kind)
	}

	fmt.Printf("Totals: files=%d, chars=%d\n", len(stats), len(finalString))
	fmt.Println("---------------------")
}

func printUsage() {
	fmt.Println("Usage: aic [command] \"prompt with @file/paths\"")
	fmt.Println("\nDefault:")
	fmt.Println("  aic \"...\"       -> Same as `aic prompt \"...\"`")
	fmt.Println("\nCommands:")
	fmt.Println("  prompt \"...\"    -> Wraps text and expands @file/paths INLINE")
	fmt.Println("  help            -> Show this help message")
}
