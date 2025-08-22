package proxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateNewFile(t *testing.T) {
	// Create a temporary directory for our test
	tmpDir := t.TempDir()

	// Define a non-existent file path
	testFile := filepath.Join(tmpDir, "new_file.txt")

	// Ensure the file doesn't exist
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatalf("Test file should not exist initially: %s", testFile)
	}

	// Create proxy manager
	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	// Step 1: Attempt to open the non-existent file
	t.Logf("Attempting to open non-existent file: %s", testFile)
	fileContent, err := pm.GetFile(testFile)
	if err != nil {
		t.Fatalf("GetFile should not fail for non-existent file: %v", err)
	}

	// Should return empty content for new file
	if fileContent.Content != "" {
		t.Errorf("Expected empty content for new file, got: %q", fileContent.Content)
	}

	// Step 2: Modify the file content
	newContent := "Hello, this is a new file!\nWith multiple lines.\n"
	t.Logf("Writing content to file: %s", testFile)

	// Step 3: Attempt to save the file
	diagnostics, err := pm.WriteFile(testFile, newContent)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Should not have any errors in diagnostics
	t.Logf("WriteFile returned %d diagnostics", len(diagnostics))

	// Step 4: Verify the file was actually created on disk
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("File was not created on disk: %s", testFile)
	}

	// Step 5: Verify the content was written correctly
	writtenContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(writtenContent) != newContent {
		t.Errorf("File content mismatch.\nExpected: %q\nGot: %q", newContent, string(writtenContent))
	}

	t.Logf("Successfully created and wrote to new file: %s", testFile)
}
