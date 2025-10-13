package editor

import (
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/clipboard"
	"veda/internal/syntax"
	"veda/internal/ui"
)

// TestPasteSyntaxHighlighting verifies that syntax highlighting is properly updated after paste
func TestPasteSyntaxHighlighting(t *testing.T) {
	// Create initial Go code
	initialCode := `package main

func main() {
	// Original line
}`

	// Create test editor
	te := NewTestEditorWithContent(80, 24, initialCode)
	te.CurrentBuffer().FilePath = "test.go"

	// Initialize syntax highlighting (per-buffer)
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	err := te.CurrentBuffer().Highlighter.SetLanguageFromFile("test.go")
	if err != nil {
		t.Fatalf("Failed to set language: %v", err)
	}
	te.CurrentBuffer().Highlighter.ParseFull([]byte(te.CurrentBuffer().Rope.String()))

	// Render the initial content
	te.render()

	// Verify syntax highlighting is working initially
	// Get the "func main" line (line 2, 0-indexed)
	funcLine := getRenderedLine(te, 2)

	// Check that "func" is highlighted as a keyword
	if !strings.Contains(funcLine, "func") {
		t.Fatalf("Line doesn't contain 'func': %s", funcLine)
	}

	// Check the style of 'f' in "func" (accounting for line numbers)
	funcStyle := getCellStyle(te, 2, "func")
	if funcStyle == nil {
		t.Fatal("Could not find 'func' style")
	}

	keywordStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxKeyword)
	if !stylesMatch(*funcStyle, keywordStyle) {
		t.Errorf("Initial 'func' keyword is not highlighted properly. Got %v, want %v", funcStyle, keywordStyle)
	}

	// Now copy a line with syntax (e.g., a print statement)
	codeToPaste := "\tfmt.Println(\"Hello, World!\")"
	err = clipboard.Copy(codeToPaste)
	if err != nil {
		t.Skipf("Clipboard not available: %v", err)
	}

	// Move cursor inside the function (line 3) and open a new line
	te.ExecuteCommand("down")            // Move to line 3
	te.ExecuteCommand("down")            // Move to line 3 (comment line)
	te.ExecuteCommand("open-line-below") // Open new line below

	// We should now be in insert mode on line 4
	// Exit insert mode and paste
	te.ExecuteCommand("exit-insert-mode")
	te.ExecuteCommand("paste-after") // Paste the code

	// Render again after paste
	te.render()

	// Debug: Print all lines to find where the paste went
	t.Logf("Buffer content after paste:")
	for i := 0; i < te.CurrentBuffer().LineCount; i++ {
		line := getRenderedLine(te, i)
		t.Logf("Line %d: %s", i, line)
	}

	// Find which line contains our pasted content
	pastedLineNum := -1
	for i := 0; i < te.CurrentBuffer().LineCount; i++ {
		line := getRenderedLine(te, i)
		if strings.Contains(line, "fmt.Println") {
			pastedLineNum = i
			break
		}
	}

	if pastedLineNum == -1 {
		t.Fatal("Could not find pasted content 'fmt.Println' in any line")
	}

	t.Logf("Found pasted content on line %d", pastedLineNum)

	// Check that "fmt" is highlighted
	fmtStyle := getCellStyle(te, pastedLineNum, "fmt")
	if fmtStyle == nil {
		t.Fatal("Could not find 'fmt' style")
	}

	normalStyle := te.config.Theme.GetStyle(te.config.Theme.Normal)
	if stylesMatch(*fmtStyle, normalStyle) {
		t.Log("Pasted 'fmt' uses the normal style (theme treats package identifiers as plain text)")
	}

	// Check that the string is highlighted
	stringContent := "Hello"
	stringStyle := getCellStyle(te, pastedLineNum, stringContent)
	if stringStyle == nil {
		t.Fatal("Could not find string style")
	}

	// Debug: Log what tokens the highlighter has for this line
	iter := buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), pastedLineNum)
	lineStart := iter.Position()
	lineEnd := iter.LineEnd()
	t.Logf("Line %d range: %d-%d", pastedLineNum, lineStart, lineEnd)
	tokens := te.CurrentBuffer().Highlighter.GetTokensForLine(lineStart, lineEnd)
	t.Logf("Tokens for pasted line: %+v", tokens)

	expectedStringStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxString)

	// Debug: Log style components
	gotFg, gotBg, gotAttr := stringStyle.Decompose()
	expectedFg, expectedBg, expectedAttr := expectedStringStyle.Decompose()
	t.Logf("Got style: fg=%v, bg=%v, attr=%v", gotFg, gotBg, gotAttr)
	t.Logf("Expected style: fg=%v, bg=%v, attr=%v", expectedFg, expectedBg, expectedAttr)

	// Check what line this is and if it's the current line
	currentLine := te.CurrentBuffer().CursorIterator.LineNumber()
	t.Logf("Current cursor line: %d, pasted line: %d", currentLine, pastedLineNum)

	// Check if the background matches current line background
	currentLineStyle := te.config.Theme.GetStyle(te.config.Theme.CurrentLine)
	_, currentLineBg, _ := currentLineStyle.Decompose()
	t.Logf("Current line bg: %v", currentLineBg)

	// The string should have the correct foreground but might have current line background
	if currentLine == pastedLineNum && gotBg == currentLineBg {
		t.Log("Background difference is due to current line highlighting - this is expected")
		// Check only foreground color
		if gotFg == expectedFg {
			t.Log("SUCCESS: String is properly highlighted (foreground color matches)")
			return // Test passes
		}
	}

	if !stylesMatch(*stringStyle, expectedStringStyle) {
		t.Errorf("Pasted string literal is not highlighted properly. Got %v, want %v", stringStyle, expectedStringStyle)

		// Now test if manually updating syntax highlighting fixes it
		t.Log("Testing if updateSyntaxHighlighting fixes the issue...")

		// Debug: Check what content the highlighter has
		t.Logf("Buffer content: %q", te.CurrentBuffer().Rope.String())
		t.Logf("Line %d content: %q", pastedLineNum, iter.Line())

		// Debug: Check if the tree is valid
		tree := te.CurrentBuffer().Highlighter.GetTree()
		if tree != nil {
			t.Logf("Tree root node type: %s", tree.RootNode().Type())
			t.Logf("Tree root node range: %d-%d", tree.RootNode().StartByte(), tree.RootNode().EndByte())
		} else {
			t.Log("Tree is nil!")
		}

		te.updateSyntaxHighlighting()
		te.render()

		// Check again
		stringStyleAfterUpdate := getCellStyle(te, pastedLineNum, stringContent)

		// Debug: Log tokens after update
		iter2 := buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), pastedLineNum)
		tokensAfterUpdate := te.CurrentBuffer().Highlighter.GetTokensForLine(iter2.Position(), iter2.LineEnd())
		t.Logf("Tokens after updateSyntaxHighlighting: %+v", tokensAfterUpdate)

		if stringStyleAfterUpdate != nil && stylesMatch(*stringStyleAfterUpdate, expectedStringStyle) {
			t.Log("SUCCESS: updateSyntaxHighlighting fixes the highlighting issue")
		} else {
			t.Log("FAILURE: Even updateSyntaxHighlighting doesn't fix it")
		}
	}
}

