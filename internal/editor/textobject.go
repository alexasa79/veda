package editor

import (
	"fmt"
	"veda/internal/buffer"
)

// TextObject represents a region of text that can be operated on
type TextObject interface {
	// GetRange returns the start and end positions of the text object
	// Returns (startLine, startChar, endLine, endChar, found)
	// If found is false, no valid text object was found at the cursor position
	GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool)
	
	// GetType returns the type of text object for debugging
	GetType() string
}

// TextObjectModifier represents how to select a text object
type TextObjectModifier string

const (
	ModifierNone   TextObjectModifier = ""
	ModifierAround TextObjectModifier = "a" // Include surrounding whitespace/delimiters
	ModifierInside TextObjectModifier = "i" // Exclude surrounding whitespace/delimiters
)

// LineTextObject represents the current line
type LineTextObject struct{}

func (l LineTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	// For line object, we select the entire current line including newline
	iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
	if iter.IsEOF() {
		return 0, 0, 0, 0, false
	}
	
	line := iter.Line()
	// Start at beginning of line, end after the newline (or end of line if last line)
	return cursorLine, 0, cursorLine, len(line), true
}

func (l LineTextObject) GetType() string {
	return "line"
}

// WordTextObject represents a word (sequence of alphanumeric chars and underscores)
type WordTextObject struct {
	modifier TextObjectModifier
}

func (w WordTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
	if iter.IsEOF() {
		return 0, 0, 0, 0, false
	}
	
	line := iter.Line()
	if cursorChar >= len(line) {
		return 0, 0, 0, 0, false
	}
	
	// For 'dw' (no modifier), delete the current word
	// If on a word character: delete the entire word (like vim's diw)
	// If on whitespace: delete from cursor through the next word
	if w.modifier == ModifierNone {
		start := cursorChar
		end := cursorChar
		
		// If we're not on a word character, we're on whitespace/punctuation
		if cursorChar < len(line) && !isWordChar(line[cursorChar]) {
			// Delete from cursor position through the next word
			// Skip whitespace/punctuation to find next word
			for end < len(line) && !isWordChar(line[end]) {
				end++
			}
			if end >= len(line) {
				return 0, 0, 0, 0, false
			}
			// Now skip the word itself
			for end < len(line) && isWordChar(line[end]) {
				end++
			}
			// start stays at cursorChar (don't jump back)
		} else {
			// We're on a word character - delete the entire word
			// Find start of word
			for start > 0 && isWordChar(line[start-1]) {
				start--
			}
			// Find end of word  
			for end < len(line) && isWordChar(line[end]) {
				end++
			}
		}
		
		return cursorLine, start, cursorLine, end, true
	}
	
	// For 'diw' and 'daw', operate on the current word
	start := cursorChar
	end := cursorChar
	
	// If we're not on a word character, find the next word
	if !isWordChar(line[cursorChar]) {
		// Find next word
		for end < len(line) && !isWordChar(line[end]) {
			end++
		}
		if end >= len(line) {
			return 0, 0, 0, 0, false
		}
		start = end
		cursorChar = end
	}
	
	// Find start of word
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	
	// Find end of word  
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	
	// For "around" modifier, include trailing whitespace
	if w.modifier == ModifierAround {
		for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
			end++
		}
	}
	
	return cursorLine, start, cursorLine, end, true
}

func (w WordTextObject) GetType() string {
	if w.modifier == ModifierAround {
		return "word-around"
	}
	return "word"
}

// WORDTextObject represents a WORD (sequence of non-whitespace chars)
type WORDTextObject struct{}

func (w WORDTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
	if iter.IsEOF() {
		return 0, 0, 0, 0, false
	}
	
	line := iter.Line()
	if cursorChar >= len(line) {
		return 0, 0, 0, 0, false
	}
	
	// For 'dW', delete from cursor to end of current WORD (similar to vim's dw)
	end := cursorChar
	
	// Skip to end of current WORD
	if cursorChar < len(line) && !isWhitespace(line[cursorChar]) {
		for end < len(line) && !isWhitespace(line[end]) {
			end++
		}
	} else {
		// If we're on whitespace, skip it
		for end < len(line) && isWhitespace(line[end]) {
			end++
		}
	}
	
	return cursorLine, cursorChar, cursorLine, end, true
}

