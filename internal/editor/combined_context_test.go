package editor

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"veda/internal/proxy"
)

// TestCombinedContext tests combining document symbols and folding ranges for complete context
func TestCombinedContext(t *testing.T) {
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
	testFile := filepath.Join(tmpDir, "test_combined.go")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Start proxy client
	client, err := proxy.NewClient()
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}
	defer client.Close()

	// Get both document symbols and folding ranges
	symbols, err := client.GetDocumentSymbols(testFile)
	if err != nil {
		t.Fatalf("Failed to get document symbols: %v", err)
	}

	ranges, err := client.GetFoldingRanges(testFile)
	if err != nil {
		t.Fatalf("Failed to get folding ranges: %v", err)
	}

	// Read the source file
	lines, err := readFileLines(testFile)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}

	// Log folding ranges for debugging
	t.Logf("Folding ranges:")
	for i, r := range ranges {
		if r.StartLine < len(lines) {
			t.Logf("  [%d] Lines %d-%d: %s", i, r.StartLine, r.EndLine, strings.TrimSpace(lines[r.StartLine]))
		}
	}

	// Test context determination at various cursor positions
	testCases := []struct {
		line     int
		expected string
		desc     string
	}{
		{6, "main", "Before for loop"},
		{10, "main > for i, item := range items > if item == \"b\"", "Inside if statement in for loop"},
		{12, "main > for i, item := range items > if item == \"b\" > if i > 0", "Inside nested if"},
		{16, "main > for i, item := range items > else", "Inside else block"},
		{21, "main > if len(items) > 0 > switch items[0] > case \"a\"", "Inside switch case in if"},
		{22, "main > if len(items) > 0 > switch items[0] > case \"a\"", "Inside case"},
		{32, "helper > for i := 0; i < 10; i++ > if i%2 == 0", "Inside if in helper's for loop"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			context := determineCombinedContext(symbols, ranges, lines, tc.line)
			t.Logf("Line %d context: %s", tc.line, context)
			if context != tc.expected {
				t.Errorf("Expected: %s", tc.expected)
			}
		})
	}
}

// readFileLines reads a file and returns its lines
func readFileLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// determineCombinedContext combines symbols and folding ranges to determine full context
func determineCombinedContext(symbols []proxy.DocumentSymbol, ranges []proxy.FoldingRange, lines []string, cursorLine int) string {
	// First, get the symbol context (functions, methods, etc.)
	symbolPath := getContextPath(symbols, cursorLine)
	
	// Then, find control flow context from folding ranges
	controlFlowPath := getControlFlowContext(ranges, lines, cursorLine)
	
	// Combine them
	if len(symbolPath) == 0 {
		return controlFlowPath
	}
	if controlFlowPath == "" {
		return strings.Join(symbolPath, " > ")
	}
	
	// Merge: symbol path + control flow within that symbol
	return strings.Join(symbolPath, " > ") + " > " + controlFlowPath
}

// getControlFlowContext determines control flow context from folding ranges
func getControlFlowContext(ranges []proxy.FoldingRange, lines []string, cursorLine int) string {
	// Find all ranges that contain the cursor line
	var containingRanges []proxy.FoldingRange
	for _, r := range ranges {
		if cursorLine >= r.StartLine && cursorLine <= r.EndLine {
			containingRanges = append(containingRanges, r)
		}
	}
	
	// Sort by start line to get proper nesting order
	// In Go, we can assume that if range A starts before range B and both contain
	// the cursor, then A contains B
	var contextParts []string
	
	// Regular expressions for detecting control flow statements
	forPattern := regexp.MustCompile(`^\s*for\s+(.*)`)
	ifPattern := regexp.MustCompile(`^\s*(?:}\s*else\s+)?if\s+(.*)`)
	elsePattern := regexp.MustCompile(`^\s*}\s*else\s*{`)
	switchPattern := regexp.MustCompile(`^\s*switch\s+(.*)`)
	casePattern := regexp.MustCompile(`^\s*case\s+(.*):|^\s*default:`)
	
	for _, r := range containingRanges {
		if r.StartLine >= len(lines) {
			continue
		}
		
		line := lines[r.StartLine]
		trimmed := strings.TrimSpace(line)
		
		// Skip function/method definitions (handled by symbols)
		if strings.HasPrefix(trimmed, "func") {
			continue
		}
		
		// Identify control flow type
		if matches := forPattern.FindStringSubmatch(line); matches != nil {
			contextParts = append(contextParts, "for " + strings.TrimSuffix(matches[1], " {"))
		} else if matches := ifPattern.FindStringSubmatch(line); matches != nil {
			contextParts = append(contextParts, "if " + strings.TrimSuffix(matches[1], " {"))
		} else if elsePattern.MatchString(line) {
			contextParts = append(contextParts, "else")
		} else if matches := switchPattern.FindStringSubmatch(line); matches != nil {
			contextParts = append(contextParts, "switch " + strings.TrimSuffix(matches[1], " {"))
		} else if matches := casePattern.FindStringSubmatch(line); matches != nil {
			if strings.Contains(line, "default:") {
				contextParts = append(contextParts, "default")
			} else {
				contextParts = append(contextParts, "case " + strings.TrimSuffix(matches[1], ":"))
			}
		}
	}
	
	return strings.Join(contextParts, " > ")
}