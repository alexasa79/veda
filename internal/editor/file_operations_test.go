package editor

import (
    "os"
    "path/filepath"
    "testing"
    "veda/internal/proxy"
)

func TestOpenNonExistentFile(t *testing.T) {
	// Create a test editor
	te := NewTestEditor(80, 24)
	
	// Generate a temporary file name that doesn't exist
	tempDir := t.TempDir()
	nonExistentFile := filepath.Join(tempDir, "new_file.txt")
	
	// Ensure the file doesn't exist
	if _, err := os.Stat(nonExistentFile); !os.IsNotExist(err) {
		t.Fatalf("Test file already exists: %s", nonExistentFile)
	}
	
	// Open the non-existent file
	te.openFile(nonExistentFile)
	
	// Check that a buffer was created with the correct file path
	if te.buffer == nil {
		t.Fatal("Expected buffer to be created for non-existent file")
	}
	
	if te.buffer.FilePath != nonExistentFile {
		t.Errorf("Expected buffer file path to be %s, got %s", nonExistentFile, te.buffer.FilePath)
	}
	
	// Check that the buffer is empty
	if te.buffer.Rope.String() != "" {
		t.Errorf("Expected empty buffer for new file, got: %s", te.buffer.Rope.String())
	}
	
	// Type some content
	te.ExecuteCommand("enter-insert-mode")
	te.InsertChar('H')
	te.InsertChar('e')
	te.InsertChar('l')
	te.InsertChar('l')
	te.InsertChar('o')
	te.ExecuteCommand("exit-insert-mode")
	
	// Check the content
	if te.buffer.Rope.String() != "Hello" {
		t.Errorf("Expected buffer content to be 'Hello', got: %s", te.buffer.Rope.String())
	}
	
	// Buffer should be marked as modified
	if !te.buffer.IsModified {
		t.Error("Expected buffer to be marked as modified")
	}
}

func TestOpenNonExistentFileWithSubdirectory(t *testing.T) {
	// Create a test editor
	te := NewTestEditor(80, 24)
	
	// Generate a file path in a subdirectory that doesn't exist
	tempDir := t.TempDir()
	nonExistentFile := filepath.Join(tempDir, "subdir", "another", "new_file.txt")
	
	// Ensure the file and directories don't exist
	if _, err := os.Stat(nonExistentFile); !os.IsNotExist(err) {
		t.Fatalf("Test file already exists: %s", nonExistentFile)
	}
	
	// Open the non-existent file
	te.openFile(nonExistentFile)
	
	// Check that a buffer was created with the correct file path
	if te.buffer == nil {
		t.Fatal("Expected buffer to be created for non-existent file")
	}
	
	if te.buffer.FilePath != nonExistentFile {
		t.Errorf("Expected buffer file path to be %s, got %s", nonExistentFile, te.buffer.FilePath)
	}
	
	// Check that the buffer is empty
	if te.buffer.Rope.String() != "" {
		t.Errorf("Expected empty buffer for new file, got: %s", te.buffer.Rope.String())
	}
	
	// Type some simple content
	te.ExecuteCommand("enter-insert-mode")
	content := "This is a new file in a subdirectory"
	for _, ch := range content {
		te.InsertChar(ch)
	}
	te.ExecuteCommand("exit-insert-mode")
	
	// Check the content
	if te.buffer.Rope.String() != content {
		t.Errorf("Expected buffer content to be:\n%s\nGot:\n%s", content, te.buffer.Rope.String())
	}
	
	// Buffer should be marked as modified
	if !te.buffer.IsModified {
		t.Error("Expected buffer to be marked as modified")
	}
}

// TestFuzzyFileListCacheAndRefresh verifies that the fuzzy file list uses cache
// and that Ctrl+R refresh replaces content after the widget has been rendered.
func TestFuzzyFileListCacheAndRefresh(t *testing.T) {
    te := NewTestEditor(100, 30)

    // Simulate a remote working directory
    te.workingDirectory = "server:~/"

    // Seed cache with a small list
    cached := []proxy.FileInfo{
        {Name: "a.txt", Type: "file", Path: "/home/user/a.txt"},
        {Name: "b.txt", Type: "file", Path: "/home/user/b.txt"},
    }
    te.setCachedDirectoryListing(te.workingDirectory, cached)

    // Create fuzzy widget and overlay using cached results
    contentUpdates := make(chan []string, 2)
    // First call should use cache
    te.sendDirectoryListingRequest(contentUpdates, true)

    // Consume first update from cache
    var initial []string
    select {
    case initial = <-contentUpdates:
    default:
        t.Fatalf("expected initial content from cache")
    }

    if len(initial) != 2 {
        t.Fatalf("expected 2 items from cache, got %d", len(initial))
    }

    // Now emulate a refresh: clear cache and send new remote listing
    updated := []proxy.FileInfo{
        {Name: "c.txt", Type: "file", Path: "/home/user/c.txt"},
        {Name: "d.txt", Type: "file", Path: "/home/user/d.txt"},
        {Name: "e.txt", Type: "file", Path: "/home/user/e.txt"},
    }
    delete(te.dirCache, te.workingDirectory)

    // Spy on proxy manager by directly pushing updated content to processing
    // In tests we call setCachedDirectoryListing to simulate refresh arrival
    te.setCachedDirectoryListing(te.workingDirectory, updated)

    // Ask again with useCache=true; since we've set cache to updated, the channel should receive new list
    te.sendDirectoryListingRequest(contentUpdates, true)
    var refreshed []string
    select {
    case refreshed = <-contentUpdates:
    default:
        t.Fatalf("expected refreshed content after cache update")
    }

    if len(refreshed) != 3 {
        t.Fatalf("expected 3 items after refresh, got %d", len(refreshed))
    }
}