package buffer

import (
	"fmt"
	"strings"
	"veda/internal/config"
	"veda/internal/proxy"
	"veda/internal/syntax"
)

// calculateLineCount returns the number of lines in the rope,
// accounting for trailing newlines. A trailing newline doesn't
// create a new line; it's part of the last line.
func calculateLineCount(rope *Rope) int {
	if rope.Length() == 0 {
		return 1 // Empty buffer has 1 line
	}

	// Number of lines is newline count + 1
	return rope.root.newlineCount + 1
}

// SignatureHelp represents function signature information from LSP
type SignatureHelp struct {
	Label           string          `json:"label"`           // The function signature
	Documentation   string          `json:"documentation"`   // Optional documentation
	Parameters      []ParameterInfo `json:"parameters"`      // Information about each parameter
	ActiveParameter int             `json:"activeParameter"` // Which parameter is currently active
}

// ParameterInfo represents information about a function parameter
type ParameterInfo struct {
	Label         string `json:"label"`         // Parameter name and type (if label is string)
	LabelStart    int    `json:"labelStart"`    // Start position in signature (if label is range)
	LabelEnd      int    `json:"labelEnd"`      // End position in signature (if label is range)
	Documentation string `json:"documentation"` // Optional parameter documentation
}

// Buffer represents a complete text buffer with metadata
type Buffer struct {
	*Rope // Embed the rope data structure

	// File information
	FilePath   string // Unique buffer identifier (legacy); may be a virtual name
	BufferName string // Human-friendly buffer name for UI (optional)
	RealPath   string // Real filesystem path for I/O (empty for virtual/temporary)
	IsModified bool

	// Cursor and view state
	CursorX        int
	ViewIterator   *BufferIterator // Current view position iterator
	CursorIterator *BufferIterator // Current cursor position iterator

	// Caching
	LineCount int // Cache of the number of lines

	// Indentation settings for this buffer
	Indentation IndentationInfo

	// Line number display width (e.g., 4 for files with 1000-9999 lines)
	LineNumberWidth int

	// Generation number for iterator validation
	Generation uint64

	// LSP availability
	HasLSP    bool // Whether LSP is available for this buffer
	LSPFailed bool // Whether LSP startup failed for this buffer

	// Function signature state
	CurrentSignature    *SignatureHelp // The current function signature from LSP
	SignatureGeneration uint64         // Generation number for signature help requests

	// Folding ranges from LSP
	FoldingRanges []proxy.FoldingRange // Cached folding ranges for this buffer

	// Undo/Redo history
	History *History // Command history for undo/redo operations

	// Visual mode state
	VisualAnchorLine   int // Line number where visual selection started
	VisualAnchorX      int // X position where visual selection started
	VisualAnchorOffset int // Byte offset where visual selection started

	// Git changes tracking
	GitChanges *proxy.FileChanges // Git changes for this buffer (added, deleted, modified lines)
	GitBranch  string             // Current git branch for this buffer

	// Diagnostics from LSP (1-based lines)
	Diagnostics []proxy.Diagnostic

	// Buffer flags
	IsReadOnly bool // Whether this buffer is read-only
	IsInternal bool // Whether this is an internal buffer (not tied to a file)

	// LSP notification callback
	OnContentChange func() // Called when buffer content changes, for LSP notifications

	// Syntax highlighting
	Highlighter *syntax.Highlighter

	// Context display cache (per-buffer)
	ContextCache []int // Cached context line numbers for this buffer
	ContextLine  int   // Line number for which context was cached (for this buffer)

	// Diff view hints (populated when buffer is part of a diff view)
	DiffHints []DiffLine

	// Session change tracking
	ChangedLines map[int]bool // Set of 0-based line numbers that were modified in this session
}

// IndentationInfo holds information about file indentation
type IndentationInfo struct {
	Type  config.IndentationType
	Width int // Number of spaces per indent level (for spaces) or 1 for tabs
}

// String returns a string representation for the status bar
func (i *IndentationInfo) String() string {
	if i.Type == config.IndentTabs {
		return "Tabs"
	}
	return fmt.Sprintf("Spaces:%d", i.Width)
}

