package proxy

import (
	"strings"
	"testing"
	"time"
)

func TestProxyManager(t *testing.T) {
	// Test that ProxyManager can be created with local proxy
	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create ProxyManager: %v", err)
	}
	defer pm.Close()
	
	// Test local file access
	t.Run("LocalFile", func(t *testing.T) {
		// Try to read a known file (this test file)
		content, err := pm.GetFile("proxy_test.go")
		if err != nil {
			t.Errorf("Failed to read local file: %v", err)
		}
		if !strings.Contains(content.Content, "TestProxyManager") {
			t.Error("File content doesn't contain expected text")
		}
	})
	
	// Test remote path detection (should fail with "not yet implemented")
	t.Run("RemotePath", func(t *testing.T) {
		_, err := pm.GetFile("testhost:/tmp/test.txt")
		if err == nil {
			t.Error("Expected error for remote path, got nil")
		}
		// For now, we expect it to fail with remote proxy creation error
		if !strings.Contains(err.Error(), "failed to create remote proxy") {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestParseRemotePath(t *testing.T) {
	tests := []struct {
		input      string
		wantHost   string
		wantPath   string
		wantRemote bool
	}{
		// Remote paths
		{"host:/path/to/file", "host", "/path/to/file", true},
		{"server:file.txt", "server", "file.txt", true},
		{"remote.host:~/documents/test.go", "remote.host", "~/documents/test.go", true},
		
		// Local paths
		{"/home/user/file.txt", "", "/home/user/file.txt", false},
		{"./file.txt", "", "./file.txt", false},
		{"file.txt", "", "file.txt", false},
		
		// Windows-style paths (should be local)
		{"C:/Windows/System32", "", "C:/Windows/System32", false},
		{"D:\\Documents\\file.txt", "", "D:\\Documents\\file.txt", false},
		
		// Edge cases
		{"host/with/slash:path", "", "host/with/slash:path", false}, // Invalid host
		{":nohost", "", ":nohost", false}, // No host part
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotHost, gotPath, gotRemote := parseRemotePath(tt.input)
			if gotHost != tt.wantHost || gotPath != tt.wantPath || gotRemote != tt.wantRemote {
				t.Errorf("parseRemotePath(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.input, gotHost, gotPath, gotRemote,
					tt.wantHost, tt.wantPath, tt.wantRemote)
			}
		})
	}
}

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