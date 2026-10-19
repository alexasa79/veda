package editor

import (
	"fmt"
	"strings"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/syntax"
)

// DiffSession represents an active diff view session with two buffers
type DiffSession struct {
	// The two buffers being compared
	BufferA *buffer.Buffer // Previous version (left pane)
	BufferB *buffer.Buffer // Current version (right pane)

	// Source content (for test compatibility)
	ASource string // Previous version source
	BSource string // Current version source

	// Syntax highlighters for each buffer
	HighlighterA *syntax.Highlighter
	HighlighterB *syntax.Highlighter

	// View state
	ActivePane int // 0 for A (left), 1 for B (right)
	PaneWidth  int // Width of each pane in characters
}

// NewDiffSession creates a new diff session with two buffers
func NewDiffSession(previousContent, currentContent, filePath string) *DiffSession {
	// Create buffers for both versions
	bufferA := buffer.NewBuffer(previousContent, filePath+"@HEAD", filePath+"@HEAD", config.IndentTabs, 4)
	bufferB := buffer.NewBuffer(currentContent, filePath+"@current", filePath+"@current", config.IndentTabs, 4)

	// Mark the previous version as read-only
	bufferA.IsReadOnly = true

	// Ensure both buffers have proper iterators
	bufferA.ViewIterator = buffer.NewBufferIterator(bufferA)
	bufferA.CursorIterator = buffer.NewBufferIterator(bufferA)
	bufferB.ViewIterator = buffer.NewBufferIterator(bufferB)
	bufferB.CursorIterator = buffer.NewBufferIterator(bufferB)

	ds := &DiffSession{
		BufferA:    bufferA,
		BufferB:    bufferB,
		ASource:    previousContent,
		BSource:    currentContent,
		ActivePane: 1, // Start with current version (B) active
	}

	// Create syntax highlighters for both buffers
	if filePath != "" {
		ds.HighlighterA = syntax.NewHighlighter()
		ds.HighlighterA.SetLanguageFromFileAndContent(filePath, []byte(previousContent))
		ds.HighlighterA.ParseFull([]byte(previousContent))

		ds.HighlighterB = syntax.NewHighlighter()
		ds.HighlighterB.SetLanguageFromFileAndContent(filePath, []byte(currentContent))
		ds.HighlighterB.ParseFull([]byte(currentContent))
	}

	return ds
}

// GetActiveBuffer returns the currently active buffer
func (ds *DiffSession) GetActiveBuffer() *buffer.Buffer {
	if ds.ActivePane == 0 {
		return ds.BufferA
	}
	return ds.BufferB
}

// GetActiveHighlighter returns the highlighter for the active pane
func (ds *DiffSession) GetActiveHighlighter() *syntax.Highlighter {
	if ds.ActivePane == 0 {
		return ds.HighlighterA
	}
	return ds.HighlighterB
}

// SetActivePane sets which pane is currently active (0=A, 1=B)
func (ds *DiffSession) SetActivePane(pane int) {
	if pane == 0 || pane == 1 {
		ds.ActivePane = pane
	}
}

// SyncViewIterators synchronizes the view iterators of both buffers to the same line
func (ds *DiffSession) SyncViewIterators(lineNum int) {
	if ds.BufferA != nil && ds.BufferA.ViewIterator != nil {
		ds.BufferA.ViewIterator = buffer.NewBufferIteratorAtLine(ds.BufferA, lineNum)
	}
	if ds.BufferB != nil && ds.BufferB.ViewIterator != nil {
		ds.BufferB.ViewIterator = buffer.NewBufferIteratorAtLine(ds.BufferB, lineNum)
	}
}

// startDiffSessionForCurrentBuffer starts a diff session for the current buffer
// This method is used by the test framework
func (e *Editor) startDiffSessionForCurrentBuffer(previousContent string) {
	if e.CurrentBuffer() == nil {
		return
	}

	currentContent := e.CurrentBuffer().Rope.String()
	filePath := e.CurrentBuffer().FilePath

	// Create the diff session
	diffSession := NewDiffSession(previousContent, currentContent, filePath)

	// Note: DiffSession attachment removed as we're using simple buffer approach
	_ = diffSession // Suppress unused variable warning
}

