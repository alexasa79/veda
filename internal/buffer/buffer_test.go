package buffer

import (
	"strings"
	"testing"
)

func TestInsert(t *testing.T) {
	r := NewRopeFromString("Hello, world!")
	r.Insert(7, "beautiful ")
	expected := "Hello, beautiful world!"
	if r.String() != expected {
		t.Errorf("Insert failed: expected %q, got %q", expected, r.String())
	}
}

func TestInsert_EmptyRope(t *testing.T) {
	r := NewRopeFromString("")
	r.Insert(0, "Hello, world!")
	expected := "Hello, world!"
	if r.String() != expected {
		t.Errorf("Insert failed: expected %q, got %q", expected, r.String())
	}
}

func TestInsert_Split(t *testing.T) {
    // Construct content so that after insertion the leaf exceeds maxLeafSize and must split.
    baseLen := maxLeafSize - 1000
    insertLen := 2000 // ensures baseLen+insertLen > maxLeafSize
    text := strings.Repeat("a", baseLen)
    r := NewRopeFromString(text)
    // Insert in the middle to force split
    r.Insert(baseLen/2, strings.Repeat("b", insertLen))

    // Root should be an internal node after split
    if r.root.IsLeaf() {
        t.Errorf("Insert failed: expected root to be an internal node, but it's a leaf")
    }

    // Check the total length
    expectedLen := baseLen + insertLen
    if r.Length() != expectedLen {
        t.Errorf("Insert failed: expected length %d, got %d", expectedLen, r.Length())
    }

    // Basic content sanity: prefix, inserted run, and suffix exist
    got := r.String()
    if len(got) != expectedLen {
        t.Fatalf("content length mismatch: got %d, want %d", len(got), expectedLen)
    }
    // Verify the inserted region is present
    mid := baseLen / 2
    if got[mid:mid+insertLen] != strings.Repeat("b", insertLen) {
        t.Errorf("Inserted content not found at expected position")
    }
}
