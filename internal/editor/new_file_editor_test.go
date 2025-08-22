package editor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"veda/internal/proxy"
)

func TestEditorCreateNewFile(t *testing.T) {
	// Create a temporary directory for our test
	tmpDir := t.TempDir()

	// Define a non-existent file path
	testFile := filepath.Join(tmpDir, "editor_new_file.txt")

	// Ensure the file doesn't exist
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Fatalf("Test file should not exist initially: %s", testFile)
	}

	// Create proxy manager
	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	// Create a test editor
	te := NewTestEditor(80, 24)
	te.Editor.proxyManager = pm

	// Step 1: Open the non-existent file using editor's openFile method
	t.Logf("Opening non-existent file via editor: %s", testFile)
	te.Editor.openFile(testFile)

	// Should have a buffer for the file
	if te.Editor.buffer == nil {
		t.Fatalf("Editor should have created a buffer for the new file")
	}

	if te.Editor.buffer.FilePath != testFile {
		t.Errorf("Buffer file path mismatch. Expected: %s, Got: %s", testFile, te.Editor.buffer.FilePath)
	}

	// Step 2: Add some content to the buffer
	t.Logf("Adding content to buffer")

	// Insert content into the buffer using the Insert method
	testContent := "Hello from editor test!\nThis is line 2.\n"
	te.Editor.buffer.Insert(0, testContent)

	// Mark the buffer as modified
	te.Editor.buffer.MarkModified()

	// Step 3: Save the file using editor's writeFile method
	t.Logf("Saving file via editor")
	err = te.Editor.writeFile([]string{})
	if err != nil {
		t.Fatalf("Editor writeFile failed: %v", err)
	}

	// Wait for async save operation to complete
	t.Logf("Waiting for async save to complete")
	// Check periodically for file existence (async save might take time)
	for i := 0; i < 50; i++ { // Wait up to 5 seconds
		if _, err := os.Stat(testFile); err == nil {
			break // File exists, save completed
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Step 4: Verify the file was actually created on disk
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("File was not created on disk by editor: %s", testFile)
	}

	// Step 5: Verify the content was written correctly
	writtenContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(writtenContent) != testContent {
		t.Errorf("File content mismatch.\nExpected: %q\nGot: %q", testContent, string(writtenContent))
	}

	// Step 6: Verify buffer is marked as saved
	if te.Editor.buffer.IsModified {
		t.Errorf("Buffer should be marked as saved after successful write")
	}

	t.Logf("Successfully created and saved new file via editor: %s", testFile)
}
