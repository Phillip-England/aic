package main

import (
	"os"

	"github.com/phillip-england/aic/aic"
)

func main() {
	if err := aic.NewCLI().Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
