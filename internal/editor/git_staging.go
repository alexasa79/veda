package editor

import (
	"fmt"
	"sort"
	"strings"
	"veda/internal/buffer"
	"veda/internal/proxy"
)

// toggleStageHunk stages or unstages the current git hunk
func (e *Editor) toggleStageHunk(args []string) error {
	if e.readOnlyDiff() {
		return fmt.Errorf("cannot stage a read-only comparison")
	}
	buf := e.CurrentBuffer()
	if buf == nil {
		return fmt.Errorf("no buffer")
	}

	// Resolve real file path and working-file lines for this hunk
	// Resolve real file path
	isDiff := e.isDiffBuffer(buf)
	realPath := buf.RealPath
	if realPath == "" {
		if pair := e.getDiffPairBuffer(buf); pair != nil {
			realPath = pair.RealPath
		}
	}

	var hunkLines []int
	if isDiff {
		// Use DiffHints to find run and map to working-file lines (RightLine)
		diffLines := buf.DiffHints
		if len(diffLines) == 0 {
			e.ShowError("git: no diff info; cannot stage hunk")
			return fmt.Errorf("no diff info")
		}
		row := buf.CursorIterator.LineNumber()
		if row < 0 || row >= len(diffLines) {
			e.ShowError("git: invalid cursor in diff (row=%d of %d)", row, len(diffLines))
			return fmt.Errorf("invalid cursor position")
		}
		// Find run bounds
		start := row
		for start > 0 && diffLines[start-1].LineType != buffer.DiffLineContext {
			start--
		}
		end := row
		for end+1 < len(diffLines) && diffLines[end+1].LineType != buffer.DiffLineContext {
			end++
		}

		// If we're in @HEAD buffer, use paired @current buffer for RightLine mapping
		sourceInfo := diffLines
		if strings.HasSuffix(buf.FilePath, "@HEAD") {
			if pair := e.getDiffPairBuffer(buf); pair != nil {
				if pi := pair.DiffHints; len(pi) > 0 && len(pi) == len(diffLines) {
					sourceInfo = pi
				}
			}
		}
		linesSet := make(map[int]struct{})
		for i := start; i <= end; i++ {
			rl := sourceInfo[i].RightLine
			if rl >= 0 {
				linesSet[rl+1] = struct{}{}
			}
		}
		// Collect and sort
		for l := range linesSet {
			hunkLines = append(hunkLines, l)
		}
		sort.Ints(hunkLines)
		if len(hunkLines) == 0 {
			e.ShowError("git: no working-file lines in diff hunk at row=%d", row)
			return fmt.Errorf("no working-file lines in hunk")
		}
	} else {
		if buf.GitChanges == nil {
			e.ShowError("git: no git changes loaded for buffer path=%s", buf.FilePath)
			return fmt.Errorf("no git changes in current buffer")
		}
		// Build hunk based on git changes arrays
		currentLine := buf.CursorIterator.LineNumber() + 1
		hunks := e.getGitChangeHunks()
		if len(hunks) == 0 {
			e.ShowError("git: no git change hunks; added=%d modified=%d deleted=%d", len(buf.GitChanges.Added), len(buf.GitChanges.Modified), len(buf.GitChanges.Deleted))
			return fmt.Errorf("no git changes in current buffer")
		}
		var hunkStart, hunkEnd int
		found := false
		for i, start := range hunks {
			if i < len(hunks)-1 {
				hunkEnd = hunks[i+1] - 1
			} else {
				allChanges := append([]int{}, buf.GitChanges.Added...)
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
			e.ShowError("git: cursor not on a git change (cursor=%d, hunks=%v)", currentLine, hunks)
			return fmt.Errorf("cursor not on a git change")
		}
		allChanges := append([]int{}, buf.GitChanges.Added...)
		allChanges = append(allChanges, buf.GitChanges.Modified...)
		sort.Ints(allChanges)
		for _, line := range allChanges {
			if line >= hunkStart && line <= hunkEnd {
				hunkLines = append(hunkLines, line)
			}
		}
		if len(hunkLines) == 0 {
			e.ShowError("git: no lines to stage (hunkStart=%d)", hunkStart)
			return fmt.Errorf("no lines to stage in hunk")
		}
	}

	// Determine staged state using the real file buffer if available
	staged := false
	if realBuf, ok := e.findBufferByRealPath(realPath); ok && realBuf != nil && realBuf.GitChanges != nil {
		for _, line := range hunkLines {
			if e.isLineStaged(realBuf, line) {
				staged = true
				break
			}
		}
	}

	var changes *proxy.FileChanges
	var err error
	if staged {
		changes, err = e.proxyManager.UnstageLines(realPath, hunkLines)
	} else {
		changes, err = e.proxyManager.StageLines(realPath, hunkLines)
	}
	if err != nil {
		e.ShowError("Failed to %s hunk: %v", map[bool]string{true: "unstage", false: "stage"}[staged], err)
		return fmt.Errorf("failed to %s hunk: %v", map[bool]string{true: "unstage", false: "stage"}[staged], err)
	}
	// Update real buffer git changes if present; if proxy returned nil, force refresh
	if changes != nil {
		// Fast path: update from proxy response
		e.updateGitChangesForRealPath(realPath, changes)
	} else {
		// Fallback: force refresh from proxy
		if fc, err2 := e.proxyManager.GetFile(realPath); err2 == nil {
			e.updateGitChangesForRealPath(realPath, &fc.Changes)
		}
	}
	return nil
}

// toggleStageLine stages or unstages the current line
func (e *Editor) toggleStageLine(args []string) error {
	if e.readOnlyDiff() {
		return fmt.Errorf("cannot stage a read-only comparison")
	}
	buf := e.CurrentBuffer()
	if buf == nil {
		return fmt.Errorf("no buffer")
	}

	isDiff := e.isDiffBuffer(buf)
	realPath := buf.RealPath
	if realPath == "" {
		if pair := e.getDiffPairBuffer(buf); pair != nil {
			realPath = pair.RealPath
		}
	}

	// Determine working-file 1-based line
	var line1 int = -1
	if isDiff {
		row := buf.CursorIterator.LineNumber()
		info := buf.DiffHints
		if row < 0 || row >= len(info) {
			e.ShowError("git: invalid cursor in diff (row=%d of %d)", row, len(info))
			return fmt.Errorf("invalid cursor position")
		}
		// For @HEAD buffer, use paired current buffer's RightLine
		dl := info[row]
		if strings.HasSuffix(buf.FilePath, "@HEAD") {
			if pair := e.getDiffPairBuffer(buf); pair != nil {
				if pi := pair.DiffHints; len(pi) > 0 && row < len(pi) {
					dl = pi[row]
				}
			}
		}
		if dl.RightLine >= 0 {
			line1 = dl.RightLine + 1
		} else {
			e.ShowError("git: selected diff row has no working-file line mapping")
			return fmt.Errorf("current row has no working-file line")
		}
	} else {
		if buf.GitChanges == nil {
			e.ShowError("git: no git changes loaded for buffer path=%s", buf.FilePath)
			return fmt.Errorf("no git changes in current buffer")
		}
		line1 = buf.CursorIterator.LineNumber() + 1
		// Verify line has change
		has := false
		for _, l := range append([]int{}, buf.GitChanges.Added...) {
			if l == line1 {
				has = true
				break
			}
		}
		if !has {
			for _, l := range buf.GitChanges.Modified {
				if l == line1 {
					has = true
					break
				}
			}
		}
		if !has {
			e.ShowError("git: current line %d has no git changes", line1)
			return fmt.Errorf("current line has no git changes")
		}
	}

	// Determine staged state from real buffer if available
	staged := false
	if realBuf, ok := e.findBufferByRealPath(realPath); ok && realBuf != nil && realBuf.GitChanges != nil {
		staged = e.isLineStaged(realBuf, line1)
	}

	var changes *proxy.FileChanges
	var err error
	if staged {
		changes, err = e.proxyManager.UnstageLines(realPath, []int{line1})
	} else {
		changes, err = e.proxyManager.StageLines(realPath, []int{line1})
	}
	if err != nil {
		e.ShowError("Failed to %s line: %v", map[bool]string{true: "unstage", false: "stage"}[staged], err)
		return fmt.Errorf("failed to %s line: %v", map[bool]string{true: "unstage", false: "stage"}[staged], err)
	}
	if changes != nil {
		e.updateGitChangesForRealPath(realPath, changes)
	} else {
		if fc, err2 := e.proxyManager.GetFile(realPath); err2 == nil {
			e.updateGitChangesForRealPath(realPath, &fc.Changes)
		}
	}
	return nil
}

// findBufferByRealPath returns the first buffer with the given RealPath
func (e *Editor) findBufferByRealPath(path string) (*buffer.Buffer, bool) {
	for _, b := range e.buffers {
		if b != nil && b.RealPath == path {
			return b, true
		}
	}
	return nil, false
}

// updateGitChangesForRealPath updates GitChanges pointer for all buffers with the same real path
func (e *Editor) updateGitChangesForRealPath(path string, changes *proxy.FileChanges) {
	if path == "" {
		return
	}
	for _, b := range e.buffers {
		if b != nil && b.RealPath == path {
			b.GitChanges = changes
		}
	}
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
