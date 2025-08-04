package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEscapePathForFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple_path",
			input:    "/home/user/file.txt",
			expected: "home-user-file.txt",
		},
		{
			name:     "path_with_spaces",
			input:    "/home/user name/my file.txt",
			expected: "home-user_name-my_file.txt",
		},
		{
			name:     "windows_style_path",
			input:    "C:\\Users\\Name\\file.txt",
			expected: "C--Users-Name-file.txt",
		},
		{
			name:     "path_with_colons",
			input:    "/home/user:name/file.txt",
			expected: "home-user-name-file.txt",
		},
		{
			name:     "no_leading_slash",
			input:    "home/user/file.txt",
			expected: "home-user-file.txt",
		},
		{
			name:     "TODO_md_example",
			input:    "/Users/sasandl/hobbies/veda/TODO.md",
			expected: "Users-sasandl-hobbies-veda-TODO.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapePathForFilename(tt.input)
			if result != tt.expected {
				t.Errorf("escapePathForFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLocalHistory_SaveFileHistory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create LocalHistory with custom base directory
	lh := &LocalHistory{
		baseDir: tempDir,
	}
	
	// Test data
	testFile := "/home/user/test.txt"
	testContent := "Hello, World!"
	
	// Save file history
	err := lh.SaveFileHistory(testFile, testContent)
	if err != nil {
		t.Fatalf("SaveFileHistory failed: %v", err)
	}
	
	// Verify directory structure
	dateDir := time.Now().Format("2006-01-02")
	historyDir := filepath.Join(tempDir, dateDir)
	
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		t.Errorf("History directory not created: %s", historyDir)
	}
	
	// Verify file was created
	files, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("Failed to read history directory: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 file in history directory, got %d", len(files))
	}
	
	// Verify filename format
	if len(files) > 0 {
		filename := files[0].Name()
		expectedPrefix := "home-user-test.txt-" + dateDir
		if !strings.HasPrefix(filename, expectedPrefix) {
			t.Errorf("Filename %q doesn't start with expected prefix %q", filename, expectedPrefix)
		}
	}
	
	// Verify file content
	if len(files) > 0 {
		filePath := filepath.Join(historyDir, files[0].Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read history file: %v", err)
		}
		
		if string(content) != testContent {
			t.Errorf("History file content = %q, want %q", string(content), testContent)
		}
	}
}

func TestLocalHistory_SaveFileHistory_Remote(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create LocalHistory with custom base directory
	lh := &LocalHistory{
		baseDir: tempDir,
	}
	
	// Test data for remote file
	testFile := "server.example.com:/home/user/test.txt"
	testContent := "Remote file content"
	
	// Save file history
	err := lh.SaveFileHistory(testFile, testContent)
	if err != nil {
		t.Fatalf("SaveFileHistory failed: %v", err)
	}
	
	// Verify file was created with hostname prefix
	dateDir := time.Now().Format("2006-01-02")
	historyDir := filepath.Join(tempDir, dateDir)
	
	files, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("Failed to read history directory: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 file in history directory, got %d", len(files))
	}
	
	// Verify filename includes hostname
	if len(files) > 0 {
		filename := files[0].Name()
		expectedPrefix := "server.example.com-home-user-test.txt-" + dateDir
		if !strings.HasPrefix(filename, expectedPrefix) {
			t.Errorf("Filename %q doesn't start with expected prefix %q", filename, expectedPrefix)
		}
	}
}

func TestLocalHistory_GetHistoryFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create LocalHistory with custom base directory
	lh := &LocalHistory{
		baseDir: tempDir,
	}
	
	// Create some history files manually
	dateDir := time.Now().Format("2006-01-02")
	historyDir := filepath.Join(tempDir, dateDir)
	os.MkdirAll(historyDir, 0755)
	
	// Create test history files
	testFile := "/home/user/test.txt"
	escapedPath := escapePathForFilename(testFile)
	
	// Create multiple history entries
	times := []string{"10-00-00", "11-00-00", "12-00-00"}
	for _, timeStr := range times {
		filename := fmt.Sprintf("%s-%s-%s", escapedPath, dateDir, timeStr)
		filePath := filepath.Join(historyDir, filename)
		os.WriteFile(filePath, []byte("content"), 0644)
	}
	
	// Also create a file for a different path
	otherFile := fmt.Sprintf("home-user-other.txt-%s-13-00-00", dateDir)
	os.WriteFile(filepath.Join(historyDir, otherFile), []byte("other"), 0644)
	
	// Get history files
	historyFiles, err := lh.GetHistoryFiles(testFile)
	if err != nil {
		t.Fatalf("GetHistoryFiles failed: %v", err)
	}
	
	// Should find 3 files for test.txt
	if len(historyFiles) != 3 {
		t.Errorf("Expected 3 history files, got %d", len(historyFiles))
	}
	
	// Verify all files are for the correct path
	for _, file := range historyFiles {
		if !strings.Contains(file, escapedPath) {
			t.Errorf("History file %q doesn't contain expected path %q", file, escapedPath)
		}
	}
}

func TestLocalHistory_CleanupOldHistory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create LocalHistory with custom base directory
	lh := &LocalHistory{
		baseDir: tempDir,
	}
	
	// Create directories for different dates
	today := time.Now()
	oldDate := today.AddDate(0, 0, -10) // 10 days ago
	recentDate := today.AddDate(0, 0, -3) // 3 days ago
	
	oldDir := filepath.Join(tempDir, oldDate.Format("2006-01-02"))
	recentDir := filepath.Join(tempDir, recentDate.Format("2006-01-02"))
	todayDir := filepath.Join(tempDir, today.Format("2006-01-02"))
	
	// Create directories and files
	for _, dir := range []string{oldDir, recentDir, todayDir} {
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "test.txt"), []byte("content"), 0644)
	}
	
	// Also create a non-date directory that should be ignored
	os.MkdirAll(filepath.Join(tempDir, "not-a-date"), 0755)
	
	// Cleanup history older than 7 days
	err := lh.CleanupOldHistory(7)
	if err != nil {
		t.Fatalf("CleanupOldHistory failed: %v", err)
	}
	
	// Verify old directory was removed
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("Old directory should have been removed")
	}
	
	// Verify recent and today directories still exist
	if _, err := os.Stat(recentDir); os.IsNotExist(err) {
		t.Error("Recent directory should still exist")
	}
	
	if _, err := os.Stat(todayDir); os.IsNotExist(err) {
		t.Error("Today's directory should still exist")
	}
	
	// Verify non-date directory was not touched
	if _, err := os.Stat(filepath.Join(tempDir, "not-a-date")); os.IsNotExist(err) {
		t.Error("Non-date directory should not have been removed")
	}
}

func TestLocalHistory_RestoreFromHistory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create LocalHistory with custom base directory
	lh := &LocalHistory{
		baseDir: tempDir,
	}
	
	// Create a history file
	historyContent := "This is the historical content"
	historyFile := filepath.Join(tempDir, "history.txt")
	os.WriteFile(historyFile, []byte(historyContent), 0644)
	
	// Restore to a new location
	targetFile := filepath.Join(tempDir, "restored.txt")
	err := lh.RestoreFromHistory(historyFile, targetFile)
	if err != nil {
		t.Fatalf("RestoreFromHistory failed: %v", err)
	}
	
	// Verify restored content
	restoredContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}
	
	if string(restoredContent) != historyContent {
		t.Errorf("Restored content = %q, want %q", string(restoredContent), historyContent)
	}
}