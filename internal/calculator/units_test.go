package calculator

import (
	"testing"
)

func TestPreprocessUnits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic SI prefixes
		{"kilo", "100k", "(100*1000)"},
		{"mega", "100m", "(100*1000000)"},
		{"giga", "100g", "(100*1000000000)"},
		{"billion", "100b", "(100*1000000000)"},
		{"tera", "100t", "(100*1000000000000)"},

		// Binary prefixes - long form
		{"kibibyte", "100kib", "(100*1024)"},
		{"mebibyte", "100mib", "(100*1048576)"},
		{"gibibyte", "100gib", "(100*1073741824)"},
		{"tebibyte", "100tib", "(100*1099511627776)"},
		{"pebibyte", "100pib", "(100*1125899906842624)"},
		{"exbibyte", "100eib", "(100*1152921504606846976)"},

		// Binary prefixes - short form
		{"ki short", "100ki", "(100*1024)"},
		{"mi short", "100mi", "(100*1048576)"},
		{"gi short", "100gi", "(100*1073741824)"},
		{"ti short", "100ti", "(100*1099511627776)"},
		{"pi short", "100pi", "(100*1125899906842624)"},
		{"ei short", "100ei", "(100*1152921504606846976)"},

		// Case insensitivity
		{"upper K", "100K", "(100*1000)"},
		{"upper M", "100M", "(100*1000000)"},
		{"upper GiB", "100GiB", "(100*1073741824)"},
		{"mixed case KiB", "100KiB", "(100*1024)"},

		// Decimal numbers
		{"decimal kilo", "3.14k", "(3.14*1000)"},
		{"decimal mega", "1.5m", "(1.5*1000000)"},

		// Division case - the key use case
		{"division by kilo", "1/1k", "1/(1*1000)"},
		{"division by mega", "100/2m", "100/(2*1000000)"},

		// Parenthesis followed by unit
		{"paren kilo", "(1+2)k", "(1+2)*1000"},
		{"paren mega", "(100/2)m", "(100/2)*1000000"},
		{"nested paren", "((1+2)*3)k", "((1+2)*3)*1000"},

		// Multiple units in expression
		{"multi units", "1k+2m", "(1*1000)+(2*1000000)"},
		{"multi units div", "1k/2k", "(1*1000)/(2*1000)"},

		// No unit - should remain unchanged
		{"no unit", "100+200", "100+200"},
		{"no unit parens", "(1+2)*3", "(1+2)*3"},

		// Edge cases
		{"empty", "", ""},
		{"just number", "100", "100"},
		{"unit at end of expr", "100k+50", "(100*1000)+50"},

		// Binary/hex prefixes should NOT be treated as units
		{"binary prefix", "0b1111", "0b1111"},
		{"binary in expr", "0b1111+1", "0b1111+1"},
		{"hex prefix", "0xff", "0xff"},
		{"octal prefix", "0o77", "0o77"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PreprocessUnits(tt.input)
			if result != tt.expected {
				t.Errorf("PreprocessUnits(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPreprocessUnitsIntegration(t *testing.T) {
	// Test that preprocessing + evaluation gives correct results
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"simple kilo", "1k", 1000},
		{"simple mega", "1m", 1000000},
		{"division by kilo", "1/1k", 0.001},
		{"division by mega", "1/1m", 0.000001},
		{"paren kilo", "(1+2)k", 3000},
		{"complex", "1k+2k", 3000},
		{"gibibyte", "1gib", 1073741824},
		{"mixed", "2*3k", 6000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preprocessed := PreprocessUnits(tt.input)
			result, err := EvaluateExpression(preprocessed)
			if err != nil {
				t.Errorf("EvaluateExpression(%q) error: %v", preprocessed, err)
				return
			}
			if result != tt.expected {
				t.Errorf("EvaluateExpression(PreprocessUnits(%q)) = %v, want %v (preprocessed: %q)",
					tt.input, result, tt.expected, preprocessed)
			}
		})
	}
}
