package editor

import (
	"fmt"
	"strings"
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

// CharacterTextObject represents a single character movement
type CharacterTextObject struct {
	Direction string // "left" or "right"
}

func (c CharacterTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
	if iter.IsEOF() {
		return 0, 0, 0, 0, false
	}
	
	line := iter.Line()
	
	if c.Direction == "left" {
		// Moving left - delete character to the left of cursor
		if cursorChar == 0 {
			// At beginning of line, can't move left
			return 0, 0, 0, 0, false
		}
		// Delete the character before the cursor
		return cursorLine, cursorChar - 1, cursorLine, cursorChar, true
	} else if c.Direction == "right" {
		// Moving right - delete character at cursor position
		if cursorChar >= len(line) {
			// At or past end of line, can't delete
			return 0, 0, 0, 0, false
		}
		// Delete the character at the cursor
		return cursorLine, cursorChar, cursorLine, cursorChar + 1, true
	}
	
	return 0, 0, 0, 0, false
}

func (c CharacterTextObject) GetType() string {
	return fmt.Sprintf("character-%s", c.Direction)
}

// LineMotionTextObject represents lines selected by up/down motion
type LineMotionTextObject struct {
	Direction string // "up" or "down"
}

func (l LineMotionTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	if l.Direction == "up" {
		// Moving up - select current line and line above
		if cursorLine == 0 {
			// At first line, can't move up
			return 0, 0, 0, 0, false
		}
		// Select from beginning of line above to end of current line
		currentIter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
		if currentIter.IsEOF() {
			return 0, 0, 0, 0, false
		}
		currentLineLen := len(currentIter.Line())
		
		return cursorLine - 1, 0, cursorLine, currentLineLen, true
	} else if l.Direction == "down" {
		// Moving down - select current line and line below
		if cursorLine >= buf.LineCount - 1 {
			// At last line, can't move down
			return 0, 0, 0, 0, false
		}
		// Select from beginning of current line to end of line below
		nextIter := buffer.NewBufferIteratorAtLine(buf, cursorLine + 1)
		if nextIter.IsEOF() {
			return 0, 0, 0, 0, false
		}
		nextLineLen := len(nextIter.Line())
		
		return cursorLine, 0, cursorLine + 1, nextLineLen, true
	}
	
	return 0, 0, 0, 0, false
}

func (l LineMotionTextObject) GetType() string {
	return fmt.Sprintf("line-motion-%s", l.Direction)
}

// MultiLineTextObject represents multiple lines starting from the current line
type MultiLineTextObject struct {
	Count int
}

func (m MultiLineTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	// Select Count lines starting from current line
	startLine := cursorLine
	endLine := cursorLine + m.Count - 1
	
	// Ensure we don't go past end of buffer
	if endLine >= buf.LineCount {
		endLine = buf.LineCount - 1
	}
	
	// Get length of last line
	iter := buffer.NewBufferIteratorAtLine(buf, endLine)
	if iter.IsEOF() {
		return 0, 0, 0, 0, false
	}
	lastLine := iter.Line()
	
	return startLine, 0, endLine, len(lastLine), true
}

func (m MultiLineTextObject) GetType() string {
	return "multiline"
}

// WordTextObject represents a word (sequence of alphanumeric chars and underscores)
type WordTextObject struct {
	modifier TextObjectModifier
	excludeTrailingSpace bool // For 'cw' which should act like 'ce'
}

func (w WordTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	// For 'dw' (no modifier), use shared logic with trailing space
	// But for 'cw', we exclude trailing space (like 'ce' in Vim)
	if w.modifier == ModifierNone {
		includeTrailing := !w.excludeTrailingSpace
		boundary := FindCurrentWordBoundary(buf, cursorLine, cursorChar, includeTrailing)
		if boundary == nil {
			return 0, 0, 0, 0, false
		}
		return boundary.StartLine, boundary.StartChar, boundary.EndLine, boundary.EndChar, true
	}
	
	// For 'diw' and 'daw', use shared logic without trailing space for 'inside'
	// and with trailing space for 'around'
	includeTrailing := w.modifier == ModifierAround
	boundary := FindCurrentWordBoundary(buf, cursorLine, cursorChar, includeTrailing)
	if boundary == nil {
		return 0, 0, 0, 0, false
	}
	
	// For 'diw', we need to handle the case where cursor is not on a word
	if w.modifier == ModifierInside || w.modifier == ModifierAround {
		iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
		if iter.IsEOF() {
			return 0, 0, 0, 0, false
		}
		line := iter.Line()
		
		// If we're not on a word character, find the next word
		if cursorChar < len(line) && !isWordCharByte(line[cursorChar]) {
			nextWord := FindNextWordBoundary(buf, cursorLine, cursorChar)
			if nextWord == nil {
				return 0, 0, 0, 0, false
			}
			// Get the boundary of the next word
			boundary = FindCurrentWordBoundary(buf, nextWord.StartLine, nextWord.StartChar, includeTrailing)
			if boundary == nil {
				return 0, 0, 0, 0, false
			}
		}
	}
	
	return boundary.StartLine, boundary.StartChar, boundary.EndLine, boundary.EndChar, true
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

// FindCharacterTextObject represents text from cursor to the next occurrence of a character
type FindCharacterTextObject struct {
	character string
}

func (f FindCharacterTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
	if iter.IsEOF() {
		return 0, 0, 0, 0, false
	}
	
	line := iter.Line()
	
	// Search for the character starting from the position after the cursor
	for i := cursorChar + 1; i < len(line); i++ {
		if string(line[i]) == f.character {
			// Include the found character (like vim's df command)
			return cursorLine, cursorChar, cursorLine, i + 1, true
		}
	}
	
	// Character not found
	return 0, 0, 0, 0, false
}

func (f FindCharacterTextObject) GetType() string {
	return fmt.Sprintf("find-%s", f.character)
}

// TillCharacterTextObject represents text from cursor to just before the next occurrence of a character
type TillCharacterTextObject struct {
	character string
}

func (t TillCharacterTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
	if iter.IsEOF() {
		return 0, 0, 0, 0, false
	}
	
	line := iter.Line()
	
	// Search for the character starting from the position after the cursor
	for i := cursorChar + 1; i < len(line); i++ {
		if string(line[i]) == t.character {
			// Stop just before the found character (like vim's dt command)
			return cursorLine, cursorChar, cursorLine, i, true
		}
	}
	
	// Character not found
	return 0, 0, 0, 0, false
}

func (t TillCharacterTextObject) GetType() string {
	return fmt.Sprintf("till-%s", t.character)
}

// SearchResultTextObject represents a search match with captured pattern
type SearchResultTextObject struct {
	editor        *Editor
	searchPattern string // Captured search pattern at time of creation
}

func NewSearchResultTextObject(editor *Editor) *SearchResultTextObject {
	// Capture only the search pattern to preserve it for repeat
	// We'll recalculate matches dynamically since buffer positions change
	return &SearchResultTextObject{
		editor:        editor,
		searchPattern: editor.searchPattern,
	}
}

func (s SearchResultTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	// Use the stored search pattern (captured at creation time)
	if s.searchPattern == "" {
		return 0, 0, 0, 0, false
	}
	
	// Recalculate matches for the stored pattern in the current buffer
	searchMatches := s.findMatchesInBuffer(buf, s.searchPattern)
	if len(searchMatches) == 0 {
		return 0, 0, 0, 0, false
	}
	
	// Get current cursor position in absolute terms
	currentPos := s.editor.getAbsolutePosition()
	
	// Find which search match we're currently on or closest to
	var matchPos int = -1
	
	// First check if we're exactly on a match
	patternLen := len(s.searchPattern)
	for _, pos := range searchMatches {
		if currentPos >= pos && currentPos < pos+patternLen {
			matchPos = pos
			break
		}
	}
	
	// If we're not on a match, find the closest one
	if matchPos == -1 {
		// Find the closest match (prefer the one after cursor, then before)
		closestDistance := int(^uint(0) >> 1) // Max int
		
		for _, pos := range searchMatches {
			distance := pos - currentPos
			if distance >= 0 && distance < closestDistance {
				matchPos = pos
				closestDistance = distance
			}
		}
		
		// If no match found after cursor, find closest before
		if matchPos == -1 {
			for _, pos := range searchMatches {
				distance := currentPos - pos
				if distance >= 0 && distance < closestDistance {
					matchPos = pos
					closestDistance = distance
				}
			}
		}
	}
	
	if matchPos == -1 {
		return 0, 0, 0, 0, false
	}
	
	// Convert absolute position to line/char coordinates
	startLine, startChar := s.absoluteToLineChar(buf, matchPos)
	endLine, endChar := s.absoluteToLineChar(buf, matchPos+patternLen)
	
	return startLine, startChar, endLine, endChar, true
}

// absoluteToLineChar converts an absolute position to line and character coordinates
func (s SearchResultTextObject) absoluteToLineChar(buf *buffer.Buffer, pos int) (int, int) {
	iter := buffer.NewBufferIterator(buf)
	lineNum := 0
	lineStart := 0
	
	// Find the line containing the position
	for !iter.IsEOF() {
		lineEnd := iter.LineEnd()
		if pos <= lineEnd {
			// Position is on this line
			return lineNum, pos - lineStart
		}
		lineStart = lineEnd + 1 // +1 for newline
		iter = iter.Next()
		lineNum++
	}
	
	// Position is past end of buffer
	return buf.LineCount - 1, 0
}

func (s SearchResultTextObject) GetType() string {
	return "search-result"
}

// findMatchesInBuffer finds all matches for the given pattern in the buffer
func (s SearchResultTextObject) findMatchesInBuffer(buf *buffer.Buffer, pattern string) []int {
	if pattern == "" {
		return []int{}
	}
	
	// Get buffer content
	content := buf.Rope.String()
	
	// Find all matches using the same logic as updateSearchMatches
	var matches []int
	pos := 0
	for {
		index := strings.Index(content[pos:], pattern)
		if index == -1 {
			break
		}
		matches = append(matches, pos+index)
		pos = pos + index + 1
	}
	
	return matches
}

