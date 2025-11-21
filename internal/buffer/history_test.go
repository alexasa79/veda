package buffer

import (
	"testing"
	"veda/internal/config"
)

func TestUndoRedoBasics(t *testing.T) {
	t.Run("simple insert and undo", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Verify buffer is properly initialized
		if buf.Rope == nil {
			t.Fatal("Buffer rope is nil after creation")
		}
		if buf.Rope.root == nil {
			t.Fatal("Buffer rope root is nil after creation")
		}

		// Insert text
		entry := NewInsertTextEntry(0, 0, "Hello")
		_, err := buf.History.Execute(buf, entry)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Verify text was inserted
		if buf.Rope.String() != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", buf.Rope.String())
		}

		// Check rope state before undo
		if buf.Rope == nil {
			t.Fatal("Buffer rope is nil before undo")
		}
		if buf.Rope.root == nil {
			t.Fatal("Buffer rope root is nil before undo")
		}

		// Undo
		err = buf.History.Undo(buf)
		if err != nil {
			t.Fatalf("Undo failed: %v", err)
		}

		// Verify text was removed
		if buf.Rope.String() != "" {
			t.Errorf("Expected empty buffer after undo, got '%s'", buf.Rope.String())
		}
	})

	t.Run("simple delete and undo", func(t *testing.T) {
		buf := NewBuffer("Hello", "", "", config.IndentTabs, 4)

		// Delete text
		entry := NewDeleteTextEntry(0, 1, "ello")
		_, err := buf.History.Execute(buf, entry)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Verify text was deleted
		if buf.Rope.String() != "H" {
			t.Errorf("Expected 'H', got '%s'", buf.Rope.String())
		}

		// Undo
		err = buf.History.Undo(buf)
		if err != nil {
			t.Fatalf("Undo failed: %v", err)
		}

		// Verify text was restored
		if buf.Rope.String() != "Hello" {
			t.Errorf("Expected 'Hello' after undo, got '%s'", buf.Rope.String())
		}
	})

	t.Run("undo then redo", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Insert text
		entry := NewInsertTextEntry(0, 0, "Test")
		_, _ = buf.History.Execute(buf, entry)

		// Undo
		buf.History.Undo(buf)
		if buf.Rope.String() != "" {
			t.Errorf("Expected empty after undo, got '%s'", buf.Rope.String())
		}

		// Redo
		err := buf.History.Redo(buf)
		if err != nil {
			t.Fatalf("Redo failed: %v", err)
		}

		// Verify text was restored
		if buf.Rope.String() != "Test" {
			t.Errorf("Expected 'Test' after redo, got '%s'", buf.Rope.String())
		}
	})

	t.Run("multiple undos and redos", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Execute multiple operations
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 0, "One"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 3, " Two"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 7, " Three"))

		if buf.Rope.String() != "One Two Three" {
			t.Errorf("Expected 'One Two Three', got '%s'", buf.Rope.String())
		}

		// Undo all
		buf.History.Undo(buf) // Remove " Three"
		if buf.Rope.String() != "One Two" {
			t.Errorf("After first undo expected 'One Two', got '%s'", buf.Rope.String())
		}

		buf.History.Undo(buf) // Remove " Two"
		if buf.Rope.String() != "One" {
			t.Errorf("After second undo expected 'One', got '%s'", buf.Rope.String())
		}

		buf.History.Undo(buf) // Remove "One"
		if buf.Rope.String() != "" {
			t.Errorf("After third undo expected empty, got '%s'", buf.Rope.String())
		}

		// Redo all
		buf.History.Redo(buf) // Add "One"
		if buf.Rope.String() != "One" {
			t.Errorf("After first redo expected 'One', got '%s'", buf.Rope.String())
		}

		buf.History.Redo(buf) // Add " Two"
		if buf.Rope.String() != "One Two" {
			t.Errorf("After second redo expected 'One Two', got '%s'", buf.Rope.String())
		}

		buf.History.Redo(buf) // Add " Three"
		if buf.Rope.String() != "One Two Three" {
			t.Errorf("After third redo expected 'One Two Three', got '%s'", buf.Rope.String())
		}
	})

	t.Run("redo stack cleared on new operation", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Add some operations
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 0, "First"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 5, " Second"))

		// Undo once
		buf.History.Undo(buf)

		// Execute new operation (should clear redo stack)
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 5, " New"))

		// Try to redo - should have no effect
		buf.History.Redo(buf)

		// Verify final state
		if buf.Rope.String() != "First New" {
			t.Errorf("Expected 'First New', got '%s'", buf.Rope.String())
		}
	})
}

