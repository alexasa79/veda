package editor

import (
	"testing"
	"veda/internal/clipboard"
	"veda/internal/events"
)

func TestNumberedYankOperations(t *testing.T) {
	t.Run("2yy yanks two lines", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "line one\nline two\nline three\nline four")
		
		// Directly call handleEvent to simulate the real key handling
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '2'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		
		// Get clipboard content
		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("Failed to get clipboard content: %v", err)
		}
		
		// Should have yanked two lines with newlines
		expected := "line one\nline two\n"
		if content != expected {
			t.Errorf("Expected clipboard content:\n%q\nGot:\n%q", expected, content)
		}
		
		// Cursor should remain at the same position
		AssertCursorAt(t, te, 0, 0)
	})
	
	t.Run("3yy yanks three lines", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "first\nsecond\nthird\nfourth\nfifth")
		
		// Move to second line
		te.ExecuteCommand("down")
		
		// Yank three lines starting from the second line
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '3'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		
		// Get clipboard content
		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("Failed to get clipboard content: %v", err)
		}
		
		// Should have yanked three lines
		expected := "second\nthird\nfourth\n"
		if content != expected {
			t.Errorf("Expected clipboard content:\n%q\nGot:\n%q", expected, content)
		}
		
		// Cursor should remain at the same position (line 1, char 0)
		AssertCursorAt(t, te, 0, 1)
	})
	
	t.Run("yank count exceeds available lines", func(t *testing.T) {
		// Create editor with only 3 lines
		te := NewTestEditorWithContent(80, 24, "one\ntwo\nthree")
		
		// Move to second line
		te.ExecuteCommand("down")
		
		// Try to yank 5 lines (more than available)
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '5'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		
		// Get clipboard content
		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("Failed to get clipboard content: %v", err)
		}
		
		// Should yank only the available lines (lines 2 and 3) with trailing newline
		expected := "two\nthree\n"
		if content != expected {
			t.Errorf("Expected clipboard content:\n%q\nGot:\n%q", expected, content)
		}
	})
	
	t.Run("1yy behaves same as yy", func(t *testing.T) {
		// Create editor with test content
		te := NewTestEditorWithContent(80, 24, "hello\nworld")
		
		// Yank one line with count
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: '1'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'y'})
		
		// Get clipboard content
		content, err := clipboard.Paste()
		if err != nil {
			t.Fatalf("Failed to get clipboard content: %v", err)
		}
		
		// Should have yanked one line with newline
		expected := "hello\n"
		if content != expected {
			t.Errorf("Expected clipboard content:\n%q\nGot:\n%q", expected, content)
		}
	})
}