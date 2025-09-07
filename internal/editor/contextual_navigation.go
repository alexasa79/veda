package editor

import (
	"fmt"
	"sort"
	"strings"
	"veda/internal/buffer"
)

// getNavigationContext returns a string describing what g n/g p will navigate
func (e *Editor) getNavigationContext() string {
	// Check if we have an active special buffer
	if e.lastActiveSpecialBuffer != "" {
		// Verify it still exists
		if _, exists := e.buffers[e.lastActiveSpecialBuffer]; exists {
			if strings.HasPrefix(e.lastActiveSpecialBuffer, "*References:") {
				if e.currentReferencesBuffer != nil && len(e.currentReferencesBuffer.References) > 0 {
					return fmt.Sprintf("refs:%s", e.currentReferencesBuffer.SymbolName)
				}
			} else if e.lastActiveSpecialBuffer == "*Diagnostics*" {
				return "diagnostics"
			} else if e.lastActiveSpecialBuffer == "*Project Changes*" {
				return "changes"
			}
		}
	}

	// Don't show git changes in status bar - g n/g p will silently navigate them
	return ""
}

// nextContextualItem jumps to the next item based on active context
func (e *Editor) nextContextualItem(args []string) error {
	// Check if we have an active special buffer
	if e.lastActiveSpecialBuffer != "" {
		// Verify it still exists
		if _, exists := e.buffers[e.lastActiveSpecialBuffer]; exists {
			if strings.HasPrefix(e.lastActiveSpecialBuffer, "*References:") {
				// Navigate references
				if e.currentReferencesBuffer != nil && len(e.currentReferencesBuffer.References) > 0 {
					return e.nextReference(args)
				}
			} else if e.lastActiveSpecialBuffer == "*Diagnostics*" {
				// Navigate to next diagnostic in *Diagnostics* buffer
				return e.nextDiagnosticInBuffer()
			} else if e.lastActiveSpecialBuffer == "*Project Changes*" {
				// Navigate to next change in *Project Changes* buffer
				return e.nextChangeInProjectBuffer()
			}
		}
	}

	// No special buffer active, navigate git changes in current file
	return e.nextGitChange()
}

// previousContextualItem jumps to the previous item based on active context
func (e *Editor) previousContextualItem(args []string) error {
	// Check if we have an active special buffer
	if e.lastActiveSpecialBuffer != "" {
		// Verify it still exists
		if _, exists := e.buffers[e.lastActiveSpecialBuffer]; exists {
			if strings.HasPrefix(e.lastActiveSpecialBuffer, "*References:") {
				// Navigate references
				if e.currentReferencesBuffer != nil && len(e.currentReferencesBuffer.References) > 0 {
					return e.previousReference(args)
				}
			} else if e.lastActiveSpecialBuffer == "*Diagnostics*" {
				// Navigate to previous diagnostic in *Diagnostics* buffer
				return e.previousDiagnosticInBuffer()
			} else if e.lastActiveSpecialBuffer == "*Project Changes*" {
				// Navigate to previous change in *Project Changes* buffer
				return e.previousChangeInProjectBuffer()
			}
		}
	}

	// No special buffer active, navigate git changes in current file
	return e.previousGitChange()
}

