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
	// Check that the string contains the expected parts
	str := iter.String()
	if !strings.Contains(str, "line=0") || !strings.Contains(str, "eof=false") || 
	   !strings.Contains(str, "start=0") || !strings.Contains(str, "end=6") {
		t.Errorf("Expected iterator string to contain line=0, eof=false, start=0, end=6, got %s", str)
	}
	
	// Test after moving to next line
	iter = iter.Next()
	str = iter.String()
	if !strings.Contains(str, "line=1") || !strings.Contains(str, "eof=false") || 
	   !strings.Contains(str, "start=7") || !strings.Contains(str, "end=13") {
		t.Errorf("Expected iterator string to contain line=1, eof=false, start=7, end=13, got %s", str)
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