func TestInsertModeSession(t *testing.T) {
	t.Run("insert mode session groups changes", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Start insert mode session
		buf.History.StartInsertSession("i")

		// Multiple operations in insert mode
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 0, "H"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 1, "e"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 2, "l"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 3, "l"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 4, "o"))

		// End insert mode session
		buf.History.EndInsertSession()

		if buf.Rope.String() != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", buf.Rope.String())
		}

		// Single undo should remove all text from insert session
		buf.History.Undo(buf)

		if buf.Rope.String() != "" {
			t.Errorf("Expected empty buffer after undo, got '%s'", buf.Rope.String())
		}

		// Single redo should restore all text
		buf.History.Redo(buf)

		if buf.Rope.String() != "Hello" {
			t.Errorf("Expected 'Hello' after redo, got '%s'", buf.Rope.String())
		}
	})

	t.Run("empty insert session not added to history", func(t *testing.T) {
		buf := NewBuffer("Initial", "", "", config.IndentTabs, 4)

		// Add a normal operation
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 7, " Text"))

		// Start and end empty insert session
		buf.History.StartInsertSession("i")
		buf.History.EndInsertSession()

		// Undo should remove " Text", not do nothing
		buf.History.Undo(buf)

		if buf.Rope.String() != "Initial" {
			t.Errorf("Expected 'Initial' after undo, got '%s'", buf.Rope.String())
		}
	})

	t.Run("insert mode with mixed operations", func(t *testing.T) {
		buf := NewBuffer("Hello", "", "", config.IndentTabs, 4)

		// Start insert mode session
		buf.History.StartInsertSession("i")

		// Mix of inserts and deletes in insert mode
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 5, " "))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 6, "W"))
		_, _ = buf.History.Execute(buf, NewDeleteTextEntry(0, 6, "W")) // Backspace
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 6, "w"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 7, "o"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 8, "r"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 9, "l"))
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 10, "d"))

		// End insert mode session
		buf.History.EndInsertSession()

		if buf.Rope.String() != "Hello world" {
			t.Errorf("Expected 'Hello world', got '%s'", buf.Rope.String())
		}

		// Single undo should revert entire insert session
		buf.History.Undo(buf)

		if buf.Rope.String() != "Hello" {
			t.Errorf("Expected 'Hello' after undo, got '%s'", buf.Rope.String())
		}
	})
}

func TestMultilineOperations(t *testing.T) {
	t.Run("insert newline and undo", func(t *testing.T) {
		buf := NewBuffer("Line1", "", "", config.IndentTabs, 4)

		// Insert newline at end of Line1
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 5, "\n"))

		// Check intermediate state
		if buf.Rope.String() != "Line1\n" {
			t.Errorf("After newline, expected 'Line1\\n', got '%s'", buf.Rope.String())
		}

		// Debug: Let's check where line 1 starts
		iter := NewBufferIteratorAtLine(buf, 1)
		t.Logf("Line 1 starts at offset %d", iter.Offset)
		t.Logf("Buffer length: %d", buf.Rope.Length())

		// Insert text on new line
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(1, 0, "Line2"))

		if buf.Rope.String() != "Line1\nLine2" {
			t.Errorf("Expected 'Line1\\nLine2', got '%s'", buf.Rope.String())
		}

		// Undo both
		buf.History.Undo(buf)
		buf.History.Undo(buf)

		if buf.Rope.String() != "Line1" {
			t.Errorf("Expected 'Line1' after undo, got '%s'", buf.Rope.String())
		}
	})

	t.Run("delete across lines", func(t *testing.T) {
		buf := NewBuffer("Line1\nLine2", "", "", config.IndentTabs, 4)

		// Delete from middle of Line1 to middle of Line2
		_, _ = buf.History.Execute(buf, NewDeleteTextEntry(0, 3, "e1\nLi"))

		if buf.Rope.String() != "Linne2" {
			t.Errorf("Expected 'Linne2', got '%s'", buf.Rope.String())
		}

		// Undo
		buf.History.Undo(buf)

		if buf.Rope.String() != "Line1\nLine2" {
			t.Errorf("Expected 'Line1\\nLine2' after undo, got '%s'", buf.Rope.String())
		}
	})
}