// getRenderedLine returns the text content of a rendered line
func getRenderedLine(te *TestEditor, lineNum int) string {
	width, _ := te.mockScreen.Size()

	// Calculate screen Y position
	contentStartY := len(te.CurrentBuffer().ContextCache)
	viewStartLine := te.CurrentBuffer().ViewIterator.LineNumber()
	screenY := contentStartY + (lineNum - viewStartLine)

	// Build the line string
	var line strings.Builder
	for x := 0; x < width; x++ {
		if row, exists := te.mockScreen.cells[screenY]; exists {
			if cell, exists := row[x]; exists && cell.Rune != 0 {
				line.WriteRune(cell.Rune)
			}
		}
	}

	return line.String()
}

// getCellStyle finds the style of a specific text in a line
func getCellStyle(te *TestEditor, lineNum int, text string) *ui.Style {
	// Calculate screen Y position
	contentStartY := len(te.CurrentBuffer().ContextCache)
	viewStartLine := te.CurrentBuffer().ViewIterator.LineNumber()
	screenY := contentStartY + (lineNum - viewStartLine)

	// Find the text in the line
	lineContent := getRenderedLine(te, lineNum)
	textIndex := strings.Index(lineContent, text)
	if textIndex == -1 {
		return nil
	}

	// Get the style at that position
	if row, exists := te.mockScreen.cells[screenY]; exists {
		if cell, exists := row[textIndex]; exists {
			return &cell.Style
		}
	}

	return nil
}

// stylesMatch compares two styles for equality
func stylesMatch(a, b ui.Style) bool {
	// Compare foreground and attributes; ignore background differences.
	// Backgrounds can vary due to current-line, rulers, or window dimming overlays
	// and are not relevant for token identity or color consistency checks.
	aFg, _, aAttr := a.Decompose()
	bFg, _, bAttr := b.Decompose()
	return aFg == bFg && aAttr == bAttr
}
