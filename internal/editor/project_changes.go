package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
)

// showProjectChanges displays all project changes in a buffer
func (e *Editor) showProjectChanges(args []string) error {
	// Get project changes from the proxy
	changes, err := e.proxyManager.GetProjectChanges(e.workingDirectory)
	if err != nil {
		LL.Error("Failed to get project changes: %v", err)
		return fmt.Errorf("failed to get project changes: %w", err)
	}

	// Create content for changes buffer
	var content strings.Builder
	content.WriteString("# Project Changes\n\n")

	if len(changes) == 0 {
		content.WriteString("No changes found.\n")
	} else {
		totalFiles := len(changes)
		totalAdded := 0
		totalDeleted := 0
		
		for _, change := range changes {
			totalAdded += len(change.AddedLines)
			totalDeleted += len(change.DeletedLines)
		}
		
		content.WriteString(fmt.Sprintf("Found changes in %d files: +%d -%d\n\n", totalFiles, totalAdded, totalDeleted))

		// Process each file's changes
		for _, change := range changes {
			// Format: [M] filename.go (+10 -5)
			statusChar := "?"
			switch change.Status {
			case "modified", "M":
				statusChar = "M"
			case "added", "A":
				statusChar = "A"
			case "deleted", "D":  
				statusChar = "D"
			case "renamed", "R":
				statusChar = "R"
			}
			
			stagedMark := ""
			if change.Staged {
				stagedMark = "●"
			}
			
			// Create clickable file link with clear status indicator
			statusIndicator := statusChar
			if stagedMark != "" {
				statusIndicator = statusChar + stagedMark
			}
			content.WriteString(fmt.Sprintf("[%s] [%s](%s) (+%d -%d)\n", 
				statusIndicator, change.File, change.File,
				len(change.AddedLines), len(change.DeletedLines)))
			
			// Show hunks with better formatting
			for _, hunk := range change.Hunks {
				// Parse the hunk header to get line counts
				var oldCount, newCount int
				fmt.Sscanf(hunk.Header, "@@ -%d,%d +%d,%d @@", &hunk.OldStart, &oldCount, &hunk.NewStart, &newCount)
				
				// Create a more descriptive hunk link
				hunkDesc := fmt.Sprintf("Lines %d-%d", hunk.NewStart, hunk.NewStart+newCount-1)
				if newCount == 1 {
					hunkDesc = fmt.Sprintf("Line %d", hunk.NewStart)
				}
				
				content.WriteString(fmt.Sprintf("  • [%s](%s#L%d)", hunkDesc, change.File, hunk.NewStart))
				
				// Count additions and deletions in this hunk
				hunkAdded := 0
				hunkDeleted := 0
				for _, line := range change.AddedLines {
					if line >= hunk.NewStart && line < hunk.NewStart+newCount {
						hunkAdded++
					}
				}
				for _, line := range change.DeletedLines {
					if line >= hunk.OldStart && line < hunk.OldStart+oldCount {
						hunkDeleted++
					}
				}
				
				if hunkAdded > 0 || hunkDeleted > 0 {
					content.WriteString(fmt.Sprintf(" (+%d -%d)", hunkAdded, hunkDeleted))
				}
				content.WriteString("\n")
			}
			
			content.WriteString("\n")
		}
	}

	// Save current buffer state
	e.saveCurrentBufferState()
	
	// Create or update changes buffer
	bufferName := "*Project Changes*"
	
	if buf, exists := e.buffers[bufferName]; exists {
		// Update existing buffer's content
		buf.Rope = buffer.NewRopeFromString(content.String())
		buf.CursorX = 0
		buf.ViewIterator = buffer.NewBufferIterator(buf)
		buf.CursorIterator = buffer.NewBufferIterator(buf)
		buf.IsReadOnly = true
		buf.IsInternal = true
		e.switchToBuffer(buf)
	} else {
		// Create new buffer
		newBuffer := buffer.NewBuffer(content.String(), bufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		newBuffer.IsReadOnly = true
		newBuffer.IsInternal = true
		e.buffers[bufferName] = newBuffer
		e.bufferOrder = append(e.bufferOrder, bufferName)
		e.switchToBuffer(newBuffer)
	}

	// Clear any existing search
	e.clearSearch()

	return nil
}