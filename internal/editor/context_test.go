package editor

import (
	"strings"
	"testing"
	"veda/internal/proxy"
)

// TestGetContextForPosition tests the context determination logic
func TestGetContextForPosition(t *testing.T) {
	tests := []struct {
		name         string
		symbols      []proxy.DocumentSymbol
		cursorLine   int
		expectedPath []string
	}{
		{
			name: "cursor in main function",
			symbols: []proxy.DocumentSymbol{
				{
					Name: "main",
					Kind: 12, // Function
					Range: proxy.Range{
						Start: proxy.Position{Line: 10, Character: 0},
						End:   proxy.Position{Line: 20, Character: 1},
					},
				},
			},
			cursorLine:   15,
			expectedPath: []string{"main"},
		},
		{
			name: "cursor in nested function",
			symbols: []proxy.DocumentSymbol{
				{
					Name: "MyStruct",
					Kind: 23, // Struct
					Range: proxy.Range{
						Start: proxy.Position{Line: 5, Character: 0},
						End:   proxy.Position{Line: 30, Character: 1},
					},
					Children: []proxy.DocumentSymbol{
						{
							Name: "MyMethod",
							Kind: 6, // Method
							Range: proxy.Range{
								Start: proxy.Position{Line: 10, Character: 0},
								End:   proxy.Position{Line: 20, Character: 1},
							},
						},
					},
				},
			},
			cursorLine:   15,
			expectedPath: []string{"MyStruct", "MyMethod"},
		},
		{
			name: "cursor between functions",
			symbols: []proxy.DocumentSymbol{
				{
					Name: "func1",
					Kind: 12,
					Range: proxy.Range{
						Start: proxy.Position{Line: 5, Character: 0},
						End:   proxy.Position{Line: 10, Character: 1},
					},
				},
				{
					Name: "func2",
					Kind: 12,
					Range: proxy.Range{
						Start: proxy.Position{Line: 15, Character: 0},
						End:   proxy.Position{Line: 20, Character: 1},
					},
				},
			},
			cursorLine:   12,
			expectedPath: []string{},
		},
		{
			name: "deeply nested context",
			symbols: []proxy.DocumentSymbol{
				{
					Name: "Package",
					Kind: 4, // Package
					Range: proxy.Range{
						Start: proxy.Position{Line: 0, Character: 0},
						End:   proxy.Position{Line: 100, Character: 0},
					},
					Children: []proxy.DocumentSymbol{
						{
							Name: "MyClass",
							Kind: 5, // Class
							Range: proxy.Range{
								Start: proxy.Position{Line: 10, Character: 0},
								End:   proxy.Position{Line: 90, Character: 1},
							},
							Children: []proxy.DocumentSymbol{
								{
									Name: "MyMethod",
									Kind: 6, // Method
									Range: proxy.Range{
										Start: proxy.Position{Line: 20, Character: 0},
										End:   proxy.Position{Line: 80, Character: 1},
									},
									Children: []proxy.DocumentSymbol{
										{
											Name: "innerFunc",
											Kind: 12, // Function
											Range: proxy.Range{
												Start: proxy.Position{Line: 30, Character: 0},
												End:   proxy.Position{Line: 70, Character: 1},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			cursorLine:   50,
			expectedPath: []string{"Package", "MyClass", "MyMethod", "innerFunc"},
		},
		{
			name:         "no symbols",
			symbols:      []proxy.DocumentSymbol{},
			cursorLine:   10,
			expectedPath: []string{},
		},
		{
			name: "cursor in if statement (typically not in symbols)",
			symbols: []proxy.DocumentSymbol{
				{
					Name: "processData",
					Kind: 12, // Function
					Range: proxy.Range{
						Start: proxy.Position{Line: 10, Character: 0},
						End:   proxy.Position{Line: 30, Character: 1},
					},
					// Note: if statements are typically NOT included as symbols
					// by most language servers
				},
			},
			cursorLine:   15, // Inside the function, possibly in an if statement
			expectedPath: []string{"processData"},
		},
		{
			name: "cursor before all symbols",
			symbols: []proxy.DocumentSymbol{
				{
					Name: "main",
					Kind: 12,
					Range: proxy.Range{
						Start: proxy.Position{Line: 10, Character: 0},
						End:   proxy.Position{Line: 20, Character: 1},
					},
				},
			},
			cursorLine:   5,
			expectedPath: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := getContextPath(tt.symbols, tt.cursorLine)
			
			if len(path) != len(tt.expectedPath) {
				t.Errorf("Expected path length %d, got %d", len(tt.expectedPath), len(path))
				t.Errorf("Expected: %v", tt.expectedPath)
				t.Errorf("Got: %v", path)
				return
			}
			
			for i, expected := range tt.expectedPath {
				if i >= len(path) || path[i] != expected {
					t.Errorf("Expected path[%d] = %q, got %q", i, expected, path[i])
				}
			}
		})
	}
}

// getContextPath determines the context path for a given cursor position
// This is the function we'll implement in the editor
func getContextPath(symbols []proxy.DocumentSymbol, cursorLine int) []string {
	var path []string
	
	var traverse func(symbols []proxy.DocumentSymbol) bool
	traverse = func(symbols []proxy.DocumentSymbol) bool {
		for _, symbol := range symbols {
			// Check if cursor is within this symbol's range
			if cursorLine >= symbol.Range.Start.Line && cursorLine <= symbol.Range.End.Line {
				path = append(path, symbol.Name)
				
				// Check children
				if len(symbol.Children) > 0 {
					if traverse(symbol.Children) {
						return true
					}
				}
				
				return true
			}
		}
		return false
	}
	
	traverse(symbols)
	return path
}

// TestFormatContextPath tests the context path formatting
func TestFormatContextPath(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		maxWidth int
		expected string
	}{
		{
			name:     "simple path",
			path:     []string{"main"},
			maxWidth: 80,
			expected: "main",
		},
		{
			name:     "nested path",
			path:     []string{"MyStruct", "MyMethod"},
			maxWidth: 80,
			expected: "MyStruct > MyMethod",
		},
		{
			name:     "deeply nested path",
			path:     []string{"Package", "MyClass", "MyMethod", "innerFunc"},
			maxWidth: 80,
			expected: "Package > MyClass > MyMethod > innerFunc",
		},
		{
			name:     "empty path",
			path:     []string{},
			maxWidth: 80,
			expected: "",
		},
		{
			name:     "path that needs truncation",
			path:     []string{"VeryLongPackageName", "VeryLongClassName", "VeryLongMethodName", "VeryLongFunctionName"},
			maxWidth: 50,
			expected: "...VeryLongMethodName > VeryLongFunctionName",
		},
		{
			name:     "path with single long item",
			path:     []string{"ThisIsAVeryLongFunctionNameThatExceedsTheMaximumWidth"},
			maxWidth: 20,
			expected: "ThisIsAVeryLongFu...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatContextPath(tt.path, tt.maxWidth)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// formatContextPath formats the context path for display
func formatContextPath(path []string, maxWidth int) string {
	if len(path) == 0 {
		return ""
	}
	
	separator := " > "
	
	// Simple case: join and check
	full := strings.Join(path, separator)
	if len(full) <= maxWidth {
		return full
	}
	
	// Need to truncate
	if len(path) == 1 {
		// Single item, just truncate it
		if len(path[0]) > maxWidth-3 {
			return path[0][:maxWidth-3] + "..."
		}
		return path[0]
	}
	
	// For simplicity, if it doesn't fit, just show "..." and as much of the end as possible
	// This ensures we always show the most relevant (deepest) context
	ellipsis := "..."
	minSize := len(ellipsis) + len(separator) + 1 // at least one char from the path
	
	if maxWidth < minSize {
		return ellipsis
	}
	
	// Start from the end and work backwards
	remaining := maxWidth - len(ellipsis)
	result := ""
	
	for i := len(path) - 1; i >= 0; i-- {
		part := path[i]
		needed := len(part)
		if i < len(path)-1 {
			needed += len(separator)
		}
		
		if len(result) + needed <= remaining {
			if i < len(path)-1 {
				result = part + separator + result
			} else {
				result = part + result
			}
		} else {
			// Can't fit this part, truncate and stop
			if len(result) > 0 {
				result = ellipsis + result
			} else {
				// Even the last item doesn't fit
				availableForLastItem := maxWidth - len(ellipsis)
				if availableForLastItem > 0 && len(path[len(path)-1]) > availableForLastItem {
					result = ellipsis + path[len(path)-1][len(path[len(path)-1])-availableForLastItem:]
				} else {
					result = ellipsis
				}
			}
			break
		}
	}
	
	// If we fit everything, no ellipsis needed
	if result == full {
		return result
	}
	
	// If result doesn't start with ellipsis, we need to add it
	if !strings.HasPrefix(result, ellipsis) && len(result) < len(full) {
		result = ellipsis + result  
	}
	
	return result
}