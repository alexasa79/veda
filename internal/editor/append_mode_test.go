package editor

import (
	"testing"
	"veda/internal/commands"
)

func TestAppendAfterCursor(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		cursorX        int
		expectedX      int
		insertText     string
		expectedResult string
	}{
		{
			name:           "append after character in middle of line",
			initialContent: "Hello World",
			cursorX:        5, // On the space
			expectedX:      6, // After the space
			insertText:     "Beautiful ",
			expectedResult: "Hello Beautiful World",
		},
		{
			name:           "append at end of line",
			initialContent: "Hello",
			cursorX:        4, // On 'o'
			expectedX:      5, // After 'o'
			insertText:     " World",
			expectedResult: "Hello World",
		},
		{
			name:           "append after first character",
			initialContent: "Test",
			cursorX:        0, // On 'T'
			expectedX:      1, // After 'T'
			insertText:     "he ",
			expectedResult: "The est",
		},
		{
			name:           "append on empty line",
			initialContent: "",
			cursorX:        0,
			expectedX:      0, // Stay at 0 on empty line
			insertText:     "Hello",
			expectedResult: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.initialContent)
			
			// Set initial cursor position
			te.buffer.CursorX = tt.cursorX
			
			// Execute append after cursor
			te.ExecuteCommand("append-after-cursor")
			
			// Check mode
			if te.mode != commands.InsertMode {
				t.Errorf("Expected to be in insert mode, but got %v", te.mode)
			}
			
			// Check cursor position
			if te.buffer.CursorX != tt.expectedX {
				t.Errorf("Expected cursor at X=%d, but got X=%d", tt.expectedX, te.buffer.CursorX)
			}
			
			// Insert text
			for _, ch := range tt.insertText {
				te.InsertChar(ch)
			}
			
			// Exit insert mode
			te.ExecuteCommand("exit-insert-mode")
			
			// Check final content
			AssertBufferEquals(t, te, tt.expectedResult)
		})
	}
}

func TestAppendAtEndOfLine(t *testing.T) {
	tests := []struct {
		name           string
		initialContent string
		cursorX        int
		insertText     string
		expectedResult string
	}{
		{
			name:           "append at end from beginning",
			initialContent: "Hello",
			cursorX:        0, // At beginning
			insertText:     " World",
			expectedResult: "Hello World",
		},
		{
			name:           "append at end from middle",
			initialContent: "Test Line",
			cursorX:        4, // In middle
			insertText:     " Here",
			expectedResult: "Test Line Here",
		},
		{
			name:           "append at end of empty line",
			initialContent: "",
			cursorX:        0,
			insertText:     "New Text",
			expectedResult: "New Text",
		},
		{
			name:           "multi-line append at end",
			initialContent: "Line1\nLine2\nLine3",
			cursorX:        0,
			insertText:     " Extended",
			expectedResult: "Line1 Extended\nLine2\nLine3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.initialContent)
			
			// Set initial cursor position
			te.buffer.CursorX = tt.cursorX
			
			// Execute append at end of line
			te.ExecuteCommand("append-at-end-of-line")
			
			// Check mode
			if te.mode != commands.InsertMode {
				t.Errorf("Expected to be in insert mode, but got %v", te.mode)
			}
			
			// Check cursor position is at end of line
			line := te.buffer.CursorIterator.Line()
			if te.buffer.CursorX != len(line) {
				t.Errorf("Expected cursor at end of line (X=%d), but got X=%d", len(line), te.buffer.CursorX)
			}
			
			// Insert text
			for _, ch := range tt.insertText {
				te.InsertChar(ch)
			}
			
			// Exit insert mode
			te.ExecuteCommand("exit-insert-mode")
			
			// Check final content
			AssertBufferEquals(t, te, tt.expectedResult)
		})
	}
}

func TestAppendAtEndOfLineMultiLine(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "First Line\nSecond Line\nThird Line")
	
	// Move to second line
	te.ExecuteCommand("down")
	AssertCursorAt(t, te, 0, 1)
	
	// Execute A command
	te.ExecuteCommand("append-at-end-of-line")
	
	// Should be at end of second line
	line := te.buffer.CursorIterator.Line()
	if te.buffer.CursorX != len(line) {
		t.Errorf("Expected cursor at end of line (X=%d), but got X=%d", len(line), te.buffer.CursorX)
	}
	
	// Insert text
	te.InsertChar(' ')
	te.InsertChar('!')
	
	// Exit insert mode
	te.ExecuteCommand("exit-insert-mode")
	
	// Check result
	AssertBufferEquals(t, te, "First Line\nSecond Line !\nThird Line")
}

func TestAppendCommandsWithUndo(t *testing.T) {
	t.Run("append after cursor with undo", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Hello")
		
		// Position cursor on 'e'
		te.buffer.CursorX = 1
		
		// Append after cursor
		te.ExecuteCommand("append-after-cursor")
		te.InsertChar('X')
		te.InsertChar('Y')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "HeXYllo")
		
		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Hello")
	})
	
	t.Run("append at end of line with undo", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Test")
		
		// Append at end of line
		te.ExecuteCommand("append-at-end-of-line")
		te.InsertChar('i')
		te.InsertChar('n')
		te.InsertChar('g')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "Testing")
		
		// Undo
		te.ExecuteCommand("undo")
		AssertBufferEquals(t, te, "Test")
	})
}

func TestAppendWithRepeat(t *testing.T) {
	t.Run("repeat append after cursor", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "One\nTwo\nThree")
		
		// Append after first character of first line
		te.ExecuteCommand("append-after-cursor")
		te.InsertChar('X')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "OXne\nTwo\nThree")
		
		// Move to second line
		te.ExecuteCommand("down")
		
		// Repeat
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "OXne\nTXwo\nThree")
	})
	
	t.Run("repeat append at end of line", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Line1\nLine2\nLine3")
		
		// Append at end of first line
		te.ExecuteCommand("append-at-end-of-line")
		te.InsertChar('!')
		te.ExecuteCommand("exit-insert-mode")
		
		AssertBufferEquals(t, te, "Line1!\nLine2\nLine3")
		
		// Move to third line
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		
		// Repeat
		te.ExecuteCommand("repeat-last-change")
		AssertBufferEquals(t, te, "Line1!\nLine2\nLine3!")
	})
}