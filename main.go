package main

import (
	"fmt"
	"os"
	"veda/internal/editor"
	"veda/internal/log"
	"veda/internal/proxy"
	"veda/internal/terminal"
)

func main() {
	// Set up logging
	log.SetupLogging("veda.log")
	defer log.CleanupLogging()

	// Set log level to TRACE for debugging
	editor.LL.MinLevel = log.TRACE
	proxy.LL.MinLevel = log.TRACE
	terminal.LL.MinLevel = log.TRACE

	ed, err := editor.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create editor: %v", err)
		os.Exit(1)
	}
	defer ed.Close()

	ed.Run()
}
