package editor

import (
	"unicode"
	"veda/internal/buffer"
)

// isWordChar returns true if the character is part of a word (alphanumeric or underscore)
func isWordChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

// isWordCharWithOptions returns true if the character is part of a word
// If includeUnderscore is false, underscores are treated as word boundaries (for W/B commands)
func isWordCharWithOptions(ch rune, includeUnderscore bool) bool {
	if includeUnderscore {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
	}
	return unicode.IsLetter(ch) || unicode.IsDigit(ch)
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

// FindPreviousWordBoundary finds the boundary of the previous word from the given position
// This implements vim's word backward navigation logic:
// - From a word: finds the start of the current word or previous word
// - From whitespace: finds the start of the previous word
// - From punctuation: finds the start of the previous word
func FindPreviousWordBoundary(buf *buffer.Buffer, startLine, startChar int) *WordBoundary {
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	if iter.IsEOF() {
		return nil
	}

	x := startChar
	line := iter.Line()

	// If we're at the beginning of the buffer, nothing to do
	if startLine == 0 && x <= 0 {
		return nil
	}

	// Move back one character first to start the search
	if x > 0 {
		x--
	} else {
		// Move to previous line
		iter = iter.Previous()
		if iter.LineNumber() == startLine {
			// Couldn't move back, we're at beginning
			return nil
		}
		prevLine := iter.Line()
		x = len(prevLine) - 1
		line = prevLine
	}

	// Skip whitespace backwards
	for !iter.IsEOF() {
		line = iter.Line()

		// Skip whitespace on current line
		for x >= 0 && x < len(line) && unicode.IsSpace(rune(line[x])) {
			x--
		}

		// If we found a non-space character, we're at a word or punctuation
		if x >= 0 && x < len(line) {
			break
		}

		// Move to previous line
		iter = iter.Previous()
		if iter.LineNumber() == startLine {
			// Couldn't move back further
			return nil
		}
		prevLine := iter.Line()
		if len(prevLine) > 0 {
			x = len(prevLine) - 1
		} else {
			x = 0
		}
	}

	// If we're at EOF or couldn't find anything, return nil
	if iter.IsEOF() || x < 0 || x >= len(line) {
		return nil
	}

	// Now we're on a non-whitespace character
	currentChar := rune(line[x])

	if isWordChar(currentChar) {
		// We're on a word character, find the start of this word
		for x > 0 && isWordChar(rune(line[x-1])) {
			x--
		}
	} else {
		// We're on punctuation - continue searching backwards for the next word character
		// This matches vim behavior where 'b' skips over punctuation to find previous words
		for x > 0 {
			x--
			if x < len(line) && isWordChar(rune(line[x])) {
				// Found a word character, find the start of this word
				for x > 0 && isWordChar(rune(line[x-1])) {
					x--
				}
				break
			}
		}
	}

	return &WordBoundary{
		StartLine: iter.LineNumber(),
		StartChar: x,
		EndLine:   iter.LineNumber(),
		EndChar:   x,
	}
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
			// On whitespace - only delete the whitespace
			// Skip whitespace
			for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
				end++
			}
			// Do NOT include the next word
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

// shouldStopAtEndOfLine determines if word-forward navigation should stop at the end of the current line
// This implements the enhanced navigation behavior where words extending to line boundaries get an intermediate stop
func shouldStopAtEndOfLine(buf *buffer.Buffer, startLine, startChar int) bool {
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	if iter.IsEOF() {
		return false
	}

	line := iter.Line()

	// Special case: if we're already at or near the end of line, don't stop
	if startChar >= len(line)-1 || len(line) == 0 {
		return false
	}

	// Find where the current word ends
	wordEndPos := startChar
	if wordEndPos < len(line) {
		currentChar := rune(line[wordEndPos])
		if isWordChar(currentChar) {
			// Skip word characters to find end of word
			for wordEndPos < len(line) && isWordChar(rune(line[wordEndPos])) {
				wordEndPos++
			}
		} else if !unicode.IsSpace(currentChar) {
			// Skip a single punctuation character
			wordEndPos++
		} else {
			// On whitespace, don't apply end-of-line logic
			return false
		}
	}

	// Only stop at end of line if the word extends to the very end of the line
	if wordEndPos < len(line) {
		return false
	}

	// Check if there's a next line and what it contains
	nextIter := iter.Next()
	if nextIter.IsEOF() {
		// End of file - always apply end-of-line logic when word extends to end
		return true
	}

	// Check if the next line has actual content (not just whitespace)
	nextLine := nextIter.Line()
	nextLineHasContent := false
	for i := 0; i < len(nextLine); i++ {
		if !unicode.IsSpace(rune(nextLine[i])) {
			nextLineHasContent = true
			break
		}
	}

	// Only apply end-of-line stopping if the next line has content
	// (skip over empty lines with original vim behavior)
	if !nextLineHasContent {
		return false
	}

	return true
}

// shouldStopAtEndOfLineWithOptions determines if word-forward navigation should stop at the end of the current line
// with underscore handling options
func shouldStopAtEndOfLineWithOptions(buf *buffer.Buffer, startLine, startChar int, includeUnderscore bool) bool {
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	if iter.IsEOF() {
		return false
	}

	line := iter.Line()

	// Special case: if we're already at or near the end of line, don't stop
	if startChar >= len(line)-1 || len(line) == 0 {
		return false
	}

	// Find where the current word ends
	wordEndPos := startChar
	if wordEndPos < len(line) {
		currentChar := rune(line[wordEndPos])
		if isWordCharWithOptions(currentChar, includeUnderscore) {
			// Skip word characters to find end of word
			for wordEndPos < len(line) && isWordCharWithOptions(rune(line[wordEndPos]), includeUnderscore) {
				wordEndPos++
			}
		} else if !unicode.IsSpace(currentChar) {
			// Skip a single punctuation character
			wordEndPos++
		} else {
			// On whitespace, don't apply end-of-line logic
			return false
		}
	}

	// Only stop at end of line if the word extends to the very end of the line
	if wordEndPos < len(line) {
		return false
	}

	// Check if there's a next line and what it contains
	nextIter := iter.Next()
	if nextIter.IsEOF() {
		// End of file - always apply end-of-line logic when word extends to end
		return true
	}

	// Check if the next line has actual content (not just whitespace)
	nextLine := nextIter.Line()
	nextLineHasContent := false
	for i := 0; i < len(nextLine); i++ {
		if !unicode.IsSpace(rune(nextLine[i])) {
			nextLineHasContent = true
			break
		}
	}

	// Only apply end-of-line stopping if the next line has content
	// (skip over empty lines with original vim behavior)
	if !nextLineHasContent {
		return false
	}

	return true
}

// FindNextWordBoundaryWithOptions finds the boundary of the next word with underscore handling options
// If includeUnderscore is false, underscores are treated as word boundaries (for W command)
func FindNextWordBoundaryWithOptions(buf *buffer.Buffer, startLine, startChar int, includeUnderscore bool) *WordBoundary {
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	if iter.IsEOF() {
		return nil
	}

	x := startChar
	line := iter.Line()

	// First, skip the current word/character
	if x < len(line) {
		currentChar := rune(line[x])
		if isWordCharWithOptions(currentChar, includeUnderscore) {
			// Skip word characters
			for x < len(line) && isWordCharWithOptions(rune(line[x]), includeUnderscore) {
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

// FindPreviousWordBoundaryWithOptions finds the boundary of the previous word with underscore handling options
// If includeUnderscore is false, underscores are treated as word boundaries (for B command)
func FindPreviousWordBoundaryWithOptions(buf *buffer.Buffer, startLine, startChar int, includeUnderscore bool) *WordBoundary {
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	if iter.IsEOF() {
		return nil
	}

	x := startChar
	line := iter.Line()

	// If we're at the beginning of the buffer, nothing to do
	if startLine == 0 && x <= 0 {
		return nil
	}

	// Move back one character first to start the search
	if x > 0 {
		x--
	} else {
		// Move to previous line
		iter = iter.Previous()
		if iter.LineNumber() == startLine {
			// Couldn't move back, we're at beginning
			return nil
		}
		prevLine := iter.Line()
		x = len(prevLine) - 1
		line = prevLine
	}

	// Skip whitespace backwards
	for !iter.IsEOF() {
		line = iter.Line()

		// Skip whitespace on current line
		for x >= 0 && x < len(line) && unicode.IsSpace(rune(line[x])) {
			x--
		}

		// If we found a non-space character, we're at a word or punctuation
		if x >= 0 && x < len(line) {
			break
		}

		// Move to previous line
		iter = iter.Previous()
		if iter.LineNumber() == startLine {
			// Couldn't move back further
			return nil
		}
		prevLine := iter.Line()
		if len(prevLine) > 0 {
			x = len(prevLine) - 1
		} else {
			x = 0
		}
	}

	// If we're at EOF or couldn't find anything, return nil
	if iter.IsEOF() || x < 0 || x >= len(line) {
		return nil
	}

	// Now we're on a non-whitespace character
	currentChar := rune(line[x])

	if isWordCharWithOptions(currentChar, includeUnderscore) {
		// We're on a word character, find the start of this word
		for x > 0 && isWordCharWithOptions(rune(line[x-1]), includeUnderscore) {
			x--
		}
	} else if !includeUnderscore && currentChar == '_' {
		// When treating underscores as boundaries, stop at underscores (they are valid word boundaries)
		// Don't move further, this underscore is our target
	} else {
		// We're on punctuation - continue searching backwards for the next word character
		for x > 0 {
			x--
			if x < len(line) && isWordCharWithOptions(rune(line[x]), includeUnderscore) {
				// Found a word character, find the start of this word
				for x > 0 && isWordCharWithOptions(rune(line[x-1]), includeUnderscore) {
					x--
				}
				break
			}
			// Also stop at underscores when they're treated as boundaries
			if !includeUnderscore && x < len(line) && rune(line[x]) == '_' {
				break
			}
		}
	}

	return &WordBoundary{
		StartLine: iter.LineNumber(),
		StartChar: x,
		EndLine:   iter.LineNumber(),
		EndChar:   x,
	}
}
