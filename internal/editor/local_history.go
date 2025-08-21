package editor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalHistory manages saving file history locally
type LocalHistory struct {
	baseDir string
}

// NewLocalHistory creates a new LocalHistory instance
func NewLocalHistory() (*LocalHistory, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".cache", "veda", "history")
	return &LocalHistory{
		baseDir: baseDir,
	}, nil
}

// SaveFileHistory saves a copy of the file to the local history
func (lh *LocalHistory) SaveFileHistory(filePath string, content string) error {
	// Get current date for directory name
	now := time.Now()
	dateDir := now.Format("2006-01-02")

	// Create history directory for today
	historyDir := filepath.Join(lh.baseDir, dateDir)
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Parse the file path to determine if it's remote
	// Note: parsePathComponents is defined in editor.go
	host, actualPath, isRemote := parsePathComponents(filePath)

	// Build the history filename
	var historyFileName string
	if isRemote {
		// For remote files, prefix with hostname
		escapedHost := strings.ReplaceAll(host, ":", "-")
		escapedPath := escapePathForFilename(actualPath)
		historyFileName = fmt.Sprintf("%s-%s-%s", escapedHost, escapedPath, now.Format("2006-01-02-15-04-05"))
	} else {
		// For local files
		escapedPath := escapePathForFilename(filePath)
		historyFileName = fmt.Sprintf("%s-%s", escapedPath, now.Format("2006-01-02-15-04-05"))
	}

	// Full path to history file
	historyFilePath := filepath.Join(historyDir, historyFileName)

	// Write the content to the history file
	if err := os.WriteFile(historyFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	LL.Debug("Saved file history: %s -> %s", filePath, historyFilePath)
	return nil
}

// escapePathForFilename converts a file path to a filename-safe string
func escapePathForFilename(path string) string {
	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	// Replace path separators with dashes
	escaped := strings.ReplaceAll(path, "/", "-")

	// Also escape other problematic characters
	escaped = strings.ReplaceAll(escaped, ":", "-")
	escaped = strings.ReplaceAll(escaped, " ", "_")
	escaped = strings.ReplaceAll(escaped, "\\", "-")

	return escaped
}

// GetHistoryFiles returns a list of history files for a given file path
func (lh *LocalHistory) GetHistoryFiles(filePath string) ([]string, error) {
	// Parse the file path
	host, actualPath, isRemote := parsePathComponents(filePath)

	// Build the prefix to search for
	var prefix string
	if isRemote {
		escapedHost := strings.ReplaceAll(host, ":", "-")
		escapedPath := escapePathForFilename(actualPath)
		prefix = fmt.Sprintf("%s-%s-", escapedHost, escapedPath)
	} else {
		escapedPath := escapePathForFilename(filePath)
		prefix = fmt.Sprintf("%s-", escapedPath)
	}

	var historyFiles []string

	// Walk through all date directories
	err := filepath.Walk(lh.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() && strings.HasPrefix(info.Name(), prefix) {
			historyFiles = append(historyFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk history directory: %w", err)
	}

	return historyFiles, nil
}

// CleanupOldHistory removes history files older than the specified number of days
func (lh *LocalHistory) CleanupOldHistory(daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	// Walk through date directories
	entries, err := os.ReadDir(lh.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to clean up
		}
		return fmt.Errorf("failed to read history directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse date from directory name
		dirDate, err := time.Parse("2006-01-02", entry.Name())
		if err != nil {
			continue // Skip directories that don't match our date format
		}

		// Remove if older than cutoff
		if dirDate.Before(cutoffDate) {
			dirPath := filepath.Join(lh.baseDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				LL.Error("Failed to remove old history directory %s: %v", dirPath, err)
			} else {
				LL.Debug("Removed old history directory: %s", dirPath)
			}
		}
	}

	return nil
}

// RestoreFromHistory restores a file from a specific history entry
func (lh *LocalHistory) RestoreFromHistory(historyFilePath string, targetPath string) error {
	// Read the history file
	content, err := os.ReadFile(historyFilePath)
	if err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}

	// Write to target path
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write restored file: %w", err)
	}

	return nil
}

// CopyHistoryFile creates a copy of a history file (useful for manual recovery)
func (lh *LocalHistory) CopyHistoryFile(historyFilePath string, destPath string) error {
	src, err := os.Open(historyFilePath)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
