package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os" // Ensure os is imported
	"os/exec"
	"strconv"
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
			fmt.Println("Initialized ./aic")
		}
	case "watch":
		if err := watcher.Start(500*time.Millisecond, 100*time.Millisecond); err != nil {
			log.Fatal(err)
		}
	case "history":
		handleHistoryCmd()
	default:
		fmt.Println("Unknown command")
	}
}

func openPromptInVi() {
	d, err := dir.OpenAiDir()
	if err != nil {
		fmt.Println("No aic dir found. Run 'aic init'")
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

func handleHistoryCmd() {

	d, err := dir.OpenAiDir()

	if err != nil {

		fmt.Println("No aic dir found. Run 'aic init'")

		return

	}



	if len(os.Args) > 2 {

		switch os.Args[2] {

		case "size":

			if len(os.Args) == 3 {

				historyFilePath := d.GetHistoryFilePath()

				fileInfo, err := os.Stat(historyFilePath)

				if err != nil {

					if os.IsNotExist(err) {

						fmt.Println("History file does not exist.")

						return

					}

					fmt.Printf("Error getting history file info: %v\n", err)

					return

				}

				fmt.Printf("History file size: %d bytes\n", fileInfo.Size())

			} else {

				fmt.Println("Usage: aic history size")

			}

			return

		case "delete":

			if len(os.Args) == 4 { // aic history delete <N>

				key, err := strconv.Atoi(os.Args[3])

				if err != nil {

					fmt.Println("Error: Invalid history index. Must be an integer.")

					return

				}

				if err := d.DeleteHistoryEntry(key); err != nil {

					fmt.Println("Error deleting history entry:", err)

				} else {

					fmt.Printf("Deleted history entry %d\n", key)

				}

			} else if len(os.Args) == 5 { // aic history delete <N> <M>

				start, err1 := strconv.Atoi(os.Args[3])

				end, err2 := strconv.Atoi(os.Args[4])

				if err1 != nil || err2 != nil {

					fmt.Println("Error: Invalid history index. Must be integers.")

					return

				}

				if err := d.DeleteHistoryRange(start, end); err != nil {

					fmt.Println("Error deleting history range:", err)

				} else {

					fmt.Printf("Deleted history entries from %d to %d\n", start, end)

				}

			} else {

				fmt.Println("Usage: aic history delete <index> | <start_index> <end_index>")

			}

			return

		case "load":

			if len(os.Args) == 4 { // aic history load <file.json>

				filePath := os.Args[3]

				data, err := os.ReadFile(filePath)

				if err != nil {

					fmt.Printf("Error reading history file '%s': %v\n", filePath, err)

					return

				}

				if err := d.LoadHistory(data); err != nil {

					fmt.Println("Error loading history:", err)

				} else {

					fmt.Println("Successfully loaded new history.")

				}

			} else {

				fmt.Println("Usage: aic history load <file.json>")

			}

			return

		}

	}



	switch len(os.Args) {

	case 2: // aic history

		history, err := d.GetAllHistory()

		if err != nil {

			fmt.Println("Error reading history:", err)

			return

		}

		jsonOutput, err := json.MarshalIndent(history, "", "  ")

		if err != nil {

			fmt.Println("Error formatting history to JSON:", err)

			return

		}

		fmt.Println(string(jsonOutput))



	case 3: // aic history <N>

		key, err := strconv.Atoi(os.Args[2])

		if err != nil {

			fmt.Println("Error: Invalid history index. Must be an integer, 'size', 'delete', or 'load'.")

			return

		}

		entry, err := d.GetHistoryEntry(key)

		if err != nil {

			fmt.Println("Error:", err)

			return

		}

		fmt.Println(entry)



	case 4: // aic history <N> <M>

		start, err1 := strconv.Atoi(os.Args[2])

		end, err2 := strconv.Atoi(os.Args[3])

		if err1 != nil || err2 != nil {

			fmt.Println("Error: Invalid history index. Must be integers.")

			return

		}

		if start > end {

			fmt.Println("Error: Start index cannot be greater than end index.")

			return

		}

		entries, err := d.GetHistoryRange(start, end)

		if err != nil {

			fmt.Println("Error:", err)

			return

		}

		jsonOutput, err := json.MarshalIndent(entries, "", "  ")

		if err != nil {

			fmt.Println("Error formatting history to JSON:", err)

			return

		}

		fmt.Println(string(jsonOutput))

	default:

		fmt.Println("Usage: aic history [size|delete|load] | [<index>] | [<start_index> <end_index>]")

	}

}
