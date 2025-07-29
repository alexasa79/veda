package editor

import (
	"testing"
	"veda/internal/commands"
	"veda/internal/events"
)

func TestCommandModeLineJump(t *testing.T) {
	// Create test editor with multiple lines
	content := `Line 1
Line 2
Line 3
Line 4
Line 5
Line 6
Line 7
Line 8
Line 9
Line 10`
	
	te := NewTestEditorWithContent(80, 24, content)
	
	// Test jumping to various line numbers
	tests := []struct {
		name          string
		command       string
		expectedLine  int
		expectedCol   int
	}{
		{"Jump to line 5", "5", 4, 0},      // 0-based, so line 5 is index 4
		{"Jump to line 1", "1", 0, 0},
		{"Jump to line 10", "10", 9, 0},
		{"Jump to line 3", "3", 2, 0},
		{"Jump beyond end", "20", 9, 0},    // Should clamp to last line
		{"Jump to line 0", "0", 0, 0},      // Should go to first line
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Enter command mode
			te.ExecuteCommand("enter-command-mode")
			if te.mode != commands.CommandMode {
				t.Fatal("Failed to enter command mode")
			}
			
			// Type the line number
			for _, ch := range test.command {
				ev := &events.KeyEvent{
					Key:  events.KeyRune,
					Rune: ch,
				}
				te.commandLine.HandleKey(ev)
			}
			
			// Press Enter to execute
			te.executeCommand()
			te.exitCommandMode()
			
			// Check cursor position
			x, y := te.GetCursorPosition()
			if x != test.expectedCol || y != test.expectedLine {
				t.Errorf("Expected cursor at (%d, %d) after command :%s, but got (%d, %d)",
					test.expectedCol, test.expectedLine, test.command, x, y)
			}
		})
	}
}

func TestCommandModeLineJumpWithExistingCursor(t *testing.T) {
	// Test that jumping to a line works correctly from different starting positions
	content := `First line
Second line
Third line
Fourth line
Fifth line`
	
	te := NewTestEditorWithContent(80, 24, content)
	
	// Start at line 3, column 5
	te.SetCursorPosition(5, 2)
	
	// Jump to line 1
	te.ExecuteCommand("enter-command-mode")
	for _, ch := range "1" {
		ev := &events.KeyEvent{
			Key:  events.KeyRune,
			Rune: ch,
		}
		te.commandLine.HandleKey(ev)
	}
	te.executeCommand()
	te.exitCommandMode()
	
	// Should be at line 1, column 0
	AssertCursorAt(t, te, 0, 0)
	
	// Now jump to line 4
	te.ExecuteCommand("enter-command-mode")
	for _, ch := range "4" {
		ev := &events.KeyEvent{
			Key:  events.KeyRune,
			Rune: ch,
		}
		te.commandLine.HandleKey(ev)
	}
	te.executeCommand()
	te.exitCommandMode()
	
	// Should be at line 4, column 0
	AssertCursorAt(t, te, 0, 3)
}

func TestCommandModeLineJumpEmptyBuffer(t *testing.T) {
	// Test jumping in an empty buffer
	te := NewTestEditorWithContent(80, 24, "")
	
	// Try to jump to line 5 in empty buffer
	te.ExecuteCommand("enter-command-mode")
	for _, ch := range "5" {
		ev := &events.KeyEvent{
			Key:  events.KeyRune,
			Rune: ch,
		}
		te.commandLine.HandleKey(ev)
	}
	te.executeCommand()
	te.exitCommandMode()
	
	// Should stay at 0,0
	AssertCursorAt(t, te, 0, 0)
}

func TestCommandModeLineJumpSingleLine(t *testing.T) {
	// Test jumping in a single-line buffer
	te := NewTestEditorWithContent(80, 24, "Only one line here")
	
	// Try to jump to line 2
	te.ExecuteCommand("enter-command-mode")
	for _, ch := range "2" {
		ev := &events.KeyEvent{
			Key:  events.KeyRune,
			Rune: ch,
		}
		te.commandLine.HandleKey(ev)
	}
	te.executeCommand()
	te.exitCommandMode()
	
	// Should stay at line 1 (index 0)
	AssertCursorAt(t, te, 0, 0)
}