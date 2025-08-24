package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
)

func TestAutoIndentCleanupBasics(t *testing.T) {
	// Create a test editor
	editor := &Editor{
		autoIndentLines: make(map[string]AutoIndentInfo),
		lastCursorLine:  -1,
		windows:         []*Window{},
		activeWindow:    0,
	}

	// Create a test buffer
	testBuffer := buffer.NewBuffer("", "test.go", config.IndentSpaces, 4)

	// Create a window with the buffer
	editor.windows = append(editor.windows, &Window{Buffer: testBuffer})
	editor.SetCurrentBuffer(testBuffer)

	// Test tracking auto-indent
	editor.trackAutoIndent(1, 4)

	// Verify tracking
	key := editor.makeAutoIndentKey("test.go", 1)
	info, exists := editor.autoIndentLines[key]
	if !exists {
		t.Error("Auto-indent should be tracked")
	}
	if info.AutoIndentLength != 4 {
		t.Errorf("Expected AutoIndentLength 4, got %d", info.AutoIndentLength)
	}
	if info.HasUserContent {
		t.Error("Should not have user content initially")
	}

	// Test marking line with user content
	editor.markLineHasUserContent(1)
	info, exists = editor.autoIndentLines[key]
	if !exists {
		t.Error("Auto-indent tracking should still exist")
	}
	if !info.HasUserContent {
		t.Error("Should have user content after marking")
	}
}

func TestAutoIndentCleanupKey(t *testing.T) {
	editor := &Editor{}

	key1 := editor.makeAutoIndentKey("test.go", 5)
	key2 := editor.makeAutoIndentKey("test.go", 5)
	key3 := editor.makeAutoIndentKey("test.go", 6)
	key4 := editor.makeAutoIndentKey("other.go", 5)

	if key1 != key2 {
		t.Error("Same file and line should produce same key")
	}
	if key1 == key3 {
		t.Error("Different lines should produce different keys")
	}
	if key1 == key4 {
		t.Error("Different files should produce different keys")
	}
}
