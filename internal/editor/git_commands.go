package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
)

// gitBlameCurrentLine shows git blame information for the current line
func (e *Editor) gitBlameCurrentLine(args []string) error {
	if e.CurrentBuffer() == nil {
		return nil
	}

	// Get current cursor position
	cursorIter := e.CurrentBuffer().CursorIterator
	if cursorIter == nil {
		return nil
	}

	lineNumber := cursorIter.LineNumber() + 1 // git blame uses 1-based line numbers

	// Get the file path
	filePath := e.CurrentBuffer().FilePath
	if filePath == "" {
		e.ShowTransientInfo("Buffer is not associated with a file")
		return nil
	}

	// Check if it's a remote file
	if host, remotePath, isRemote := parsePathComponents(filePath); isRemote {
		// For remote files, we need to run git blame on the remote host
		result, err := e.proxyManager.RunGitBlame(host, remotePath, lineNumber)
		if err != nil {
			e.ShowTransientInfo("Git blame failed: %v", err)
			return nil
		}
		e.showBlameResult(result)
	} else {
		// For local files, run git blame locally
		result, err := e.proxyManager.RunGitBlame("", filePath, lineNumber)
		if err != nil {
			e.ShowTransientInfo("Git blame failed: %v", err)
			return nil
		}
		e.showBlameResult(result)
	}

	return nil
}

// showBlameResult formats and displays the git blame result
func (e *Editor) showBlameResult(result map[string]interface{}) {
	// Extract blame information from the result
	if success, ok := result["success"].(bool); !ok || !success {
		if errMsg, ok := result["error"].(string); ok {
			e.ShowTransientInfo("Git blame failed: %s", errMsg)
		} else {
			e.ShowTransientInfo("Git blame failed")
		}
		return
	}

	// Extract author, date, and commit message
	author, _ := result["author"].(string)
	date, _ := result["date"].(string)
	message, _ := result["message"].(string)
	commit, _ := result["commit"].(string)

	if author == "" && date == "" && message == "" {
		e.ShowTransientInfo("No git blame information available")
		return
	}

	// Format the blame information
	// Show commit hash (first 7 chars), author, date, and first line of message
	if commit != "" && len(commit) > 7 {
		commit = commit[:7]
	}

	// Get just the first line of the commit message
	if idx := strings.Index(message, "\n"); idx > 0 {
		message = message[:idx]
	}

	// Trim whitespace from message
	message = strings.TrimSpace(message)

	// Format the output
	blameInfo := fmt.Sprintf("%s | %s | %s | %s", commit, author, date, message)
	e.ShowTransientInfo("%s", blameInfo)
}

// gitShowCommit shows the full commit information for the current line in a new buffer
func (e *Editor) gitShowCommit(args []string) error {
	return e.gitShowCommitInternal(false)
}

// gitShowCommitNewWindow shows the full commit information for the current line in a new window
func (e *Editor) gitShowCommitNewWindow(args []string) error {
	return e.gitShowCommitInternal(true)
}

// gitShowCommitInternal is the internal implementation for showing commit info
func (e *Editor) gitShowCommitInternal(openInNewWindow bool) error {
	if e.CurrentBuffer() == nil {
		return nil
	}

	// Get current cursor position
	cursorIter := e.CurrentBuffer().CursorIterator
	if cursorIter == nil {
		return nil
	}

	lineNumber := cursorIter.LineNumber() + 1 // git blame uses 1-based line numbers

	// Get the file path
	filePath := e.CurrentBuffer().FilePath
	if filePath == "" {
		e.ShowError("Buffer is not associated with a file")
		return nil
	}

	// Check if it's a remote file
	var result map[string]interface{}
	var err error
	if host, remotePath, isRemote := parsePathComponents(filePath); isRemote {
		// For remote files, we need to run git commands on the remote host
		result, err = e.proxyManager.GetGitCommitInfo(host, remotePath, lineNumber)
	} else {
		// For local files, run git commands locally
		result, err = e.proxyManager.GetGitCommitInfo("", filePath, lineNumber)
	}

	if err != nil {
		e.ShowError("Failed to get commit info: %v", err)
		return nil
	}

	// Check for success
	if success, ok := result["success"].(bool); !ok || !success {
		if errMsg, ok := result["error"].(string); ok {
			e.ShowError("Git commit info failed: %s", errMsg)
		} else {
			e.ShowError("Git commit info failed")
		}
		return nil
	}

	// Extract commit info and full content
	fullContent, _ := result["full_content"].(string)
	commitInfo, _ := result["commit_info"].(map[string]interface{})

	if fullContent == "" {
		e.ShowError("No commit information available")
		return nil
	}

	// Create buffer name based on commit hash
	bufferName := "*Git Commit*"
	if commitInfo != nil {
		if hash, ok := commitInfo["hash"].(string); ok && len(hash) >= 7 {
			bufferName = fmt.Sprintf("*Git Commit %s*", hash[:7])
		}
	}

	// Save current buffer state before switching
	e.saveCurrentBufferState()

	// Create or update the commit buffer
	var commitBuffer *buffer.Buffer
	if buf, exists := e.buffers[bufferName]; exists {
		// Update existing buffer
		buf.Rope = buffer.NewRopeFromString(fullContent)
		buf.CursorX = 0
		buf.ViewIterator = buffer.NewBufferIterator(buf)
		buf.CursorIterator = buffer.NewBufferIterator(buf)
		commitBuffer = buf
	} else {
		// Create new buffer
		commitBuffer = buffer.NewBuffer(fullContent, bufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		commitBuffer.IsReadOnly = true
		commitBuffer.IsInternal = true
		e.buffers[bufferName] = commitBuffer
		e.bufferOrder = append(e.bufferOrder, bufferName)
	}

	// Switch to the commit buffer
	if openInNewWindow {
		// Create a new window at the far right
		e.createRightmostWindow(e.CurrentBuffer())
		// Switch to the commit buffer in the new window
		e.switchToBuffer(commitBuffer)
	} else {
		// Switch in current window
		e.switchToBuffer(commitBuffer)
	}
	
	// Clear any existing search
	e.clearSearch()

	return nil
}