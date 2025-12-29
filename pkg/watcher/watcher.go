package watcher

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/phillip-england/aic/pkg/dir"
	"github.com/phillip-england/aic/pkg/interpreter"
	"github.com/phillip-england/aic/pkg/llmactions"
)

func Start(pollInterval, debounce time.Duration) error {
	// We need the AiDir here to know the WorkingDir for the clipboard watcher
	aiDir, err := dir.OpenAiDir()
	if err != nil {
		aiDir, _ = dir.NewAiDir(false)
	}
	
	// Pass aiDir to WatchClipboard
	go WatchClipboard(pollInterval, aiDir) 
	
	// Pass aiDir to WatchPrompt
	return WatchPrompt(pollInterval, debounce, aiDir)
}

// Updated signature: accepts *dir.AiDir
func WatchClipboard(pollInterval time.Duration, d *dir.AiDir) {
	fmt.Println("Watching Clipboard for Agent Actions...")
	
	workingDir := "."
	if d != nil {
		workingDir = d.WorkingDir
	}

	lastClip, _ := clipboard.ReadAll()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		currentClip, err := clipboard.ReadAll()
		if err != nil {
			continue
		}
		if currentClip != lastClip {
			lastClip = currentClip
			// Pass workingDir to ProcessClipboard
			llmactions.ProcessClipboard(currentClip, workingDir)
		}
	}
}

// Updated signature: accepts *dir.AiDir to reuse the instance
func WatchPrompt(pollInterval, debounce time.Duration, aiDir *dir.AiDir) error {
	if aiDir == nil {
		// Fallback if Start() failed to init it
		var err error
		aiDir, err = dir.OpenAiDir()
		if err != nil {
			aiDir, _ = dir.NewAiDir(false)
		}
	}
	
	if aiDir == nil {
		return fmt.Errorf("could not open or create ai directory")
	}

	interp := interpreter.New(aiDir)
	fmt.Printf("Watching %s...\n", aiDir.PromptPath())
	
	// ... remainder of WatchPrompt is unchanged ...
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	var lastMod time.Time
	if info, err := os.Stat(aiDir.PromptPath()); err == nil {
		lastMod = info.ModTime()
	}
	var pending bool
	var pendingSince time.Time
	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
			info, err := os.Stat(aiDir.PromptPath())
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				pending = true
				pendingSince = time.Now()
			}
			if pending && time.Since(pendingSince) > debounce {
				pending = false
				raw, err := aiDir.ReadPrompt()
				if err != nil {
					fmt.Println("Error reading prompt:", err)
					continue
				}
				if isBodyEmpty(raw) {
					continue
				}
				fmt.Println("Processing change...")
				if err := interp.Run(raw); err != nil {
					fmt.Println("Error:", err)
				} else {
					fmt.Println("Done.")
					if i, e := os.Stat(aiDir.PromptPath()); e == nil {
						lastMod = i.ModTime()
					}
				}
			}
		}
	}
}

// ... helper unchanged ...
func isBodyEmpty(s string) bool {
	parts := strings.Split(s, "---")
	if len(parts) < 3 {
		return true
	}
	return strings.TrimSpace(parts[2]) == ""
}