// StartDiffViewForCurrentBuffer starts a diff view for the current buffer against git HEAD
func (e *Editor) StartDiffViewForCurrentBuffer() error {
	LL.Debug("StartDiffViewForCurrentBuffer: Starting diff view")

	if e.CurrentBuffer() == nil || !e.CurrentBuffer().HasFile() {
		LL.Debug("StartDiffViewForCurrentBuffer: No valid file buffer")
		return fmt.Errorf("no file buffer to diff")
	}

	if e.proxyManager == nil {
		LL.Debug("StartDiffViewForCurrentBuffer: No proxy manager available")
		return fmt.Errorf("proxy manager not available")
	}

	originalBuffer := e.CurrentBuffer()
	filePath := originalBuffer.FilePath
	currentContent := originalBuffer.Rope.String()
	currentCursorLine := originalBuffer.CursorIterator.LineNumber()
	currentCursorX := originalBuffer.CursorX
	cursorVisualLine, _ := e.GetCursorVisualPosition()
	LL.Debug("StartDiffViewForCurrentBuffer: File=%s, CurrentContentLen=%d", filePath, len(currentContent))

	// Get the previous version from git
	previousContent, err := e.proxyManager.GetGitFileVersion(filePath, "HEAD")
	if err != nil {
		LL.Debug("StartDiffViewForCurrentBuffer: Git error: %v", err)
		return fmt.Errorf("failed to get git version: %w", err)
	}
	LL.Debug("StartDiffViewForCurrentBuffer: Git content retrieved, PreviousContentLen=%d", len(previousContent))

	// Compute diff to create aligned content with same number of lines
	diffResult := e.computeDiffViaProxy(previousContent, currentContent)
	alignedHeadContent := createBufferContent(diffResult.LeftLines)
	alignedCurrentContent := createBufferContent(diffResult.RightLines)
	diffCursorLine := rightDiffRow(diffResult.RightLines, currentCursorLine)

	LL.Debug("StartDiffViewForCurrentBuffer: Aligned content created - Head lines: %d, Current lines: %d",
		len(diffResult.LeftLines), len(diffResult.RightLines))

	// Create HEAD buffer name and right buffer name
	headBufferName := filePath + "@HEAD"
	currentBufferName := filePath + "@current"

	// Create or update HEAD buffer
	if headBuffer, exists := e.buffers[headBufferName]; exists {
		// Update existing buffer
		headBuffer.Rope = buffer.NewRopeFromString(alignedHeadContent)
		headBuffer.CursorX = 0
		headBuffer.ViewIterator = buffer.NewBufferIterator(headBuffer)
		headBuffer.CursorIterator = buffer.NewBufferIterator(headBuffer)
		headBuffer.BufferName = headBufferName
		headBuffer.RealPath = filePath
		headBuffer.IsReadOnly = true
	} else {
		// Create new HEAD buffer
		headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, headBufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		headBuffer.IsReadOnly = true
		headBuffer.IsInternal = false // Can be moved between windows
		headBuffer.BufferName = headBufferName
		headBuffer.RealPath = filePath
		e.buffers[headBufferName] = headBuffer
		e.bufferOrder = append(e.bufferOrder, headBufferName)
	}

	// Ensure syntax highlighter exists and is updated for HEAD buffer
	headBuffer := e.buffers[headBufferName]

	// Always update or create the highlighter with the new content
	headHighlighter := syntax.NewHighlighter()
	headHighlighter.SetLanguageFromFileAndContent(filePath, []byte(alignedHeadContent))
	headHighlighter.ParseFull([]byte(alignedHeadContent))
	headBuffer.Highlighter = headHighlighter

	headBuffer.DiffHints = diffResult.LeftLines

	// Create or update current buffer copy
	if currentBuffer, exists := e.buffers[currentBufferName]; exists {
		// Update existing buffer
		currentBuffer.Rope = buffer.NewRopeFromString(alignedCurrentContent)
		currentBuffer.CursorX = currentCursorX
		currentBuffer.ViewIterator = buffer.NewBufferIteratorAtLine(currentBuffer, diffCursorLine)
		currentBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(currentBuffer, diffCursorLine)
		currentBuffer.BufferName = currentBufferName
		currentBuffer.RealPath = filePath
		// Share GitChanges with original buffer so gutter reflects staging
		currentBuffer.GitChanges = originalBuffer.GitChanges
	} else {
		// Create new current buffer copy
		currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, currentBufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		currentBuffer.IsReadOnly = false
		currentBuffer.IsInternal = false // Can be moved between windows
		currentBuffer.CursorX = currentCursorX
		currentBuffer.ViewIterator = buffer.NewBufferIteratorAtLine(currentBuffer, diffCursorLine)
		currentBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(currentBuffer, diffCursorLine)
		currentBuffer.BufferName = currentBufferName
		currentBuffer.RealPath = filePath
		// Share GitChanges with original buffer so gutter reflects staging
		currentBuffer.GitChanges = originalBuffer.GitChanges
		e.buffers[currentBufferName] = currentBuffer
		e.bufferOrder = append(e.bufferOrder, currentBufferName)
	}

	// Ensure syntax highlighter exists and is updated for current buffer
	currentBuffer := e.buffers[currentBufferName]

	// Always update or create the highlighter with the new content
	currentHighlighter := syntax.NewHighlighter()
	currentHighlighter.SetLanguageFromFileAndContent(filePath, []byte(alignedCurrentContent))
	currentHighlighter.ParseFull([]byte(alignedCurrentContent))
	currentBuffer.Highlighter = currentHighlighter

	currentBuffer.DiffHints = diffResult.RightLines

	// Set up DiffBufferView on the current window (single-window diff).
	headBuffer = e.buffers[headBufferName]
	currentBuffer = e.buffers[currentBufferName]

	if e.activeWindow < 0 || e.activeWindow >= len(e.windows) {
		e.windows = append(e.windows, newWindow(nil))
		e.activeWindow = len(e.windows) - 1
	}
	win := e.windows[e.activeWindow]

	diffView := &DiffBufferView{
		LeftBuffer:  headBuffer,
		RightBuffer: currentBuffer,
		ActivePane:  1, // Focus on current (right) by default
		RealPath:    filePath,
	}

	// Remove the SingleBufferView for the original file from Views
	// (the DiffBufferView replaces it).
	for i, v := range win.Views {
		if sv, ok := v.(*SingleBufferView); ok && sv.Buf != nil && sv.Buf.FilePath == filePath {
			win.Views = append(win.Views[:i], win.Views[i+1:]...)
			break
		}
	}

	// Add the DiffBufferView to Views
	win.Views = append(win.Views, diffView)

	win.View = diffView
	win.Buffer = currentBuffer
	win.addBufferToWindow(headBuffer)
	win.addBufferToWindow(currentBuffer)
	e.switchToBuffer(currentBuffer)
	e.preserveCursorVisualLine(cursorVisualLine)

	// Sync cursors between panes
	diffView.SyncCursors()
	win.ViewLine = currentBuffer.ViewIterator.LineNumber()
	win.CursorLine = currentBuffer.CursorIterator.LineNumber()
	win.CursorX = currentBuffer.CursorX

	LL.Debug("StartDiffViewForCurrentBuffer: Diff view setup complete")

	return nil
}

