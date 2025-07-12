package buffer

import (
	"fmt"
	"strings"
	"veda/internal/config"
)

// Buffer represents a complete text buffer with metadata
type Buffer struct {
	*Rope // Embed the rope data structure
	
	// File information
	FilePath    string
	IsModified  bool
	
	// Cursor and view state
	CursorX        int
	ViewIterator   *BufferIterator  // Saved view position iterator
	CursorIterator *BufferIterator  // Saved cursor position iterator
	
	// Caching
	LineOffsets  []int  // Cache of byte offsets for each line
	OffsetsDirty bool   // Whether line offsets need recalculation
	LineCount    int    // Cache of the number of lines
	
	// Indentation settings for this buffer
	Indentation  IndentationInfo
	
	// Line number display width (e.g., 4 for files with 1000-9999 lines)
	LineNumberWidth int
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
		ViewIterator: nil,  // Will be created when buffer is first used
		LineOffsets:  []int{},
		OffsetsDirty: true,
		LineCount:    len(rope.Lines()),
		Indentation: IndentationInfo{
			Type:  defaultIndentType,
			Width: defaultIndentWidth,
		},
	}
	b.UpdateLineNumberWidth()
	// Create initial iterator
	b.ViewIterator = NewBufferIterator(b)
	return b
}

// SaveCursorState saves the current cursor and view position
func (b *Buffer) SaveCursorState(x int, viewIterator *BufferIterator, cursorIterator *BufferIterator) {
	b.CursorX = x
	b.ViewIterator = viewIterator
	b.CursorIterator = cursorIterator
}

// RestoreCursorState returns the saved cursor and view position
func (b *Buffer) RestoreCursorState() (x int, viewIterator *BufferIterator, cursorIterator *BufferIterator) {
	// Ensure we have valid iterators
	if b.ViewIterator == nil {
		b.ViewIterator = NewBufferIterator(b)
	}
	if b.CursorIterator == nil {
		b.CursorIterator = NewBufferIterator(b)
	}
	return b.CursorX, b.ViewIterator, b.CursorIterator
}

// MarkModified marks the buffer as modified
func (b *Buffer) MarkModified() {
	b.IsModified = true
	b.OffsetsDirty = true
	b.LineCount = len(b.Lines())
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