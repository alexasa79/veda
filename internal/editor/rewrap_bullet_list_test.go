package editor

import (
	"strings"
	"testing"

	"veda/internal/events"
)

// TestRewrapBulletListFromLastLine tests that rewrapping a bullet list item
// from its last line does not incorrectly indent the bullet marker.
func TestRewrapBulletListFromLastLine(t *testing.T) {
	// Create editor with width > 110 characters as specified
	te := NewTestEditorWithContent(120, 24, "")
	te.CurrentBuffer().FilePath = "test.md"

	// Set vertical ruler to control wrap width (e.g., 80 characters)
	te.config.VerticalRulers = []int{80}

	// Enter insert mode
	te.TypeKeys("i")

	// Type a multiline bullet list item
	// The text should be long enough to span multiple lines when rewrapped
	bulletText := "- This is a bullet list item that spans multiple lines. " +
		"It contains enough text to require wrapping when the paragraph " +
		"is reformatted. We want to test that the bullet marker stays " +
		"at column zero."
	te.TypeKeys(bulletText)

	// Exit insert mode using PressKey for proper escape handling
	te.PressKey(events.KeyEscape, 0)

	t.Logf("Buffer before rewrap:\n%s", te.GetBufferContent())
	t.Logf("Cursor at line %d, col %d", te.CurrentBuffer().CursorIterator.LineNumber(), te.CurrentBuffer().CursorX)

	// The content should be on a single line initially
	// Now rewrap it so it becomes multiple lines using alt+q
	te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'q', Modifiers: events.ModAlt})

	t.Logf("Buffer after first rewrap:\n%s", te.GetBufferContent())

	// Count the lines after rewrap
	content := te.GetBufferContent()
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected multiple lines after rewrap, got %d lines", len(lines))
	}

	t.Logf("Number of lines after rewrap: %d", len(lines))

	// Go to the last non-empty line of the bullet list item
	lastLineIndex := len(lines) - 1
	for lastLineIndex > 0 && strings.TrimSpace(lines[lastLineIndex]) == "" {
		lastLineIndex--
	}

	// Move cursor to the last line using 'j' keys
	currentLine := te.CurrentBuffer().CursorIterator.LineNumber()
	for currentLine < lastLineIndex {
		te.TypeKeys("j")
		currentLine = te.CurrentBuffer().CursorIterator.LineNumber()
	}

	t.Logf("Cursor now at line %d (last line index: %d)", currentLine, lastLineIndex)
	t.Logf("Last line content: %q", lines[lastLineIndex])

	// Now rewrap from the last line - this is where the bug occurs
	te.handleEvent(&events.KeyEvent{Key: events.KeyRune, Rune: 'q', Modifiers: events.ModAlt})

	t.Logf("Buffer after rewrap from last line:\n%s", te.GetBufferContent())

	// Check that the bullet "-" is still at column 0
	finalContent := te.GetBufferContent()
	finalLines := strings.Split(finalContent, "\n")

	if len(finalLines) == 0 {
		t.Fatal("Buffer is empty after rewrap")
	}

	firstLine := finalLines[0]
	t.Logf("First line after rewrap: %q", firstLine)

	// The first character should be '-' (the bullet marker)
	if len(firstLine) == 0 {
		t.Fatal("First line is empty after rewrap")
	}

	if firstLine[0] != '-' {
		t.Errorf("BUG REPRODUCED: Bullet marker '-' is not at column 0. First line: %q", firstLine)
	}

	// Also check that there's no leading whitespace before the bullet
	if strings.HasPrefix(firstLine, " ") || strings.HasPrefix(firstLine, "\t") {
		t.Errorf("BUG REPRODUCED: Bullet marker has unexpected leading whitespace. First line: %q", firstLine)
	}
}
