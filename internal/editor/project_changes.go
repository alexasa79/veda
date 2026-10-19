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
				// Pre-fetch file changes to link the diff to the first changed line.
				resolvedPath := e.resolvePath(change.File)
				var fileChanges *proxy.FileChanges
				if fc, err := e.proxyManager.GetFile(resolvedPath, false); err == nil {
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

				modified := 0
				if fileChanges != nil {
					modified = len(fileChanges.Modified)
				}
				writeChangeRow(&content, changeRow{
					Path:     change.RelativePath,
					DiffLink: fmt.Sprintf("%s#diff#L%d", change.File, anchorLine),
					FileLink: change.File, Status: change.Status, Staged: change.Staged,
					Added: len(change.AddedLines), Deleted: len(change.DeletedLines), Modified: modified,
				})
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
