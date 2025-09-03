package editor

import (
	"fmt"
	"sort"
	"veda/internal/buffer"
	"veda/internal/proxy"
)

// toggleStageHunk stages or unstages the current git hunk
func (e *Editor) toggleStageHunk(args []string) error {
	buf := e.CurrentBuffer()
	if buf == nil || buf.GitChanges == nil {
		return fmt.Errorf("no git changes in current buffer")
	}

	// Get current line (1-based)
	currentLine := buf.CursorIterator.LineNumber() + 1

	// Find the hunk that contains the current line
	hunks := e.getGitChangeHunks()
	if len(hunks) == 0 {
		return fmt.Errorf("no git changes in current buffer")
	}

	// Find which hunk we're in
	var hunkStart, hunkEnd int
	found := false
	for i, start := range hunks {
		// Determine the end of this hunk
		if i < len(hunks)-1 {
			hunkEnd = hunks[i+1] - 1
		} else {
			// Last hunk - find the last changed line
			allChanges := make([]int, 0)
			allChanges = append(allChanges, buf.GitChanges.Added...)
			allChanges = append(allChanges, buf.GitChanges.Modified...)
			if len(allChanges) > 0 {
				sort.Ints(allChanges)
				hunkEnd = allChanges[len(allChanges)-1]
			} else {
				hunkEnd = start
			}
		}

		if currentLine >= start && currentLine <= hunkEnd {
			hunkStart = start
			found = true
			break
		}
	}

	if !found {
		// Current line is not in a changed hunk
		return fmt.Errorf("cursor not on a git change")
	}

	// Collect all lines in this hunk
	hunkLines := make([]int, 0)
	allChanges := make([]int, 0)
	allChanges = append(allChanges, buf.GitChanges.Added...)
	allChanges = append(allChanges, buf.GitChanges.Modified...)
	sort.Ints(allChanges)

	for _, line := range allChanges {
		if line >= hunkStart && line <= hunkEnd {
			hunkLines = append(hunkLines, line)
		}
	}

	if len(hunkLines) == 0 {
		return fmt.Errorf("no lines to stage in hunk")
	}

	// Check if any lines in the hunk are already staged
	staged := false
	for _, line := range hunkLines {
		if e.isLineStaged(buf, line) {
			staged = true
			break
		}
	}

	// Toggle staging state
	var changes *proxy.FileChanges
	var err error
	if staged {
		// Unstage the hunk
		changes, err = e.proxyManager.UnstageLines(buf.FilePath, hunkLines)
	} else {
		// Stage the hunk
		changes, err = e.proxyManager.StageLines(buf.FilePath, hunkLines)
	}

	if err != nil {
		return fmt.Errorf("failed to %s hunk: %v", map[bool]string{true: "unstage", false: "stage"}[staged], err)
	}

	// Update buffer's git changes
	buf.GitChanges = changes
	return nil
}

// toggleStageLine stages or unstages the current line
func (e *Editor) toggleStageLine(args []string) error {
	buf := e.CurrentBuffer()
	if buf == nil || buf.GitChanges == nil {
		return fmt.Errorf("no git changes in current buffer")
	}

	// Get current line (1-based)
	currentLine := buf.CursorIterator.LineNumber() + 1

	// Check if current line has changes
	hasChange := false
	for _, line := range buf.GitChanges.Added {
		if line == currentLine {
			hasChange = true
			break
		}
	}
	if !hasChange {
		for _, line := range buf.GitChanges.Modified {
			if line == currentLine {
				hasChange = true
				break
			}
		}
	}

	if !hasChange {
		return fmt.Errorf("current line has no git changes")
	}

	// Check if line is already staged
	staged := e.isLineStaged(buf, currentLine)

	// Toggle staging state
	var changes *proxy.FileChanges
	var err error
	if staged {
		// Unstage the line
		changes, err = e.proxyManager.UnstageLines(buf.FilePath, []int{currentLine})
	} else {
		// Stage the line
		changes, err = e.proxyManager.StageLines(buf.FilePath, []int{currentLine})
	}

	if err != nil {
		return fmt.Errorf("failed to %s line: %v", map[bool]string{true: "unstage", false: "stage"}[staged], err)
	}

	// Update buffer's git changes
	buf.GitChanges = changes
	return nil
}

// isLineStaged checks if a line is staged
func (e *Editor) isLineStaged(buf *buffer.Buffer, line int) bool {
	if buf.GitChanges == nil {
		return false
	}

	// Check if line is in staged arrays
	for _, stagedLine := range buf.GitChanges.StagedAdded {
		if stagedLine == line {
			return true
		}
	}
	for _, stagedLine := range buf.GitChanges.StagedModified {
		if stagedLine == line {
			return true
		}
	}
	return false
}