// NewBuffer creates a new buffer with default settings.
// bufferPath is the logical/display identifier (may be virtual), realPath is the filesystem path for I/O (empty for virtual/unnamed buffers).
func NewBuffer(content string, bufferPath string, realPath string, defaultIndentType config.IndentationType, defaultIndentWidth int) *Buffer {
	rope := NewRopeFromString(content)
	b := &Buffer{
		Rope:           rope,
		FilePath:       bufferPath,
		BufferName:     bufferPath,
		RealPath:       realPath,
		IsModified:     false,
		CursorX:        0,
		ViewIterator:   nil, // Will be created when buffer is first used
		CursorIterator: nil, // Will be created when buffer is first used
		LineCount:      calculateLineCount(rope),
		Indentation: IndentationInfo{
			Type:  defaultIndentType,
			Width: defaultIndentWidth,
		},
		Generation:         0,            // Start with generation 0
		History:            NewHistory(), // Initialize undo/redo history
		VisualAnchorLine:   -1,           // -1 means no visual selection
		VisualAnchorX:      -1,
		VisualAnchorOffset: -1,
	}
	// Initialize per-buffer highlighter
	b.Highlighter = syntax.NewHighlighter()
	// Initialize context cache fields
	b.ContextCache = nil
	b.ContextLine = -1
	// Initialize changed lines tracking
	b.ChangedLines = make(map[int]bool)
	b.UpdateLineNumberWidth()
	// Create initial iterators
	b.ViewIterator = NewBufferIterator(b)
	b.CursorIterator = NewBufferIterator(b)
	return b
}

// GetDiagnosticForLine returns the highest-severity diagnostic for a 1-based line.
// Returns (nil, false) if no diagnostic exists for that line.
func (b *Buffer) GetDiagnosticForLine(lineNum int) (*proxy.Diagnostic, bool) {
	if b == nil || len(b.Diagnostics) == 0 {
		return nil, false
	}
	// Higher rank is more severe
	severityRank := func(s string) int {
		switch s {
		case "error":
			return 4
		case "warning":
			return 3
		case "info":
			return 2
		case "hint":
			return 1
		default:
			return 0
		}
	}
	bestIdx := -1
	bestRank := 0
	for i := range b.Diagnostics {
		d := &b.Diagnostics[i]
		if d.Line != lineNum {
			continue
		}
		r := severityRank(d.Severity)
		if r > bestRank {
			bestRank = r
			bestIdx = i
		}
	}
	if bestIdx >= 0 {
		return &b.Diagnostics[bestIdx], true
	}
	return nil, false
}

// MarkModified marks the buffer as modified
func (b *Buffer) MarkModified() {
	b.IsModified = true
	b.LineCount = calculateLineCount(b.Rope)
	b.UpdateLineNumberWidth()
}

// MarkSaved marks the buffer as saved (not modified)
func (b *Buffer) MarkSaved() {
	b.IsModified = false
}

// HasFile returns true if this buffer is backed by a real file
func (b *Buffer) HasFile() bool {
	return b.RealPath != ""
}

// DisplayName returns a user-friendly name for the buffer
func (b *Buffer) DisplayName() string {
	if b.BufferName != "" {
		return b.BufferName
	}
	if b.FilePath == "" {
		return "[No Name]"
	}
	return b.FilePath
}

// FileName returns just the filename without the path
func (b *Buffer) FileName() string {
	if b.FilePath == "" {
		return "[No Name]"
	}
	// Find the last slash
	lastSlash := -1
	for i := len(b.FilePath) - 1; i >= 0; i-- {
		if b.FilePath[i] == '/' {
			lastSlash = i
			break
		}
	}
	if lastSlash >= 0 && lastSlash < len(b.FilePath)-1 {
		return b.FilePath[lastSlash+1:]
	}
	return b.FilePath
}

// SetIndentationType sets the indentation type for this buffer
func (b *Buffer) SetIndentationType(indentType config.IndentationType) {
	b.Indentation.Type = indentType
	if indentType == config.IndentTabs {
		b.Indentation.Width = 1
	}
}

// SetIndentationWidth sets the indentation width (for spaces)
func (b *Buffer) SetIndentationWidth(width int) {
	if width > 0 && width <= 8 {
		b.Indentation.Width = width
		if b.Indentation.Type == config.IndentTabs {
			b.Indentation.Type = config.IndentSpaces
		}
	}
}

