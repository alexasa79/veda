package editor

import (
	"fmt"
	"sort"
	"strconv"
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
	// In a diff view, navigate hunks within the diff rather than dispatching
	// to project changes (which would call openFile and exit the diff).
	if e.isDiffViewActive() {
		return e.nextGitChange()
	}

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
				// Navigate to next change across the project (open file and jump to hunk)
				return e.nextProjectChange()
			}
		}
	}

	// No special buffer active, navigate git changes in current file
	return e.nextGitChange()
}

// previousContextualItem jumps to the previous item based on active context
func (e *Editor) previousContextualItem(args []string) error {
	// In a diff view, navigate hunks within the diff rather than dispatching
	// to project changes (which would call openFile and exit the diff).
	if e.isDiffViewActive() {
		return e.previousGitChange()
	}

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
				// Navigate to previous change across the project (open file and jump to hunk)
				return e.previousProjectChange()
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
		// If we're in diff view, derive hunks from DiffHints instead
		if buf != nil {
			if diffLines := buf.DiffHints; len(diffLines) > 0 {
				hunks := make([]int, 0)
				inRun := false
				runStart := 0
				for i, dl := range diffLines {
					isChange := dl.LineType != buffer.DiffLineContext
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

// parseProjectChangeLink extracts (path, line) from a markdown link in the form "...](PATH#LNUM)".
// Returns ok=false if the line doesn't contain a link with a line anchor.
func parseProjectChangeLink(line string) (string, int, bool) {
	// Find parens containing the URL
	start := strings.IndexByte(line, '(')
	end := strings.IndexByte(line, ')')
	if start < 0 || end < 0 || end <= start+1 {
		return "", 0, false
	}
	target := line[start+1 : end]
	// Expect optional #LNNN anchor
	if i := strings.Index(target, "#L"); i >= 0 {
		path := target[:i]
		numStr := target[i+2:]
		if n, err := strconv.Atoi(numStr); err == nil && n > 0 {
			return path, n, true
		}
		return "", 0, false
	}
	return "", 0, false
}

// getProjectChangesProgress returns (currentIndex, total) of project hunks based
// on the selection position inside the *Project Changes* buffer. currentIndex is
// 1-based and counts the number of hunk links up to and including the selected line.
func (e *Editor) getProjectChangesProgress() (int, int, bool) {
	buf, exists := e.buffers["*Project Changes*"]
	if !exists || buf == nil {
		return 0, 0, false
	}
	sel := buf.CursorIterator.LineNumber()
	// First, count total hunk links
	total := 0
	for i := 0; i < buf.LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		if _, _, ok := parseProjectChangeLink(iter.Line()); ok {
			total++
		}
	}
	if total == 0 {
		return 0, 0, false
	}
	// Only show progress if the selection is on a hunk link
	selIter := buffer.NewBufferIteratorAtLine(buf, sel)
	if _, _, ok := parseProjectChangeLink(selIter.Line()); !ok {
		return 0, total, false
	}
	// Compute 1-based index of the selected link
	idx := 0
	for i := 0; i <= sel; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		if _, _, ok := parseProjectChangeLink(iter.Line()); ok {
			idx++
		}
	}
	if idx < 1 {
		return 0, total, false
	}
	return idx, total, true
}

// getDiagnosticsProgress returns (currentIndex, total) of diagnostic entries
// within the *Diagnostics* buffer based on the current selection. It treats
// any line with a markdown file link that includes a #L anchor as an entry.
func (e *Editor) getDiagnosticsProgress() (int, int, bool) {
	buf, exists := e.buffers["*Diagnostics*"]
	if !exists || buf == nil {
		return 0, 0, false
	}
	sel := buf.CursorIterator.LineNumber()
	// Count total diagnostic entries (lines with a link + #L anchor)
	total := 0
	for i := 0; i < buf.LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if strings.Contains(line, "](") && strings.Contains(line, "#L") {
			total++
		}
	}
	if total == 0 {
		return 0, 0, false
	}
	// Only show progress if selection is on a diagnostic entry line
	selIter := buffer.NewBufferIteratorAtLine(buf, sel)
	selLine := selIter.Line()
	if !(strings.Contains(selLine, "](") && strings.Contains(selLine, "#L")) {
		return 0, total, false
	}
	// Compute 1-based index
	idx := 0
	for i := 0; i <= sel; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if strings.Contains(line, "](") && strings.Contains(line, "#L") {
			idx++
		}
	}
	if idx < 1 {
		return 0, total, false
	}
	return idx, total, true
}

