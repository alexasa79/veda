package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/events"
	"veda/internal/proxy"
)

// showProjectChanges displays all project changes in a buffer
func (e *Editor) showProjectChanges(args []string) error {
	// Show spinner and status while we fetch and build
	opID := "project-changes"
	e.startAsyncOperation(opID, "loading project changes")
	e.connectionStatus = "Loading project changes"

	go func(workingDir string) {
		defer e.endAsyncOperation(opID)
		defer func() { e.connectionStatus = "" }()

		// Fetch changes
		changes, err := e.proxyManager.GetProjectChanges(workingDir)
		if err != nil {
			msg := fmt.Sprintf("Failed to get project changes: %v", err)
			e.asyncEvents <- events.NewCustomEvent("apply", func(ed *Editor) { ed.ShowError("%s", msg) })
			return
		}

		// Build content off-UI thread
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

			for _, change := range changes {
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
				statusIndicator := statusChar
				if stagedMark != "" {
					statusIndicator = statusChar + stagedMark
				}
				filename := change.File
				if strings.Contains(filename, "/") {
					parts := strings.Split(filename, "/")
					filename = parts[len(parts)-1]
				}
				content.WriteString(fmt.Sprintf("[%s] [%s](%s) (+%d -%d)\n", statusIndicator, filename, change.File, len(change.AddedLines), len(change.DeletedLines)))

				// Pre-fetch file changes to refine anchors
				resolvedPath := e.resolvePath(change.File)
				var fileChanges *proxy.FileChanges
				if fc, err := e.proxyManager.GetFile(resolvedPath); err == nil {
					fileChanges = &fc.Changes
				}

				for _, hunk := range change.Hunks {
					var oldCount, newCount int
					fmt.Sscanf(hunk.Header, "@@ -%d,%d +%d,%d @@", &hunk.OldStart, &oldCount, &hunk.NewStart, &newCount)
					anchorLine := hunk.NewStart
					minChanged := 0
					maxChanged := 0
					if fileChanges != nil {
						inRange := func(v int) bool { return v >= hunk.NewStart && v < hunk.NewStart+newCount }
						for _, v := range fileChanges.Added {
							if inRange(v) {
								if minChanged == 0 || v < minChanged {
									minChanged = v
								}
								if v > maxChanged {
									maxChanged = v
								}
							}
						}
						for _, v := range fileChanges.Modified {
							if inRange(v) {
								if minChanged == 0 || v < minChanged {
									minChanged = v
								}
								if v > maxChanged {
									maxChanged = v
								}
							}
						}
						if minChanged > 0 {
							anchorLine = minChanged
						}
					}
					hunkStart := anchorLine
					hunkEnd := hunk.NewStart + newCount - 1
					if maxChanged > 0 {
						hunkEnd = maxChanged
					}
					hunkDesc := fmt.Sprintf("Lines %d-%d", hunkStart, hunkEnd)
					if hunkStart == hunkEnd {
						hunkDesc = fmt.Sprintf("Line %d", hunkStart)
					}
					content.WriteString(fmt.Sprintf("  • [%s](%s#L%d)", hunkDesc, change.File, anchorLine))
					// Count adds/deletes within hunk
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

		// Apply buffer updates on main thread
		final := content.String()
		e.asyncEvents <- events.NewCustomEvent("apply", func(ed *Editor) {
			// Save current buffer state
			ed.saveCurrentBufferState()
			bufferName := "*Project Changes*"
			if buf, exists := ed.buffers[bufferName]; exists {
				buf.Rope = buffer.NewRopeFromString(final)
				buf.CursorX = 0
				buf.ViewIterator = buffer.NewBufferIterator(buf)
				buf.CursorIterator = buffer.NewBufferIterator(buf)
				buf.IsReadOnly = true
				buf.IsInternal = true
				buf.RealPath = ""
				ed.switchToBuffer(buf)
			} else {
				newBuffer := buffer.NewBuffer(final, bufferName, bufferName, ed.config.DefaultIndentType, ed.config.DefaultIndentWidth)
				newBuffer.IsReadOnly = true
				newBuffer.IsInternal = true
				newBuffer.RealPath = ""
				ed.buffers[bufferName] = newBuffer
				ed.bufferOrder = append(ed.bufferOrder, bufferName)
				ed.switchToBuffer(newBuffer)
			}
			ed.clearSearch()
		})
	}(e.workingDirectory)

	return nil
}
