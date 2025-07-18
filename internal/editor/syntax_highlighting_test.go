package editor

import (
	"testing"
)

func TestSyntaxHighlightingWithNewline(t *testing.T) {
	t.Run("insert newline updates syntax highlighting", func(t *testing.T) {
		// Create a test editor with Go code
		te := NewTestEditorWithContent(80, 24, "")
		
		// Set up a Go file to enable syntax highlighting
		te.buffer.FilePath = "test.go"
		te.highlighter.SetLanguageFromFile("test.go")
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// Type some Go code
		code := "package main"
		for _, ch := range code {
			te.InsertChar(ch)
		}
		
		// Verify content
		AssertBufferEquals(t, te, "package main")
		AssertCursorAt(t, te, 12, 0) // After "main"
		
		// Insert a newline
		te.ExecuteCommand("insert-newline")
		
		// Verify newline was inserted (no indentation since current line has none)
		AssertBufferEquals(t, te, "package main\n")
		AssertCursorAt(t, te, 0, 1) // Beginning of new line
		
		// Type more code on the new line
		// Type "import "
		for _, ch := range "import " {
			te.InsertChar(ch)
		}
		
		// Type opening quote - this will create a pair ""
		te.InsertChar('"')
		
		// Type "fmt"
		for _, ch := range "fmt" {
			te.InsertChar(ch)
		}
		
		// Move past the auto-paired closing quote
		te.ExecuteCommand("right")
		
		// Verify the complete content
		AssertBufferEquals(t, te, "package main\nimport \"fmt\"")
		AssertCursorAt(t, te, 11, 1) // After the closing quote
		
		// Force a render to trigger syntax highlighting
		// This would have crashed before the fix
		te.drawEditor()
		te.drawStatusBar()
		
		// Verify we can still render without crashing
		lines := te.GetScreenContent()
		if len(lines) == 0 {
			t.Error("Expected screen content, but got none")
		}
	})
	
	t.Run("syntax tokens are bounded correctly", func(t *testing.T) {
		// Create editor with some Go code
		te := NewTestEditorWithContent(80, 24, "func main() { fmt.Println(\"Hello\") }")
		
		// Set up Go file
		te.buffer.FilePath = "test.go"
		te.highlighter.SetLanguageFromFile("test.go")
		te.highlighter.Parse([]byte(te.buffer.String()))
		
		// Move cursor to middle of line
		te.SetCursorPosition(13, 0) // After "{"
		
		// Enter insert mode and insert newline
		te.ExecuteCommand("enter-insert-mode")
		te.ExecuteCommand("insert-newline")
		
		// Add some content on new line
		te.InsertChar('\t') // Tab for indentation
		
		// Verify content (no auto-indent, then manual tab)
		expected := "func main() {\n\t fmt.Println(\"Hello\") }"
		AssertBufferEquals(t, te, expected)
		
		// Force render - should not crash
		te.drawEditor()
		te.drawStatusBar()
		
		// Verify rendering worked
		lines := te.GetScreenContent()
		if len(lines) < 2 {
			t.Error("Expected at least 2 lines on screen")
		}
	})
	
	t.Run("rapid edits with syntax highlighting", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "var x = 10")
		
		// Enable Go syntax highlighting
		te.buffer.FilePath = "test.go"
		te.highlighter.SetLanguageFromFile("test.go")
		te.highlighter.Parse([]byte(te.buffer.String()))
		
		// Move to end of line
		te.buffer.CursorX = 10
		te.ExecuteCommand("enter-insert-mode")
		
		// Make several rapid edits
		te.ExecuteCommand("insert-newline")
		te.InsertChar('v')
		te.InsertChar('a')
		te.InsertChar('r')
		te.InsertChar(' ')
		te.InsertChar('y')
		te.InsertChar(' ')
		te.InsertChar('=')
		te.InsertChar(' ')
		te.InsertChar('2')
		te.InsertChar('0')
		
		// Verify content
		AssertBufferEquals(t, te, "var x = 10\nvar y = 20")
		
		// Delete some characters
		for i := 0; i < 3; i++ {
			te.ExecuteCommand("backspace")
		}
		
		AssertBufferEquals(t, te, "var x = 10\nvar y =")
		
		// Force multiple renders
		for i := 0; i < 5; i++ {
			te.drawEditor()
			te.drawStatusBar()
		}
		
		// Should not crash
		lines := te.GetScreenContent()
		if len(lines) < 2 {
			t.Error("Expected at least 2 lines after edits")
		}
	})
}

func TestEmptyBufferWithSyntaxHighlighting(t *testing.T) {
	t.Run("empty buffer with syntax highlighting enabled", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "")
		
		// Enable syntax highlighting on empty buffer
		te.buffer.FilePath = "test.go"
		te.highlighter.SetLanguageFromFile("test.go")
		
		// Should not crash when rendering empty buffer
		te.drawEditor()
		te.drawStatusBar()
		
		// Enter insert mode and type
		te.ExecuteCommand("enter-insert-mode")
		te.InsertChar('p')
		te.InsertChar('k')
		te.InsertChar('g')
		
		// Should handle partial keywords
		te.drawEditor()
		AssertBufferEquals(t, te, "pkg")
	})
}