// nextProjectChange navigates to the next change across the project by opening the
// referenced file and jumping to the hunk line. Uses the selection position in
// the *Project Changes* buffer to keep progress.
func (e *Editor) nextProjectChange() error {
	buf, exists := e.buffers["*Project Changes*"]
	if !exists {
		return fmt.Errorf("project changes buffer not open")
	}
	// Start after current selection
	start := buf.CursorIterator.LineNumber() + 1
	// Search forward for a link with a line anchor (hunk entry)
	for i := start; i < buf.LineCount; i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if path, ln, ok := parseProjectChangeLink(line); ok {
			// Check if file exists; skip deleted files
			if attrs, err := e.proxyManager.GetFileAttributes(path); err == nil {
				if !attrs.Exists {
					// Skip this deleted entry and continue scanning
					continue
				}
			}

			// Update selection for next call
			buf.CursorIterator = iter
			buf.CursorX = 0
			// Open file and jump to the first actual changed line at/after hunk start
			e.openFile(path)
			target := ln
			if e.CurrentBuffer() != nil && e.CurrentBuffer().GitChanges != nil {
				// Find the minimal added/modified line >= ln
				minLine := 0
				for _, v := range e.CurrentBuffer().GitChanges.Added {
					if v >= ln && (minLine == 0 || v < minLine) {
						minLine = v
					}
				}
				for _, v := range e.CurrentBuffer().GitChanges.Modified {
					if v >= ln && (minLine == 0 || v < minLine) {
						minLine = v
					}
				}
				if minLine > 0 {
					target = minLine
				}
			}
			e.goToLine(target - 1)
			return nil
		}
	}
	// Wrap around to top
	for i := 0; i <= buf.CursorIterator.LineNumber(); i++ {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if path, ln, ok := parseProjectChangeLink(line); ok {
			if attrs, err := e.proxyManager.GetFileAttributes(path); err == nil {
				if !attrs.Exists {
					continue
				}
			}
			buf.CursorIterator = iter
			buf.CursorX = 0
			e.openFile(path)
			target := ln
			if e.CurrentBuffer() != nil && e.CurrentBuffer().GitChanges != nil {
				minLine := 0
				for _, v := range e.CurrentBuffer().GitChanges.Added {
					if v >= ln && (minLine == 0 || v < minLine) {
						minLine = v
					}
				}
				for _, v := range e.CurrentBuffer().GitChanges.Modified {
					if v >= ln && (minLine == 0 || v < minLine) {
						minLine = v
					}
				}
				if minLine > 0 {
					target = minLine
				}
			}
			e.goToLine(target - 1)
			return nil
		}
	}
	return fmt.Errorf("no project changes to navigate")
}

// previousProjectChange navigates to the previous project change (wraps around).
func (e *Editor) previousProjectChange() error {
	buf, exists := e.buffers["*Project Changes*"]
	if !exists {
		return fmt.Errorf("project changes buffer not open")
	}
	start := buf.CursorIterator.LineNumber() - 1
	for i := start; i >= 0; i-- {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if path, ln, ok := parseProjectChangeLink(line); ok {
			if attrs, err := e.proxyManager.GetFileAttributes(path); err == nil {
				if !attrs.Exists {
					continue
				}
			}
			buf.CursorIterator = iter
			buf.CursorX = 0
			e.openFile(path)
			target := ln
			if e.CurrentBuffer() != nil && e.CurrentBuffer().GitChanges != nil {
				minLine := 0
				for _, v := range e.CurrentBuffer().GitChanges.Added {
					if v >= ln && (minLine == 0 || v < minLine) {
						minLine = v
					}
				}
				for _, v := range e.CurrentBuffer().GitChanges.Modified {
					if v >= ln && (minLine == 0 || v < minLine) {
						minLine = v
					}
				}
				if minLine > 0 {
					target = minLine
				}
			}
			e.goToLine(target - 1)
			return nil
		}
	}
	// Wrap to bottom
	for i := buf.LineCount - 1; i >= buf.CursorIterator.LineNumber(); i-- {
		iter := buffer.NewBufferIteratorAtLine(buf, i)
		line := iter.Line()
		if path, ln, ok := parseProjectChangeLink(line); ok {
			buf.CursorIterator = iter
			buf.CursorX = 0
			e.openFile(path)
			e.goToLine(ln - 1)
			return nil
		}
	}
	return fmt.Errorf("no project changes to navigate")
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
