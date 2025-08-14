package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	fmt.Println("Key Sequence Detector")
	fmt.Println("====================")
	fmt.Println("Press any key combinations to see the raw byte sequences.")
	fmt.Println("Press Ctrl+C to exit.")
	fmt.Println()

	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Put terminal in raw mode
	if err := setRawMode(); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting raw mode: %v\n", err)
		os.Exit(1)
	}
	defer restoreTerminal()

	// Buffer to collect input
	buf := make([]byte, 32)
	var pending []byte
	var lastReadTime time.Time

	for {
		select {
		case <-sigChan:
			fmt.Println("\nExiting...")
			return
		default:
			// Set read timeout to avoid blocking forever
			if err := syscall.SetNonblock(int(os.Stdin.Fd()), true); err == nil {
				n, err := os.Stdin.Read(buf)
				if err == nil && n > 0 {
					pending = append(pending, buf[:n]...)
					lastReadTime = time.Now()
				} else if len(pending) > 0 && time.Since(lastReadTime) > 100*time.Millisecond {
					// Process accumulated bytes after timeout
					displayKeySequence(pending)
					pending = nil
				}
				// Reset to blocking mode briefly to avoid busy loop
				syscall.SetNonblock(int(os.Stdin.Fd()), false)
			}
			
			// Small delay to prevent busy loop
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func displayKeySequence(bytes []byte) {
	fmt.Printf("Key pressed: ")
	
	// Display hex representation
	fmt.Printf("Hex: ")
	for i, b := range bytes {
		if i > 0 {
			fmt.Printf(" ")
		}
		fmt.Printf("%02x", b)
	}
	
	// Display as escape sequence format for Go code
	fmt.Printf(" | Go bytes: {")
	for i, b := range bytes {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("'\\x%02x'", b)
	}
	fmt.Printf("}")
	
	// Display readable interpretation
	fmt.Printf(" | Chars: ")
	for _, b := range bytes {
		if b >= 32 && b <= 126 {
			fmt.Printf("%c", b)
		} else if b == 27 {
			fmt.Printf("ESC")
		} else if b == 13 {
			fmt.Printf("CR")
		} else if b == 10 {
			fmt.Printf("LF")
		} else if b == 9 {
			fmt.Printf("TAB")
		} else if b == 127 {
			fmt.Printf("DEL")
		} else if b == 8 {
			fmt.Printf("BS")
		} else {
			fmt.Printf("\\x%02x", b)
		}
	}
	
	fmt.Println()
}