// createSecondWindow creates a second window if it doesn't exist
func (e *Editor) createSecondWindow() {
	if len(e.windows) >= 2 {
		return
	}

	// Create a new empty window
	newWin := newWindow(nil)
	e.windows = append(e.windows, newWin)

	// Recompute layout to accommodate the new window
	e.recomputeWindowsLayout()
}

// diffCurrentBuffer command handler for opening diff view
func (e *Editor) readOnlyDiff() bool {
	v := e.findDiffViewForBuffer(e.CurrentBuffer())
	return v != nil && v.LeftBuffer.IsReadOnly && v.RightBuffer.IsReadOnly
}

func (e *Editor) diffCurrentBuffer(args []string) error {
	return e.StartDiffViewForCurrentBuffer()
}

// isDiffBuffer checks if a buffer is part of a diff pair
func (e *Editor) isDiffBuffer(buf *buffer.Buffer) bool {
	if buf == nil {
		return false
	}
	return buf.DiffHints != nil
}

// findDiffViewForBuffer returns the DiffBufferView containing the given buffer, or nil.
// Searches all Views in all windows, not just the active View.
func (e *Editor) findDiffViewForBuffer(buf *buffer.Buffer) *DiffBufferView {
	if buf == nil {
		return nil
	}
	for _, w := range e.windows {
		if w == nil {
			continue
		}
		for _, v := range w.Views {
			if dv, ok := v.(*DiffBufferView); ok {
				if dv.LeftBuffer == buf || dv.RightBuffer == buf {
					return dv
				}
			}
		}
	}
	return nil
}

// getDiffPairBuffer returns the paired diff buffer if it exists
func (e *Editor) getDiffPairBuffer(buf *buffer.Buffer) *buffer.Buffer {
	if buf == nil {
		return nil
	}

	// Check DiffBufferView first
	if dv := e.findDiffViewForBuffer(buf); dv != nil {
		if dv.LeftBuffer == buf {
			return dv.RightBuffer
		}
		return dv.LeftBuffer
	}

	// Legacy fallback: string-suffix matching
	if strings.HasSuffix(buf.FilePath, "@HEAD") {
		currentPath := strings.TrimSuffix(buf.FilePath, "@HEAD") + "@current"
		return e.buffers[currentPath]
	} else if strings.HasSuffix(buf.FilePath, "@current") {
		headPath := strings.TrimSuffix(buf.FilePath, "@current") + "@HEAD"
		return e.buffers[headPath]
	}

	return nil
}

