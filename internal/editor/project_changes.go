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
			content.WriteString(fmt.Sprintf("Found changes in %d files.\n\n", len(changes)))

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
				content.WriteString(fmt.Sprintf("## %s [%s]\n", filename, statusIndicator))

				// Pre-fetch file changes to link the diff to the first changed line.
				resolvedPath := e.resolvePath(change.File)
				var fileChanges *proxy.FileChanges
				if fc, err := e.proxyManager.GetFile(resolvedPath); err == nil {
					fileChanges = &fc.Changes
				}

				anchorLine := 1
				if len(change.Hunks) > 0 {
					anchorLine = change.Hunks[0].NewStart
				}
				if fileChanges != nil {
					firstChangedLine := 0
					for _, lines := range [][]int{fileChanges.Added, fileChanges.Modified} {
						for _, line := range lines {
							if line > 0 && (firstChangedLine == 0 || line < firstChangedLine) {
								firstChangedLine = line
							}
						}
					}
					if firstChangedLine > 0 {
						anchorLine = firstChangedLine
					}
				}
				if anchorLine < 1 {
					anchorLine = 1
				}

				content.WriteString(fmt.Sprintf("[Diff](%s#diff#L%d)\n", change.File, anchorLine))
				content.WriteString(fmt.Sprintf("[File](%s)\n", change.File))
				hunkLabel := "hunks"
				if len(change.Hunks) == 1 {
					hunkLabel = "hunk"
				}
				content.WriteString(fmt.Sprintf("%d %s, +%d -%d", len(change.Hunks), hunkLabel, len(change.AddedLines), len(change.DeletedLines)))
				if fileChanges != nil && len(fileChanges.Modified) > 0 {
					content.WriteString(fmt.Sprintf(", %d modified", len(fileChanges.Modified)))
				}
				content.WriteString("\n\n")
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
