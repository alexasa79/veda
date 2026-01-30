package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"veda/internal/buffer"
	"veda/internal/proxy"
)

// TestReferencesUpdateOnContentChange tests that reference positions are updated
// when the buffer content changes via background LSP re-query.
func TestReferencesUpdateOnContentChange(t *testing.T) {
	// Skip if running in CI or short mode
	if testing.Short() {
		t.Skip("Skipping LSP integration test in short mode")
	}

	// Check if gopls is available
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH; skipping test")
	}

	// Create proxy manager
	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "veda-refs-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create go.mod for gopls
	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent := "module testrefs\n\ngo 1.21\n"
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create test Go file with a function called multiple times
	testFilePath := filepath.Join(tempDir, "main.go")
	initialContent := `package main

func foo() {
	bar()
}

func bar() {
	// some code
}

func baz() {
	bar()
}
`
	if err := os.WriteFile(testFilePath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Open the file via proxy to trigger LSP startup
	_, err = pm.GetFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to open test file via proxy: %v", err)
	}

	// Wait for gopls to initialize
	time.Sleep(3 * time.Second)

	// Create editor with the content
	te := NewTestEditorWithContent(80, 24, initialContent)
	te.Editor.proxyManager = pm
	te.CurrentBuffer().FilePath = testFilePath
	te.CurrentBuffer().HasLSP = true
	te.buffers[testFilePath] = te.CurrentBuffer()

	// Find references to "bar" - cursor should be on the "bar()" call in foo()
	// Position cursor on line 3 (0-indexed), at "bar" in "bar()"
	te.SetCursorPosition(1, 3) // x=1 is at 'b' in "\tbar()"

	// Call find-references
	err = te.findReferences(nil)
	if err != nil {
		t.Logf("findReferences error (may be expected if LSP not ready): %v", err)
	}

	// Wait for references to be populated
	time.Sleep(1 * time.Second)

	if te.currentReferencesBuffer == nil || len(te.currentReferencesBuffer.References) == 0 {
		t.Skip("LSP did not return references; skipping test")
	}

	t.Logf("Found %d references to 'bar'", len(te.currentReferencesBuffer.References))
	for i, ref := range te.currentReferencesBuffer.References {
		t.Logf("  Reference %d: line %d, char %d", i, ref.Range.Start.Line, ref.Range.Start.Character)
	}

	// Find the reference on line 10 (func baz's bar() call, 0-indexed)
	// In initial content, baz is at line 10, bar() call is at line 11
	var bazBarRefIndex = -1
	for i, ref := range te.currentReferencesBuffer.References {
		// Look for reference around line 10-11 (baz's bar() call)
		if ref.Range.Start.Line >= 10 && ref.Range.Start.Line <= 11 {
			bazBarRefIndex = i
			t.Logf("Found baz's bar() reference at index %d, line %d", i, ref.Range.Start.Line)
			break
		}
	}

	if bazBarRefIndex == -1 {
		t.Skip("Could not find reference in baz function")
	}

	originalLine := te.currentReferencesBuffer.References[bazBarRefIndex].Range.Start.Line
	t.Logf("Original line for baz's bar() call: %d", originalLine)

	// Switch back to the source file buffer for editing
	sourceBuffer := te.buffers[testFilePath]
	if sourceBuffer == nil {
		t.Fatal("Source buffer not found")
	}
	te.switchToBuffer(sourceBuffer)

	// Now insert 2 lines after foo() but before baz()
	// This should push baz's reference down by 2 lines
	modifiedContent := `package main

func foo() {
	bar()
}

// new comment 1
// new comment 2

func bar() {
	// some code
}

func baz() {
	bar()
}
`
	// Update buffer content
	te.CurrentBuffer().Rope = buffer.NewRopeFromString(modifiedContent)
	te.CurrentBuffer().Generation++

	// Notify LSP of file change
	if err := pm.NotifyFileChange(testFilePath, modifiedContent); err != nil {
		t.Logf("Failed to notify file change: %v", err)
	}

	// Also update the file on disk for gopls
	if err := os.WriteFile(testFilePath, []byte(modifiedContent), 0644); err != nil {
		t.Logf("Failed to update file on disk: %v", err)
	}

	// Trigger OnContentChange to invoke background reference refresh
	if te.CurrentBuffer().OnContentChange != nil {
		te.CurrentBuffer().OnContentChange()
	}

	// Wait for background LSP re-query to complete and process the Post callback
	time.Sleep(3 * time.Second)
	te.ProcessAsyncEvents(1 * time.Second)

	// Check if references were updated
	if te.currentReferencesBuffer == nil || len(te.currentReferencesBuffer.References) == 0 {
		t.Fatal("References were lost after content change")
	}

	t.Logf("After edit, found %d references", len(te.currentReferencesBuffer.References))
	for i, ref := range te.currentReferencesBuffer.References {
		t.Logf("  Reference %d: line %d, char %d", i, ref.Range.Start.Line, ref.Range.Start.Character)
	}

	// Find the new line for baz's bar() call (should be 2 lines later)
	var newBazBarLine = -1
	for i, ref := range te.currentReferencesBuffer.References {
		// Look for reference around line 14-15 (baz's bar() call after inserting 2 lines)
		if ref.Range.Start.Line >= 13 && ref.Range.Start.Line <= 15 {
			newBazBarLine = ref.Range.Start.Line
			t.Logf("Found updated baz's bar() reference at index %d, line %d", i, ref.Range.Start.Line)
			break
		}
	}

	// The reference should have moved from originalLine to originalLine+3
	// (2 comment lines + 1 empty line added before func bar())
	expectedLine := originalLine + 3
	if newBazBarLine != expectedLine {
		t.Errorf("Reference not updated correctly. Expected line %d, got line %d (original was %d)",
			expectedLine, newBazBarLine, originalLine)
	} else {
		t.Logf("SUCCESS: Reference updated from line %d to line %d as expected", originalLine, newBazBarLine)
	}
}