// getGitChangeHunks returns the start lines of each hunk (group of contiguous changed lines)
func (e *Editor) getGitChangeHunks() []int {
    buf := e.CurrentBuffer()
    if buf == nil || buf.GitChanges == nil {
        // If we're in diff view, derive hunks from diffLineInfo instead
        if buf != nil {
            if diffLines, ok := e.diffLineInfo[buf]; ok && len(diffLines) > 0 {
                hunks := make([]int, 0)
                inRun := false
                runStart := 0
                for i, dl := range diffLines {
                    isChange := dl.LineType != DiffLineContext
                    if isChange && !inRun {
                        inRun = true
                        runStart = i
                    } else if !isChange && inRun {
                        // Close the run [runStart, i)
                        // Prefer the first non-filler line within the run for this buffer
                        target := runStart
                        for j := runStart; j < i; j++ {
                            if !diffLines[j].IsFillerLine {
                                target = j
                                break
                            }
                        }
                        hunks = append(hunks, target+1) // 1-based
                        inRun = false
                    }
                }
                // Handle trailing run to EOF
                if inRun {
                    target := runStart
                    for j := runStart; j < len(diffLines); j++ {
                        if !diffLines[j].IsFillerLine {
                            target = j
                            break
                        }
                    }
                    hunks = append(hunks, target+1)
                }
                return hunks
            }
        }
        return nil
    }

	// Combine and sort all changed lines
	allChanges := make([]int, 0)
	allChanges = append(allChanges, buf.GitChanges.Added...)
	allChanges = append(allChanges, buf.GitChanges.Modified...)

	if len(allChanges) == 0 {
		return nil
	}

	sort.Ints(allChanges)

	// Group into hunks (contiguous line groups)
	hunks := make([]int, 0)
	if len(allChanges) > 0 {
		hunks = append(hunks, allChanges[0])
		for i := 1; i < len(allChanges); i++ {
			// If there's a gap of more than 1 line, it's a new hunk
			if allChanges[i] > allChanges[i-1]+1 {
				hunks = append(hunks, allChanges[i])
			}
		}
	}

	return hunks
}

// nextGitChange jumps to the next git change hunk in the current buffer
func (e *Editor) nextGitChange() error {
	hunks := e.getGitChangeHunks()
	if len(hunks) == 0 {
		return fmt.Errorf("no git changes in current buffer")
	}

	// Get current line (1-based)
	currentLine := e.CurrentBuffer().CursorIterator.LineNumber() + 1

	// Find next hunk
	for _, hunkStart := range hunks {
		if hunkStart > currentLine {
			// Jump to this hunk (goToLine expects 0-based)
			e.goToLine(hunkStart - 1)
			return nil
		}
	}

	// Wrap around to first hunk
	e.goToLine(hunks[0] - 1)
	return nil
}

// previousGitChange jumps to the previous git change hunk in the current buffer
func (e *Editor) previousGitChange() error {
	hunks := e.getGitChangeHunks()
	if len(hunks) == 0 {
		return fmt.Errorf("no git changes in current buffer")
	}

	// Get current line (1-based)
	currentLine := e.CurrentBuffer().CursorIterator.LineNumber() + 1

	// Find the current hunk we're in (if any)
	currentHunkIdx := -1
	for i, hunkStart := range hunks {
		if hunkStart <= currentLine {
			currentHunkIdx = i
		} else {
			break
		}
	}

	// If we're at or after the start of a hunk, go to the previous hunk
	if currentHunkIdx > 0 {
		// If we're at the start of the current hunk, go to previous
		// Otherwise, go to the start of current hunk first
		if hunks[currentHunkIdx] == currentLine {
			e.goToLine(hunks[currentHunkIdx-1] - 1)
		} else {
			e.goToLine(hunks[currentHunkIdx] - 1)
		}
		return nil
	}

	// We're before all hunks or at the first hunk - wrap to last
	e.goToLine(hunks[len(hunks)-1] - 1)
	return nil
}

// nextDiagnosticInBuffer navigates to next diagnostic entry in *Diagnostics* buffer
func (e *Editor) nextDiagnosticInBuffer() error {
	buf, exists := e.buffers["*Diagnostics*"]
	if !exists {
		return fmt.Errorf("diagnostics buffer not open")
	}

	// Switch to diagnostics buffer if not current
	if e.CurrentBuffer() != buf {
		e.switchToBuffer(buf)
	}

	// Move to next line that starts with a file path (diagnostic entry)
	currentLine := buf.CursorIterator.LineNumber()
	for i := currentLine + 1; i < buf.LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		// Check if line contains a markdown link (diagnostic entry)
		if strings.Contains(line, "](") && strings.Contains(line, "#L") {
			buf.CursorIterator = iter
			buf.CursorX = 0
			return nil
		}
	}

	// Wrap around to first diagnostic
	for i := 0; i <= currentLine; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if strings.Contains(line, "](") && strings.Contains(line, "#L") {
			buf.CursorIterator = iter
			buf.CursorX = 0
			return nil
		}
	}

	return fmt.Errorf("no diagnostics to navigate")
}

