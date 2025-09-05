package editor

import (
	"fmt"
	"strings"
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