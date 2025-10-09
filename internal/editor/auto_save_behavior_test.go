package editor

import (
	"testing"
	"time"
	"veda/internal/events"
)

func TestAutoSaveMarksBufferSaved(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello")
	defer func() {
		if te.autoSaveManager != nil {
			te.autoSaveManager.Stop()
		}
	}()

	buf := te.CurrentBuffer()
	if buf == nil {
		t.Fatal("expected current buffer")
	}

	for {
		select {
		case <-te.asyncEvents:
		default:
			goto drained
		}
	}
drained:

	// Ensure auto-save writes to a temporary directory for test isolation.
	autoSaveDir := t.TempDir()
	if te.autoSaveManager != nil {
		te.autoSaveManager.autoSaveDir = autoSaveDir
	}

	// Simulate a modification so the buffer is marked dirty.
	buf.Insert(buf.Length(), "!")
	te.markBufferDirty()

	if !buf.IsModified {
		t.Fatal("expected buffer to report unsaved changes")
	}

	// Trigger auto-save for this buffer and wait for it to finish.
	te.performAutoSave(buf.FilePath)
	te.pendingSaves.Wait()

	foundRender := false
	timeout := time.After(500 * time.Millisecond)
	for !foundRender {
		select {
		case ev := <-te.asyncEvents:
			if ce, ok := ev.(*events.CustomEvent); ok && ce.Type == "render" {
				foundRender = true
			}
		case <-timeout:
			t.Fatal("expected auto-save to request render")
		}
	}

	if buf.IsModified {
		t.Fatal("expected auto-save to clear unsaved flag")
	}
}

func TestWriteFileClearsUnsavedFlag(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "hello")
	defer func() {
		if te.autoSaveManager != nil {
			te.autoSaveManager.Stop()
		}
	}()

	buf := te.CurrentBuffer()
	if buf == nil {
		t.Fatal("expected current buffer")
	}

	autoSaveDir := t.TempDir()
	if te.autoSaveManager != nil {
		te.autoSaveManager.autoSaveDir = autoSaveDir
	}

	buf.Insert(buf.Length(), "!")

	if !buf.IsModified {
		t.Fatal("expected buffer to report unsaved changes")
	}

	if err := te.writeFile(nil); err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}

	te.pendingSaves.Wait()

	if buf.IsModified {
		t.Fatal("expected save to clear unsaved flag")
	}
}
