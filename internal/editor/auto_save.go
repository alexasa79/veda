package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AutoSaveManager handles local auto-save functionality to protect against data loss
type AutoSaveManager struct {
	autoSaveDir   string
	saveInterval  time.Duration
	minEditDelay  time.Duration
	lastSaveTimes map[string]time.Time // Maps file paths to last save times
	pendingSaves  map[string]time.Time // Maps file paths to when they were marked dirty
	mutex         sync.RWMutex
	stopChan      chan struct{}
	done          chan struct{}
	saveCallback  func(string) // Callback to request editor to save a file
}

// NewAutoSaveManager creates a new auto-save manager
func NewAutoSaveManager() *AutoSaveManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		LL.Error("Failed to get user home directory: %v", err)
		homeDir = "/tmp"
	}

	autoSaveDir := filepath.Join(homeDir, ".cache", "veda", "auto-save")

	// Create auto-save directory if it doesn't exist
	if err := os.MkdirAll(autoSaveDir, 0755); err != nil {
		LL.Error("Failed to create auto-save directory: %v", err)
	}

	manager := &AutoSaveManager{
		autoSaveDir:   autoSaveDir,
		saveInterval:  10 * time.Second, // Save every 10 seconds
		minEditDelay:  2 * time.Second,  // Wait at least 2 seconds after last edit
		lastSaveTimes: make(map[string]time.Time),
		pendingSaves:  make(map[string]time.Time),
		stopChan:      make(chan struct{}),
		done:          make(chan struct{}),
	}

	// Start the auto-save loop
	go manager.saveLoop()

	return manager
}

// getAutoSavePath returns the local auto-save path for a file
// Uses the same file naming logic as local history but without timestamps
func (asm *AutoSaveManager) getAutoSavePath(filePath string) string {
	if filePath == "" {
		return ""
	}

	// Parse the file path to determine if it's remote (same logic as local history)
	host, actualPath, isRemote := parsePathComponents(filePath)

	// Build the auto-save filename using the same logic as local history
	var autoSaveFileName string
	if isRemote {
		// For remote files, prefix with hostname
		escapedHost := strings.ReplaceAll(host, ":", "-")
		escapedPath := escapePathForFilename(actualPath)
		autoSaveFileName = fmt.Sprintf("%s-%s.auto-save", escapedHost, escapedPath)
	} else {
		// For local files, use the same escaping as local history
		escapedPath := escapePathForFilename(filePath)
		autoSaveFileName = fmt.Sprintf("%s.auto-save", escapedPath)
	}

	return filepath.Join(asm.autoSaveDir, autoSaveFileName)
}

// Note: We use the same escapePathForFilename function as local history
// It's defined in local_history.go and handles path escaping consistently

// MarkDirty marks a file as having unsaved changes
func (asm *AutoSaveManager) MarkDirty(filePath string) {
	if filePath == "" {
		return
	}

	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	asm.pendingSaves[filePath] = time.Now()
	LL.Debug("Marked file as dirty for auto-save: %s", filePath)
}

// GetPendingDirtyTime returns the time when the file was marked dirty, if pending
func (asm *AutoSaveManager) GetPendingDirtyTime(filePath string) (time.Time, bool) {
	if filePath == "" {
		return time.Time{}, false
	}

	asm.mutex.RLock()
	defer asm.mutex.RUnlock()

	dirtyTime, exists := asm.pendingSaves[filePath]
	return dirtyTime, exists
}

// MarkAutoSaved records that auto-save completed successfully for the given file
func (asm *AutoSaveManager) MarkAutoSaved(filePath string, dirtyTime time.Time) {
	if filePath == "" {
		return
	}

	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	if pendingTime, exists := asm.pendingSaves[filePath]; exists {
		if !dirtyTime.IsZero() && !pendingTime.Equal(dirtyTime) {
			// Changes were made after this auto-save was requested; keep pending entry
			return
		}
		delete(asm.pendingSaves, filePath)
	}

	asm.lastSaveTimes[filePath] = time.Now()
}

