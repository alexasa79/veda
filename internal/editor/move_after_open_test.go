package editor

import (
    "testing"
    "veda/internal/proxy"
)

// TestMoveRightAfterFinishOpenFile_ClearsSourceWindowBuffer ensures that when a file
// is opened via finishOpenFile (new buffer path) and then moved right, the source
// window no longer shows the moved buffer and does not retain it in its list.
func TestMoveRightAfterFinishOpenFile_ClearsSourceWindowBuffer(t *testing.T) {
    te := NewTestEditor(120, 40)

    // Open a file using the same code path as production (finishOpenFile)
    changes := &proxy.FileChanges{Added: []int{}, Deleted: []int{}, Modified: []int{}}
    te.finishOpenFile("/tmp/test-move-right.txt", "hello world", changes, "main", nil)

    // Invariants should hold after opening
    assertWindowBufferInvariants(t, te.Editor)

    moved := te.CurrentBuffer()
    if moved == nil || moved.FilePath != "/tmp/test-move-right.txt" {
        t.Fatalf("expected current buffer to be the newly opened file, got %+v", moved)
    }

    // Move buffer to the right (creates a second window when only one exists)
    if err := te.ExecuteCommand("move-buffer-right"); err != nil {
        t.Fatalf("move-buffer-right failed: %v", err)
    }

    if len(te.windows) != 2 {
        t.Fatalf("expected 2 windows after move, got %d", len(te.windows))
    }

    // Invariants should hold after movement
    assertWindowBufferInvariants(t, te.Editor)

    // Left window should not show the moved buffer
    if te.windows[0].Buffer == moved {
        t.Fatalf("left window still shows the moved buffer")
    }

    // And the moved buffer should not be present in left window's list
    for _, b := range te.windows[0].Buffers {
        if b == moved {
            t.Fatalf("left window still contains the moved buffer in its list")
        }
    }

    // Right window must show the moved buffer
    if te.windows[1].Buffer != moved {
        t.Fatalf("right window does not show the moved buffer")
    }
}
