package proxy

import (
	"os"
	"path/filepath"
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
		{":nohost", "", ":nohost", false},                           // No host part
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

	// Try a simple operation - get attributes of current directory
	t.Log("Attempting to get directory attributes...")
	attrs, err := client.GetFileAttributes("../../")
	if err != nil {
		t.Fatalf("GetFileAttributes failed: %v", err)
	}

	t.Logf("Got attributes for: %s", attrs.Path)

	// Verify it's a directory
	if !attrs.IsDirectory {
		t.Error("Expected path to be a directory, but it wasn't")
	}
	if !attrs.Exists {
		t.Error("Expected path to exist, but it doesn't")
	}
}

func TestListDirectoryFiltered(t *testing.T) {
	// Create a temporary directory with mixed file types
	tempDir := t.TempDir()

	// Create some supported files
	supportedTestFiles := []string{
		"test.go",
		"script.py",
		"code.c",
		"view_controller.m",
		"Makefile",
		"README.md",
		"config.json",
		"template.j2",
	}

	// Create some unsupported files
	unsupportedTestFiles := []string{
		"image.png",
		"document.pdf",
		"binary.exe",
		"archive.zip",
		"video.mp4",
	}

	// Create all test files
	for _, f := range supportedTestFiles {
		if err := os.WriteFile(filepath.Join(tempDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}
	for _, f := range unsupportedTestFiles {
		if err := os.WriteFile(filepath.Join(tempDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}

	// Start proxy client
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}
	defer client.Close()

	// Test listing with "supported" filter
	supportedFiles, err := client.ListDirectoryRecursiveFiltered(tempDir, "supported")
	if err != nil {
		t.Fatalf("ListDirectoryRecursiveFiltered (supported) failed: %v", err)
	}

	// Test listing with "all" filter
	allFiles, err := client.ListDirectoryRecursiveFiltered(tempDir, "all")
	if err != nil {
		t.Fatalf("ListDirectoryRecursiveFiltered (all) failed: %v", err)
	}

	t.Logf("Supported files: %d", len(supportedFiles))
	t.Logf("All files: %d", len(allFiles))

	// Verify that supported files is a subset of all files
	if len(supportedFiles) > len(allFiles) {
		t.Errorf("Expected supported files (%d) to be less than or equal to all files (%d)",
			len(supportedFiles), len(allFiles))
	}

	// Check that filtering is working correctly
	expectedSupportedCount := len(supportedTestFiles)
	expectedAllCount := len(supportedTestFiles) + len(unsupportedTestFiles)

	if len(supportedFiles) != expectedSupportedCount {
		t.Errorf("Expected %d supported files, got %d", expectedSupportedCount, len(supportedFiles))
	}

	if len(allFiles) != expectedAllCount {
		t.Errorf("Expected %d total files, got %d", expectedAllCount, len(allFiles))
	}

	// Verify that supported files only contain expected extensions
	supportedExts := map[string]bool{
		".go": true, ".mod": true, ".sum": true,
		".py": true, ".pyw": true, ".pyi": true,
		".c": true, ".h": true, ".m": true, ".cpp": true, ".cc": true, ".cxx": true,
		".hpp": true, ".hh": true, ".hxx": true,
		".sh": true, ".bash": true, ".zsh": true,
		".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
		".toml": true, ".ini": true, ".cfg": true, ".conf": true,
		".j2":   true,
		".html": true, ".htm": true, ".css": true, ".js": true,
		".xml": true, ".gitignore": true, ".dockerignore": true, ".env": true,
	}

	for _, file := range supportedFiles {
		if file.Type == "file" {
			// Check file extension
			ext := ""
			for i := len(file.Name) - 1; i >= 0; i-- {
				if file.Name[i] == '.' {
					ext = file.Name[i:]
					break
				}
			}

			// Some files don't have extensions (like Makefile)
			if ext != "" && !supportedExts[ext] {
				// Check if it's a known filename without extension
				knownFiles := map[string]bool{
					"Makefile": true, "makefile": true, "Dockerfile": true,
					"README": true, "LICENSE": true, "CHANGELOG": true,
				}
				if !knownFiles[file.Name] {
					t.Logf("Unexpected file in supported list: %s (ext: %s)", file.Name, ext)
				}
			}
		}
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
