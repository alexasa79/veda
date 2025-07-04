package buffer

import (
	"strings"
	"testing"
)

func TestInsert(t *testing.T) {
	b := NewBufferFromString("Hello, world!")
	b.Insert(7, "beautiful ")
	expected := "Hello, beautiful world!"
	if b.String() != expected {
		t.Errorf("Insert failed: expected %q, got %q", expected, b.String())
	}
}

func TestInsert_EmptyBuffer(t *testing.T) {
	b := NewBufferFromString("")
	b.Insert(0, "Hello, world!")
	expected := "Hello, world!"
	if b.String() != expected {
		t.Errorf("Insert failed: expected %q, got %q", expected, b.String())
	}
}

func TestInsert_Split(t *testing.T) {
	// Create a string that is larger than minLeafSize but smaller than maxLeafSize
	text := strings.Repeat("a", 3000)
	b := NewBufferFromString(text)
	// Insert another string to trigger a split
	b.Insert(1500, strings.Repeat("b", 3000))

	// Check that the root is a ConcatNode
	if _, ok := b.root.(*ConcatNode); !ok {
		t.Errorf("Insert failed: expected root to be a ConcatNode, got %T", b.root)
	}

	// Check the length
	expectedLen := 6000
	if b.Length() != expectedLen {
		t.Errorf("Insert failed: expected length %d, got %d", expectedLen, b.Length())
	}
}