// HasAutoSave checks if an auto-save file exists and is newer than the original
func (asm *AutoSaveManager) HasAutoSave(filePath string, originalModTime time.Time) bool {
	if filePath == "" {
		return false
	}

	autoSavePath := asm.getAutoSavePath(filePath)
	if autoSavePath == "" {
		return false
	}

	info, err := os.Stat(autoSavePath)
	if err != nil {
		return false
	}

	// Check if auto-save is newer than original file
	return info.ModTime().After(originalModTime)
}

// GetAutoSaveContent reads the content of an auto-save file
func (asm *AutoSaveManager) GetAutoSaveContent(filePath string) (string, error) {
	autoSavePath := asm.getAutoSavePath(filePath)
	if autoSavePath == "" {
		return "", fmt.Errorf("invalid file path")
	}

	content, err := os.ReadFile(autoSavePath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// saveFile saves content to the auto-save location
func (asm *AutoSaveManager) saveFile(filePath, content string) error {
	autoSavePath := asm.getAutoSavePath(filePath)
	if autoSavePath == "" {
		return fmt.Errorf("invalid file path")
	}

	// Write to a temporary file first, then rename for atomic operation
	tempPath := autoSavePath + ".tmp"

	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return err
	}

	if err := os.Rename(tempPath, autoSavePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return err
	}

	LL.Debug("Auto-saved file: %s -> %s", filePath, autoSavePath)
	return nil
}

// CleanupAutoSave removes the auto-save file (called when file is successfully saved)
func (asm *AutoSaveManager) CleanupAutoSave(filePath string) {
	if filePath == "" {
		return
	}

	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	// Remove from pending saves
	delete(asm.pendingSaves, filePath)
	delete(asm.lastSaveTimes, filePath)

	// Remove auto-save file
	autoSavePath := asm.getAutoSavePath(filePath)
	if autoSavePath != "" {
		if err := os.Remove(autoSavePath); err != nil && !os.IsNotExist(err) {
			LL.Error("Failed to remove auto-save file %s: %v", autoSavePath, err)
		} else {
			LL.Debug("Cleaned up auto-save file: %s", autoSavePath)
		}
	}
}

// saveLoop runs the auto-save background process
func (asm *AutoSaveManager) saveLoop() {
	defer close(asm.done)

	ticker := time.NewTicker(asm.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-asm.stopChan:
			return
		case <-ticker.C:
			asm.processPendingSaves()
		}
	}
}

// processPendingSaves checks for files that need auto-saving
func (asm *AutoSaveManager) processPendingSaves() {
	asm.mutex.Lock()
	now := time.Now()
	filesToSave := make([]string, 0)

	// Find files that are ready to be saved
	for filePath, dirtyTime := range asm.pendingSaves {
		if lastSave, exists := asm.lastSaveTimes[filePath]; exists {
			// Don't save too frequently
			if now.Sub(lastSave) < asm.saveInterval {
				continue
			}
		}

		// Wait for minimum delay since last edit
		if now.Sub(dirtyTime) >= asm.minEditDelay {
			filesToSave = append(filesToSave, filePath)
		}
	}
	asm.mutex.Unlock()

	// Save the files (outside of lock to avoid blocking)
	for _, filePath := range filesToSave {
		asm.requestAutoSave(filePath)
	}
}

// requestAutoSave requests the editor to perform an auto-save for a specific file
func (asm *AutoSaveManager) requestAutoSave(filePath string) {
	// This will be called by the editor to perform the actual save
	if asm.saveCallback != nil {
		asm.saveCallback(filePath)
	}
}

// SetSaveCallback sets the callback function for saving files
func (asm *AutoSaveManager) SetSaveCallback(callback func(string)) {
	asm.saveCallback = callback
}

// Stop stops the auto-save manager
func (asm *AutoSaveManager) Stop() {
	close(asm.stopChan)
	<-asm.done // Wait for saveLoop to finish
}

// GetAutoSaveInfo returns information about auto-save files
func (asm *AutoSaveManager) GetAutoSaveInfo(filePath string) (exists bool, modTime time.Time, err error) {
	autoSavePath := asm.getAutoSavePath(filePath)
	if autoSavePath == "" {
		return false, time.Time{}, nil
	}

	info, err := os.Stat(autoSavePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, err
	}

	return true, info.ModTime(), nil
}