func TestCursorPosition(t *testing.T) {
	t.Run("cursor position after insert and undo", func(t *testing.T) {
		buf := NewBuffer("Test", "", "", config.IndentTabs, 4)

		// Insert at position 2
		entry := NewInsertTextEntry(0, 2, "X")
		_, _ = buf.History.Execute(buf, entry)

		// Check cursor moved forward
		if buf.CursorX != 3 {
			t.Errorf("Expected cursor at 3, got %d", buf.CursorX)
		}

		// Undo
		buf.History.Undo(buf)

		// Check cursor restored
		if buf.CursorX != 2 {
			t.Errorf("Expected cursor at 2 after undo, got %d", buf.CursorX)
		}
	})

	t.Run("cursor position after delete and undo", func(t *testing.T) {
		buf := NewBuffer("Test", "", "", config.IndentTabs, 4)

		// Move cursor to position 4
		buf.CursorIterator = NewBufferIterator(buf)
		buf.CursorX = 4

		// Delete character before cursor
		entry := NewDeleteTextEntry(0, 3, "t")
		_, _ = buf.History.Execute(buf, entry)

		// Check cursor moved back
		if buf.CursorX != 3 {
			t.Errorf("Expected cursor at 3, got %d", buf.CursorX)
		}

		// Undo
		buf.History.Undo(buf)

		// Check cursor restored
		if buf.CursorX != 3 {
			t.Errorf("Expected cursor at 3 after undo, got %d", buf.CursorX)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("undo on empty history", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Should not panic or error
		err := buf.History.Undo(buf)
		if err != nil {
			t.Errorf("Undo on empty history returned error: %v", err)
		}
	})

	t.Run("redo on empty redo stack", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Should not panic or error
		err := buf.History.Redo(buf)
		if err != nil {
			t.Errorf("Redo on empty stack returned error: %v", err)
		}
	})

	t.Run("operations at buffer boundaries", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Insert at beginning
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 0, "Start"))

		// Insert at end
		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 5, "End"))

		if buf.Rope.String() != "StartEnd" {
			t.Errorf("Expected 'StartEnd', got '%s'", buf.Rope.String())
		}

		// Undo both
		buf.History.Undo(buf)
		buf.History.Undo(buf)

		if buf.Rope.String() != "" {
			t.Errorf("Expected empty buffer, got '%s'", buf.Rope.String())
		}
	})

	t.Run("large text operations", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Insert large text
		largeText := ""
		for i := 0; i < 100; i++ {
			largeText += "Line " + string(rune('0'+i%10)) + "\n"
		}

		_, _ = buf.History.Execute(buf, NewInsertTextEntry(0, 0, largeText))

		// Verify insertion
		if buf.Rope.String() != largeText {
			t.Errorf("Large text insertion failed")
		}

		// Undo
		buf.History.Undo(buf)

		if buf.Rope.String() != "" {
			t.Errorf("Expected empty buffer after undo, got non-empty")
		}

		// Redo
		buf.History.Redo(buf)

		if buf.Rope.String() != largeText {
			t.Errorf("Large text redo failed")
		}
	})
}

func TestCompositeEntry(t *testing.T) {
	t.Run("composite entry basic operations", func(t *testing.T) {
		buf := NewBuffer("", "", "", config.IndentTabs, 4)

		// Create composite entry
		composite := NewCompositeEntry("Test composite")
		composite.Add(NewInsertTextEntry(0, 0, "A"))
		composite.Add(NewInsertTextEntry(0, 1, "B"))
		composite.Add(NewInsertTextEntry(0, 2, "C"))

		// Execute composite
		err := composite.Execute(buf)
		if err != nil {
			t.Fatalf("Composite execute failed: %v", err)
		}

		if buf.Rope.String() != "ABC" {
			t.Errorf("Expected 'ABC', got '%s'", buf.Rope.String())
		}

		// Undo composite
		err = composite.Undo(buf)
		if err != nil {
			t.Fatalf("Composite undo failed: %v", err)
		}

		if buf.Rope.String() != "" {
			t.Errorf("Expected empty after composite undo, got '%s'", buf.Rope.String())
		}
	})

	t.Run("composite with mixed operations", func(t *testing.T) {
		buf := NewBuffer("Original", "", "", config.IndentTabs, 4)

		// Create composite with inserts and deletes
		composite := NewCompositeEntry("Mixed operations")
		composite.Add(NewInsertTextEntry(0, 8, " Text"))    // "Original Text"
		composite.Add(NewDeleteTextEntry(0, 0, "Original")) // " Text"
		composite.Add(NewInsertTextEntry(0, 0, "New"))      // "New Text"

		// Execute
		composite.Execute(buf)

		if buf.Rope.String() != "New Text" {
			t.Errorf("Expected 'New Text', got '%s'", buf.Rope.String())
		}

		// Undo
		composite.Undo(buf)

		if buf.Rope.String() != "Original" {
			t.Errorf("Expected 'Original' after undo, got '%s'", buf.Rope.String())
		}
	})
}

func TestHistoryDescriptions(t *testing.T) {
	t.Run("entry descriptions", func(t *testing.T) {
		// Test insert descriptions
		insert1 := NewInsertTextEntry(0, 0, "a")
		if insert1.Description() != "Insert char: a" {
			t.Errorf("Wrong description for single char insert: %s", insert1.Description())
		}

		insert2 := NewInsertTextEntry(0, 0, "hello")
		if insert2.Description() != "Insert text: hello" {
			t.Errorf("Wrong description for text insert: %s", insert2.Description())
		}

		// Test delete descriptions
		delete1 := NewDeleteTextEntry(0, 0, "x")
		if delete1.Description() != "Delete char: x" {
			t.Errorf("Wrong description for single char delete: %s", delete1.Description())
		}

		delete2 := NewDeleteTextEntry(0, 0, "world")
		if delete2.Description() != "Delete text: world" {
			t.Errorf("Wrong description for text delete: %s", delete2.Description())
		}

		// Test composite description
		composite := NewCompositeEntry("Test composite")
		if composite.Description() != "Test composite" {
			t.Errorf("Wrong composite description: %s", composite.Description())
		}
	})
}
