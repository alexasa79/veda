package main

import (
	"vidya/internal/editor"
	"fmt"
	"os"
)

func main() {
	ed, err := editor.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create editor: %v", err)
		os.Exit(1)
	}
	defer ed.Close()

	ed.Run()
}
