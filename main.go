package main

import (
	"fmt"
	"log"
	"os" // Ensure os is imported
	"os/exec"
	"time"

	"github.com/phillip-england/aic/pkg/dir"
	"github.com/phillip-england/aic/pkg/watcher"
)

func main() {
	if len(os.Args) < 2 {
		openPromptInVi()
		return
	}

	switch os.Args[1] {
	case "init":
		_, err := dir.NewAiDir(true)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Println("Initialized ./.aic")
		}
	case "watch":
		if err := watcher.Start(500*time.Millisecond, 100*time.Millisecond); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Println("Unknown command")
	}
}

func openPromptInVi() {
	d, err := dir.OpenAiDir()
	if err != nil {
		fmt.Println("No .aic dir found. Run 'aic init'")
		return
	}
	cmd := exec.Command("vi", d.PromptPath())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error opening vi: %v\n", err)
	}
}
