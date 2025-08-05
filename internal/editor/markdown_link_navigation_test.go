package editor

import (
	"fmt"
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestMarkdownLinkNavigation(t *testing.T) {
	t.Run("follow markdown link without line number", func(t *testing.T) {
		// Create editor with a markdown link
		te := NewTestEditorWithContent(80, 24, "Check [this file](test.go) for details")
		te.buffer.FilePath = "test.md"
		
		// Move cursor to the link (accounting for markdown rendering)
		// In normal mode, this renders as "Check this file for details"
		// Move cursor to "this" which starts at position 6
		for i := 0; i < 6; i++ {
			te.ExecuteCommand("right")
		}
		
		// Create a mock file to open
		te.buffers["test.go"] = buffer.NewBuffer("package main\n\nfunc main() {\n\t// Test file\n}", "test.go", config.IndentSpaces, 4)
		
		// Press Enter to follow the link
		te.ExecuteCommand("jump-to-reference")
		
		// Should have opened test.go
		if te.buffer.FilePath != "test.go" {
			t.Errorf("Expected to open test.go, but current file is %s", te.buffer.FilePath)
		}
		
		// Should be at the beginning of the file
		x, y := te.GetCursorPosition()
		if x != 0 || y != 0 {
			t.Errorf("Expected cursor at (0, 0), got (%d, %d)", x, y)
		}
	})
	
	t.Run("follow markdown link with line number", func(t *testing.T) {
		// Create editor with a markdown link that includes a line number
		te := NewTestEditorWithContent(80, 24, "See [function](main.go#L10) for implementation")
		te.buffer.FilePath = "README.md"
		
		// Create a target file with multiple lines
		content := ""
		for i := 1; i <= 20; i++ {
			content += fmt.Sprintf("Line %d\n", i)
		}
		te.buffers["main.go"] = buffer.NewBuffer(content, "main.go", config.IndentSpaces, 4)
		
		// Move cursor to the link
		// In normal mode, this renders as "See function for implementation"
		// Move cursor to "function" which starts at position 4
		for i := 0; i < 4; i++ {
			te.ExecuteCommand("right")
		}
		
		// Press Enter to follow the link
		te.ExecuteCommand("jump-to-reference")
		
		// Should have opened main.go
		if te.buffer.FilePath != "main.go" {
			t.Errorf("Expected to open main.go, but current file is %s", te.buffer.FilePath)
		}
		
		// Should be at line 10 (0-based index 9)
		lineNum := te.buffer.CursorIterator.LineNumber()
		if lineNum != 9 {
			t.Errorf("Expected to be at line 10 (index 9), but at line %d", lineNum+1)
		}
		
		// Cursor should be at beginning of the line
		if te.buffer.CursorX != 0 {
			t.Errorf("Expected cursor X to be 0, got %d", te.buffer.CursorX)
		}
	})
	
	t.Run("cursor not on link does nothing", func(t *testing.T) {
		// Create editor with a markdown link
		te := NewTestEditorWithContent(80, 24, "Check [this file](test.go) for details")
		te.buffer.FilePath = "test.md"
		originalFile := te.buffer.FilePath
		
		// Move cursor to a position not on the link
		// Move to the end of the line
		te.ExecuteCommand("end-of-line")
		
		// Press Enter
		te.ExecuteCommand("jump-to-reference")
		
		// Should still be in the same file
		if te.buffer.FilePath != originalFile {
			t.Errorf("File should not have changed, but changed from %s to %s", originalFile, te.buffer.FilePath)
		}
	})
	
	t.Run("follow link in insert mode", func(t *testing.T) {
		// Create editor with a markdown link
		te := NewTestEditorWithContent(80, 24, "Check [this file](test.go) for details")
		te.buffer.FilePath = "test.md"
		
		// Enter insert mode
		te.ExecuteCommand("enter-insert-mode")
		
		// In insert mode, the full markdown syntax is shown
		// Move cursor to be within the link syntax
		for i := 0; i < 10; i++ { // Move to somewhere in "[this file]"
			te.ExecuteCommand("right-insert")
		}
		
		// Create a mock file to open
		te.buffers["test.go"] = buffer.NewBuffer("// Test file", "test.go", config.IndentSpaces, 4)
		
		// Exit insert mode first
		te.ExecuteCommand("exit-insert-mode")
		
		// Press Enter to follow the link
		te.ExecuteCommand("jump-to-reference")
		
		// Should have opened test.go
		if te.buffer.FilePath != "test.go" {
			t.Errorf("Expected to open test.go, but current file is %s", te.buffer.FilePath)
		}
	})
	
	t.Run("navigation history is preserved", func(t *testing.T) {
		// Create editor with a markdown link
		te := NewTestEditorWithContent(80, 24, "Go to [target](target.go#L5)")
		te.buffer.FilePath = "source.md"
		
		// Create target file
		te.buffers["target.go"] = buffer.NewBuffer("Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6", "target.go", config.IndentSpaces, 4)
		
		// Move cursor to the link
		for i := 0; i < 6; i++ { // Move to "target"
			te.ExecuteCommand("right")
		}
		
		// Follow the link
		te.ExecuteCommand("jump-to-reference")
		
		// Should be in target.go at line 5
		if te.buffer.FilePath != "target.go" {
			t.Errorf("Expected to be in target.go, but in %s", te.buffer.FilePath)
		}
		
		lineNum := te.buffer.CursorIterator.LineNumber()
		if lineNum != 4 { // Line 5 is index 4
			t.Errorf("Expected to be at line 5 (index 4), but at line %d", lineNum+1)
		}
		
		// Go back
		te.ExecuteCommand("go-back")
		
		// Should be back in source.md
		if te.buffer.FilePath != "source.md" {
			t.Errorf("Expected to be back in source.md, but in %s", te.buffer.FilePath)
		}
	})
}