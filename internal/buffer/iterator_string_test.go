package buffer

import (
	"strings"
	"testing"
	"veda/internal/config"
)

func TestIteratorString(t *testing.T) {
	// Test with a simple buffer
	content := "Line 1\nLine 2\nLine 3"
	buf := NewBuffer(content, "test.txt", config.IndentSpaces, 4)
	
	// Test nil iterator
	var nilIter *BufferIterator
	if nilIter.String() != "BufferIterator(nil)" {
		t.Errorf("Expected nil iterator to return 'BufferIterator(nil)', got %s", nilIter.String())
	}
	
	// Test iterator at beginning
	iter := NewBufferIterator(buf)
	expected := "BufferIterator(line=0, eof=false, pos=0, start=0, end=6)"
	if iter.String() != expected {
		t.Errorf("Expected %s, got %s", expected, iter.String())
	}
	
	// Test after moving to next line
	iter = iter.Next()
	expected = "BufferIterator(line=1, eof=false, pos=7, start=7, end=13)"
	if iter.String() != expected {
		t.Errorf("Expected %s, got %s", expected, iter.String())
	}
	
	// Test at EOF
	iter = NewBufferIteratorAtEnd(buf)
	// Should show eof=true
	if !strings.Contains(iter.String(), "eof=true") {
		t.Errorf("Expected EOF iterator to contain 'eof=true', got %s", iter.String())
	}
	
	// Test with printf
	t.Logf("Iterator formatted with %%v: %v", iter)
	t.Logf("Iterator formatted with %%s: %s", iter)
}

