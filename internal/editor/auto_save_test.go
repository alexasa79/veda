package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAutoSavePathGeneration(t *testing.T) {
	// Create a temporary auto-save manager for testing
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	manager := &AutoSaveManager{
		autoSaveDir: filepath.Join(homeDir, ".cache", "veda", "auto-save"),
	}

	// NOTE: In actual usage, file paths should be resolved to absolute paths
	// before being passed to the auto-save system (via editor.resolvePath())

	tests := []struct {
		name           string
		inputPath      string
		expectedSuffix string
		shouldBeHidden bool
	}{
		{
			name:           "regular local file",
			inputPath:      "/home/user/project/main.go",
			expectedSuffix: ".auto-save",
			shouldBeHidden: false,
		},
		{
			name:           "hidden file with dot prefix",
			inputPath:      "/home/user/.gitignore",
			expectedSuffix: ".auto-save",
			shouldBeHidden: false, // Absolute paths don't create hidden files
		},
		{
			name:           "config file",
			inputPath:      "/home/user/.config/settings.json",
			expectedSuffix: ".auto-save",
			shouldBeHidden: false, // Absolute paths don't create hidden files
		},
		{
			name:           "remote file",
			inputPath:      "server.com:/home/user/file.go",
			expectedSuffix: ".auto-save",
			shouldBeHidden: false,
		},
		{
			name:           "remote hidden file",
			inputPath:      "server.com:/home/user/.bashrc",
			expectedSuffix: ".auto-save",
			shouldBeHidden: false, // Remote files have hostname prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			autoSavePath := manager.getAutoSavePath(tt.inputPath)

			// Check that the path is not empty
			if autoSavePath == "" {
				t.Errorf("getAutoSavePath returned empty path for %s", tt.inputPath)
				return
			}

			// Extract the filename
			filename := filepath.Base(autoSavePath)

			// Check if it ends with the expected suffix
			if !strings.HasSuffix(filename, tt.expectedSuffix) {
				t.Errorf("Expected filename to end with %s, got: %s", tt.expectedSuffix, filename)
			}

			// Check if the file would be hidden (starts with .)
			isHidden := strings.HasPrefix(filename, ".")
			if isHidden != tt.shouldBeHidden {
				if isHidden {
					t.Errorf("Auto-save file %s would be hidden (starts with .), this is unexpected for path: %s", filename, tt.inputPath)
				} else if tt.shouldBeHidden {
					t.Errorf("Expected auto-save file %s to be hidden for path: %s, but it's not", filename, tt.inputPath)
				}
			}

			// Verify the path contains the auto-save directory
			if !strings.Contains(autoSavePath, "auto-save") {
				t.Errorf("Auto-save path should contain 'auto-save' directory: %s", autoSavePath)
			}

			t.Logf("Input: %s -> Auto-save: %s", tt.inputPath, filename)
		})
	}
}

// TestAutoSaveEscapePathForFilename removed - we now use the same function as local_history.go

func TestAutoSaveFileNamingConsistency(t *testing.T) {
	manager := &AutoSaveManager{
		autoSaveDir: "/test/auto-save",
	}

	// Test that local and remote versions of the same file have different names
	localPath := "/home/user/project/main.go"
	remotePath := "server.com:/home/user/project/main.go"

	localAutoSave := manager.getAutoSavePath(localPath)
	remoteAutoSave := manager.getAutoSavePath(remotePath)

	localFilename := filepath.Base(localAutoSave)
	remoteFilename := filepath.Base(remoteAutoSave)

	if localFilename == remoteFilename {
		t.Errorf("Local and remote auto-save files should have different names: local=%s, remote=%s", localFilename, remoteFilename)
	}

	// Both should end with .auto-save
	if !strings.HasSuffix(localFilename, ".auto-save") {
		t.Errorf("Local auto-save filename should end with .auto-save: %s", localFilename)
	}
	if !strings.HasSuffix(remoteFilename, ".auto-save") {
		t.Errorf("Remote auto-save filename should end with .auto-save: %s", remoteFilename)
	}

	// Remote should have hostname prefix
	if !strings.HasPrefix(remoteFilename, "server.com-") {
		t.Errorf("Remote auto-save filename should start with hostname: %s", remoteFilename)
	}

	t.Logf("Local: %s", localFilename)
	t.Logf("Remote: %s", remoteFilename)
}

