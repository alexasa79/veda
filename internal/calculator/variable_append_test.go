package calculator

import (
	"testing"
)

func TestVariableInProcessLineAppend(t *testing.T) {
	tests := []struct {
		name      string
		setup     func()
		line      string
		cursorPos int
		expected  string
	}{
		{
			name: "evaluate $? after calculation",
			setup: func() {
				GetDefaultVariables().Clear()
				EvaluateExpression("100")
			},
			line:      "$?",
			cursorPos: 2,
			expected:  "$? → 100",
		},
		{
			name: "use $? in expression",
			setup: func() {
				GetDefaultVariables().Clear()
				EvaluateExpression("50")
			},
			line:      "$? * 2",
			cursorPos: 6,
			expected:  "$? * 2 → 100",
		},
		{
			name: "chain with $?",
			setup: func() {
				GetDefaultVariables().Clear()
				EvaluateExpression("10")
			},
			line:      "5 + $?",
			cursorPos: 6,
			expected:  "5 + $? → 15",
		},
		{
			name: "multiple calculations update $?",
			setup: func() {
				GetDefaultVariables().Clear()
				EvaluateExpression("100")
				EvaluateExpression("200") // This should update $?
			},
			line:      "$?",
			cursorPos: 2,
			expected:  "$? → 200",
		},
		{
			name: "formatted output with $?",
			setup: func() {
				GetDefaultVariables().Clear()
				EvaluateExpression("255")
			},
			line:      "$? hex",
			cursorPos: 6,
			expected:  "$? hex → 0xff",
		},
		{
			name: "comma separated with $?",
			setup: func() {
				GetDefaultVariables().Clear()
				EvaluateExpression("1000000")
			},
			line:      "$? cs",
			cursorPos: 5,
			expected:  "$? cs → 1,000,000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setup != nil {
				tt.setup()
			}

			// Process line
			result, err := ProcessLineAppend(tt.line, tt.cursorPos)
			if err != nil {
				t.Errorf("ProcessLineAppend failed: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestChainedCalculationsUpdateVariable(t *testing.T) {
	// Clear variables
	GetDefaultVariables().Clear()

	// First calculation
	line1 := "100 + 50"
	result1, err := ProcessLineAppend(line1, len(line1))
	if err != nil {
		t.Fatalf("First calculation failed: %v", err)
	}
	
	expectedResult1 := "100 + 50 → 150"
	if result1 != expectedResult1 {
		t.Errorf("First calc: expected %q, got %q", expectedResult1, result1)
	}

	// Check that $? was updated
	varResult, err := EvaluateExpression("$?")
	if err != nil {
		t.Fatalf("Failed to evaluate $?: %v", err)
	}
	if varResult != 150 {
		t.Errorf("$? should be 150, got %f", varResult)
	}

	// Use $? in next calculation
	line2 := "$? * 2"
	result2, err := ProcessLineAppend(line2, len(line2))
	if err != nil {
		t.Fatalf("Second calculation failed: %v", err)
	}

	expectedResult2 := "$? * 2 → 300"
	if result2 != expectedResult2 {
		t.Errorf("Second calc: expected %q, got %q", expectedResult2, result2)
	}

	// Check that $? was updated again
	varResult2, err := EvaluateExpression("$?")
	if err != nil {
		t.Fatalf("Failed to evaluate $? after second calc: %v", err)
	}
	if varResult2 != 300 {
		t.Errorf("$? should be 300, got %f", varResult2)
	}
}