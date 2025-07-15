package proxy

import (
	"testing"
	"time"
)

func TestProxyBasic(t *testing.T) {
	// Basic test to ensure proxy starts correctly
	t.Log("Testing basic proxy functionality...")

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}
	defer client.Close()

	// Try a simple operation - list current directory
	t.Log("Attempting to list directory...")
	files, err := client.ListDirectory("../../")
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}

	t.Logf("Listed %d files in current directory", len(files))

	// Should at least find main.go
	foundMain := false
	for _, file := range files {
		if file.Name == "main.go" {
			foundMain = true
			break
		}
	}

	if !foundMain {
		t.Error("Expected to find main.go in current directory")
	}
}

func TestGetDocumentSymbols(t *testing.T) {
	// Skip if running in CI or if gopls is not available
	if testing.Short() {
		t.Skip("Skipping LSP test in short mode")
	}

	// Start proxy client
	t.Log("Creating proxy client...")
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}
	defer client.Close()

	// Give the proxy time to start
	t.Log("Waiting for proxy to initialize...")
	time.Sleep(500 * time.Millisecond)

	// Test GetDocumentSymbols
	t.Run("GetDocumentSymbols", func(t *testing.T) {
		// Use main.go as our test file - need full path from the root
		testFile := "../../main.go"

		t.Logf("Calling GetDocumentSymbols for %s...", testFile)
		symbols, err := client.GetDocumentSymbols(testFile)
		if err != nil {
			t.Fatalf("GetDocumentSymbols failed: %v", err)
		}

		t.Logf("GetDocumentSymbols succeeded, got %d symbols", len(symbols))

		// Verify we got symbols
		if len(symbols) == 0 {
			t.Error("Expected to get document symbols, got none")
		}

		// Check for main function at minimum
		foundMain := false
		for _, symbol := range symbols {
			if symbol.Name == "main" {
				foundMain = true
				break
			}
		}

		if !foundMain {
			t.Error("Expected to find 'main' function symbol")
		}

		// Log all symbols found for debugging
		t.Logf("Found %d symbols:", len(symbols))
		for i, symbol := range symbols {
			t.Logf("  [%d] %s (kind: %d) at line %d-%d",
				i, symbol.Name, symbol.Kind,
				symbol.Range.Start.Line, symbol.Range.End.Line)

			// Log children if any
			if len(symbol.Children) > 0 {
				t.Logf("      Children: %d", len(symbol.Children))
				for _, child := range symbol.Children {
					t.Logf("        - %s (kind: %d)", child.Name, child.Kind)
				}
			}
		}
	})
}