// GetIndentationString returns the string to use for one level of indentation
func (b *Buffer) GetIndentationString() string {
	if b.Indentation.Type == config.IndentTabs {
		return "\t"
	}
	return strings.Repeat(" ", b.Indentation.Width)
}

// LineIndentInfo holds information about a line's indentation
type LineIndentInfo struct {
	Level          int    // The indentation level (in terms of file's indent width)
	Spaces         int    // Total number of spaces
	Tabs           int    // Total number of tabs
	IsContinuation bool   // Whether this appears to be a continuation line
	ActualPrefix   string // The actual whitespace prefix
}

// GetLineIndentInfo analyzes a line and returns its indentation information
func (b *Buffer) GetLineIndentInfo(line string) LineIndentInfo {
	info := LineIndentInfo{}

	// Count leading whitespace
	for i, ch := range line {
		if ch == ' ' {
			info.Spaces++
		} else if ch == '\t' {
			info.Tabs++
		} else {
			info.ActualPrefix = line[:i]
			break
		}
	}

	// If entire line is whitespace
	if info.ActualPrefix == "" && len(line) > 0 && (info.Spaces > 0 || info.Tabs > 0) {
		info.ActualPrefix = line
	}

	// Calculate level based on file's indentation settings
	if b.Indentation.Type == config.IndentTabs {
		// For tabs, each tab is one level
		info.Level = info.Tabs
		// If there are spaces mixed with tabs, it might be continuation
		if info.Spaces > 0 {
			info.IsContinuation = true
		}
	} else {
		// For spaces, divide by the indent width
		if b.Indentation.Width > 0 {
			info.Level = info.Spaces / b.Indentation.Width
			// Check if it's a continuation line (not a multiple of indent width)
			if info.Spaces%b.Indentation.Width != 0 {
				info.IsContinuation = true
			}
		}
		// Mixed tabs and spaces in a spaces file suggests continuation
		if info.Tabs > 0 {
			info.IsContinuation = true
		}
	}

	return info
}

// GetCurrentLineIndentInfo returns detailed indentation info for the line at the iterator position
func (b *Buffer) GetCurrentLineIndentInfo(iter *BufferIterator) LineIndentInfo {
	if iter.IsEOF() {
		return LineIndentInfo{}
	}

	return b.GetLineIndentInfo(iter.Line())
}

// UpdateLineNumberWidth calculates and caches the width needed for line numbers
func (b *Buffer) UpdateLineNumberWidth() {
	if b.LineCount == 0 {
		// Minimum width: 1 digit + 1 padding + 2 indicator columns
		b.LineNumberWidth = 4
	} else {
		// Calculate the width needed for the largest line number
		digitsWidth := len(fmt.Sprintf("%d", b.LineCount))
		// Total = digits + 1 padding + 2 indicator columns
		b.LineNumberWidth = digitsWidth + 3
	}
}

// Insert inserts text at the given position and increments the generation
func (b *Buffer) Insert(pos int, text string) {
	// Track which line is being modified
	lineNum := b.GetLineAtPosition(pos)

	b.Rope.Insert(pos, text)
	b.Generation++ // Increment generation when buffer is modified
	b.MarkModified()

	// Mark the line(s) as changed
	b.MarkLineChanged(lineNum)
	// If text contains newlines, mark those new lines as changed too
	newlineCount := 0
	for _, ch := range text {
		if ch == '\n' {
			newlineCount++
		}
	}
	if newlineCount > 0 {
		b.MarkLinesChanged(lineNum, lineNum+newlineCount)
	}

	// Notify LSP if callback is set
	if b.OnContentChange != nil {
		b.OnContentChange()
	}
}

// Delete removes characters between start and end positions and increments the generation
func (b *Buffer) Delete(start, end int) {
	// Track which lines are being modified
	startLine := b.GetLineAtPosition(start)

	b.Rope.Delete(start, end)
	b.Generation++ // Increment generation when buffer is modified
	b.MarkModified()

	// Mark the affected line(s) as changed
	// After deletion, we only mark the line where deletion occurred
	b.MarkLineChanged(startLine)

	// Notify LSP if callback is set
	if b.OnContentChange != nil {
		b.OnContentChange()
	}
}

