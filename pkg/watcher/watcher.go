package watcher

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/phillip-england/aic/pkg/dir"
	"github.com/phillip-england/aic/pkg/interpreter"
)

func Start(pollInterval, debounce time.Duration) error {
	aiDir, err := dir.OpenAiDir()
	if err != nil {
		aiDir, _ = dir.NewAiDir(false)
	}

	return WatchPrompt(pollInterval, debounce, aiDir)
}

func WatchPrompt(pollInterval, debounce time.Duration, aiDir *dir.AiDir) error {
	if aiDir == nil {
		var err error
		aiDir, err = dir.OpenAiDir()
		if err != nil {
			aiDir, _ = dir.NewAiDir(false)
		}
	}
	if aiDir == nil {
		return fmt.Errorf("could not open or create .aic directory")
	}

	interp := interpreter.New(aiDir)
	fmt.Printf("Watching %s...\n", aiDir.PromptPath())

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
					if err := aiDir.StashPrompt(raw); err != nil {
						fmt.Println("Error stashing prompt:", err)
					}
					// Update lastMod to avoid re-triggering immediately if file mod time drifted
					if i, e := os.Stat(aiDir.PromptPath()); e == nil {
						lastMod = i.ModTime()
					}
				}
			}
		}
	}
}

func isBodyEmpty(s string) bool {
	parts := strings.Split(s, "---")
	if len(parts) < 3 {
		return true
	}
	return strings.TrimSpace(parts[2]) == ""
}
