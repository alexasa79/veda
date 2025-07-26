package editor

import (
	"unicode"
	"veda/internal/buffer"
)

// isWordChar returns true if the character is part of a word (alphanumeric or underscore)
func isWordChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

// isWordCharByte is a convenience function for byte input
func isWordCharByte(ch byte) bool {
	return isWordChar(rune(ch))
}

// isWhitespace returns true if the character is whitespace
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

// WordBoundary represents the start and end of a word
type WordBoundary struct {
	StartLine int
	StartChar int
	EndLine   int
	EndChar   int
}

// FindNextWordBoundary finds the boundary of the next word from the given position
// This implements vim's word navigation logic:
// - From a word: finds the start of the next word
// - From whitespace: finds the start of the next word
// - From punctuation: finds the start of the next item after this punctuation
func FindNextWordBoundary(buf *buffer.Buffer, startLine, startChar int) *WordBoundary {
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	if iter.IsEOF() {
		return nil
	}
	
	x := startChar
	line := iter.Line()
	
	// First, skip the current word/character
	if x < len(line) {
		currentChar := rune(line[x])
		if isWordChar(currentChar) {
			// Skip word characters
			for x < len(line) && isWordChar(rune(line[x])) {
				x++
			}
		} else if !unicode.IsSpace(currentChar) {
			// Skip a single punctuation character
			x++
		} else {
			// On whitespace, move forward
			x++
		}
	}
	
	// Now skip any whitespace to find the next word
	for !iter.IsEOF() {
		line = iter.Line()
		
		// Skip whitespace on current line
		for x < len(line) && unicode.IsSpace(rune(line[x])) {
			x++
		}
		
		// If we found a non-space character, we're at the next word
		if x < len(line) {
			return &WordBoundary{
				StartLine: iter.LineNumber(),
				StartChar: x,
				EndLine:   iter.LineNumber(),
				EndChar:   x,
			}
		}
		
		// Move to next line
		iter = iter.Next()
		if !iter.IsEOF() {
			x = 0
		}
	}
	
	return nil
}

// FindCurrentWordBoundary finds the boundary of the word at the current position
// Used by text objects to determine what to delete/change
// includeTrailingSpace: if true, includes trailing whitespace (for 'dw')
func FindCurrentWordBoundary(buf *buffer.Buffer, cursorLine, cursorChar int, includeTrailingSpace bool) *WordBoundary {
	iter := buffer.NewBufferIteratorAtLine(buf, cursorLine)
	if iter.IsEOF() {
		return nil
	}
	
	line := iter.Line()
	if cursorChar >= len(line) {
		return nil
	}
	
	start := cursorChar
	end := cursorChar
	
	if cursorChar < len(line) {
		currentChar := line[cursorChar]
		
		if isWordCharByte(currentChar) {
			// On a word character - find the entire word
			// Find start of word
			for start > 0 && isWordCharByte(line[start-1]) {
				start--
			}
			// Find end of word  
			for end < len(line) && isWordCharByte(line[end]) {
				end++
			}
			// Include trailing whitespace if requested
			if includeTrailingSpace {
				for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
					end++
				}
			}
		} else if currentChar == ' ' || currentChar == '\t' {
			// On whitespace - include whitespace + next word
			// Skip whitespace
			for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
				end++
			}
			// If we hit a word, include it too
			if end < len(line) && isWordCharByte(line[end]) {
				for end < len(line) && isWordCharByte(line[end]) {
					end++
				}
			}
		} else {
			// On punctuation - include punctuation + trailing whitespace
			end = cursorChar + 1
			// Include trailing whitespace if requested
			if includeTrailingSpace {
				for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
					end++
				}
			}
		}
	}
	
	return &WordBoundary{
		StartLine: cursorLine,
		StartChar: start,
		EndLine:   cursorLine,
		EndChar:   end,
	}
}