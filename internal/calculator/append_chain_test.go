package calculator

import (
	"testing"
)

func TestProcessLineAppendChaining(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		cursorPos int
		expected  string
		desc      string
	}{
		{
			name:      "simple calculation",
			line:      "1,000**2",
			cursorPos: 8,
			expected:  "1,000**2 → 1,000,000",
			desc:      "Should append result with comma formatting",
		},
		{
			name:      "chain first power",
			line:      "1,000**2 → 1,000,000",
			cursorPos: 8,  // Cursor at end of first expression "1,000**2"
			expected:  "1,000**2 → 1,000,000 → 1,000,000",
			desc:      "Should recalculate when cursor at end of expression",
		},
		{
			name:      "chain second power - from result",
			line:      "1,000**2 → 1,000,000**2",
			cursorPos: 25,  // Cursor at end of line
			expected:  "1,000**2 → 1,000,000**2 → 1,000,000,000,000",
			desc:      "Should calculate power of the result",
		},
		{
			name:      "multiple chains",
			line:      "100**2 → 10,000**2",
			cursorPos: 20,  // Cursor at end of line
			expected:  "100**2 → 10,000**2 → 100,000,000",
			desc:      "Should handle multiple arrows correctly",
		},
		{
			name:      "chain with different operation",
			line:      "1,000**2 → 1,000,000 + 1000",
			cursorPos: 29,  // Cursor at end of line
			expected:  "1,000**2 → 1,000,000 + 1000 → 1,001,000",
			desc:      "Should handle different operations in chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessLineAppend(tt.line, tt.cursorPos)
			if err != nil {
				t.Errorf("ProcessLineAppend failed: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("%s:\nExpected: %q\nGot:      %q", tt.desc, tt.expected, result)
			}
		})
	}
}

func TestFindExpressionAfterArrow(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		cursorPos int
		wantExpr  string
		wantValue float64
		desc      string
	}{
		{
			name:      "expression after arrow",
			line:      "1,000**2 → 1,000,000",
			cursorPos: 22,  // At end of line
			wantExpr:  "1,000,000",
			wantValue: 1000000,
			desc:      "Should find the result after arrow",
		},
		{
			name:      "expression with operation after arrow",
			line:      "1,000**2 → 1,000,000**2",
			cursorPos: 25,  // At end of line
			wantExpr:  "1,000,000**2",
			wantValue: 1000000000000,
			desc:      "Should evaluate operation on result",
		},
		{
			name:      "no arrow",
			line:      "1,000,000**2",
			cursorPos: 12,
			wantExpr:  "1,000,000**2",
			wantValue: 1000000000000,
			desc:      "Should work without arrow",
		},
		{
			name:      "multiple arrows - last expression",
			line:      "10**2 → 100**2 → 10,000",
			cursorPos: 27,  // At end of line
			wantExpr:  "10,000",
			wantValue: 10000,
			desc:      "Should find expression after last arrow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindExpressionFromPosition(tt.line, tt.cursorPos)
			if err != nil {
				t.Errorf("FindExpressionFromPosition failed: %v", err)
				return
			}

			if result.Expression != tt.wantExpr {
				t.Errorf("%s:\nExpected expression: %q\nGot: %q", 
					tt.desc, tt.wantExpr, result.Expression)
			}

			if result.Value != tt.wantValue {
				t.Errorf("%s:\nExpected value: %f\nGot: %f", 
					tt.desc, tt.wantValue, result.Value)
			}
		})
	}
}

// Test the specific problematic case
func TestProblematicChain(t *testing.T) {
	// Simulate the exact sequence that causes the issue
	
	// Step 1: Calculate 1,000**2
	line1 := "1,000**2"
	result1, err := ProcessLineAppend(line1, len(line1))
	if err != nil {
		t.Fatalf("Step 1 failed: %v", err)
	}
	t.Logf("Step 1: %s", result1)
	
	// Expected: "1,000**2 → 1,000,000"
	if result1 != "1,000**2 → 1,000,000" {
		t.Errorf("Step 1: Expected '1,000**2 → 1,000,000', got %q", result1)
	}
	
	// Step 2: Now add **2 to the result and calculate
	line2 := result1 + "**2"  // "1,000**2 → 1,000,000**2"
	result2, err := ProcessLineAppend(line2, len(line2))
	if err != nil {
		t.Fatalf("Step 2 failed: %v", err)
	}
	t.Logf("Step 2: %s", result2)
	
	// Expected: "1,000**2 → 1,000,000**2 → 1,000,000,000,000"
	expectedResult := "1,000**2 → 1,000,000**2 → 1,000,000,000,000"
	if result2 != expectedResult {
		t.Errorf("Step 2: Expected %q, got %q", expectedResult, result2)
	}
	
	// Verify the result is not 0
	if result2 == "1,000**2 → 1,000,000**2 → 0" {
		t.Error("Result incorrectly evaluated to 0")
	}
}