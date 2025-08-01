package buffer

import (
	"fmt"
	"strings"
	"veda/internal/config"
	"veda/internal/proxy"
)

// calculateLineCount returns the number of lines in the rope,
// accounting for trailing newlines. A trailing newline doesn't
// create a new line; it's part of the last line.
func calculateLineCount(rope *Rope) int {
	if rope.Length() == 0 {
		return 1 // Empty buffer has 1 line
	}
	
	// Number of lines is newline count + 1
	lineCount := rope.root.newlineCount + 1
	
	// If buffer ends with newline, don't count it as creating a new line
	if rope.Length() > 0 {
		// Get the last character
		lastChar := rope.ByteAt(rope.Length() - 1)
		if lastChar == '\n' && lineCount > 1 {
			lineCount--
		}
	}
	
	return lineCount
}

// SignatureHelp represents function signature information from LSP
type SignatureHelp struct {
	Label           string                 `json:"label"`          // The function signature
	Documentation   string                 `json:"documentation"`  // Optional documentation
	Parameters      []ParameterInfo        `json:"parameters"`     // Information about each parameter
	ActiveParameter int                    `json:"activeParameter"` // Which parameter is currently active
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
	FilePath    string
	IsModified  bool

	// Cursor and view state
	CursorX        int
	ViewIterator   *BufferIterator  // Current view position iterator
	CursorIterator *BufferIterator  // Current cursor position iterator

	// Caching
	LineCount    int    // Cache of the number of lines

	// Indentation settings for this buffer
	Indentation  IndentationInfo

	// Line number display width (e.g., 4 for files with 1000-9999 lines)
	LineNumberWidth int

	// Generation number for iterator validation
	Generation uint64

	// LSP availability
	HasLSP bool  // Whether LSP is available for this buffer

	// Function signature state
	CurrentSignature *SignatureHelp // The current function signature from LSP
	SignatureGeneration uint64    // Generation number for signature help requests

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

// NewBuffer creates a new buffer with default settings
func NewBuffer(content string, filePath string, defaultIndentType config.IndentationType, defaultIndentWidth int) *Buffer {
	rope := NewRopeFromString(content)
	b := &Buffer{
		Rope:         rope,
		FilePath:     filePath,
		IsModified:   false,
		CursorX:      0,
		ViewIterator:   nil,  // Will be created when buffer is first used
		CursorIterator: nil,  // Will be created when buffer is first used
		LineCount:    calculateLineCount(rope),
		Indentation: IndentationInfo{
			Type:  defaultIndentType,
			Width: defaultIndentWidth,
		},
		Generation:   0,  // Start with generation 0
		History:      NewHistory(), // Initialize undo/redo history
		VisualAnchorLine:   -1, // -1 means no visual selection
		VisualAnchorX:      -1,
		VisualAnchorOffset: -1,
	}
	b.UpdateLineNumberWidth()
	// Create initial iterators
	b.ViewIterator = NewBufferIterator(b)
	b.CursorIterator = NewBufferIterator(b)
	return b
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
	return b.FilePath != "" && b.FilePath != "[No Name]"
}

// DisplayName returns a user-friendly name for the buffer
func (b *Buffer) DisplayName() string {
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
	Level            int    // The indentation level (in terms of file's indent width)
	Spaces           int    // Total number of spaces
	Tabs             int    // Total number of tabs
	IsContinuation   bool   // Whether this appears to be a continuation line
	ActualPrefix     string // The actual whitespace prefix
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
			if info.Spaces % b.Indentation.Width != 0 {
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
		b.LineNumberWidth = 2 // Minimum width for "1 "
	} else {
		// Calculate the width needed for the largest line number
		width := len(fmt.Sprintf("%d", b.LineCount))
		b.LineNumberWidth = width + 1 // +1 for padding space
	}
}

// Insert inserts text at the given position and increments the generation
func (b *Buffer) Insert(pos int, text string) {
	b.Rope.Insert(pos, text)
	b.Generation++ // Increment generation when buffer is modified
	b.MarkModified()
}

// Delete removes characters between start and end positions and increments the generation
func (b *Buffer) Delete(start, end int) {
	b.Rope.Delete(start, end)
	b.Generation++ // Increment generation when buffer is modified
	b.MarkModified()
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
