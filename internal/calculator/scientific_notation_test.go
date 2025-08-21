package calculator

import (
	"math/big"
	"strings"
	"testing"
)

func TestScientificNotationPrevention(t *testing.T) {
	tests := []struct {
		name             string
		expr             string
		shouldNotContain []string // These should NOT be in the output
	}{
		{
			name:             "very large number multiplication",
			expr:             "123456789012345678901234567890 * 987654321098765432109876543210",
			shouldNotContain: []string{"e+", "E+", "e-", "E-"}, // Should not contain scientific notation
		},
		{
			name:             "very small number division",
			expr:             "1 / 123456789012345678901234567890123456789012345678901234567890",
			shouldNotContain: []string{"e+", "E+", "e-", "E-"}, // Should not contain scientific notation
		},
		{
			name:             "very large factorial-like calculation",
			expr:             "1000000000000000000000 * 1000000000000000000000",
			shouldNotContain: []string{"e+", "E+", "e-", "E-"}, // Should not contain scientific notation
		},
		{
			name:             "huge number with many digits",
			expr:             "999999999999999999999999999999999999999999999999999999999999",
			shouldNotContain: []string{"e+", "E+", "e-", "E-"}, // Should not contain scientific notation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionWithResult(tt.expr)
			if err != nil {
				t.Errorf("EvaluateExpressionWithResult(%q) failed: %v", tt.expr, err)
				return
			}

			output := result.String()
			t.Logf("Expression: %s", tt.expr)
			t.Logf("Result: %s", output)

			// Check that output doesn't contain scientific notation patterns
			for _, pattern := range tt.shouldNotContain {
				if strings.Contains(output, pattern) {
					t.Errorf("Output %q should not contain scientific notation pattern %q", output, pattern)
				}
			}

			// Additional check: result should be all digits (with possible negative sign and decimal point)
			validChars := "0123456789.-"
			for _, char := range output {
				if !strings.ContainsRune(validChars, char) {
					t.Errorf("Output %q contains invalid character %q for plain decimal format", output, string(char))
				}
			}
		})
	}
}

func TestFormatDefaultWithLargeNumbers(t *testing.T) {
	tests := []struct {
		name  string
		value string // String representation of big number
	}{
		{
			name:  "large integer",
			value: "123456789012345678901234567890",
		},
		{
			name:  "result of 2^100",
			value: "1267650600228229401496703205376", // 2^100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create big.Float from string with high precision
			bigFloat := new(big.Float)
			bigFloat.SetPrec(256) // Set high precision for large numbers
			bigFloat.SetString(tt.value)

			result := NewCalculationResult(bigFloat)
			output := result.String()

			t.Logf("Input: %s, Output: %s", tt.value, output)

			// Ensure no scientific notation
			if strings.Contains(output, "e") || strings.Contains(output, "E") {
				t.Errorf("Output %q contains scientific notation", output)
			}

			// Ensure the output is a valid decimal number
			validChars := "0123456789.-"
			for _, char := range output {
				if !strings.ContainsRune(validChars, char) {
					t.Errorf("Output %q contains invalid character %q for plain decimal format", output, string(char))
				}
			}
		})
	}
}

func TestBigFloatTextGFormatProducesScientificNotation(t *testing.T) {
	// This test demonstrates that big.Float.Text('g', 10) produces scientific notation
	// for very large numbers, which is the problem we need to fix

	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "very large number",
			value: "123456789012345678901234567890123456789012345678901234567890",
		},
		{
			name:  "very small number",
			value: "0.000000000000000000000000000000000000000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bigFloat := new(big.Float)
			bigFloat.SetString(tt.value)

			// Test the current behavior of big.Float.Text('g', 10)
			gFormat := bigFloat.Text('g', 10)
			t.Logf("big.Float.Text('g', 10) for %s: %s", tt.value, gFormat)

			// Check if it contains scientific notation
			hasScientific := strings.Contains(gFormat, "e") || strings.Contains(gFormat, "E")

			if hasScientific {
				t.Logf("CONFIRMED: big.Float.Text('g', 10) produces scientific notation: %s", gFormat)
			} else {
				t.Logf("big.Float.Text('g', 10) does NOT produce scientific notation: %s", gFormat)
			}
		})
	}
}
