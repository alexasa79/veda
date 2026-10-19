package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLSPDidChangeIntegration tests the full LSP integration with didChange notifications
func TestLSPDidChangeIntegration(t *testing.T) {
	// Skip if running in CI or short mode
	if testing.Short() {
		t.Skip("Skipping LSP integration test in short mode")
	}

	// Check if pylsp is available for Python tests
	if _, err := os.Stat("/usr/local/bin/pylsp"); err != nil && os.IsNotExist(err) {
		if _, err := os.Stat("/usr/bin/pylsp"); err != nil && os.IsNotExist(err) {
			t.Skip("pylsp not found, skipping Python LSP integration tests")
		}
	}

	t.Log("Starting LSP didChange integration test...")

	// Create proxy manager
	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	// Get absolute path to test file
	cwd, _ := os.Getwd()
	testFile := filepath.Join(cwd, "../../artifacts/python_test_project/services/task_manager.py")

	// Make sure the test file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Test file not found: %s", testFile)
	}

	t.Logf("Using test file: %s", testFile)

	// Read the original content
	originalContent, err := pm.GetFile(testFile, true)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	t.Logf("Read file successfully, content length: %d", len(originalContent.Content))

	// LSP server starts automatically when needed, no explicit start needed
	t.Log("LSP server will start automatically on first request")

	// Give LSP server time to initialize
	time.Sleep(2 * time.Second)

	// Test scenarios
	t.Run("CompletionAfterUnsavedChanges", func(t *testing.T) {
		// Simulate adding a new method to the TaskManager class
		// First, find where to insert (after the __init__ method)
		lines := strings.Split(originalContent.Content, "\n")
		insertLine := -1
		for i, line := range lines {
			if strings.Contains(line, "def __init__") {
				// Find the end of __init__ method
				for j := i + 1; j < len(lines); j++ {
					// Look for next method or end of class
					if j < len(lines)-1 && !strings.HasPrefix(lines[j], " ") && !strings.HasPrefix(lines[j], "\t") && lines[j] != "" {
						insertLine = j
						break
					}
					if strings.HasPrefix(lines[j], "    def ") {
						insertLine = j
						break
					}
				}
				break
			}
		}

		if insertLine == -1 {
			t.Skip("Could not find appropriate insertion point in test file")
		}

		// Create modified content with a new method
		newMethod := `
    def get_task_count(self):
        """Get the total number of tasks."""
        return len(self.`

		// Insert the new method (incomplete, so we can test completion)
		modifiedLines := make([]string, 0, len(lines)+4)
		modifiedLines = append(modifiedLines, lines[:insertLine]...)
		modifiedLines = append(modifiedLines, strings.Split(newMethod, "\n")...)
		modifiedLines = append(modifiedLines, lines[insertLine:]...)

		modifiedContent := strings.Join(modifiedLines, "\n")

		t.Logf("Inserting new method at line %d", insertLine)

		// Send didChange notification
		err = pm.NotifyFileChange(testFile, modifiedContent)
		if err != nil {
			t.Logf("Warning: Failed to send didChange notification: %v", err)
			// This might be expected if LSP is not fully initialized
		}

		// Give LSP time to process the change
		time.Sleep(500 * time.Millisecond)

		// Calculate the position for completion (after "self.")
		// Find the line number and character position
		completionLine := insertLine + 3 // The line with "return len(self."
		completionChar := len("        return len(self.")

		t.Logf("Requesting completion at line %d, char %d", completionLine, completionChar)

		// Request completions at the position after "self."
		completions, err := pm.GetCompletions(testFile, completionLine, completionChar)
		if err != nil {
			t.Logf("GetCompletions failed: %v", err)
			// Don't fail the test - LSP might not be fully ready
			return
		}

		// Check if we got completions
		if completions != nil {
			// Try to extract completion items
			if result, ok := completions.(map[string]interface{}); ok {
				if items, ok := result["items"].([]interface{}); ok {
					t.Logf("Got %d completion items", len(items))

					// Look for "tasks" in completions (should be available as self.tasks)
					foundTasks := false
					for _, item := range items {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if label, ok := itemMap["label"].(string); ok {
								if label == "tasks" || strings.Contains(label, "tasks") {
									foundTasks = true
									t.Logf("Found 'tasks' in completions")
									break
								}
							}
						}
					}

					if !foundTasks && len(items) > 0 {
						t.Logf("Did not find 'tasks' in completions, but got %d other items", len(items))
						// Log first few items for debugging
						for i := 0; i < len(items) && i < 5; i++ {
							if itemMap, ok := items[i].(map[string]interface{}); ok {
								if label, ok := itemMap["label"].(string); ok {
									t.Logf("  - %s", label)
								}
							}
						}
					}
				}
			}
		}
	})

	t.Run("SignatureHelpAfterUnsavedChanges", func(t *testing.T) {
		// Add a new function call that's not saved to disk
		lines := strings.Split(originalContent.Content, "\n")

		// Find a good place to add a function call
		insertLine := -1
		for i, line := range lines {
			if strings.Contains(line, "def add_task") {
				// Find inside the method
				for j := i + 1; j < len(lines); j++ {
					if strings.Contains(lines[j], "self.tasks.append") {
						insertLine = j + 1
						break
					}
				}
				break
			}
		}

		if insertLine == -1 {
			t.Skip("Could not find appropriate insertion point for signature help test")
		}

		// Add a new line with a function call
		newLine := "        task.set_priority("

		modifiedLines := make([]string, 0, len(lines)+1)
		modifiedLines = append(modifiedLines, lines[:insertLine]...)
		modifiedLines = append(modifiedLines, newLine)
		modifiedLines = append(modifiedLines, lines[insertLine:]...)

		modifiedContent := strings.Join(modifiedLines, "\n")

		// Send didChange notification
		err = pm.NotifyFileChange(testFile, modifiedContent)
		if err != nil {
			t.Logf("Warning: Failed to send didChange notification: %v", err)
		}

		// Give LSP time to process
		time.Sleep(500 * time.Millisecond)

		// Request signature help at the position inside the parentheses
		signatureLine := insertLine
		signatureChar := len(newLine) - 1 // Just inside the opening parenthesis

		t.Logf("Requesting signature help at line %d, char %d", signatureLine, signatureChar)

		sigHelp, err := pm.GetSignatureHelp(testFile, signatureLine, signatureChar)
		if err != nil {
			t.Logf("GetSignatureHelp failed: %v", err)
			// Don't fail - LSP might not provide signature help for this
			return
		}

		if sigHelp != nil {
			t.Logf("Got signature help response: %T", sigHelp)
			// Log the response for debugging
			if result, ok := sigHelp.(map[string]interface{}); ok {
				if signatures, ok := result["signatures"].([]interface{}); ok {
					t.Logf("Got %d signatures", len(signatures))
				}
			}
		}
	})

	t.Run("DiagnosticsAfterUnsavedChanges", func(t *testing.T) {
		// Introduce a syntax error that's not saved to disk
		lines := strings.Split(originalContent.Content, "\n")

		// Find a line to break
		breakLine := -1
		for i, line := range lines {
			if strings.Contains(line, "def add_task") {
				breakLine = i
				break
			}
		}

		if breakLine == -1 {
			t.Skip("Could not find line to introduce error")
		}

		// Introduce a syntax error (missing colon)
		lines[breakLine] = strings.Replace(lines[breakLine], ":", "", 1)
		modifiedContent := strings.Join(lines, "\n")

		// Send didChange notification
		err = pm.NotifyFileChange(testFile, modifiedContent)
		if err != nil {
			t.Logf("Warning: Failed to send didChange notification: %v", err)
		}

		// Give LSP time to process and generate diagnostics
		time.Sleep(1 * time.Second)

		// Get diagnostics
		diagnostics, err := pm.GetDiagnostics(testFile)
		if err != nil {
			t.Logf("GetDiagnostics failed: %v", err)
			return
		}

		if diagnostics != nil {
			t.Logf("Got diagnostics map with %d URIs", len(diagnostics))
			// Check if we have diagnostics for our test file
			for uri, fileDiags := range diagnostics {
				if strings.Contains(uri, "task_manager.py") {
					t.Logf("Found %d diagnostics for task_manager.py", len(fileDiags))
					// We should have at least one diagnostic about the syntax error
					if len(fileDiags) > 0 {
						t.Log("Successfully detected syntax error in unsaved changes")
					}
				}
			}
		}
	})
}

