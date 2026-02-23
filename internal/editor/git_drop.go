package editor

import (
	"fmt"
	"sort"
	"strings"
	"veda/internal/buffer"
)

// dropChangeHunk reverts the current git hunk back to HEAD
func (e *Editor) dropChangeHunk(args []string) error {
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
	if realPath == "" {
		// Fall back to buffer path if it's a normal file buffer
		realPath = buf.FilePath
	}

	var hunkLines []int
	if isDiff {
		// Use diff info to find contiguous change block and map to working-file lines
		diffLines := buf.DiffHints
		if len(diffLines) == 0 {
			e.ShowError("git: no diff info; cannot drop hunk")
			return fmt.Errorf("no diff info")
		}
		row := buf.CursorIterator.LineNumber()
		if row < 0 || row >= len(diffLines) {
			e.ShowError("git: invalid cursor in diff (row=%d of %d)", row, len(diffLines))
			return fmt.Errorf("invalid cursor position")
		}
		// Expand to change run [start,end]
		start := row
		for start > 0 && diffLines[start-1].LineType != buffer.DiffLineContext {
			start--
		}
		end := row
		for end+1 < len(diffLines) && diffLines[end+1].LineType != buffer.DiffLineContext {
			end++
		}

		// For @HEAD, use paired @current buffer mapping
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
			e.ShowError("git: no lines to drop (hunkStart=%d)", hunkStart)
			return fmt.Errorf("no lines to drop in hunk")
		}
	}

	// Revert these lines back to HEAD
	if _, err := e.proxyManager.DropLines(realPath, hunkLines); err != nil {
		e.ShowError("Failed to drop hunk: %v", err)
		return fmt.Errorf("failed to drop hunk: %v", err)
	}

	// Reload file content from disk to reflect changes in buffers with this path
	if realPath != "" {
		e.performReload(realPath)
	}
	return nil
}

// dropChangeLine reverts the current line change back to HEAD
func (e *Editor) dropChangeLine(args []string) error {
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
	if realPath == "" {
		realPath = buf.FilePath
	}

	// Determine working-file 1-based line to drop
	var line1 int = -1
	if isDiff {
		row := buf.CursorIterator.LineNumber()
		info := buf.DiffHints
		if row < 0 || row >= len(info) {
			e.ShowError("git: invalid cursor in diff (row=%d of %d)", row, len(info))
			return fmt.Errorf("invalid cursor position")
		}
		dl := info[row]
		// For @HEAD, use mapping from the paired current buffer
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
		// Ensure this line is a changed (added/modified) line
		has := false
		for _, l := range buf.GitChanges.Added {
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

	if _, err := e.proxyManager.DropLines(realPath, []int{line1}); err != nil {
		e.ShowError("Failed to drop line: %v", err)
		return fmt.Errorf("failed to drop line: %v", err)
	}

	// Reload file content so buffer reflects the reverted line
	if realPath != "" {
		e.performReload(realPath)
	}
	return nil
}
