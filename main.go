package main

import (
	"veda/internal/editor"
	"veda/internal/log"
	"veda/internal/proxy"
	"fmt"
	"os"
)

func main() {
	// Set up logging
	log.SetupLogging("veda.log")
	defer log.CleanupLogging()
	
	// Set log level to TRACE for debugging
	editor.LL.MinLevel = log.TRACE
	proxy.LL.MinLevel = log.TRACE
	
	ed, err := editor.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create editor: %v", err)
		os.Exit(1)
	}
	defer ed.Close()

	ed.Run()
}