// syncDiffCursors synchronizes cursor Y positions and ViewIterator between diff buffer pairs.
func (e *Editor) syncDiffCursors() {
	// Check for DiffBufferView on the active window
	if e.activeWindow >= 0 && e.activeWindow < len(e.windows) {
		if dv, ok := e.windows[e.activeWindow].View.(*DiffBufferView); ok {
			dv.SyncCursors()
			return
		}
	}

	// Legacy two-window fallback
	if e.CurrentBuffer() == nil || !e.isDiffBuffer(e.CurrentBuffer()) {
		return
	}

	pairBuffer := e.getDiffPairBuffer(e.CurrentBuffer())
	if pairBuffer == nil {
		return
	}

	currentLine := e.CurrentBuffer().CursorIterator.LineNumber()
	pairBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(pairBuffer, currentLine)

	currentViewLine := e.CurrentBuffer().ViewIterator.LineNumber()
	pairBuffer.ViewIterator = buffer.NewBufferIteratorAtLine(pairBuffer, currentViewLine)

	for _, window := range e.windows {
		if window.Buffer == pairBuffer {
			window.CursorLine = currentLine
			window.ViewLine = currentViewLine
			break
		}
	}
}

// computeDiffViaProxy prefers proxy-backed (difflib) alignment; falls back internally
func (e *Editor) computeDiffViaProxy(leftText, rightText string) *buffer.DiffResult {
	if e == nil || e.proxyManager == nil {
		return computeDiff(leftText, rightText)
	}
	aligned, err := e.proxyManager.ComputeTextDiff(leftText, rightText)
	if err != nil || len(aligned) == 0 {
		return computeDiff(leftText, rightText)
	}
	res := &buffer.DiffResult{LeftLines: make([]buffer.DiffLine, 0, len(aligned)), RightLines: make([]buffer.DiffLine, 0, len(aligned))}
	curL, curR := 0, 0
	for _, it := range aligned {
		switch it.Tag {
		case "equal":
			res.LeftLines = append(res.LeftLines, buffer.DiffLine{Content: it.Left, LineType: buffer.DiffLineContext, LeftLine: curL, RightLine: curR, IsFillerLine: false})
			res.RightLines = append(res.RightLines, buffer.DiffLine{Content: it.Right, LineType: buffer.DiffLineContext, LeftLine: curL, RightLine: curR, IsFillerLine: false})
			curL++
			curR++
		case "insert":
			res.LeftLines = append(res.LeftLines, buffer.DiffLine{Content: "", LineType: buffer.DiffLineDeleted, LeftLine: -1, RightLine: -1, IsFillerLine: true})
			res.RightLines = append(res.RightLines, buffer.DiffLine{Content: it.Right, LineType: buffer.DiffLineAdded, LeftLine: -1, RightLine: curR, IsFillerLine: false})
			curR++
		case "delete":
			res.LeftLines = append(res.LeftLines, buffer.DiffLine{Content: it.Left, LineType: buffer.DiffLineAdded, LeftLine: curL, RightLine: -1, IsFillerLine: false})
			res.RightLines = append(res.RightLines, buffer.DiffLine{Content: "", LineType: buffer.DiffLineDeleted, LeftLine: -1, RightLine: -1, IsFillerLine: true})
			curL++
		case "replace":
			wr := computeWordDiff(it.Left, it.Right)
			res.LeftLines = append(res.LeftLines, buffer.DiffLine{Content: it.Left, LineType: buffer.DiffLineModified, LeftLine: curL, RightLine: curR, IsFillerLine: false, WordRanges: wr})
			res.RightLines = append(res.RightLines, buffer.DiffLine{Content: it.Right, LineType: buffer.DiffLineModified, LeftLine: curL, RightLine: curR, IsFillerLine: false, WordRanges: wr})
			curL++
			curR++
		default:
			// Fallback to context to avoid crashing on unknown tags
			res.LeftLines = append(res.LeftLines, buffer.DiffLine{Content: it.Left, LineType: buffer.DiffLineContext, LeftLine: curL, RightLine: curR, IsFillerLine: false})
			res.RightLines = append(res.RightLines, buffer.DiffLine{Content: it.Right, LineType: buffer.DiffLineContext, LeftLine: curL, RightLine: curR, IsFillerLine: false})
			curL++
			curR++
		}
	}
	return res
}

