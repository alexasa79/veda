package main

import (
	"fmt"
	"os"
	"veda/internal/editor"
	"veda/internal/log"
)

func main() {
	// Set up logging
	log.SetupLogging("veda.log")
	defer log.CleanupLogging()

	ed, err := editor.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create editor: %v", err)
		os.Exit(1)
	}
	defer ed.Close()

	// Check if a file was provided as command-line argument
	if len(os.Args) > 1 {
		filePath := os.Args[1]
		ed.OpenFile(filePath)
	}

	ed.Run()
}