// AppendLine appends a line to the end of the buffer
func (b *Buffer) AppendLine(line string) {
	// Temporarily disable read-only for internal buffers
	wasReadOnly := b.IsReadOnly
	if b.IsInternal && b.IsReadOnly {
		b.IsReadOnly = false
	}

	// If buffer is not empty and doesn't end with newline, add one
	if b.Length() > 0 && b.ByteAt(b.Length()-1) != '\n' {
		b.Insert(b.Length(), "\n")
	}

	// Append the line with a newline
	b.Insert(b.Length(), line+"\n")

	// Restore read-only status
	if wasReadOnly {
		b.IsReadOnly = true
	}
}

// MarkLineChanged marks a line as changed in the current session
func (b *Buffer) MarkLineChanged(lineNumber int) {
	if b.ChangedLines == nil {
		b.ChangedLines = make(map[int]bool)
	}
	b.ChangedLines[lineNumber] = true
}

// MarkLinesChanged marks a range of lines as changed in the current session
func (b *Buffer) MarkLinesChanged(startLine, endLine int) {
	if b.ChangedLines == nil {
		b.ChangedLines = make(map[int]bool)
	}
	for line := startLine; line <= endLine; line++ {
		b.ChangedLines[line] = true
	}
}

// GetLineAtPosition returns the 0-based line number at the given byte position
func (b *Buffer) GetLineAtPosition(pos int) int {
	// Count newlines up to the position
	lineNum := 0
	for i := 0; i < pos && i < b.Length(); i++ {
		if b.ByteAt(i) == '\n' {
			lineNum++
		}
	}
	return lineNum
}

// RemoveTrailingSpacesFromChangedLines removes trailing whitespace from all lines marked as changed
func (b *Buffer) RemoveTrailingSpacesFromChangedLines() {
	if len(b.ChangedLines) == 0 {
		return
	}

	// Save cursor and view positions before modifications
	var savedCursorLine, savedCursorX int
	var savedViewLine int
	if b.CursorIterator != nil && b.CursorIterator.Generation == b.Generation {
		savedCursorLine = b.CursorIterator.LineNumber()
		savedCursorX = b.CursorX
	}
	if b.ViewIterator != nil && b.ViewIterator.Generation == b.Generation {
		savedViewLine = b.ViewIterator.LineNumber()
	}

	// Convert map to sorted slice to process lines in order
	lines := make([]int, 0, len(b.ChangedLines))
	for line := range b.ChangedLines {
		lines = append(lines, line)
	}
	// Sort lines in reverse order to avoid line number shifts
	for i := 0; i < len(lines); i++ {
		for j := i + 1; j < len(lines); j++ {
			if lines[i] < lines[j] {
				lines[i], lines[j] = lines[j], lines[i]
			}
		}
	}

	// Process each changed line
	for _, lineNum := range lines {
		iter := NewBufferIteratorAtLine(b, lineNum)
		if iter.IsEOF() {
			continue
		}

		line := iter.Line()
		// Find trailing whitespace
		trimmedLen := len(line)
		for trimmedLen > 0 && (line[trimmedLen-1] == ' ' || line[trimmedLen-1] == '\t') {
			trimmedLen--
		}

		// If there's trailing whitespace, remove it
		if trimmedLen < len(line) {
			startPos := iter.Offset
			endPos := startPos + len(line)
			// Use the rope methods directly to avoid triggering line tracking again
			b.Rope.Delete(startPos+trimmedLen, endPos)
			b.Generation++ // Increment generation since buffer is modified
			b.MarkModified()
		}
	}

	// Restore iterators after all modifications
	if b.CursorIterator != nil {
		b.CursorIterator = NewBufferIteratorAtLine(b, savedCursorLine)
		b.CursorX = savedCursorX
	}
	if b.ViewIterator != nil {
		b.ViewIterator = NewBufferIteratorAtLine(b, savedViewLine)
	}
}

// String implements the fmt.Stringer interface for Buffer
func (b *Buffer) String() string {
	if b == nil {
		return "Buffer(nil)"
	}

	filename := b.FileName()
	lines := b.LineCount
	chars := b.Rope.Length()

	return fmt.Sprintf("Buffer(file=%s, lines=%d, chars=%d)", filename, lines, chars)
}
