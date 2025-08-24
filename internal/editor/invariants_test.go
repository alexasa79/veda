package editor

import (
    "testing"
    "veda/internal/buffer"
)

// assertWindowBufferInvariants verifies per-window buffer invariants:
// - If w.Buffer != nil, it must be present in w.Buffers
// - If len(w.Buffers) == 0, then w.Buffer must be nil
// - No duplicates in w.Buffers
func assertWindowBufferInvariants(t *testing.T, e *Editor) {
    t.Helper()
    for i, w := range e.windows {
        if w == nil {
            continue
        }

        // No duplicates
        seen := make(map[*buffer.Buffer]bool)
        for j, b := range w.Buffers {
            if b == nil {
                t.Fatalf("window %d has nil buffer entry at index %d", i, j)
            }
            if seen[b] {
                t.Fatalf("window %d has duplicate buffer entries for %s", i, b.DisplayName())
            }
            seen[b] = true
        }

        // If list is empty, active must be nil
        if len(w.Buffers) == 0 {
            if w.Buffer != nil {
                t.Fatalf("window %d has no buffers but active Buffer is set to %s", i, w.Buffer.DisplayName())
            }
            continue
        }

        // If active is set, it must be in the list
        if w.Buffer != nil {
            if !seen[w.Buffer] {
                t.Fatalf("window %d active Buffer %s not present in Buffers list", i, w.Buffer.DisplayName())
            }
        }
    }
}