// computeDiff compares two texts and returns a buffer.DiffResult with aligned lines
// computeWordDiff computes character-level differences between two strings
// Uses a simple prefix/suffix matching approach for better accuracy
func computeWordDiff(left, right string) []buffer.DiffRange {
	var ranges []buffer.DiffRange

	leftRunes := []rune(left)
	rightRunes := []rune(right)

	// Handle simple cases
	if len(leftRunes) == 0 {
		// Everything in right is new
		if len(rightRunes) > 0 {
			ranges = append(ranges, buffer.DiffRange{Start: 0, End: len(rightRunes)})
		}
		return ranges
	}

	if len(rightRunes) == 0 {
		// Everything was deleted, nothing to highlight in right
		return ranges
	}

	// Find common prefix
	commonPrefixLen := 0
	minLen := len(leftRunes)
	if len(rightRunes) < minLen {
		minLen = len(rightRunes)
	}

	for i := 0; i < minLen; i++ {
		if leftRunes[i] == rightRunes[i] {
			commonPrefixLen++
		} else {
			break
		}
	}

	// Find common suffix
	commonSuffixLen := 0
	leftIdx := len(leftRunes) - 1
	rightIdx := len(rightRunes) - 1

	for leftIdx >= commonPrefixLen && rightIdx >= commonPrefixLen &&
		leftRunes[leftIdx] == rightRunes[rightIdx] {
		commonSuffixLen++
		leftIdx--
		rightIdx--
	}

	// Calculate the differing middle section
	rightStart := commonPrefixLen
	rightEnd := len(rightRunes) - commonSuffixLen

	// If there's a differing section, add it as a range
	if rightStart < rightEnd {
		ranges = append(ranges, buffer.DiffRange{Start: rightStart, End: rightEnd})
	}

	return ranges
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// findCharLCS finds longest common subsequence at character level
func findCharLCS(left, right []rune) []LCSMatch {
	leftLen := len(left)
	rightLen := len(right)

	if leftLen == 0 || rightLen == 0 {
		return []LCSMatch{}
	}

	// Create LCS table
	lcs := make([][]int, leftLen+1)
	for i := range lcs {
		lcs[i] = make([]int, rightLen+1)
	}

	// Fill LCS table
	for i := 1; i <= leftLen; i++ {
		for j := 1; j <= rightLen; j++ {
			if left[i-1] == right[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else {
				lcs[i][j] = max(lcs[i-1][j], lcs[i][j-1])
			}
		}
	}

	// Backtrack to find LCS
	var matches []LCSMatch
	i, j := leftLen, rightLen
	for i > 0 && j > 0 {
		if left[i-1] == right[j-1] {
			matches = append([]LCSMatch{{LeftIdx: i - 1, RightIdx: j - 1}}, matches...)
			i--
			j--
		} else if lcs[i-1][j] > lcs[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return matches
}

func computeDiff(leftText, rightText string) *buffer.DiffResult {
	leftLines := strings.Split(leftText, "\n")
	rightLines := strings.Split(rightText, "\n")

	// Remove trailing empty line if it exists (common when splitting text ending with \n)
	if len(leftLines) > 0 && leftLines[len(leftLines)-1] == "" {
		leftLines = leftLines[:len(leftLines)-1]
	}
	if len(rightLines) > 0 && rightLines[len(rightLines)-1] == "" {
		rightLines = rightLines[:len(rightLines)-1]
	}

	// For now, use a simple line-by-line alignment
	// This ensures both sides have exactly the same number of lines
	return alignLines(leftLines, rightLines)
}

// alignLines creates aligned buffer.DiffResult ensuring both sides have equal line counts
func alignLines(leftLines, rightLines []string) *buffer.DiffResult {
	result := &buffer.DiffResult{
		LeftLines:  make([]buffer.DiffLine, 0),
		RightLines: make([]buffer.DiffLine, 0),
	}

	// Use dynamic programming to find longest common subsequence
	lcs := findLCS(leftLines, rightLines)

	// Trace through both sequences to create aligned output
	leftIdx := 0
	rightIdx := 0
	lcsIdx := 0

	for leftIdx < len(leftLines) || rightIdx < len(rightLines) {
		// Check if we have a match from LCS
		if lcsIdx < len(lcs) &&
			leftIdx < len(leftLines) && rightIdx < len(rightLines) &&
			lcs[lcsIdx].LeftIdx == leftIdx && lcs[lcsIdx].RightIdx == rightIdx {
			// Common line - context
			line := leftLines[leftIdx]
			result.LeftLines = append(result.LeftLines, buffer.DiffLine{
				Content:      line,
				LineType:     buffer.DiffLineContext,
				LeftLine:     leftIdx,
				RightLine:    rightIdx,
				IsFillerLine: false,
			})
			result.RightLines = append(result.RightLines, buffer.DiffLine{
				Content:      line,
				LineType:     buffer.DiffLineContext,
				LeftLine:     leftIdx,
				RightLine:    rightIdx,
				IsFillerLine: false,
			})
			leftIdx++
			rightIdx++
			lcsIdx++
		} else if leftIdx < len(leftLines) &&
			(rightIdx >= len(rightLines) ||
				(lcsIdx < len(lcs) && leftIdx < lcs[lcsIdx].LeftIdx)) {
			// Line only in left (present in HEAD, missing in current)
			// Show red (Deleted) on the left, green (Added filler) on the right
			result.LeftLines = append(result.LeftLines, buffer.DiffLine{
				Content:      leftLines[leftIdx],
				LineType:     buffer.DiffLineDeleted,
				LeftLine:     leftIdx,
				RightLine:    -1,
				IsFillerLine: false,
			})
			result.RightLines = append(result.RightLines, buffer.DiffLine{
				Content:      "",
				LineType:     buffer.DiffLineAdded, // Filler opposite of left-side Deleted
				LeftLine:     -1,
				RightLine:    -1,
				IsFillerLine: true,
			})
			leftIdx++
		} else if rightIdx < len(rightLines) {
			// Line only in right (present in current, missing in HEAD)
			// Show red (Deleted filler) on the left, green (Added) on the right
			result.LeftLines = append(result.LeftLines, buffer.DiffLine{
				Content:      "",
				LineType:     buffer.DiffLineDeleted, // Filler opposite of right-side Added
				LeftLine:     -1,
				RightLine:    -1,
				IsFillerLine: true,
			})
			result.RightLines = append(result.RightLines, buffer.DiffLine{
				Content:      rightLines[rightIdx],
				LineType:     buffer.DiffLineAdded,
				LeftLine:     -1,
				RightLine:    rightIdx,
				IsFillerLine: false,
			})
			rightIdx++
		}
	}

	// Post-process to detect modifications (delete+insert pairs)
	result = detectModifications(result)

	return result
}

// LCSMatch represents a match in the longest common subsequence
type LCSMatch struct {
	LeftIdx  int
	RightIdx int
}

// findLCS finds the longest common subsequence between two string slices
func findLCS(leftLines, rightLines []string) []LCSMatch {
	m, n := len(leftLines), len(rightLines)
	if m == 0 || n == 0 {
		return []LCSMatch{}
	}

	// DP table: dp[i][j] = length of LCS for leftLines[0:i] and rightLines[0:j]
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Fill DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if leftLines[i-1] == rightLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	// Backtrack to find actual LCS
	var matches []LCSMatch
	i, j := m, n
	for i > 0 && j > 0 {
		if leftLines[i-1] == rightLines[j-1] {
			matches = append([]LCSMatch{{LeftIdx: i - 1, RightIdx: j - 1}}, matches...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return matches
}

// detectModifications post-processes the diff result to convert delete+insert pairs into modifications
func detectModifications(result *buffer.DiffResult) *buffer.DiffResult {
	if len(result.LeftLines) != len(result.RightLines) {
		return result // Safety check - should always be equal
	}

	newResult := &buffer.DiffResult{
		LeftLines:  make([]buffer.DiffLine, 0, len(result.LeftLines)),
		RightLines: make([]buffer.DiffLine, 0, len(result.RightLines)),
	}

	i := 0
	for i < len(result.LeftLines) {
		leftLine := result.LeftLines[i]
		rightLine := result.RightLines[i]

		// Detect a delete+insert pair represented with asymmetric types:
		// i:   Left=Added (real), Right=Deleted (filler)
		// i+1: Left=Deleted (filler), Right=Added (real)
		// Merge replace pattern into a single Modified line on both sides.
		// Pattern A (our current aligner semantics):
		//   i:   Left=Deleted(real), Right=Added(filler)
		//   i+1: Left=Deleted(filler), Right=Added(real)
		if i+1 < len(result.LeftLines) &&
			leftLine.LineType == buffer.DiffLineDeleted && !leftLine.IsFillerLine &&
			rightLine.LineType == buffer.DiffLineAdded && rightLine.IsFillerLine &&
			result.LeftLines[i+1].LineType == buffer.DiffLineDeleted && result.LeftLines[i+1].IsFillerLine &&
			result.RightLines[i+1].LineType == buffer.DiffLineAdded && !result.RightLines[i+1].IsFillerLine {

			nextRightReal := result.RightLines[i+1]
			wordRanges := computeWordDiff(leftLine.Content, nextRightReal.Content)

			newResult.LeftLines = append(newResult.LeftLines, buffer.DiffLine{
				Content:      leftLine.Content,
				LineType:     buffer.DiffLineModified,
				LeftLine:     leftLine.LeftLine,
				RightLine:    nextRightReal.RightLine,
				IsFillerLine: false,
				WordRanges:   wordRanges,
			})
			newResult.RightLines = append(newResult.RightLines, buffer.DiffLine{
				Content:      nextRightReal.Content,
				LineType:     buffer.DiffLineModified,
				LeftLine:     leftLine.LeftLine,
				RightLine:    nextRightReal.RightLine,
				IsFillerLine: false,
				WordRanges:   wordRanges,
			})
			i += 2
			continue
		}
		// Pattern B (legacy orientation):
		//   i:   Left=Added(real), Right=Deleted(filler)
		//   i+1: Left=Deleted(filler), Right=Added(real)
		if i+1 < len(result.LeftLines) &&
			leftLine.LineType == buffer.DiffLineAdded && !leftLine.IsFillerLine &&
			rightLine.LineType == buffer.DiffLineDeleted && rightLine.IsFillerLine &&
			result.LeftLines[i+1].LineType == buffer.DiffLineDeleted && result.LeftLines[i+1].IsFillerLine &&
			result.RightLines[i+1].LineType == buffer.DiffLineAdded && !result.RightLines[i+1].IsFillerLine {

			nextRightReal := result.RightLines[i+1]
			wordRanges := computeWordDiff(leftLine.Content, nextRightReal.Content)

			newResult.LeftLines = append(newResult.LeftLines, buffer.DiffLine{
				Content:      leftLine.Content,
				LineType:     buffer.DiffLineModified,
				LeftLine:     leftLine.LeftLine,
				RightLine:    nextRightReal.RightLine,
				IsFillerLine: false,
				WordRanges:   wordRanges,
			})
			newResult.RightLines = append(newResult.RightLines, buffer.DiffLine{
				Content:      nextRightReal.Content,
				LineType:     buffer.DiffLineModified,
				LeftLine:     leftLine.LeftLine,
				RightLine:    nextRightReal.RightLine,
				IsFillerLine: false,
				WordRanges:   wordRanges,
			})
			i += 2
			continue
		}

		// Keep the line as-is
		newResult.LeftLines = append(newResult.LeftLines, leftLine)
		newResult.RightLines = append(newResult.RightLines, rightLine)
		i++
	}

	return newResult
}

// isDiffViewActive checks if any windows contain diff views (in their Views list)
func (e *Editor) isDiffViewActive() bool {
	for _, window := range e.windows {
		if window == nil {
			continue
		}
		for _, v := range window.Views {
			if v != nil && v.IsDiff() {
				return true
			}
		}
		// Legacy: check buffer DiffHints
		if window.Buffer != nil && e.isDiffBuffer(window.Buffer) {
			return true
		}
	}
	return false
}

// exitDiffViewIfActive closes the diff buffer pair if a diff view is active.
func (e *Editor) exitDiffViewIfActive() {
	// Search all windows' Views lists for a DiffBufferView
	for _, window := range e.windows {
		if window == nil {
			continue
		}
		var dv *DiffBufferView
		dvIdx := -1
		for i, v := range window.Views {
			if d, ok := v.(*DiffBufferView); ok {
				dv = d
				dvIdx = i
				break
			}
		}
		if dv == nil {
			continue
		}
		// Remove DiffBufferView from Views list
		window.Views = append(window.Views[:dvIdx], window.Views[dvIdx+1:]...)
		// Remove diff buffers from editor
		for _, buf := range dv.AllBuffers() {
			if buf != nil {
				buf.DiffHints = nil
				buf.Highlighter = nil
				delete(e.buffers, buf.FilePath)
				e.bufferOrder = removeFromSlice(e.bufferOrder, buf.FilePath)
				window.removeBufferFromWindow(buf)
			}
		}
		// Restore SingleBufferView for the original file
		if origBuf := e.buffers[dv.RealPath]; origBuf != nil {
			if window.findViewForBuffer(origBuf) < 0 {
				window.Views = append(window.Views, &SingleBufferView{Buf: origBuf})
			}
		}
		// If window has no buffer left, try to find one
		if window.Buffer == nil {
			if origBuf := e.buffers[dv.RealPath]; origBuf != nil {
				window.setActiveBuffer(origBuf)
				e.switchToBuffer(origBuf)
			} else if len(e.buffers) > 0 {
				for _, buf := range e.buffers {
					window.setActiveBuffer(buf)
					e.switchToBuffer(buf)
					break
				}
			} else {
				newBuf := e.createUnnamedBuffer()
				window.setActiveBuffer(newBuf)
				e.switchToBuffer(newBuf)
			}
		} else {
			e.switchToBuffer(window.Buffer)
		}
		e.recomputeWindowsLayout()
		return
	}

	// Legacy two-window fallback
	if !e.isDiffViewActive() {
		return
	}

	diffBuf := e.CurrentBuffer()
	if diffBuf == nil || !e.isDiffBuffer(diffBuf) {
		diffBuf = nil
		for _, window := range e.windows {
			if window != nil && window.Buffer != nil && e.isDiffBuffer(window.Buffer) {
				diffBuf = window.Buffer
				break
			}
		}
	}

	if diffBuf == nil {
		return
	}

	_ = e.closeDiffBufferPair(diffBuf.FilePath)
}

// createBufferContent creates the actual text content for a buffer from buffer.DiffLines
func createBufferContent(diffLines []buffer.DiffLine) string {
	var lines []string
	for _, diffLine := range diffLines {
		lines = append(lines, diffLine.Content)
	}
	return strings.Join(lines, "\n")
}

// refreshDiffBuffersForRealPath updates existing diff buffers (path@HEAD and path@current)
// after the working file content changes. It recomputes alignment and updates
// iterators, highlighters, and diff metadata in place without creating windows.
func (e *Editor) refreshDiffBuffersForRealPath(path string, currentContent string) {
	if path == "" {
		return
	}

	// Try DiffBufferView first
	var headBuf, curBuf *buffer.Buffer
	for _, w := range e.windows {
		if w == nil {
			continue
		}
		if dv, ok := w.View.(*DiffBufferView); ok && dv.RealPath == path {
			headBuf = dv.LeftBuffer
			curBuf = dv.RightBuffer
			break
		}
	}

	// Legacy fallback: name convention
	if headBuf == nil || curBuf == nil {
		headName := path + "@HEAD"
		curName := path + "@current"
		headBuf = e.buffers[headName]
		curBuf = e.buffers[curName]
	}

	if headBuf == nil || curBuf == nil {
		return
	}

	// Get HEAD content
	prevContent, err := e.proxyManager.GetGitFileVersion(path, "HEAD")
	if err != nil {
		return
	}

	// Compute aligned diff
	diffResult := e.computeDiffViaProxy(prevContent, currentContent)
	alignedHead := createBufferContent(diffResult.LeftLines)
	alignedCur := createBufferContent(diffResult.RightLines)

	// Preserve cursor and view lines
	prevHeadCurLine := headBuf.CursorIterator.LineNumber()
	prevHeadViewLine := headBuf.ViewIterator.LineNumber()
	prevCurCurLine := curBuf.CursorIterator.LineNumber()
	prevCurViewLine := curBuf.ViewIterator.LineNumber()
	prevCurX := curBuf.CursorX

	// Update ropes
	headBuf.Rope = buffer.NewRopeFromString(alignedHead)
	curBuf.Rope = buffer.NewRopeFromString(alignedCur)

	// Clamp and restore iterators
	maxHead := headBuf.LineCount - 1
	if prevHeadCurLine > maxHead {
		prevHeadCurLine = maxHead
	}
	maxCur := curBuf.LineCount - 1
	if prevCurCurLine > maxCur {
		prevCurCurLine = maxCur
	}
	headBuf.ViewIterator = buffer.NewBufferIteratorAtLine(headBuf, prevHeadViewLine)
	headBuf.CursorIterator = buffer.NewBufferIteratorAtLine(headBuf, prevHeadCurLine)
	curBuf.ViewIterator = buffer.NewBufferIteratorAtLine(curBuf, prevCurViewLine)
	curBuf.CursorIterator = buffer.NewBufferIteratorAtLine(curBuf, prevCurCurLine)
	// Restore cursor X conservatively within line bounds
	line := curBuf.CursorIterator.Line()
	if prevCurX > len(line) {
		prevCurX = len(line)
	}
	curBuf.CursorX = prevCurX

	// Update diff metadata
	headBuf.DiffHints = diffResult.LeftLines
	curBuf.DiffHints = diffResult.RightLines

	// Update highlighters
	headHL := syntax.NewHighlighter()
	headHL.SetLanguageFromFileAndContent(path, []byte(alignedHead))
	headHL.ParseFull([]byte(alignedHead))
	headBuf.Highlighter = headHL

	curHL := syntax.NewHighlighter()
	curHL.SetLanguageFromFileAndContent(path, []byte(alignedCur))
	curHL.ParseFull([]byte(alignedCur))
	curBuf.Highlighter = curHL

	// Keep GitChanges in sync for the right pane
	if baseBuf, exists := e.buffers[path]; exists {
		curBuf.GitChanges = baseBuf.GitChanges
	}

	// Ensure paired cursor/view sync respects current active buffer
	e.syncDiffCursors()
}
