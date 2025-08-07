package calculator

import (
	"math"
	"testing"
)

func TestBasicArithmetic(t *testing.T) {
	tests := []struct {
		expr     string
		expected float64
	}{
		{"1+1", 2},
		{"2*3", 6},
		{"10/2", 5},
		{"10-3", 7},
		{"2+3*4", 14},        // Precedence test
		{"(2+3)*4", 20},      // Parentheses
		{"2**3", 8},          // Power with **
		{"2**3**2", 512},     // Right-associative power: 2**(3**2) = 2**9 = 512
		{"10/2*3", 15},       // Left-to-right: (10/2)*3 = 5*3 = 15
		{"-5", -5},           // Unary minus
		{"+5", 5},            // Unary plus
		{"-(2+3)", -5},       // Unary minus with parentheses
		{"2*-3", -6},         // Unary minus in expression
		{"1.5+2.5", 4},       // Floating point
		{"3.14*2", 6.28},     // Floating point
		{"1e3+1", 1001},      // Scientific notation
		{"2.5e2", 250},       // Scientific notation
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expr)
			if err != nil {
				t.Errorf("Failed to evaluate %q: %v", tt.expr, err)
				return
			}
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("Expression %q = %v, expected %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestNumberBases(t *testing.T) {
	tests := []struct {
		expr     string
		expected float64
	}{
		{"0xFF", 255},
		{"0xff", 255},
		{"0x10", 16},
		{"0xA+0xB", 21},
		{"0o10", 8},
		{"0o77", 63},
		{"0b1010", 10},
		{"0b1111", 15},
		{"0b101+0b010", 7},
		{"0x10+0o10+0b10", 26}, // 16 + 8 + 2
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expr)
			if err != nil {
				t.Errorf("Failed to evaluate %q: %v", tt.expr, err)
				return
			}
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("Expression %q = %v, expected %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestComplexExpressions(t *testing.T) {
	tests := []struct {
		expr     string
		expected float64
	}{
		{"(2+3)*(4+5)", 45},
		{"2*(3+4*(5+6))", 94},
		{"((1+2)*3)+4", 13},
		{"2**(3+1)", 16},
		{"(0xFF-0xF0)/0xF", 1},
		{"1+2+3+4+5", 15},
		{"100-50-25-10", 15},
		{"2*3*4*5", 120},
		{"1000/10/10/5", 2},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expr)
			if err != nil {
				t.Errorf("Failed to evaluate %q: %v", tt.expr, err)
				return
			}
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("Expression %q = %v, expected %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestFindExpression(t *testing.T) {
	tests := []struct {
		line     string
		pos      int
		wantExpr string
		wantVal  float64
		wantErr  bool
	}{
		{"1+1", 3, "1+1", 2, false},
		{"The answer is 2+3", 17, "2+3", 5, false},
		{"1+1 → 2", 3, "1+1", 2, false},
		{"1+1 → 2+3", 11, "2+3", 5, false}, // Position 11 is after the full "2+3" (11 bytes total)
		{"Calculate: 10*5+2", 17, "10*5+2", 52, false},
		{"x = 5*6; y = 7", 7, "5*6", 30, false},
		{"array[2+3] = 10", 9, "2+3", 5, false},
		{"No expression here", 18, "", 0, true},
		{"2**3**2", 9, "2**3**2", 512, false},
		{"  3 + 4  ", 9, "3 + 4", 7, false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result, err := FindExpressionFromPosition(tt.line, tt.pos)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %q at pos %d, but got result: %+v", tt.line, tt.pos, result)
				}
				return
			}
			if err != nil {
				t.Errorf("Failed to find expression in %q at pos %d: %v", tt.line, tt.pos, err)
				return
			}
			if result.Expression != tt.wantExpr {
				t.Errorf("Found expression %q, expected %q", result.Expression, tt.wantExpr)
			}
			if math.Abs(result.Value-tt.wantVal) > 0.0001 {
				t.Errorf("Expression %q = %v, expected %v", result.Expression, result.Value, tt.wantVal)
			}
		})
	}
}

func TestProcessLineAppend(t *testing.T) {
	tests := []struct {
		line     string
		pos      int
		expected string
		wantErr  bool
	}{
		{"1+1", 3, "1+1 → 2", false},
		{"Calculate: 2*3", 14, "Calculate: 2*3 → 6", false},
		{"1+1 → 2+3", 11, "1+1 → 2+3 → 5", false},
		{"Result: 0xFF", 12, "Result: 0xFF → 255", false},
		{"2**3", 4, "2**3 → 8", false},
		{"10/2", 4, "10/2 → 5", false},
		{"3.5*2", 5, "3.5*2 → 7", false},
		{"No math here", 12, "No math here", true},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result, err := ProcessLineAppend(tt.line, tt.pos)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %q at pos %d, but got: %q", tt.line, tt.pos, result)
				}
				return
			}
			if err != nil {
				t.Errorf("ProcessLineAppend(%q, %d) failed: %v", tt.line, tt.pos, err)
				return
			}
			if result != tt.expected {
				t.Errorf("ProcessLineAppend(%q, %d) = %q, expected %q", tt.line, tt.pos, result, tt.expected)
			}
		})
	}
}

func TestProcessLineReplace(t *testing.T) {
	tests := []struct {
		line        string
		pos         int
		expectedLine string
		expectedPos int
		wantErr     bool
	}{
		{"1+1", 3, "2", 1, false},
		{"Calculate: 2*3", 14, "Calculate: 6", 12, false},
		{"x = 10/2", 8, "x = 5", 5, false},
		{"0xFF+1", 6, "256", 3, false},
		{"The answer is 3.5*2", 19, "The answer is 7", 15, false},
		{"No math", 7, "No math", 7, true},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			newLine, newPos, err := ProcessLineReplace(tt.line, tt.pos)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %q at pos %d", tt.line, tt.pos)
				}
				return
			}
			if err != nil {
				t.Errorf("ProcessLineReplace(%q, %d) failed: %v", tt.line, tt.pos, err)
				return
			}
			if newLine != tt.expectedLine {
				t.Errorf("ProcessLineReplace(%q, %d) line = %q, expected %q", tt.line, tt.pos, newLine, tt.expectedLine)
			}
			if newPos != tt.expectedPos {
				t.Errorf("ProcessLineReplace(%q, %d) pos = %d, expected %d", tt.line, tt.pos, newPos, tt.expectedPos)
			}
		})
	}
}

func TestFormatResult(t *testing.T) {
	tests := []struct {
		value    float64
		expected string
	}{
		{2, "2"},
		{2.5, "2.5"},
		{2.0, "2"},
		{1000, "1000"},
		{3.14159, "3.14159"},
		{0.1, "0.1"},
		{-5, "-5"},
		{-3.5, "-3.5"},
		{1e10, "10000000000"},
		{1.23456789012, "1.2345678901"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatResult(tt.value)
			if result != tt.expected {
				t.Errorf("FormatResult(%v) = %q, expected %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestInvalidExpressions(t *testing.T) {
	tests := []string{
		"",
		"()",
		"1+",
		"+",
		"*5",
		"1++2",
		"(1+2",
		"1+2)",
		"1/0",
		"((1+2)",
		"2+*3",
		"0x",
		"0xG",
		"0o8",
		"0o9",
		"0b2",
		"0b",
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvaluateExpression(expr)
			if err == nil {
				t.Errorf("Expected error for invalid expression %q", expr)
			}
		})
	}
}