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
	
	// Find the innermost folding range that contains the cursor
	var bestRange *proxy.FoldingRange
	
	for i := range buf.FoldingRanges {
		r := &buf.FoldingRanges[i]
		
		// Check if cursor is within this range
		if cursorLine >= r.StartLine && cursorLine <= r.EndLine {
			// If we haven't found a range yet, or this range is smaller (more specific)
			if bestRange == nil || (r.EndLine - r.StartLine) < (bestRange.EndLine - bestRange.StartLine) {
				bestRange = r
			}
		}
	}
	
	if bestRange == nil {
		return 0, 0, 0, 0, false
	}
	
	// For 'inside' modifier, we want to exclude the first and last lines
	// For 'around' modifier, we want the entire range
	if b.modifier == ModifierInside {
		// Adjust to exclude the opening and closing lines
		startLine := bestRange.StartLine + 1
		endLine := bestRange.EndLine - 1
		
		// Make sure we have at least one line
		if startLine > endLine {
			return 0, 0, 0, 0, false
		}
		
		// Start at beginning of first inner line
		startChar := 0
		
		// Find the end character of the last inner line
		iter := buffer.NewBufferIteratorAtLine(buf, endLine)
		if iter.IsEOF() {
			return 0, 0, 0, 0, false
		}
		line := iter.Line()
		endChar := len(line)
		
		return startLine, startChar, endLine, endChar, true
	}
	
	// For 'around', return the entire range
	startChar := 0
	if bestRange.StartCharacter != nil {
		startChar = *bestRange.StartCharacter
	}
	
	endChar := 0
	if bestRange.EndCharacter != nil {
		endChar = *bestRange.EndCharacter
	} else {
		// If no end character specified, use end of line
		iter := buffer.NewBufferIteratorAtLine(buf, bestRange.EndLine)
		if !iter.IsEOF() {
			line := iter.Line()
			endChar = len(line)
		}
	}
	
	return bestRange.StartLine, startChar, bestRange.EndLine, endChar, true
}

func (b BlockTextObject) GetType() string {
	if b.modifier == ModifierInside {
		return "block-inside"
	}
	return "block-around"
}

// ArgumentTextObject represents a function argument
type ArgumentTextObject struct {
	modifier TextObjectModifier
	proxyManager *proxy.ProxyManager
}

func NewArgumentTextObject(modifier TextObjectModifier, proxyManager *proxy.ProxyManager) *ArgumentTextObject {
	return &ArgumentTextObject{
		modifier: modifier,
		proxyManager: proxyManager,
	}
}

func (a ArgumentTextObject) GetRange(buf *buffer.Buffer, cursorLine, cursorChar int) (int, int, int, int, bool) {
	if a.proxyManager == nil || buf.FilePath == "" {
		return 0, 0, 0, 0, false
	}
	
	// Get document symbols to find the enclosing function
	symbols, err := a.proxyManager.GetDocumentSymbols(buf.FilePath)
	if err != nil {
		return 0, 0, 0, 0, false
	}
	
	// Find the function containing the cursor
	functionSymbol := findEnclosingFunction(symbols, cursorLine, cursorChar)
	if functionSymbol == nil {
		return 0, 0, 0, 0, false
	}
	
	// Get the function's parameter list from the buffer
	// This is a simplified implementation - a full implementation would
	// parse the argument list properly
	startLine := functionSymbol.Range.Start.Line
	startChar := functionSymbol.Range.Start.Character
	
	// Find the opening parenthesis
	iter := buffer.NewBufferIteratorAtLine(buf, startLine)
	if iter.IsEOF() {
		return 0, 0, 0, 0, false
	}
	
	line := iter.Line()
	parenStart := -1
	for i := startChar; i < len(line); i++ {
		if line[i] == '(' {
			parenStart = i
			break
		}
	}
	
	if parenStart == -1 {
		return 0, 0, 0, 0, false
	}
	
	// For now, return a simple range - a full implementation would
	// parse arguments and find the specific argument at cursor position
	if a.modifier == ModifierInside {
		return startLine, parenStart + 1, startLine, parenStart + 5, true
	}
	return startLine, parenStart, startLine, parenStart + 6, true
}

func (a ArgumentTextObject) GetType() string {
	if a.modifier == ModifierInside {
		return "argument-inside"
	}
	return "argument-around"
}

// FunctionTextObject represents an entire function
type FunctionTextObject struct {
	modifier TextObjectModifier
	proxyManager *proxy.ProxyManager
}

func NewFunctionTextObject(modifier TextObjectModifier, proxyManager *proxy.ProxyManager) *FunctionTextObject {
	return &FunctionTextObject{
		modifier: modifier,
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