// previousDiagnosticInBuffer navigates to previous diagnostic entry in *Diagnostics* buffer
func (e *Editor) previousDiagnosticInBuffer() error {
	buf, exists := e.buffers["*Diagnostics*"]
	if !exists {
		return fmt.Errorf("diagnostics buffer not open")
	}

	// Switch to diagnostics buffer if not current
	if e.CurrentBuffer() != buf {
		e.switchToBuffer(buf)
	}

	// Move to previous line that starts with a file path (diagnostic entry)
	currentLine := buf.CursorIterator.LineNumber()
	for i := currentLine - 1; i >= 0; i-- {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		// Check if line contains a markdown link (diagnostic entry)
		if strings.Contains(line, "](") && strings.Contains(line, "#L") {
			buf.CursorIterator = iter
			buf.CursorX = 0
			return nil
		}
	}

	// Wrap around to last diagnostic
	for i := buf.LineCount - 1; i >= currentLine; i-- {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if strings.Contains(line, "](") && strings.Contains(line, "#L") {
			buf.CursorIterator = iter
			buf.CursorX = 0
			return nil
		}
	}

	return fmt.Errorf("no diagnostics to navigate")
}

// nextChangeInProjectBuffer navigates to next change entry in *Project Changes* buffer
func (e *Editor) nextChangeInProjectBuffer() error {
	buf, exists := e.buffers["*Project Changes*"]
	if !exists {
		return fmt.Errorf("project changes buffer not open")
	}

	// Switch to project changes buffer if not current
	if e.CurrentBuffer() != buf {
		e.switchToBuffer(buf)
	}

	// Move to next line that contains a file entry
	currentLine := buf.CursorIterator.LineNumber()
	for i := currentLine + 1; i < buf.LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		// Check if line contains a markdown link (file entry)
		if strings.Contains(line, "](") && strings.Contains(line, ")") {
			buf.CursorIterator = iter
			buf.CursorX = 0
			return nil
		}
	}

	// Wrap around to first change
	for i := 0; i <= currentLine; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if strings.Contains(line, "](") && strings.Contains(line, ")") {
			buf.CursorIterator = iter
			buf.CursorX = 0
			return nil
		}
	}

	return fmt.Errorf("no changes to navigate")
}

// previousChangeInProjectBuffer navigates to previous change entry in *Project Changes* buffer
func (e *Editor) previousChangeInProjectBuffer() error {
	buf, exists := e.buffers["*Project Changes*"]
	if !exists {
		return fmt.Errorf("project changes buffer not open")
	}

	// Switch to project changes buffer if not current
	if e.CurrentBuffer() != buf {
		e.switchToBuffer(buf)
	}

	// Move to previous line that contains a file entry
	currentLine := buf.CursorIterator.LineNumber()
	for i := currentLine - 1; i >= 0; i-- {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		// Check if line contains a markdown link (file entry)
		if strings.Contains(line, "](") && strings.Contains(line, ")") {
			buf.CursorIterator = iter
			buf.CursorX = 0
			return nil
		}
	}

	// Wrap around to last change
	for i := buf.LineCount - 1; i >= currentLine; i-- {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if strings.Contains(line, "](") && strings.Contains(line, ")") {
			buf.CursorIterator = iter
			buf.CursorX = 0
			return nil
		}
	}

	return fmt.Errorf("no changes to navigate")
}
