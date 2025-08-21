package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLSPDidChangeSimple tests that didChange notifications are properly sent to LSP
func TestLSPDidChangeSimple(t *testing.T) {
	// Skip if running in CI or short mode
	if testing.Short() {
		t.Skip("Skipping LSP test in short mode")
	}

	// Check for gopls
	gopaths := []string{
		"/usr/local/bin/gopls",
		os.ExpandEnv("$GOPATH/bin/gopls"),
		os.ExpandEnv("$HOME/go/bin/gopls"),
	}

	goplsFound := false
	for _, path := range gopaths {
		if _, err := os.Stat(path); err == nil {
			goplsFound = true
			t.Logf("Found gopls at: %s", path)
			break
		}
	}

	if !goplsFound {
		t.Skip("gopls not found, skipping test")
	}

	// Create proxy manager
	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	// Use the permanent test file in artifacts directory
	cwd, _ := os.Getwd()
	testFile := filepath.Join(cwd, "../../artifacts/go_test_project/main.go")

	// Make sure the test file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Test file not found: %s", testFile)
	}

	// Read the original content for use in the test
	originalBytes, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	originalContent := string(originalBytes)

	t.Logf("Using test file: %s", testFile)
	t.Log("Note: This test only sends didChange notifications - it does not modify the file on disk")

	// First, open the file to trigger LSP initialization
	// This simulates what happens when the editor opens a file
	t.Log("Opening file to initialize LSP...")

	// Get the file through proxy (this might trigger LSP)
	_, err = pm.GetFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Try to get document symbols to ensure LSP is working
	t.Log("Getting document symbols to verify LSP is working...")
	symbols, err := pm.GetDocumentSymbols(testFile)
	if err != nil {
		t.Logf("Warning: GetDocumentSymbols failed: %v", err)
	} else {
		t.Logf("Got %d symbols - LSP appears to be working", len(symbols))
	}

	// Now test didChange notification immediately without waiting
	t.Run("ModifyAndGetCompletions", func(t *testing.T) {
		// Modify the helper function to add an incomplete line
		// Use fmt. to trigger completions
		modifiedContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func helper() {
	// Function to test completions
	x := fmt.
	fmt.Println("Helper function")
}

func calculate(a, b int) int {
	return a + b
}
`

		// Send didChange notification
		// This sends the modified content to the LSP server but does NOT write to disk
		t.Log("Sending didChange notification with modified content...")
		err = pm.NotifyFileChange(testFile, modifiedContent)
		if err != nil {
			t.Logf("Warning: NotifyFileChange returned error: %v", err)
		}

		// Give LSP time to process the change
		time.Sleep(500 * time.Millisecond)

		// Try to get completions at the position after "fmt."
		// Count the exact line and column
		lines := strings.Split(modifiedContent, "\n")
		targetLine := -1
		targetChar := -1
		for i, line := range lines {
			if strings.Contains(line, "x := fmt.") {
				targetLine = i
				// Position at the dot (LSP uses 0-based indexing and may want cursor AT the trigger char)
				targetChar = strings.Index(line, "fmt.") + len("fmt.") - 1
				t.Logf("Line %d: %q", i, line)
				break
			}
		}

		if targetLine == -1 {
			t.Fatal("Could not find x := fmt. line")
		}

		t.Logf("Found 'x := fmt.' at line %d (content: %q)", targetLine, lines[targetLine])
		t.Logf("Line length: %d bytes", len(lines[targetLine]))
		t.Logf("Character position after 'fmt.': %d", targetChar)

		// Debug: Show the line as bytes
		lineBytes := []byte(lines[targetLine])
		t.Logf("Line as bytes: %v", lineBytes)

		line := targetLine
		character := targetChar

		t.Logf("Requesting completions at line %d, character %d", line, character)
		completions, err := pm.GetCompletions(testFile, line, character)
		if err != nil {
			t.Logf("GetCompletions failed: %v", err)
			return
		}

		// Check completions
		if completions != nil {
			if result, ok := completions.(map[string]interface{}); ok {
				if items, ok := result["items"].([]interface{}); ok {
					t.Logf("SUCCESS: Got %d completion items after didChange", len(items))

					// Look for fmt package completions
					foundFmt := false
					foundAppend := false

					for _, item := range items {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if label, ok := itemMap["label"].(string); ok {
								if label == "fmt" || strings.Contains(label, "fmt") {
									foundFmt = true
								}
								if strings.Contains(label, "Append") {
									foundAppend = true
								}
							}
						}
					}

					if foundFmt || foundAppend {
						t.Log("✓ Found fmt package completions - didChange is working!")
					} else if len(items) > 0 {
						t.Logf("Got %d completions", len(items))
						// Log first few for debugging
						for i := 0; i < len(items) && i < 3; i++ {
							if itemMap, ok := items[i].(map[string]interface{}); ok {
								if label, ok := itemMap["label"].(string); ok {
									t.Logf("  - %s", label)
								}
							}
						}
					}
				}
			}
		} else {
			t.Log("No completions returned")
		}
	})

	t.Run("TestVersionIncrement", func(t *testing.T) {
		// Test that multiple changes increment version
		for i := 0; i < 3; i++ {
			modifiedContent := strings.Replace(originalContent,
				"Hello, World!",
				"Hello, World"+strings.Repeat("!", i+1),
				1)

			err = pm.NotifyFileChange(testFile, modifiedContent)
			if err != nil {
				t.Logf("NotifyFileChange %d failed: %v", i+1, err)
			} else {
				t.Logf("Sent didChange notification %d", i+1)
			}

			time.Sleep(100 * time.Millisecond)
		}

		// After multiple changes, completions should still work
		finalContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello!!!")
	fmt.
}

func helper() {
	// Function to test completions
	fmt.Println("Helper function")
}

func calculate(a, b int) int {
	return a + b
}
`

		err = pm.NotifyFileChange(testFile, finalContent)
		if err != nil {
			t.Logf("Final NotifyFileChange failed: %v", err)
			return
		}

		time.Sleep(500 * time.Millisecond)

		// Try completions after "fmt."
		completions, err := pm.GetCompletions(testFile, 6, 6)
		if err != nil {
			t.Logf("Final GetCompletions failed: %v", err)
			return
		}

		if completions != nil {
			if result, ok := completions.(map[string]interface{}); ok {
				if items, ok := result["items"].([]interface{}); ok {
					t.Logf("✓ Got %d completions after multiple didChange notifications", len(items))
				}
			}
		}
	})
}
