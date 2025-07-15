package editor

import (
	"fmt"
	"strings"
	"testing"
	"veda/internal/proxy"
)

func TestGetContextLines(t *testing.T) {
	// Test source code
	source := `package main

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
}`

	lines := strings.Split(source, "\n")
	
	// Simulated folding ranges from gopls (based on earlier test output)
	ranges := []proxy.FoldingRange{
		{StartLine: 4, EndLine: 27},   // func main() {
		{StartLine: 5, EndLine: 5},     // items := []string{"a", "b", "c"}
		{StartLine: 7, EndLine: 17},    // for i, item := range items {
		{StartLine: 8, EndLine: 14},    // if item == "b" {
		{StartLine: 11, EndLine: 13},   // if i > 0 {
		{StartLine: 14, EndLine: 16},   // } else {
		{StartLine: 19, EndLine: 26},   // if len(items) > 0 {
		{StartLine: 20, EndLine: 25},   // switch items[0] {
		{StartLine: 21, EndLine: 22},   // case "a":
		{StartLine: 23, EndLine: 24},   // default:
		{StartLine: 29, EndLine: 35},   // func helper() {
		{StartLine: 30, EndLine: 34},   // for i := 0; i < 10; i++ {
		{StartLine: 31, EndLine: 33},   // if i%2 == 0 {
	}
	
	tests := []struct {
		line         int
		maxLines     int
		expectedLines []string
	}{
		{
			line:     6,
			maxLines: 3,
			expectedLines: []string{
				"func main()",
			},
		},
		{
			line:     10,
			maxLines: 3,
			expectedLines: []string{
				"func main()",
				"\tfor i, item := range items",
				"\t\tif item == \"b\"",
			},
		},
		{
			line:     12,
			maxLines: 4,
			expectedLines: []string{
				"func main()",
				"\tfor i, item := range items",
				"\t\tif item == \"b\"",
				"\t\t\tif i > 0",
			},
		},
		{
			line:     15,
			maxLines: 3,
			expectedLines: []string{
				"func main()",
				"\tfor i, item := range items",
				"\t\t} else",
			},
		},
		{
			line:     22,
			maxLines: 5,
			expectedLines: []string{
				"func main()",
				"\tif len(items) > 0",
				"\t\tswitch items[0]",
				"\t\tcase \"a\":",
			},
		},
		{
			line:     32,
			maxLines: 3,
			expectedLines: []string{
				"func helper()",
				"\tfor i := 0; i < 10; i++",
				"\t\tif i%2 == 0",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Line %d", tt.line), func(t *testing.T) {
			contextLines := getContextLines(ranges, lines, tt.line, tt.maxLines)
			
			if len(contextLines) != len(tt.expectedLines) {
				t.Errorf("Line %d: got %d context lines, want %d", tt.line, len(contextLines), len(tt.expectedLines))
				t.Errorf("Got: %v", contextLines)
				t.Errorf("Expected: %v", tt.expectedLines)
				return
			}
			
			for i, line := range contextLines {
				if line != tt.expectedLines[i] {
					t.Errorf("Line %d, context[%d]: got %q, want %q", tt.line, i, line, tt.expectedLines[i])
				}
			}
		})
	}
}