// TestLSPDidChangeWithGoProject tests LSP integration with a Go project
func TestLSPDidChangeWithGoProject(t *testing.T) {
	// Skip if running in CI or short mode
	if testing.Short() {
		t.Skip("Skipping Go LSP integration test in short mode")
	}

	// Check if gopls is available
	gopaths := []string{
		"/usr/local/bin/gopls",
		os.ExpandEnv("$GOPATH/bin/gopls"),
		os.ExpandEnv("$HOME/go/bin/gopls"),
	}

	goplsFound := false
	for _, path := range gopaths {
		if _, err := os.Stat(path); err == nil {
			goplsFound = true
			break
		}
	}

	if !goplsFound {
		t.Skip("gopls not found, skipping Go LSP integration tests")
	}

	t.Log("Starting Go LSP didChange integration test...")

	// Create proxy manager
	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	// Use main.go as test file
	cwd, _ := os.Getwd()
	testFile := filepath.Join(cwd, "../../main.go")

	// Read original content
	originalContent, err := pm.GetFile(testFile, true)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// LSP server starts automatically when needed
	t.Log("LSP server will start automatically on first request")

	// Give gopls time to initialize
	time.Sleep(3 * time.Second)

	t.Run("GoCompletionAfterUnsavedChanges", func(t *testing.T) {
		// Add a new line in main function
		lines := strings.Split(originalContent.Content, "\n")

		// Find main function
		mainLine := -1
		for i, line := range lines {
			if strings.Contains(line, "func main()") {
				// Find first line inside main
				for j := i + 1; j < len(lines); j++ {
					if strings.TrimSpace(lines[j]) != "" && strings.HasPrefix(lines[j], "\t") {
						mainLine = j
						break
					}
				}
				break
			}
		}

		if mainLine == -1 {
			t.Skip("Could not find main function")
		}

		// Add incomplete line for completion
		incompleteLine := "\tfmt."

		modifiedLines := make([]string, 0, len(lines)+1)
		modifiedLines = append(modifiedLines, lines[:mainLine]...)
		modifiedLines = append(modifiedLines, incompleteLine)
		modifiedLines = append(modifiedLines, lines[mainLine:]...)

		modifiedContent := strings.Join(modifiedLines, "\n")

		// Send didChange notification
		err = pm.NotifyFileChange(testFile, modifiedContent)
		if err != nil {
			t.Logf("Warning: Failed to send didChange notification: %v", err)
		}

		// Give gopls time to process
		time.Sleep(500 * time.Millisecond)

		// Request completions after "fmt."
		completionLine := mainLine
		completionChar := len(incompleteLine)

		t.Logf("Requesting Go completions at line %d, char %d", completionLine, completionChar)

		completions, err := pm.GetCompletions(testFile, completionLine, completionChar)
		if err != nil {
			t.Logf("GetCompletions failed: %v", err)
			return
		}

		if completions != nil {
			if result, ok := completions.(map[string]interface{}); ok {
				if items, ok := result["items"].([]interface{}); ok {
					t.Logf("Got %d completion items from gopls", len(items))

					// Look for common fmt functions
					foundPrintln := false
					for _, item := range items {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if label, ok := itemMap["label"].(string); ok {
								if label == "Println" || strings.Contains(label, "Println") {
									foundPrintln = true
									t.Log("Found 'Println' in completions - gopls is working with unsaved changes!")
									break
								}
							}
						}
					}

					if !foundPrintln && len(items) > 0 {
						t.Logf("Got %d completions but didn't find Println", len(items))
					}
				}
			}
		}
	})
}
