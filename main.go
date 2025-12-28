package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/phillip-england/aic/pkg/dir"
	"github.com/phillip-england/aic/pkg/interpreter"
	"github.com/phillip-england/aic/pkg/watcher"
)

func main() {
	if len(os.Args) < 2 {
		runOnce()
		return
	}

	switch os.Args[1] {
	case "init":
		_, err := dir.NewAiDir(true)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Initialized ./ai")
		}
	case "watch":
	if err := watcher.Start(500*time.Millisecond, 100*time.Millisecond); err != nil {
			log.Fatal(err)
	}
	default:
		fmt.Println("Unknown command")
	}
}

func runOnce() {
	d, err := dir.OpenAiDir()
	if err != nil {
		fmt.Println("No ai dir found. Run 'aic init'")
		return
	}
	i := interpreter.New(d)
	raw, _ := d.ReadPrompt()
	i.Run(raw)
}