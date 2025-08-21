package editor

import (
	"veda/internal/buffer"
	"veda/internal/proxy"
)

// BlockTextObject represents a code block (as defined by LSP folding ranges)
type BlockTextObject struct {
	modifier TextObjectModifier
}

func NewBlockTextObject(modifier TextObjectModifier) *BlockTextObject {
	return &BlockTextObject{
		modifier: modifier,
	}
}

func (b BlockTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	// Use cached folding ranges from the buffer
	if len(buf.FoldingRanges) == 0 {
		return 0, 0, 0, 0, false
	}

	// Find the best folding range that contains the cursor
	// Heuristics:
	// - Prefer multi-line structural blocks over single-line ranges
	// - When both are multi-line or both single-line, prefer smaller (more specific)
	var bestRange *proxy.FoldingRange
	for i := range buf.FoldingRanges {
		r := &buf.FoldingRanges[i]
		if cursorLine < r.StartLine || cursorLine > r.EndLine {
			continue
		}

		isMultiLine := r.EndLine > r.StartLine
		if bestRange == nil {
			bestRange = r
			continue
		}

		bestIsMultiLine := bestRange.EndLine > bestRange.StartLine
		currentSize := r.EndLine - r.StartLine
		bestSize := bestRange.EndLine - bestRange.StartLine

		if isMultiLine && !bestIsMultiLine {
			bestRange = r
			continue
		}
		if isMultiLine == bestIsMultiLine {
			if currentSize < bestSize {
				bestRange = r
			}
		}
	}

	if bestRange == nil {
		return 0, 0, 0, 0, false
	}

	// Determine whether boundary lines contain braces
	braceAtStart := lineContainsRune(buf, bestRange.StartLine, '{')
	braceAtEnd := lineContainsRune(buf, bestRange.EndLine, '}')

	if b.modifier == ModifierInside {
		// Use folding range as the body. If folding range already includes braces
		// on boundary lines, exclude them by trimming one line from each side.
		startLine := bestRange.StartLine
		endLine := bestRange.EndLine
		if braceAtStart {
			startLine++
		}
		if braceAtEnd {
			endLine--
		}
		if startLine > endLine {
			return 0, 0, 0, 0, false
		}
		// Inside selects full lines
		endChar := lineLen(buf, endLine)
		return startLine, 0, endLine, endChar, true
	}

	// Around: expand to include header (opening brace line) and closing brace line
	startLine := bestRange.StartLine
	endLine := bestRange.EndLine

	if !braceAtStart && startLine-1 >= 0 && lineContainsRune(buf, startLine-1, '{') {
		startLine--
	}
	if !braceAtEnd && endLine+1 < buf.LineCount && lineContainsRune(buf, endLine+1, '}') {
		endLine++
	}

	return startLine, 0, endLine, lineLen(buf, endLine), true
}

func (b BlockTextObject) GetType() string {
	if b.modifier == ModifierInside {
		return "block-inside"
	}
	return "block-around"
}

// FunctionTextObject represents an entire function
type FunctionTextObject struct {
	modifier     TextObjectModifier
	proxyManager *proxy.ProxyManager
}

func NewFunctionTextObject(modifier TextObjectModifier, proxyManager *proxy.ProxyManager) *FunctionTextObject {
	return &FunctionTextObject{
		modifier:     modifier,
		proxyManager: proxyManager,
	}
}

