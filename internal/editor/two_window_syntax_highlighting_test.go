package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/buffer"
	"veda/internal/syntax"
	"veda/internal/ui"
)

func TestTwoWindowSyntaxHighlightingBug(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	t.Run("syntax highlighting stays correct when editing with two windows", func(t *testing.T) {
		// Create first Go file content
		file1Content := `package main

import "fmt"

func main() {
	fmt.Println("Hello from file 1")
	x := 42
	if x > 0 {
		fmt.Printf("x is %d", x)
	}
}`

		// Create test editor with first file
		te := NewTestEditorWithContent(120, 30, file1Content)
		te.minActiveWindowWidth = 0 // Disable minimum width for equal window distribution
		te.CurrentBuffer().FilePath = "test1.go"

		// Initialize syntax highlighting for first file (per-buffer)
		te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
		err := te.CurrentBuffer().Highlighter.SetLanguageFromFile("test1.go")
		if err != nil {
			t.Skipf("Skipping test: Go syntax highlighting not available: %v", err)
		}
		te.CurrentBuffer().Highlighter.ParseFull([]byte(file1Content))

		// Force render to ensure syntax highlighting is applied
		te.render()

		// Verify initial syntax highlighting of "func" keyword in first file
		expectedKeywordStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxKeyword)
		initialFuncStyle := getCellStyle(te, 4, "func")
		if initialFuncStyle == nil {
			t.Fatal("Could not find 'func' keyword in first file")
		}

		initialFuncFg, _, _ := initialFuncStyle.Decompose()
		expectedFg, _, _ := expectedKeywordStyle.Decompose()

		if initialFuncFg != expectedFg {
			t.Logf("Initial 'func' color: %v, expected: %v", initialFuncFg, expectedFg)
		} else {
			t.Logf("Initial 'func' has correct color: %v", initialFuncFg)
		}

		// Simulate opening a second file in a new window (like Shift+Enter from file browser)
		// Create second file content
		file2Content := `package test

import (
	"strings"
	"testing"
)

var globalVar = "test"

func TestFunction() {
	result := strings.Contains("hello", "ell")
	if result {
		testing.Log("found substring")
	}
}`

		// Create a new buffer and window to simulate Shift+Enter behavior
		secondBuffer := buffer.NewBuffer(file2Content, "test2.go", "test2.go", te.config.DefaultIndentType, te.config.DefaultIndentWidth)

		// Set up the buffer properly
		secondBuffer.ViewIterator = buffer.NewBufferIterator(secondBuffer)
		secondBuffer.CursorIterator = buffer.NewBufferIterator(secondBuffer)
		secondBuffer.OnContentChange = func() {
			te.updateSearchMatchesOnContentChange()
		}
		te.buffers["test2.go"] = secondBuffer
		te.bufferOrder = append(te.bufferOrder, "test2.go")

		// Ensure the first window has the correct buffer reference (the original buffer)
		originalBuffer := te.CurrentBuffer()
		if len(te.windows) > 0 {
			te.windows[0].setActiveBuffer(originalBuffer)
		}

		// Create a new window for the second file (simulating Shift+Enter)
		newWin := newWindow(secondBuffer)
		te.windows = append(te.windows, newWin)
		te.activeWindow = len(te.windows) - 1 // Switch to the new window
		// Recompute layout now that we have multiple windows
		te.recomputeWindowsLayout()

		// Update the editor's active buffer to match the active window
		te.SetCurrentBuffer(te.windows[te.activeWindow].Buffer)

		// Set up syntax highlighting for the new buffer (per-buffer)
		if te.CurrentBuffer() != nil {
			te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
			te.CurrentBuffer().Highlighter.SetLanguageFromFile("test2.go")
			te.CurrentBuffer().Highlighter.ParseFull([]byte(file2Content))
		}

		// Force render after setting up second window
		te.render()

		// Now we have two windows: Window 0 (test1.go) and Window 1 (test2.go)
		// We're currently in Window 1 (right window with second file)

		// Verify syntax highlighting works in the second file initially
		lines := strings.Split(te.GetBufferContent(), "\n")
		testFuncLineNum := -1
		for i, line := range lines {
			if strings.Contains(line, "func TestFunction") {
				testFuncLineNum = i
				break
			}
		}

		if testFuncLineNum == -1 {
			t.Fatal("Could not find 'func TestFunction' line in second file")
		}

		funcStyleInSecondFile := getCellStyle(te, testFuncLineNum, "func")
		if funcStyleInSecondFile != nil {
			funcFgInSecondFile, _, _ := funcStyleInSecondFile.Decompose()
			if funcFgInSecondFile == expectedFg {
				t.Logf("SUCCESS: 'func' keyword in second file has correct color: %v", funcFgInSecondFile)
			}
		}

		// Now switch back to the FIRST window (left) to edit there - this is the key scenario!
		t.Logf("Switching back to first window to edit (this is where the bug should occur)")
		te.activeWindow = 0
		te.SetCurrentBuffer(te.windows[te.activeWindow].Buffer)

		// Update syntax highlighting for the first file (per-buffer)
		if te.CurrentBuffer() != nil {
			te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
			te.CurrentBuffer().Highlighter.SetLanguageFromFile("test1.go")
			te.CurrentBuffer().Highlighter.ParseFull([]byte(file1Content))
		}

		te.render()

		// Start editing in the FIRST window (left) - this should trigger the bug
		firstFileLines := strings.Split(te.GetBufferContent(), "\n")
		printlnLineNum := -1
		for i, line := range firstFileLines {
			if strings.Contains(line, "fmt.Println") {
				printlnLineNum = i
				break
			}
		}

		if printlnLineNum == -1 {
			t.Fatal("Could not find fmt.Println line in first file")
		}

		// Navigate to end of the fmt.Println line
		te.SetCursorPosition(0, printlnLineNum)
		te.ExecuteCommand("end-of-line")

		// Perform multiple rapid edits to try to trigger the bug
		te.ExecuteCommand("enter-insert-mode")
		te.ExecuteCommand("insert-newline")

		// Insert multiple lines rapidly
		for i := 0; i < 3; i++ {
			lineCode := fmt.Sprintf("\tvar x%d := %d", i, i*10)
			for _, ch := range lineCode {
				te.InsertChar(ch)
			}
			te.ExecuteCommand("insert-newline")
		}

		// Add some keywords that should be highlighted
		keywordLine := "\tfor i := 0; i < 10; i++ {"
		for _, ch := range keywordLine {
			te.InsertChar(ch)
		}
		te.ExecuteCommand("insert-newline")

		closingLine := "\t\treturn"
		for _, ch := range closingLine {
			te.InsertChar(ch)
		}
		te.ExecuteCommand("insert-newline")

		finalLine := "\t}"
		for _, ch := range finalLine {
			te.InsertChar(ch)
		}

		// Exit insert mode
		te.PressKey(KeyEscape, 0)

		// Force render after editing in the first window
		te.render()

		t.Logf("After editing in the first window (left), checking syntax highlighting...")

		// The buffer has already been updated by the editing operations above
		// Now check if syntax highlighting is correct in the edited first file

		// Get the updated content after editing
		editedFirstFileLines := strings.Split(te.GetBufferContent(), "\n")
		t.Logf("First file content after editing:")
		for i, line := range editedFirstFileLines {
			if strings.Contains(line, "func") || i <= 8 {
				t.Logf("Line %d: %s", i, line)
			}
		}

		// Find the func main line in the first file (after editing)
		var firstFuncLineNum int = -1
		for i, line := range editedFirstFileLines {
			if strings.Contains(line, "func main") {
				firstFuncLineNum = i
				break
			}
		}

		if firstFuncLineNum == -1 {
			t.Fatal("Could not find 'func main' line in first file after editing")
		}

		// Debug: Check screen rendering for the first file after editing
		t.Logf("Looking for func on line %d in first file after editing", firstFuncLineNum)
		renderedFirstLine := getRenderedLine(te, firstFuncLineNum)
		t.Logf("Rendered line %d: %s", firstFuncLineNum, renderedFirstLine)

		// The main test: Check if syntax highlighting colors are correct after editing with two windows
		// This is where the style/color bug should manifest

		// Find where "func main" appears on the screen (it might not be at the expected line due to rendering issues)
		allScreenLines := te.GetScreenContent()
		funcScreenLine := -1
		for i, line := range allScreenLines {
			if strings.Contains(line, "func main") {
				funcScreenLine = i
				break
			}
		}

		var firstFuncStyleAfterSwitch *ui.Style
		if funcScreenLine != -1 {
			// Found "func main" on screen - check its syntax highlighting
			t.Logf("Found 'func main' on screen line %d", funcScreenLine)
			firstFuncStyleAfterSwitch = getCellStyle(te, funcScreenLine, "func")
		} else {
			// Try the original approach in case the content is correctly aligned
			firstFuncStyleAfterSwitch = getCellStyle(te, firstFuncLineNum, "func")
		}

		if firstFuncStyleAfterSwitch == nil {
			t.Logf("Could not find 'func' keyword with correct style on screen - this indicates the syntax highlighting bug")

			// Debug: Show where func appears in buffer vs screen
			t.Logf("Buffer line %d: '%s'", firstFuncLineNum, firstFileLines[firstFuncLineNum])
			t.Logf("Screen line %d: '%s'", firstFuncLineNum, strings.TrimSpace(renderedFirstLine))
			if funcScreenLine != -1 {
				t.Logf("'func main' found on screen line %d: '%s'", funcScreenLine, allScreenLines[funcScreenLine])
			}

			// This could indicate either content misalignment or style application issues
			t.Error("SYNTAX HIGHLIGHTING BUG: Unable to find correctly styled 'func' keyword after editing with two windows")
			return
		}

		firstFuncFgAfterEditing, _, _ := firstFuncStyleAfterSwitch.Decompose()
		if firstFuncFgAfterEditing != expectedFg {
			t.Errorf("BUG REPRODUCED: 'func' keyword lost correct color after editing in left window with two windows open. Got %v, expected %v", firstFuncFgAfterEditing, expectedFg)

			// Debug information to help understand the issue
			t.Logf("Expected keyword color: %v", expectedFg)
			t.Logf("Actual func color: %v", firstFuncFgAfterEditing)

			// Check syntax highlighting tokens
			funcIter := buffer.NewBufferIteratorAtLine(te.CurrentBuffer(), firstFuncLineNum)
			funcLineStart := funcIter.Position()
			funcLineEnd := funcIter.LineEnd()
			if te.CurrentBuffer() != nil && te.CurrentBuffer().Highlighter != nil {
				funcTokens := te.CurrentBuffer().Highlighter.GetTokensForLine(funcLineStart, funcLineEnd)
				t.Logf("Syntax tokens for func line: %+v", funcTokens)
			}

		} else {
			t.Logf("SUCCESS: 'func' keyword maintained correct color after editing in left window: %v", firstFuncFgAfterEditing)
		}

		// Also check other keywords we added to see if they have correct highlighting
		t.Logf("Checking additional keywords added during editing...")

		// Check for 'var' keyword (note: go-tree-sitter may classify this as type or variable in this context)
		for i, line := range editedFirstFileLines {
			if strings.Contains(line, "var x0") {
				varStyle := getCellStyle(te, i, "var")
				if varStyle != nil {
					varFg, _, _ := varStyle.Decompose()
					// Accept either keyword, type, or normal text style for 'var' to tolerate parser recovery
					typeStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxType)
					typeFg, _, _ := typeStyle.Decompose()
					normalStyle := te.config.Theme.GetStyle(te.config.Theme.SyntaxText)
					normalFg, _, _ := normalStyle.Decompose()
					if varFg == expectedFg || varFg == typeFg || varFg == normalFg {
						t.Logf("SUCCESS: 'var' token has acceptable color: %v", varFg)
					} else {
						t.Errorf("BUG: 'var' token has wrong color: got %v, expected one of [%v,%v,%v]", varFg, expectedFg, typeFg, normalFg)
					}
				} else {
					t.Logf("Could not find 'var' keyword style on line %d", i)
				}
				break
			}
		}

		// Check for 'for' keyword
		for i, line := range editedFirstFileLines {
			if strings.Contains(line, "for i :=") {
				forStyle := getCellStyle(te, i, "for")
				if forStyle != nil {
					forFg, _, _ := forStyle.Decompose()
					if forFg == expectedFg {
						t.Logf("SUCCESS: 'for' keyword has correct color: %v", forFg)
					} else {
						t.Errorf("BUG: 'for' keyword has wrong color: got %v, expected %v", forFg, expectedFg)
					}
				} else {
					t.Logf("Could not find 'for' keyword style on line %d", i)
				}
				break
			}
		}

		// Check for 'return' keyword
		for i, line := range editedFirstFileLines {
			if strings.Contains(line, "return") {
				returnStyle := getCellStyle(te, i, "return")
				if returnStyle != nil {
					returnFg, _, _ := returnStyle.Decompose()
					if returnFg == expectedFg {
						t.Logf("SUCCESS: 'return' keyword has correct color: %v", returnFg)
					} else {
						t.Errorf("BUG: 'return' keyword has wrong color: got %v, expected %v", returnFg, expectedFg)
					}
				} else {
					t.Logf("Could not find 'return' keyword style on line %d", i)
				}
				break
			}
		}
	})
}