func (w WORDTextObject) GetType() string {
	return "WORD"
}

// BracketTextObject represents text within brackets
type BracketTextObject struct {
	openBracket  rune
	closeBracket rune
	modifier     TextObjectModifier
}

func NewBracketTextObject(bracket rune, modifier TextObjectModifier) *BracketTextObject {
	var open, close rune
	switch bracket {
	case '{', '}':
		open, close = '{', '}'
	case '[', ']':
		open, close = '[', ']'
	case '(', ')':
		open, close = '(', ')'
	default:
		return nil
	}
	
	return &BracketTextObject{
		openBracket:  open,
		closeBracket: close,
		modifier:     modifier,
	}
}

func (b BracketTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	// Find matching brackets using a simple algorithm
	// This is a simplified version - a full implementation would handle nesting
	
	// First, find the opening bracket by searching backwards
	startLine, startChar := b.findOpeningBracket(buf, cursorLine, cursorChar)
	if startLine == -1 {
		return 0, 0, 0, 0, false
	}
	
	// Then find the closing bracket
	endLine, endChar := b.findClosingBracket(buf, startLine, startChar)
	if endLine == -1 {
		return 0, 0, 0, 0, false
	}
	
	// Adjust based on modifier
	if b.modifier == ModifierInside {
		// Exclude the brackets themselves
		startChar++
		// Handle newline after opening bracket
		iter := buffer.NewBufferIteratorAtLine(buf, startLine)
		line := iter.Line()
		if startChar > len(line) {
			startLine++
			startChar = 0
		}
	} else {
		// Include the brackets
		endChar++
	}
	
	return startLine, startChar, endLine, endChar, true
}

func (b BracketTextObject) findOpeningBracket(buf *buffer.Buffer, startLine, startChar int) (int, int) {
	// Search backwards for opening bracket
	depth := 0
	line := startLine
	char := startChar
	
	for line >= 0 {
		iter := buffer.NewBufferIteratorAtLine(buf, line)
		if iter.IsEOF() {
			break
		}
		
		lineStr := iter.Line()
		if line == startLine {
			char = min(char, len(lineStr)-1)
		} else {
			char = len(lineStr) - 1
		}
		
		for char >= 0 {
			if char < len(lineStr) {
				if rune(lineStr[char]) == b.closeBracket {
					depth++
				} else if rune(lineStr[char]) == b.openBracket {
					if depth == 0 {
						return line, char
					}
					depth--
				}
			}
			char--
		}
		
		line--
	}
	
	return -1, -1
}

func (b BracketTextObject) findClosingBracket(buf *buffer.Buffer, startLine, startChar int) (int, int) {
	// Search forwards for closing bracket
	depth := 1 // We start inside the brackets
	line := startLine
	char := startChar + 1
	
	lineCount := buf.LineCount
	
	for line < lineCount {
		iter := buffer.NewBufferIteratorAtLine(buf, line)
		if iter.IsEOF() {
			break
		}
		
		lineStr := iter.Line()
		if line == startLine {
			char = max(char, startChar+1)
		} else {
			char = 0
		}
		
		for char < len(lineStr) {
			if rune(lineStr[char]) == b.openBracket {
				depth++
			} else if rune(lineStr[char]) == b.closeBracket {
				depth--
				if depth == 0 {
					return line, char
				}
			}
			char++
		}
		
		line++
	}
	
	return -1, -1
}

func (b BracketTextObject) GetType() string {
	bracket := string(b.openBracket)
	if b.modifier == ModifierInside {
		return fmt.Sprintf("inside-%s", bracket)
	}
	return fmt.Sprintf("around-%s", bracket)
}

// Helper functions
func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || 
		(ch >= 'A' && ch <= 'Z') || 
		(ch >= '0' && ch <= '9') || 
		ch == '_'
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}