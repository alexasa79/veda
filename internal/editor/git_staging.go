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
    buf := e.CurrentBuffer()
    if buf == nil {
        return fmt.Errorf("no buffer")
    }

    // Resolve real file path and working-file lines for this hunk
    realPath := buf.FilePath
    // Detect diff buffers and convert to real path
    isDiff := e.isDiffBuffer(buf)
    if strings.HasSuffix(realPath, "@current") || strings.HasSuffix(realPath, "@HEAD") {
        realPath = strings.TrimSuffix(strings.TrimSuffix(realPath, "@current"), "@HEAD")
    }

    var hunkLines []int
    if isDiff {
        // Use diffLineInfo to find run and map to working-file lines (RightLine)
        diffLines := e.diffLineInfo[buf]
        if len(diffLines) == 0 {
            return fmt.Errorf("no diff info")
        }
        row := buf.CursorIterator.LineNumber()
        if row < 0 || row >= len(diffLines) {
            return fmt.Errorf("invalid cursor position")
        }
        // Find run bounds
        start := row
        for start > 0 && diffLines[start-1].LineType != DiffLineContext {
            start--
        }
        end := row
        for end+1 < len(diffLines) && diffLines[end+1].LineType != DiffLineContext {
            end++
        }

        // If we're in @HEAD buffer, use paired @current buffer for RightLine mapping
        sourceInfo := diffLines
        if strings.HasSuffix(buf.FilePath, "@HEAD") {
            if pair := e.getDiffPairBuffer(buf); pair != nil {
                if pi, ok := e.diffLineInfo[pair]; ok && len(pi) == len(diffLines) {
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
            return fmt.Errorf("no working-file lines in hunk")
        }
    } else {
        if buf.GitChanges == nil {
            return fmt.Errorf("no git changes in current buffer")
        }
        // Build hunk based on git changes arrays
        currentLine := buf.CursorIterator.LineNumber() + 1
        hunks := e.getGitChangeHunks()
        if len(hunks) == 0 {
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
            return fmt.Errorf("no lines to stage in hunk")
        }
    }

    // Determine staged state using the real file buffer if available
    staged := false
    if realBuf, ok := e.buffers[realPath]; ok && realBuf != nil && realBuf.GitChanges != nil {
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
        return fmt.Errorf("failed to %s hunk: %v", map[bool]string{true: "unstage", false: "stage"}[staged], err)
    }
    // Update real buffer git changes if present
    if realBuf, ok := e.buffers[realPath]; ok && realBuf != nil {
        realBuf.GitChanges = changes
    }
    return nil
}

// toggleStageLine stages or unstages the current line
func (e *Editor) toggleStageLine(args []string) error {
    buf := e.CurrentBuffer()
    if buf == nil {
        return fmt.Errorf("no buffer")
    }

    realPath := buf.FilePath
    isDiff := e.isDiffBuffer(buf)
    if strings.HasSuffix(realPath, "@current") || strings.HasSuffix(realPath, "@HEAD") {
        realPath = strings.TrimSuffix(strings.TrimSuffix(realPath, "@current"), "@HEAD")
    }

    // Determine working-file 1-based line
    var line1 int = -1
    if isDiff {
        row := buf.CursorIterator.LineNumber()
        info := e.diffLineInfo[buf]
        if row < 0 || row >= len(info) {
            return fmt.Errorf("invalid cursor position")
        }
        // For @HEAD buffer, use paired current buffer’s RightLine
        dl := info[row]
        if strings.HasSuffix(buf.FilePath, "@HEAD") {
            if pair := e.getDiffPairBuffer(buf); pair != nil {
                if pi, ok := e.diffLineInfo[pair]; ok && row < len(pi) {
                    dl = pi[row]
                }
            }
        }
        if dl.RightLine >= 0 {
            line1 = dl.RightLine + 1
        } else {
            return fmt.Errorf("current row has no working-file line")
        }
    } else {
        if buf.GitChanges == nil {
            return fmt.Errorf("no git changes in current buffer")
        }
        line1 = buf.CursorIterator.LineNumber() + 1
        // Verify line has change
        has := false
        for _, l := range append([]int{}, buf.GitChanges.Added...) {
            if l == line1 { has = true; break }
        }
        if !has {
            for _, l := range buf.GitChanges.Modified { if l == line1 { has = true; break } }
        }
        if !has { return fmt.Errorf("current line has no git changes") }
    }

    // Determine staged state from real buffer if available
    staged := false
    if realBuf, ok := e.buffers[realPath]; ok && realBuf != nil && realBuf.GitChanges != nil {
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
        return fmt.Errorf("failed to %s line: %v", map[bool]string{true: "unstage", false: "stage"}[staged], err)
    }
    if realBuf, ok := e.buffers[realPath]; ok && realBuf != nil {
        realBuf.GitChanges = changes
    }
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
