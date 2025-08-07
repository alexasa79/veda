package calculator

import (
	"strings"
	"testing"
)

func TestPrecisionDirective(t *testing.T) {
	tests := []struct {
		expr           string
		expectedResult string
		minDigits      int // minimum number of decimal digits expected
	}{
		// Basic precision tests
		{"1/3 pre10", "0.3333333333", 10},
		{"pre10 1/3", "0.3333333333", 10},
		{"1/pre 10 3", "0.3333333333", 10},
		
		// Higher precision
		{"1/3 pre50", "0.33333333333333333333333333333333333333333333333333", 50},
		{"1/7 pre20", "0.14285714285714285714", 20},
		
		// Precision with spaces
		{"1/3 pre 10", "0.3333333333", 10},
		{"pre 10 1/3", "0.3333333333", 10},
		
		// Complex expressions with precision
		{"(1/3 + 1/7) pre15", "0.476190476190476", 15},
		{"pre15 (1/3 + 1/7)", "0.476190476190476", 15},
		
		// Integer results with precision (should show decimal places)
		{"2+2 pre5", "4.00000", 5},
		{"10/5 pre3", "2.000", 3},
	}
	
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvaluateExpressionWithResult(tt.expr)
			if err != nil {
				t.Errorf("Failed to evaluate %q: %v", tt.expr, err)
				return
			}
			
			resultStr := result.String()
			
			// Check if result starts with expected prefix
			if !strings.HasPrefix(resultStr, tt.expectedResult) {
				t.Errorf("Expression %q = %q, expected to start with %q", 
					tt.expr, resultStr, tt.expectedResult)
			}
			
			// Count decimal places
			parts := strings.Split(resultStr, ".")
			if len(parts) == 2 {
				decimalPlaces := len(parts[1])
				if decimalPlaces < tt.minDigits {
					t.Errorf("Expression %q has %d decimal places, expected at least %d",
						tt.expr, decimalPlaces, tt.minDigits)
				}
			} else if tt.minDigits > 0 {
				t.Errorf("Expression %q has no decimal point, expected %d decimal places",
					tt.expr, tt.minDigits)
			}
		})
	}
}

func TestPreprocessExpression(t *testing.T) {
	tests := []struct {
		input         string
		expectedExpr  string
		expectedPrec  int
		expectedFound bool
	}{
		// Basic cases
		{"1/3 pre100", "1/3", 100, true},
		{"pre100 1/3", "1/3", 100, true},
		{"1/pre 100 3", "1/3", 100, true},
		
		// With spaces
		{"1/3 pre 100", "1/3", 100, true},
		{"pre 100 1/3", "1/3", 100, true},
		
		// Multiple pre directives (last one wins)
		{"pre50 1/3 pre100", "1/3", 100, true},
		
		// No pre directive
		{"1/3", "1/3", -1, false},
		{"1+2+3", "1+2+3", -1, false},
		
		// Pre in middle of expression
		{"1 + pre10 2/3", "1+2/3", 10, true},
		{"(1 pre5 + 2)/3", "(1+2)/3", 5, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, prec, found := preprocessExpression(tt.input)
			
			if expr != tt.expectedExpr {
				t.Errorf("preprocessExpression(%q) expr = %q, expected %q",
					tt.input, expr, tt.expectedExpr)
			}
			
			if prec != tt.expectedPrec {
				t.Errorf("preprocessExpression(%q) precision = %d, expected %d",
					tt.input, prec, tt.expectedPrec)
			}
			
			if found != tt.expectedFound {
				t.Errorf("preprocessExpression(%q) found = %v, expected %v",
					tt.input, found, tt.expectedFound)
			}
		})
	}
}

func TestProcessLineWithPrecision(t *testing.T) {
	tests := []struct {
		line           string
		pos            int
		expectedAppend string
		hasMinDigits   int
	}{
		{"1/3 pre10", 10, "1/3 pre10 → 0.3333333333", 10},
		{"Calculate: 1/7 pre15", 20, "Calculate: 1/7 pre15 → 0.142857142857142", 15},
		{"pre20 2/3", 9, "pre20 2/3 → 0.66666666666666666666", 20},
	}
	
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result, err := ProcessLineAppend(tt.line, tt.pos)
			if err != nil {
				t.Errorf("ProcessLineAppend(%q, %d) failed: %v", tt.line, tt.pos, err)
				return
			}
			
			// Check that the result contains the arrow
			if !strings.Contains(result, "→") {
				t.Errorf("Result %q doesn't contain arrow", result)
				return
			}
			
			// Extract the calculated part
			parts := strings.Split(result, "→")
			if len(parts) != 2 {
				t.Errorf("Result %q should have exactly one arrow", result)
				return
			}
			
			calculated := strings.TrimSpace(parts[1])
			
			// Check decimal places
			decParts := strings.Split(calculated, ".")
			if len(decParts) == 2 {
				decimalPlaces := len(decParts[1])
				if decimalPlaces < tt.hasMinDigits {
					t.Errorf("Result %q has %d decimal places, expected at least %d",
						calculated, decimalPlaces, tt.hasMinDigits)
				}
			}
		})
	}
}