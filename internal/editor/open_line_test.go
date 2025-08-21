package editor

import (
	"strings"
	"testing"
	"veda/internal/commands"
)

func TestOpenLineBelow(t *testing.T) {
	t.Run("open line below in middle of file", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"Second line",
			"Third line",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)

		// Move to second line
		te.SetCursorPosition(5, 1)

		// Execute open-line-below
		err := te.ExecuteCommand("open-line-below")
		if err != nil {
			t.Fatalf("open-line-below failed: %v", err)
		}

		// Should be in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected to be in insert mode, but mode is %v", te.mode)
		}

		// Should be on new empty line
		AssertCursorAt(t, te, 0, 2)

		// Buffer should have new empty line
		expected := strings.Join([]string{
			"First line",
			"Second line",
			"",
			"Third line",
		}, "\n")
		AssertBufferEquals(t, te, expected)
	})

	t.Run("open line below at end of file", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"Last line",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)

		// Move to last line
		te.SetCursorPosition(3, 1)

		// Execute open-line-below
		err := te.ExecuteCommand("open-line-below")
		if err != nil {
			t.Fatalf("open-line-below failed: %v", err)
		}

		// Should be on new line at end
		AssertCursorAt(t, te, 0, 2)

		// Buffer should have new empty line at end
		expected := strings.Join([]string{
			"First line",
			"Last line",
			"",
		}, "\n")
		AssertBufferEquals(t, te, expected)
	})

	t.Run("open line below on empty line", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"",
			"Third line",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)

		// Move to empty line
		te.SetCursorPosition(0, 1)

		// Execute open-line-below
		err := te.ExecuteCommand("open-line-below")
		if err != nil {
			t.Fatalf("open-line-below failed: %v", err)
		}

		// Should be on new empty line
		AssertCursorAt(t, te, 0, 2)

		// Buffer should have new empty line
		expected := strings.Join([]string{
			"First line",
			"",
			"",
			"Third line",
		}, "\n")
		AssertBufferEquals(t, te, expected)
	})

	t.Run("open line below in single line file", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Only line")

		// Execute open-line-below
		err := te.ExecuteCommand("open-line-below")
		if err != nil {
			t.Fatalf("open-line-below failed: %v", err)
		}

		// Should be on new line
		AssertCursorAt(t, te, 0, 1)

		// Buffer should have new empty line
		expected := strings.Join([]string{
			"Only line",
			"",
		}, "\n")
		AssertBufferEquals(t, te, expected)
	})
}

func TestOpenLineAbove(t *testing.T) {
	t.Run("open line above in middle of file", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"Second line",
			"Third line",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)

		// Move to second line
		te.SetCursorPosition(5, 1)

		// Execute open-line-above
		err := te.ExecuteCommand("open-line-above")
		if err != nil {
			t.Fatalf("open-line-above failed: %v", err)
		}

		// Should be in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected to be in insert mode, but mode is %v", te.mode)
		}

		// Should be on new empty line (same Y position as before)
		AssertCursorAt(t, te, 0, 1)

		// Buffer should have new empty line
		expected := strings.Join([]string{
			"First line",
			"",
			"Second line",
			"Third line",
		}, "\n")
		AssertBufferEquals(t, te, expected)
	})

	t.Run("open line above at beginning of file", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"Second line",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)

		// At first line
		te.SetCursorPosition(3, 0)

		// Execute open-line-above
		err := te.ExecuteCommand("open-line-above")
		if err != nil {
			t.Fatalf("open-line-above failed: %v", err)
		}

		// Should be on new first line
		AssertCursorAt(t, te, 0, 0)

		// Buffer should have new empty line at beginning
		expected := strings.Join([]string{
			"",
			"First line",
			"Second line",
		}, "\n")
		AssertBufferEquals(t, te, expected)
	})

	t.Run("open line above on empty line", func(t *testing.T) {
		content := strings.Join([]string{
			"First line",
			"",
			"Third line",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)

		// Move to empty line
		te.SetCursorPosition(0, 1)

		// Execute open-line-above
		err := te.ExecuteCommand("open-line-above")
		if err != nil {
			t.Fatalf("open-line-above failed: %v", err)
		}

		// Should be on new empty line
		AssertCursorAt(t, te, 0, 1)

		// Buffer should have new empty line
		expected := strings.Join([]string{
			"First line",
			"",
			"",
			"Third line",
		}, "\n")
		AssertBufferEquals(t, te, expected)
	})

	t.Run("open line above in single line file", func(t *testing.T) {
		te := NewTestEditorWithContent(80, 24, "Only line")

		// Execute open-line-above
		err := te.ExecuteCommand("open-line-above")
		if err != nil {
			t.Fatalf("open-line-above failed: %v", err)
		}

		// Should be on new first line
		AssertCursorAt(t, te, 0, 0)

		// Buffer should have new empty line at beginning
		expected := strings.Join([]string{
			"",
			"Only line",
		}, "\n")
		AssertBufferEquals(t, te, expected)
	})
}

func TestOpenLineKeyBindings(t *testing.T) {
	t.Run("o key binding", func(t *testing.T) {
		content := strings.Join([]string{
			"Line 1",
			"Line 2",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)

		// Start in normal mode
		if te.mode != commands.NormalMode {
			t.Fatalf("Expected to start in normal mode")
		}

		// Execute 'o' command
		err := te.ExecuteCommand("open-line-below")
		if err != nil {
			t.Fatalf("open-line-below failed: %v", err)
		}

		// Should be in insert mode on new line
		if te.mode != commands.InsertMode {
			t.Errorf("Expected to be in insert mode after 'o'")
		}
		AssertCursorAt(t, te, 0, 1)
	})

	t.Run("O key binding", func(t *testing.T) {
		content := strings.Join([]string{
			"Line 1",
			"Line 2",
		}, "\n")

		te := NewTestEditorWithContent(80, 24, content)

		// Move to second line
		te.SetCursorPosition(0, 1)

		// Start in normal mode
		if te.mode != commands.NormalMode {
			t.Fatalf("Expected to start in normal mode")
		}

		// Execute 'O' command
		err := te.ExecuteCommand("open-line-above")
		if err != nil {
			t.Fatalf("open-line-above failed: %v", err)
		}

		// Should be in insert mode on new line above
		if te.mode != commands.InsertMode {
			t.Errorf("Expected to be in insert mode after 'O'")
		}
		AssertCursorAt(t, te, 0, 1)
	})
}
