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
	minEditDelay  time.Duration
	lastSaveTimes map[string]time.Time // Maps file paths to last save times
	pendingSaves  map[string]time.Time // Maps file paths to when they were marked dirty
	timers        map[string]*time.Timer
	mutex         sync.RWMutex
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

	return &AutoSaveManager{
		autoSaveDir:   autoSaveDir,
		minEditDelay:  2 * time.Second,
		lastSaveTimes: make(map[string]time.Time),
		pendingSaves:  make(map[string]time.Time),
		timers:        make(map[string]*time.Timer),
	}
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

// scheduleAutoSaveLocked schedules an auto-save timer for the given file.
func (asm *AutoSaveManager) scheduleAutoSaveLocked(filePath string, dirtyTime time.Time) {
	if asm.timers == nil {
		asm.timers = make(map[string]*time.Timer)
	}

	if timer, exists := asm.timers[filePath]; exists {
		timer.Stop()
		delete(asm.timers, filePath)
	}

	delay := asm.minEditDelay - time.Since(dirtyTime)
	if delay < 0 {
		delay = 0
	}

	var timer *time.Timer
	timer = time.AfterFunc(delay, func() {
		asm.handleAutoSaveTimer(filePath, dirtyTime, timer)
	})

	asm.timers[filePath] = timer
}

// handleAutoSaveTimer is invoked when the per-file timer fires.
func (asm *AutoSaveManager) handleAutoSaveTimer(filePath string, dirtyTime time.Time, timer *time.Timer) {
	asm.mutex.Lock()
	currentTimer, timerExists := asm.timers[filePath]
	if timerExists && currentTimer == timer {
		delete(asm.timers, filePath)
	}

	pendingTime, pendingExists := asm.pendingSaves[filePath]
	asm.mutex.Unlock()

	if !pendingExists || !pendingTime.Equal(dirtyTime) {
		return
	}

	asm.requestAutoSave(filePath)
}

// cancelTimerLocked stops and removes an active timer for the given file.
func (asm *AutoSaveManager) cancelTimerLocked(filePath string) {
	if asm.timers == nil {
		return
	}
	if timer, exists := asm.timers[filePath]; exists {
		timer.Stop()
		delete(asm.timers, filePath)
	}
}

// MarkDirty marks a file as having unsaved changes
func (asm *AutoSaveManager) MarkDirty(filePath string) {
	if filePath == "" {
		return
	}

	now := time.Now()

	asm.mutex.Lock()
	if asm.pendingSaves == nil {
		asm.pendingSaves = make(map[string]time.Time)
	}
	asm.pendingSaves[filePath] = now
	asm.scheduleAutoSaveLocked(filePath, now)
	asm.mutex.Unlock()

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

	if asm.pendingSaves != nil {
		if pendingTime, exists := asm.pendingSaves[filePath]; exists {
			if !dirtyTime.IsZero() && !pendingTime.Equal(dirtyTime) {
				// Changes were made after this auto-save was requested; keep pending entry
				return
			}
			delete(asm.pendingSaves, filePath)
		}
	}

	asm.cancelTimerLocked(filePath)
	if asm.lastSaveTimes == nil {
		asm.lastSaveTimes = make(map[string]time.Time)
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

	asm.cancelTimerLocked(filePath)
	if asm.pendingSaves != nil {
		delete(asm.pendingSaves, filePath)
	}
	if asm.lastSaveTimes != nil {
		delete(asm.lastSaveTimes, filePath)
	}

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

// requestAutoSave requests the editor to perform an auto-save for a specific file
func (asm *AutoSaveManager) requestAutoSave(filePath string) {
	asm.mutex.RLock()
	callback := asm.saveCallback
	asm.mutex.RUnlock()
	if callback != nil {
		callback(filePath)
	}
}

// SetSaveCallback sets the callback function for saving files
func (asm *AutoSaveManager) SetSaveCallback(callback func(string)) {
	asm.mutex.Lock()
	defer asm.mutex.Unlock()
	asm.saveCallback = callback
}

// Stop stops the auto-save manager
func (asm *AutoSaveManager) Stop() {
	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	for path, timer := range asm.timers {
		timer.Stop()
		delete(asm.timers, path)
	}

	asm.saveCallback = nil
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
