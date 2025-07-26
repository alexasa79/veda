package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/proxy"
)

func TestBlockTextObjectWithFoldingRanges(t *testing.T) {
	t.Run("dib with cached folding ranges", func(t *testing.T) {
		content := `func main() {
	if x > 0 {
		println("positive")
	}
}`
		
		// Create buffer with content
		buf := buffer.NewBuffer(content, "test.go", config.IndentSpaces, 4)
		
		// Add mock folding ranges (simulating what LSP would provide)
		buf.FoldingRanges = []proxy.FoldingRange{
			{StartLine: 0, EndLine: 4}, // main function
			{StartLine: 1, EndLine: 3}, // if block
		}
		
		// Create block text object with "inside" modifier
		blockObj := NewBlockTextObject(ModifierInside)
		
		// Test getting range when cursor is inside the if block
		startLine, startChar, endLine, _, found := blockObj.GetRange(buf, 2, 0)
		
		if !found {
			t.Error("Expected to find block range")
		}
		
		// For "inside" modifier, we expect to get lines 2-2 (excluding the if line and closing brace)
		if startLine != 2 || endLine != 2 {
			t.Errorf("Expected range lines 2-2, got %d-%d", startLine, endLine)
		}
		
		if startChar != 0 {
			t.Errorf("Expected start char 0, got %d", startChar)
		}
	})
	
	t.Run("dab with cached folding ranges", func(t *testing.T) {
		content := `func main() {
	if x > 0 {
		println("positive")
	}
}`
		
		// Create buffer with content
		buf := buffer.NewBuffer(content, "test.go", config.IndentSpaces, 4)
		
		// Add mock folding ranges
		buf.FoldingRanges = []proxy.FoldingRange{
			{StartLine: 0, EndLine: 4}, // main function
			{StartLine: 1, EndLine: 3}, // if block
		}
		
		// Create block text object with "around" modifier
		blockObj := NewBlockTextObject(ModifierAround)
		
		// Get range when cursor is inside the if block
		startLine, startChar, endLine, endChar, found := blockObj.GetRange(buf, 2, 0)
		
		if !found {
			t.Error("Expected to find block range")
		}
		
		// For "around" modifier, we expect to get the entire if block (lines 1-3)
		if startLine != 1 || endLine != 3 {
			t.Errorf("Expected range lines 1-3, got %d-%d", startLine, endLine)
		}
		
		if startChar != 0 {
			t.Errorf("Expected start char 0, got %d", startChar)
		}
		
		// End char should be end of line 3
		iter := buffer.NewBufferIteratorAtLine(buf, 3)
		expectedEndChar := len(iter.Line())
		if endChar != expectedEndChar {
			t.Errorf("Expected end char %d, got %d", expectedEndChar, endChar)
		}
	})
	
	t.Run("nested blocks select innermost", func(t *testing.T) {
		content := `func process() {
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			println(i)
		}
	}
}`
		
		// Create buffer with content
		buf := buffer.NewBuffer(content, "test.go", config.IndentSpaces, 4)
		
		// Add mock folding ranges
		buf.FoldingRanges = []proxy.FoldingRange{
			{StartLine: 0, EndLine: 6}, // function
			{StartLine: 1, EndLine: 5}, // for loop
			{StartLine: 2, EndLine: 4}, // if block
		}
		
		// Create block text object
		blockObj := NewBlockTextObject(ModifierAround)
		
		// Cursor on line 3 (inside the if block)
		startLine, _, endLine, _, found := blockObj.GetRange(buf, 3, 0)
		
		if !found {
			t.Error("Expected to find block range")
		}
		
		// Should select the innermost block (if statement)
		if startLine != 2 || endLine != 4 {
			t.Errorf("Expected innermost block (lines 2-4), got %d-%d", startLine, endLine)
		}
	})
}