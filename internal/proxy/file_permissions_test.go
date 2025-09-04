package proxy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFilePreservesExecutablePermission(t *testing.T) {
	// Skip if on Windows where executable permissions work differently
	if os.PathSeparator == '\\' {
		t.Skip("Skipping on Windows")
	}

	// Create proxy client
	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}
	defer client.Close()

	// Create a temporary directory for our test
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_script.sh")

	// Initial content for the file
	initialContent := "#!/bin/bash\necho 'Hello, World!'\n"
	modifiedContent := "#!/bin/bash\necho 'Modified World!'\n"

	// Write the initial file with executable permissions
	err = os.WriteFile(testFile, []byte(initialContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify initial permissions are executable
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat initial file: %v", err)
	}
	initialMode := info.Mode()
	if initialMode.Perm()&0111 == 0 {
		t.Fatal("Initial file should have executable permissions")
	}
	t.Logf("Initial file permissions: %v", initialMode.Perm())

	// Save the file through the proxy (simulating editor save)
	_, _, err = client.WriteFile(testFile, modifiedContent)
	if err != nil {
		t.Fatalf("Failed to write file through proxy: %v", err)
	}

	// Check if the file still has executable permissions
	info, err = os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file after proxy write: %v", err)
	}
	finalMode := info.Mode()
	t.Logf("Final file permissions: %v", finalMode.Perm())

	// Verify the executable bit is preserved
	if finalMode.Perm()&0111 == 0 {
		t.Error("File lost executable permissions after save through proxy")
	}

	// Verify the content was updated
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}
	if string(content) != modifiedContent {
		t.Errorf("File content not updated correctly.\nExpected: %q\nGot: %q",
			modifiedContent, string(content))
	}

	// Test with non-executable file
	t.Run("NonExecutableFile", func(t *testing.T) {
		nonExecFile := filepath.Join(tempDir, "regular_file.txt")

		// Create non-executable file
		err = os.WriteFile(nonExecFile, []byte("initial"), 0644)
		if err != nil {
			t.Fatalf("Failed to create non-executable file: %v", err)
		}

		// Verify it's not executable
		info, err := os.Stat(nonExecFile)
		if err != nil {
			t.Fatalf("Failed to stat non-executable file: %v", err)
		}
		if info.Mode().Perm()&0111 != 0 {
			t.Fatal("File should not have executable permissions")
		}

		// Write through proxy
		_, _, err = client.WriteFile(nonExecFile, "modified")
		if err != nil {
			t.Fatalf("Failed to write non-executable file: %v", err)
		}

		// Verify it's still not executable
		info, err = os.Stat(nonExecFile)
		if err != nil {
			t.Fatalf("Failed to stat non-executable file after write: %v", err)
		}
		if info.Mode().Perm()&0111 != 0 {
			t.Error("Non-executable file gained executable permissions")
		}
	})
}