func (f FunctionTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	if f.proxyManager == nil || buf.FilePath == "" {
		return 0, 0, 0, 0, false
	}

	// Get document symbols
	symbols, err := f.proxyManager.GetDocumentSymbols(buf.FilePath)
	if err != nil {
		return 0, 0, 0, 0, false
	}

	// Find the function containing the cursor
	functionSymbol := findEnclosingFunction(symbols, cursorLine, cursorChar)
	if functionSymbol == nil {
		return 0, 0, 0, 0, false
	}

	if f.modifier == ModifierInside {
		// For 'inside', we want just the function body
		// This is a simplified approach - find the opening brace and return from there
		startLine := functionSymbol.Range.Start.Line

		// Find the opening brace
		foundBrace := false
		braceLine := startLine
		braceChar := 0

		for line := startLine; line <= functionSymbol.Range.End.Line; line++ {
			iter := buffer.NewBufferIteratorAtLine(buf, line)
			if iter.IsEOF() {
				break
			}
			lineStr := iter.Line()
			for i, ch := range lineStr {
				if ch == '{' {
					braceLine = line
					braceChar = i + 1
					if braceChar >= len(lineStr) {
						// Opening brace at end of line, start from next line
						braceLine++
						braceChar = 0
					}
					foundBrace = true
					break
				}
			}
			if foundBrace {
				break
			}
		}

		if !foundBrace {
			// No body found, return the whole function
			return functionSymbol.Range.Start.Line, functionSymbol.Range.Start.Character,
				functionSymbol.Range.End.Line, functionSymbol.Range.End.Character, true
		}

		// Find the closing brace (simplified - just use the function end)
		endLine := functionSymbol.Range.End.Line
		endChar := functionSymbol.Range.End.Character

		// Try to exclude the closing brace
		iter := buffer.NewBufferIteratorAtLine(buf, endLine)
		if !iter.IsEOF() {
			line := iter.Line()
			if endChar > 0 && endChar <= len(line) && line[endChar-1] == '}' {
				endChar--
			}
		}

		return braceLine, braceChar, endLine, endChar, true
	}

	// For 'around', return the entire function
	return functionSymbol.Range.Start.Line, functionSymbol.Range.Start.Character,
		functionSymbol.Range.End.Line, functionSymbol.Range.End.Character, true
}

func (f FunctionTextObject) GetType() string {
	if f.modifier == ModifierInside {
		return "function-inside"
	}
	return "function-around"
}

// Helper function to find the function symbol that contains the given position
func findEnclosingFunction(symbols []proxy.DocumentSymbol, line, character int) *proxy.DocumentSymbol {
	for i := range symbols {
		symbol := &symbols[i]

		// Check if this symbol is a function/method and contains the position
		if (symbol.Kind == 12 || symbol.Kind == 6) && // Function or Method
			positionInRange(line, character, symbol.Range) {

			// Check children for more specific match
			if len(symbol.Children) > 0 {
				if child := findEnclosingFunction(symbol.Children, line, character); child != nil {
					return child
				}
			}

			return symbol
		}

		// Recurse into children
		if len(symbol.Children) > 0 {
			if result := findEnclosingFunction(symbol.Children, line, character); result != nil {
				return result
			}
		}
	}

	return nil
}

// Helper function to check if a position is within a range
func positionInRange(line, character int, r proxy.Range) bool {
	if line < r.Start.Line || line > r.End.Line {
		return false
	}

	if line == r.Start.Line && character < r.Start.Character {
		return false
	}

	if line == r.End.Line && character > r.End.Character {
		return false
	}

	return true
}

// lineContainsRune reports whether the given line contains the target rune.
func lineContainsRune(buf *buffer.Buffer, lineNumber int, target rune) bool {
	if lineNumber < 0 || lineNumber >= buf.LineCount {
		return false
	}
	iter := buffer.NewBufferIteratorAtLine(buf, lineNumber)
	if iter.IsEOF() {
		return false
	}
	line := iter.Line()
	for i := 0; i < len(line); i++ {
		if rune(line[i]) == target {
			return true
		}
	}
	return false
}

// lineLen returns the length of the given line or 0 if out of bounds.
func lineLen(buf *buffer.Buffer, lineNumber int) int {
	if lineNumber < 0 || lineNumber >= buf.LineCount {
		return 0
	}
	iter := buffer.NewBufferIteratorAtLine(buf, lineNumber)
	if iter.IsEOF() {
		return 0
	}
	return len(iter.Line())
}
