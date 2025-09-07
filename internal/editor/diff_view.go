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
	bufferA := buffer.NewBuffer(previousContent, filePath+"@HEAD", config.IndentTabs, 4)
	bufferB := buffer.NewBuffer(currentContent, filePath+"@current", config.IndentTabs, 4)

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
		ds.HighlighterA.SetLanguageFromFile(filePath)
		ds.HighlighterA.ParseFull([]byte(previousContent))

		ds.HighlighterB = syntax.NewHighlighter()
		ds.HighlighterB.SetLanguageFromFile(filePath)
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

// syncScrollInDiffMode synchronizes scrolling between both panes in diff mode
func (e *Editor) syncScrollInDiffMode() {
	if len(e.windows) != 2 {
		return
	}

	// Get the current view line from the active buffer
	if e.CurrentBuffer() == nil {
		return
	}

	currentViewLine := e.CurrentBuffer().ViewIterator.LineNumber()

	// Update both windows to reflect the synchronized view
	if e.windows[0] != nil && e.windows[0].Buffer != nil {
		e.windows[0].ViewLine = currentViewLine
	}
	if e.windows[1] != nil && e.windows[1].Buffer != nil {
		e.windows[1].ViewLine = currentViewLine
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

	filePath := e.CurrentBuffer().FilePath
	currentContent := e.CurrentBuffer().Rope.String()
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

	LL.Debug("StartDiffViewForCurrentBuffer: Aligned content created - Head lines: %d, Current lines: %d",
		len(diffResult.LeftLines), len(diffResult.RightLines))

	// Create HEAD buffer name
	headBufferName := filePath + "@HEAD"
	currentBufferName := filePath + "@current"

	// Create or update HEAD buffer
	if headBuffer, exists := e.buffers[headBufferName]; exists {
		// Update existing buffer
		headBuffer.Rope = buffer.NewRopeFromString(alignedHeadContent)
		headBuffer.CursorX = 0
		headBuffer.ViewIterator = buffer.NewBufferIterator(headBuffer)
		headBuffer.CursorIterator = buffer.NewBufferIterator(headBuffer)
	} else {
		// Create new HEAD buffer
		headBuffer := buffer.NewBuffer(alignedHeadContent, headBufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		headBuffer.IsReadOnly = true
		headBuffer.IsInternal = false // Can be moved between windows
		e.buffers[headBufferName] = headBuffer
		e.bufferOrder = append(e.bufferOrder, headBufferName)
	}

	// Ensure syntax highlighter exists and is updated for HEAD buffer
	headBuffer := e.buffers[headBufferName]
	if e.bufferHighlighters == nil {
		e.bufferHighlighters = make(map[*buffer.Buffer]*syntax.Highlighter)
	}

	// Always update or create the highlighter with the new content
	headHighlighter := syntax.NewHighlighter()
	headHighlighter.SetLanguageFromFile(filePath) // Use original file path for language detection
	headHighlighter.ParseFull([]byte(alignedHeadContent))
	e.bufferHighlighters[headBuffer] = headHighlighter
	headBuffer.Highlighter = headHighlighter // Set the highlighter directly on the buffer

	// Store diff line information for head buffer
	e.diffLineInfo[headBuffer] = diffResult.LeftLines

	// Get current cursor position from the original buffer
	currentCursorLine := e.CurrentBuffer().CursorIterator.LineNumber()

	// Create or update current buffer copy
	if currentBuffer, exists := e.buffers[currentBufferName]; exists {
		// Update existing buffer
		currentBuffer.Rope = buffer.NewRopeFromString(alignedCurrentContent)
		currentBuffer.CursorX = e.CurrentBuffer().CursorX
		currentBuffer.ViewIterator = buffer.NewBufferIteratorAtLine(currentBuffer, currentCursorLine)
		currentBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(currentBuffer, currentCursorLine)
	} else {
		// Create new current buffer copy
		currentBuffer := buffer.NewBuffer(alignedCurrentContent, currentBufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		currentBuffer.IsReadOnly = false
		currentBuffer.IsInternal = false // Can be moved between windows
		currentBuffer.CursorX = e.CurrentBuffer().CursorX
		currentBuffer.ViewIterator = buffer.NewBufferIteratorAtLine(currentBuffer, currentCursorLine)
		currentBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(currentBuffer, currentCursorLine)
		e.buffers[currentBufferName] = currentBuffer
		e.bufferOrder = append(e.bufferOrder, currentBufferName)
	}

	// Ensure syntax highlighter exists and is updated for current buffer
	currentBuffer := e.buffers[currentBufferName]

	// Always update or create the highlighter with the new content
	currentHighlighter := syntax.NewHighlighter()
	currentHighlighter.SetLanguageFromFile(filePath) // Use original file path for language detection
	currentHighlighter.ParseFull([]byte(alignedCurrentContent))
	e.bufferHighlighters[currentBuffer] = currentHighlighter
	currentBuffer.Highlighter = currentHighlighter // Set the highlighter directly on the buffer

	// Store diff line information for current buffer
	e.diffLineInfo[currentBuffer] = diffResult.RightLines

    // Place HEAD in the same window as the original buffer (do NOT remove
    // the original from that window's buffer list), and open the current
    // diff copy in a window to the right.

    // 1) Add HEAD buffer to the active (left) window and make it active there.
    if e.activeWindow < 0 || e.activeWindow >= len(e.windows) {
        // Safety: ensure we have at least one window
        e.windows = append(e.windows, newWindow(nil))
        e.activeWindow = len(e.windows) - 1
    }
    leftIdx := e.activeWindow
    e.windows[leftIdx].addBufferToWindow(e.buffers[headBufferName])

    // 2) Create or insert a window immediately to the right for the current copy.
    // Insert at position leftIdx+1 to keep relative order of existing windows.
    rightWin := newWindow(e.buffers[currentBufferName])
    if leftIdx+1 >= len(e.windows) {
        e.windows = append(e.windows, rightWin)
    } else {
        e.windows = append(e.windows[:leftIdx+1], append([]*Window{rightWin}, e.windows[leftIdx+1:]...)...)
    }

    // Activate the right diff window (current buffer copy)
    e.activeWindow = leftIdx + 1
    e.switchToBuffer(e.buffers[currentBufferName])

    // Recompute window layout to apply diff-specific width rules
    e.recomputeWindowsLayout()

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
func (e *Editor) diffCurrentBuffer(args []string) error {
	return e.StartDiffViewForCurrentBuffer()
}

// isDiffBuffer checks if a buffer is part of a diff pair
func (e *Editor) isDiffBuffer(buf *buffer.Buffer) bool {
	if buf == nil {
		return false
	}
	return strings.HasSuffix(buf.FilePath, "@HEAD") || strings.HasSuffix(buf.FilePath, "@current")
}

// getDiffPairBuffer returns the paired diff buffer if it exists
func (e *Editor) getDiffPairBuffer(buf *buffer.Buffer) *buffer.Buffer {
	if buf == nil {
		return nil
	}

	if strings.HasSuffix(buf.FilePath, "@HEAD") {
		// This is HEAD buffer, find the @current pair
		currentPath := strings.TrimSuffix(buf.FilePath, "@HEAD") + "@current"
		return e.buffers[currentPath]
	} else if strings.HasSuffix(buf.FilePath, "@current") {
		// This is current buffer, find the @HEAD pair
		headPath := strings.TrimSuffix(buf.FilePath, "@current") + "@HEAD"
		return e.buffers[headPath]
	}

	return nil
}

// syncDiffCursors synchronizes cursor Y positions and ViewIterator between diff buffer pairs
func (e *Editor) syncDiffCursors() {
	if e.CurrentBuffer() == nil || !e.isDiffBuffer(e.CurrentBuffer()) {
		return
	}

	pairBuffer := e.getDiffPairBuffer(e.CurrentBuffer())
	if pairBuffer == nil {
		return
	}

	// Get current cursor Y position (line number)
	currentLine := e.CurrentBuffer().CursorIterator.LineNumber()

	// Sync the paired buffer's cursor Y position (but not X position)
	pairBuffer.CursorIterator = buffer.NewBufferIteratorAtLine(pairBuffer, currentLine)
	// Note: We deliberately do NOT sync CursorX as content may differ

	// Sync the ViewIterator for scrolling - this is critical for proper scrolling
	currentViewLine := e.CurrentBuffer().ViewIterator.LineNumber()
	pairBuffer.ViewIterator = buffer.NewBufferIteratorAtLine(pairBuffer, currentViewLine)

	// Also sync the cursor and view position in the paired window
	for _, window := range e.windows {
		if window.Buffer == pairBuffer {
			window.CursorLine = currentLine
			// Note: We deliberately do NOT sync window.CursorX
			window.ViewLine = currentViewLine // Sync ViewLine for proper scrolling
			break
		}
	}
}

// DiffLine represents a single line in the diff with its status
type DiffLine struct {
	Content      string
	LineType     DiffLineType
	LeftLine     int         // Line number in left file (0-based), -1 if not present
	RightLine    int         // Line number in right file (0-based), -1 if not present
	IsFillerLine bool        // True for empty lines added for alignment
	WordRanges   []DiffRange // Character ranges that differ within the line (for DiffLineModified)
}

// DiffRange represents a character range within a line that differs
type DiffRange struct {
	Start int // Start character position (inclusive)
	End   int // End character position (exclusive)
}

// DiffLineType indicates the type of difference
type DiffLineType int

const (
	DiffLineContext  DiffLineType = iota // Unchanged line
	DiffLineAdded                        // Line added in right file
	DiffLineDeleted                      // Line removed from left file
	DiffLineModified                     // Line changed between files
)

// DiffResult contains the result of comparing two texts
type DiffResult struct {
    LeftLines  []DiffLine // Lines for left side display
    RightLines []DiffLine // Lines for right side display
}

// computeDiffViaProxy prefers proxy-backed (difflib) alignment; falls back internally
func (e *Editor) computeDiffViaProxy(leftText, rightText string) *DiffResult {
    if e == nil || e.proxyManager == nil {
        return computeDiff(leftText, rightText)
    }
    aligned, err := e.proxyManager.ComputeTextDiff(leftText, rightText)
    if err != nil || len(aligned) == 0 {
        return computeDiff(leftText, rightText)
    }
    res := &DiffResult{LeftLines: make([]DiffLine, 0, len(aligned)), RightLines: make([]DiffLine, 0, len(aligned))}
    curL, curR := 0, 0
    for _, it := range aligned {
        switch it.Tag {
        case "equal":
            res.LeftLines = append(res.LeftLines, DiffLine{Content: it.Left, LineType: DiffLineContext, LeftLine: curL, RightLine: curR, IsFillerLine: false})
            res.RightLines = append(res.RightLines, DiffLine{Content: it.Right, LineType: DiffLineContext, LeftLine: curL, RightLine: curR, IsFillerLine: false})
            curL++
            curR++
        case "insert":
            res.LeftLines = append(res.LeftLines, DiffLine{Content: "", LineType: DiffLineDeleted, LeftLine: -1, RightLine: -1, IsFillerLine: true})
            res.RightLines = append(res.RightLines, DiffLine{Content: it.Right, LineType: DiffLineAdded, LeftLine: -1, RightLine: curR, IsFillerLine: false})
            curR++
        case "delete":
            res.LeftLines = append(res.LeftLines, DiffLine{Content: it.Left, LineType: DiffLineAdded, LeftLine: curL, RightLine: -1, IsFillerLine: false})
            res.RightLines = append(res.RightLines, DiffLine{Content: "", LineType: DiffLineDeleted, LeftLine: -1, RightLine: -1, IsFillerLine: true})
            curL++
        case "replace":
            wr := computeWordDiff(it.Left, it.Right)
            res.LeftLines = append(res.LeftLines, DiffLine{Content: it.Left, LineType: DiffLineModified, LeftLine: curL, RightLine: curR, IsFillerLine: false, WordRanges: wr})
            res.RightLines = append(res.RightLines, DiffLine{Content: it.Right, LineType: DiffLineModified, LeftLine: curL, RightLine: curR, IsFillerLine: false, WordRanges: wr})
            curL++
            curR++
        default:
            // Fallback to context to avoid crashing on unknown tags
            res.LeftLines = append(res.LeftLines, DiffLine{Content: it.Left, LineType: DiffLineContext, LeftLine: curL, RightLine: curR, IsFillerLine: false})
            res.RightLines = append(res.RightLines, DiffLine{Content: it.Right, LineType: DiffLineContext, LeftLine: curL, RightLine: curR, IsFillerLine: false})
            curL++
            curR++
        }
    }
    return res
}

// computeDiff compares two texts and returns a DiffResult with aligned lines
// computeWordDiff computes character-level differences between two strings
// Uses a simple prefix/suffix matching approach for better accuracy
func computeWordDiff(left, right string) []DiffRange {
	var ranges []DiffRange

	leftRunes := []rune(left)
	rightRunes := []rune(right)

	// Handle simple cases
	if len(leftRunes) == 0 {
		// Everything in right is new
		if len(rightRunes) > 0 {
			ranges = append(ranges, DiffRange{Start: 0, End: len(rightRunes)})
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
		ranges = append(ranges, DiffRange{Start: rightStart, End: rightEnd})
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

func computeDiff(leftText, rightText string) *DiffResult {
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

// alignLines creates aligned DiffResult ensuring both sides have equal line counts
func alignLines(leftLines, rightLines []string) *DiffResult {
	result := &DiffResult{
		LeftLines:  make([]DiffLine, 0),
		RightLines: make([]DiffLine, 0),
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
            result.LeftLines = append(result.LeftLines, DiffLine{
                Content:      line,
                LineType:     DiffLineContext,
                LeftLine:     leftIdx,
                RightLine:    rightIdx,
                IsFillerLine: false,
            })
            result.RightLines = append(result.RightLines, DiffLine{
                Content:      line,
                LineType:     DiffLineContext,
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
            // Show green (Added) on the left, red (Deleted filler) on the right
            result.LeftLines = append(result.LeftLines, DiffLine{
                Content:      leftLines[leftIdx],
                LineType:     DiffLineAdded,
                LeftLine:     leftIdx,
                RightLine:    -1,
                IsFillerLine: false,
            })
            result.RightLines = append(result.RightLines, DiffLine{
                Content:      "",
                LineType:     DiffLineDeleted, // Filler opposite of left-side Added
                LeftLine:     -1,
                RightLine:    -1,
                IsFillerLine: true,
            })
            leftIdx++
        } else if rightIdx < len(rightLines) {
            // Line only in right (present in current, missing in HEAD)
            // Show red (Deleted filler) on the left, green (Added) on the right
            result.LeftLines = append(result.LeftLines, DiffLine{
                Content:      "",
                LineType:     DiffLineDeleted, // Filler opposite of right-side Added
                LeftLine:     -1,
                RightLine:    -1,
                IsFillerLine: true,
            })
            result.RightLines = append(result.RightLines, DiffLine{
                Content:      rightLines[rightIdx],
                LineType:     DiffLineAdded,
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
func detectModifications(result *DiffResult) *DiffResult {
	if len(result.LeftLines) != len(result.RightLines) {
		return result // Safety check - should always be equal
	}

	newResult := &DiffResult{
		LeftLines:  make([]DiffLine, 0, len(result.LeftLines)),
		RightLines: make([]DiffLine, 0, len(result.RightLines)),
	}

	i := 0
    for i < len(result.LeftLines) {
        leftLine := result.LeftLines[i]
        rightLine := result.RightLines[i]

        // Detect a delete+insert pair represented with asymmetric types:
        // i:   Left=Added (real), Right=Deleted (filler)
        // i+1: Left=Deleted (filler), Right=Added (real)
        if i+1 < len(result.LeftLines) &&
            leftLine.LineType == DiffLineAdded && !leftLine.IsFillerLine &&
            rightLine.LineType == DiffLineDeleted && rightLine.IsFillerLine &&
            result.LeftLines[i+1].LineType == DiffLineDeleted && result.LeftLines[i+1].IsFillerLine &&
            result.RightLines[i+1].LineType == DiffLineAdded && !result.RightLines[i+1].IsFillerLine {

            nextRightLine := result.RightLines[i+1]
            wordRanges := computeWordDiff(leftLine.Content, nextRightLine.Content)

            newResult.LeftLines = append(newResult.LeftLines, DiffLine{
                Content:      leftLine.Content,
                LineType:     DiffLineModified,
                LeftLine:     leftLine.LeftLine,
                RightLine:    nextRightLine.RightLine,
                IsFillerLine: false,
                WordRanges:   wordRanges,
            })
            newResult.RightLines = append(newResult.RightLines, DiffLine{
                Content:      nextRightLine.Content,
                LineType:     DiffLineModified,
                LeftLine:     leftLine.LeftLine,
                RightLine:    nextRightLine.RightLine,
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

// isDiffViewActive checks if any windows contain diff buffers (indicating active diff view)
func (e *Editor) isDiffViewActive() bool {
	for _, window := range e.windows {
		if window != nil && window.Buffer != nil && e.isDiffBuffer(window.Buffer) {
			return true
		}
	}
	return false
}

// findDiffWindowPair returns the indices of the two windows containing the diff pair buffers
// Returns (-1, -1) if no diff pair is found
func (e *Editor) findDiffWindowPair() (int, int) {
	leftIdx := -1
	rightIdx := -1

	for i, window := range e.windows {
		if window != nil && window.Buffer != nil {
			if strings.HasSuffix(window.Buffer.FilePath, "@HEAD") {
				leftIdx = i
			} else if strings.HasSuffix(window.Buffer.FilePath, "@current") {
				rightIdx = i
			}
		}
	}

	// Only return valid indices if both parts of the diff are found
	if leftIdx >= 0 && rightIdx >= 0 {
		return leftIdx, rightIdx
	}
	return -1, -1
}

// createBufferContent creates the actual text content for a buffer from DiffLines
func createBufferContent(diffLines []DiffLine) string {
	var lines []string
	for _, diffLine := range diffLines {
		lines = append(lines, diffLine.Content)
	}
	return strings.Join(lines, "\n")
}
