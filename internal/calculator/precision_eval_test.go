package calculator

import (
	"strings"
	"testing"
)

func TestPrecisionEvaluation(t *testing.T) {
	tests := []struct {
		expr           string
		expectedPrefix string // Start of expected result
		minDecimals    int    // Minimum decimal places expected
	}{
		{"1/3 pre100", "0.3333333333", 100},
		{"pre100 1/3", "0.3333333333", 100},
		{"1/pre 100 3", "0.3333333333", 100},
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
			if !strings.HasPrefix(resultStr, tt.expectedPrefix) {
				t.Errorf("Expression %q = %q, expected to start with %q",
					tt.expr, resultStr, tt.expectedPrefix)
			}

			// Check decimal places
			parts := strings.Split(resultStr, ".")
			if len(parts) == 2 {
				decimals := len(parts[1])
				if decimals < tt.minDecimals {
					t.Errorf("Expression %q has %d decimal places, expected at least %d",
						tt.expr, decimals, tt.minDecimals)
				}
			}
		})
	}
}

func TestActualPrecisionExpressions(t *testing.T) {
	// Test that various forms of precision specification work
	tests := []struct {
		expr        string
		shouldWork  bool
		description string
	}{
		{"1/3 pre10", true, "postfix with no space"},
		{"1/3 pre 10", true, "postfix with space"},
		{"pre10 1/3", true, "prefix with no space"},
		{"pre 10 1/3", true, "prefix with space"},
		{"1/pre 10 3", true, "inline with spaces"},
		{"(1+2)/3 pre20", true, "complex expression with precision"},
		{"pre20 (1+2)/3", true, "precision before complex expression"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result, err := EvaluateExpressionWithResult(tt.expr)
			if tt.shouldWork {
				if err != nil {
					t.Errorf("Expression %q failed: %v", tt.expr, err)
				} else {
					t.Logf("Expression %q = %s", tt.expr, result.String())
				}
			} else {
				if err == nil {
					t.Errorf("Expression %q should have failed but got: %s", tt.expr, result.String())
				}
			}
		})
	}
}
