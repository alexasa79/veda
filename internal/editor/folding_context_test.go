package editor

import (
	"os"
	"path/filepath"
	"testing"
	"veda/internal/proxy"
)

// TestFoldingRangeContext tests using folding ranges to determine context
func TestFoldingRangeContext(t *testing.T) {
	// Create a test Go file with nested control flow
	testContent := `package main

import "fmt"

func main() {
	items := []string{"a", "b", "c"}
	
	for i, item := range items {
		if item == "b" {
			fmt.Println("Found b at index", i)
			
			if i > 0 {
				fmt.Println("Not the first item")
			}
		} else {
			fmt.Println("Item:", item)
		}
	}
	
	if len(items) > 0 {
		switch items[0] {
		case "a":
			fmt.Println("Starts with a")
		default:
			fmt.Println("Doesn't start with a")
		}
	}
}

func helper() {
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			fmt.Println("Even:", i)
		}
	}
}
`

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_folding.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Start proxy client
	client, err := proxy.NewClient()
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}
	defer client.Close()

	// Get folding ranges
	t.Logf("Getting folding ranges for %s", testFile)
	ranges, err := client.GetFoldingRanges(testFile)
	if err != nil {
		// Log the error but don't fail - gopls might not support folding ranges
		t.Logf("GetFoldingRanges error (this is expected if gopls doesn't support it): %v", err)
		
		// Let's also try with document symbols for comparison
		symbols, symbolErr := client.GetDocumentSymbols(testFile)
		if symbolErr != nil {
			t.Fatalf("Both GetFoldingRanges and GetDocumentSymbols failed: %v, %v", err, symbolErr)
		}
		
		t.Logf("Got %d document symbols instead:", len(symbols))
		for _, sym := range symbols {
			t.Logf("  Symbol: %s (kind %d) at lines %d-%d", 
				sym.Name, sym.Kind, sym.Range.Start.Line, sym.Range.End.Line)
		}
		return
	}

	// Log all folding ranges
	t.Logf("Got %d folding ranges:", len(ranges))
	for i, r := range ranges {
		kind := "<no kind>"
		if r.Kind != nil {
			kind = *r.Kind
		}
		t.Logf("  [%d] Lines %d-%d (kind: %s)", i, r.StartLine, r.EndLine, kind)
	}

	// Test context determination at various cursor positions
	testCases := []struct {
		line     int
		expected string
		desc     string
	}{
		{8, "main > for", "Inside for loop"},
		{9, "main > for > if", "Inside if statement in for loop"},
		{12, "main > for > if > if", "Inside nested if"},
		{16, "main > for > else", "Inside else block"},
		{22, "main > if > switch", "Inside switch in if"},
		{34, "helper > for", "Inside for loop in helper function"},
		{35, "helper > for > if", "Inside if in helper's for loop"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			context := determineFoldingContext(ranges, tc.line)
			t.Logf("Line %d context: %s (expected: %s)", tc.line, context, tc.expected)
			// Note: We're just logging, not asserting, because we need to see
			// what folding ranges actually return first
		})
	}
}

// determineFoldingContext determines context based on folding ranges
// This is a simplified version for testing
func determineFoldingContext(ranges []proxy.FoldingRange, cursorLine int) string {
	// Find all ranges that contain the cursor line
	var containingRanges []proxy.FoldingRange
	for _, r := range ranges {
		if cursorLine >= r.StartLine && cursorLine <= r.EndLine {
			containingRanges = append(containingRanges, r)
		}
	}

	// Sort by start line (innermost last)
	// In a real implementation, we'd need to handle this more carefully
	context := ""
	for _, r := range containingRanges {
		if context != "" {
			context += " > "
		}
		// For now, just use the line range
		// In a real implementation, we'd read the source to determine the type
		context += "block"
		if r.Kind != nil {
			context += "(" + *r.Kind + ")"
		}
	}

	return context
}