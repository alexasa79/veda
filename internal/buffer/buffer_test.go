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
	// Create a string that is larger than minLeafSize but smaller than maxLeafSize
	text := strings.Repeat("a", 3000)
	r := NewRopeFromString(text)
	// Insert another string to trigger a split
	r.Insert(1500, strings.Repeat("b", 3000))

	// Check that the root is an internal node (not a leaf)
	if r.root.IsLeaf() {
		t.Errorf("Insert failed: expected root to be an internal node, but it's a leaf")
	}

	// Check the length
	expectedLen := 6000
	if r.Length() != expectedLen {
		t.Errorf("Insert failed: expected length %d, got %d", expectedLen, r.Length())
	}
}