func TestAutoSaveCleanupOnBufferClose(t *testing.T) {
	// Create a temporary auto-save directory for testing
	tempDir := t.TempDir()

	manager := &AutoSaveManager{
		autoSaveDir:   tempDir,
		lastSaveTimes: make(map[string]time.Time),
		pendingSaves:  make(map[string]time.Time),
	}

	// Test file paths
	testFiles := []string{
		"/home/user/test.txt",
		"/home/user/.hidden",
		"server.com:/remote/file.go",
	}

	// Create auto-save files for each test file
	for _, filePath := range testFiles {
		autoSavePath := manager.getAutoSavePath(filePath)
		content := fmt.Sprintf("Auto-save content for %s", filePath)

		// Create the auto-save file
		err := os.WriteFile(autoSavePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create auto-save file %s: %v", autoSavePath, err)
		}

		// Mark as dirty to simulate pending changes
		manager.MarkDirty(filePath)

		// Verify the file exists
		if _, err := os.Stat(autoSavePath); os.IsNotExist(err) {
			t.Errorf("Auto-save file should exist before cleanup: %s", autoSavePath)
		}
	}

	// Test cleanup for each file
	for _, filePath := range testFiles {
		autoSavePath := manager.getAutoSavePath(filePath)

		// Clean up the auto-save file
		manager.CleanupAutoSave(filePath)

		// Verify the file is deleted
		if _, err := os.Stat(autoSavePath); !os.IsNotExist(err) {
			t.Errorf("Auto-save file should be deleted after cleanup: %s", autoSavePath)
		}

		// Verify it's removed from pending saves
		manager.mutex.RLock()
		if _, exists := manager.pendingSaves[filePath]; exists {
			t.Errorf("File should be removed from pendingSaves after cleanup: %s", filePath)
		}
		if _, exists := manager.lastSaveTimes[filePath]; exists {
			t.Errorf("File should be removed from lastSaveTimes after cleanup: %s", filePath)
		}
		manager.mutex.RUnlock()
	}
}

func TestAutoSaveCleanupOnSuccessfulSave(t *testing.T) {
	// This tests the cleanup that should happen when a file is successfully saved
	// via the writeFile function
	tempDir := t.TempDir()

	manager := &AutoSaveManager{
		autoSaveDir:   tempDir,
		lastSaveTimes: make(map[string]time.Time),
		pendingSaves:  make(map[string]time.Time),
	}

	filePath := "/home/user/test.txt"
	autoSavePath := manager.getAutoSavePath(filePath)

	// Create an auto-save file
	content := "Test content with unsaved changes"
	err := os.WriteFile(autoSavePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create auto-save file: %v", err)
	}

	// Mark as dirty
	manager.MarkDirty(filePath)

	// Verify the auto-save file exists
	if _, err := os.Stat(autoSavePath); os.IsNotExist(err) {
		t.Fatal("Auto-save file should exist before cleanup")
	}

	// Simulate successful save (this is what writeFile does)
	manager.CleanupAutoSave(filePath)

	// Verify cleanup
	if _, err := os.Stat(autoSavePath); !os.IsNotExist(err) {
		t.Error("Auto-save file should be deleted after successful save")
	}

	// Verify tracking is cleaned up
	manager.mutex.RLock()
	if _, exists := manager.pendingSaves[filePath]; exists {
		t.Error("File should be removed from pendingSaves after successful save")
	}
	if _, exists := manager.lastSaveTimes[filePath]; exists {
		t.Error("File should be removed from lastSaveTimes after successful save")
	}
	manager.mutex.RUnlock()
}

func TestAutoSaveCleanupOnAllQuitOperations(t *testing.T) {
	// This tests that ALL quit operations clean up auto-save files
	// The goal is that when the user quits without saving, they don't want to be
	// prompted about recovery next time they open the file.
	tempDir := t.TempDir()

	manager := &AutoSaveManager{
		autoSaveDir:   tempDir,
		lastSaveTimes: make(map[string]time.Time),
		pendingSaves:  make(map[string]time.Time),
	}

	testFiles := []string{
		"/home/user/file1.txt",
		"/home/user/file2.go",
		"/home/user/.config",
	}

	// Create auto-save files for multiple files (simulating multiple open buffers)
	for _, filePath := range testFiles {
		autoSavePath := manager.getAutoSavePath(filePath)
		content := fmt.Sprintf("Unsaved changes for %s", filePath)

		// Create the auto-save file
		err := os.WriteFile(autoSavePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create auto-save file %s: %v", autoSavePath, err)
		}

		// Mark as dirty
		manager.MarkDirty(filePath)
	}

	// Verify all auto-save files exist before cleanup
	for _, filePath := range testFiles {
		autoSavePath := manager.getAutoSavePath(filePath)
		if _, err := os.Stat(autoSavePath); os.IsNotExist(err) {
			t.Errorf("Auto-save file should exist before cleanup: %s", autoSavePath)
		}
	}

	// Simulate force quit cleaning up ALL auto-save files
	// This is what actuallyQuit() does when user uses :Q, :qa, :q!, etc.
	for _, filePath := range testFiles {
		manager.CleanupAutoSave(filePath)
	}

	// Verify ALL auto-save files are deleted
	for _, filePath := range testFiles {
		autoSavePath := manager.getAutoSavePath(filePath)
		if _, err := os.Stat(autoSavePath); !os.IsNotExist(err) {
			t.Errorf("Auto-save file should be deleted after force quit: %s", autoSavePath)
		}
	}

	// Verify all tracking is cleaned up
	manager.mutex.RLock()
	if len(manager.pendingSaves) != 0 {
		t.Errorf("All files should be removed from pendingSaves after force quit, but %d remain", len(manager.pendingSaves))
	}
	if len(manager.lastSaveTimes) != 0 {
		t.Errorf("All files should be removed from lastSaveTimes after force quit, but %d remain", len(manager.lastSaveTimes))
	}
	manager.mutex.RUnlock()

	t.Log("SUCCESS: All auto-save files cleaned up on quit - user won't be prompted for recovery")
}
