package editor

import (
	"testing"
)

func TestLSPTextObjects(t *testing.T) {
	t.Run("dab deletes around block", func(t *testing.T) {
		// This test requires LSP server support which may not be available in test environment
		t.Skip("LSP text objects require running language server")
		
		content := `func main() {
	if x > 0 {
		println("positive")
	}
}`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move cursor inside the if block
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		
		// Execute dab
		te.TypeKeys("dab")
		
		// The if block should be deleted
		expected := `func main() {
}`
		AssertBufferEquals(t, te, expected)
	})
	
	t.Run("dib deletes inside block", func(t *testing.T) {
		// This test requires LSP server support which may not be available in test environment
		t.Skip("LSP text objects require running language server")
		
		content := `func main() {
	if x > 0 {
		println("positive")
		return
	}
}`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move cursor inside the if block
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		
		// Execute dib
		te.TypeKeys("dib")
		
		// The content inside the if block should be deleted
		expected := `func main() {
	if x > 0 {
	}
}`
		AssertBufferEquals(t, te, expected)
	})
	
	t.Run("daf deletes around function", func(t *testing.T) {
		// This test requires LSP server support which may not be available in test environment
		t.Skip("LSP text objects require running language server")
		
		content := `package main

func helper() {
	return 42
}

func main() {
	helper()
}`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move cursor inside helper function
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		
		// Execute daf
		te.TypeKeys("daf")
		
		// The helper function should be deleted
		expected := `package main

func main() {
	helper()
}`
		AssertBufferEquals(t, te, expected)
	})
	
	t.Run("dif deletes inside function", func(t *testing.T) {
		// This test requires LSP server support which may not be available in test environment
		t.Skip("LSP text objects require running language server")
		
		content := `func calculate(x, y int) int {
	sum := x + y
	product := x * y
	return sum + product
}`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move cursor inside function
		te.ExecuteCommand("down")
		
		// Execute dif
		te.TypeKeys("dif")
		
		// The function body should be deleted
		expected := `func calculate(x, y int) int {
}`
		AssertBufferEquals(t, te, expected)
	})
	
	t.Run("daa deletes around argument", func(t *testing.T) {
		// This test requires LSP server support which may not be available in test environment
		t.Skip("LSP text objects require running language server")
		
		content := `result := calculate(10, 20, 30)`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move cursor to second argument
		for i := 0; i < 24; i++ {
			te.ExecuteCommand("right")
		}
		
		// Execute daa
		te.TypeKeys("daa")
		
		// The second argument and surrounding commas should be deleted
		expected := `result := calculate(10, 30)`
		AssertBufferEquals(t, te, expected)
	})
	
	t.Run("dia deletes inside argument", func(t *testing.T) {
		// This test requires LSP server support which may not be available in test environment
		t.Skip("LSP text objects require running language server")
		
		content := `result := calculate(firstValue, secondValue, thirdValue)`
		te := NewTestEditorWithContent(80, 24, content)
		
		// Move cursor to second argument
		for i := 0; i < 32; i++ {
			te.ExecuteCommand("right")
		}
		
		// Execute dia
		te.TypeKeys("dia")
		
		// Only the argument text should be deleted, not the commas
		expected := `result := calculate(firstValue, , thirdValue)`
		AssertBufferEquals(t, te, expected)
	})
}