package calculator

import (
	"testing"
)

func TestCommaSeparatedNumbers(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		// Basic comma-separated input
		{
			name:     "simple comma separated",
			expr:     "1,000",
			expected: "1,000",
		},
		{
			name:     "larger comma separated",
			expr:     "1,000,000",
			expected: "1,000,000",
		},
		{
			name:     "comma separated with decimal",
			expr:     "1,234.567",
			expected: "1,234.567",
		},
		{
			name:     "comma separated in calculation",
			expr:     "1,000 + 500",
			expected: "1,500",
		},
		{
			name:     "mixed comma and plain",
			expr:     "1,000 + 500",
			expected: "1,500",
		},
		{
			name:     "all comma separated",
			expr:     "1,000 + 2,000",
			expected: "3,000",
		},
		// cs format specifier
		{
			name:     "cs format basic",
			expr:     "1000 cs",
			expected: "1,000",
		},
		{
			name:     "cs format larger",
			expr:     "1000000 cs",
			expected: "1,000,000",
		},
		{
			name:     "cs format with decimal",
			expr:     "1234.567891 cs",
			expected: "1,234.567,891",
		},
		{
			name:     "cs format with calculation",
			expr:     "1000 + 500 cs",
			expected: "1,500",
		},
		{
			name:     "cs format negative",
			expr:     "-1000 cs",
			expected: "-1,000",
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
			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestUnderscoreSeparatedNumbers(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		// Basic underscore-separated input
		{
			name:     "simple underscore separated",
			expr:     "1_000",
			expected: "1_000",
		},
		{
			name:     "larger underscore separated",
			expr:     "1_000_000",
			expected: "1_000_000",
		},
		{
			name:     "underscore separated with decimal",
			expr:     "1_234.567",
			expected: "1_234.567",
		},
		{
			name:     "underscore separated in calculation",
			expr:     "1_000 + 500",
			expected: "1_500",
		},
		{
			name:     "all underscore separated",
			expr:     "1_000 + 2_000",
			expected: "3_000",
		},
		// us format specifier
		{
			name:     "us format basic",
			expr:     "1000 us",
			expected: "1_000",
		},
		{
			name:     "us format larger",
			expr:     "1000000 us",
			expected: "1_000_000",
		},
		{
			name:     "us format with decimal",
			expr:     "1234.567891 us",
			expected: "1_234.567_891",
		},
		{
			name:     "us format with calculation",
			expr:     "1000 + 500 us",
			expected: "1_500",
		},
		{
			name:     "us format negative",
			expr:     "-1000 us",
			expected: "-1_000",
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
			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestMixedSeparatorBehavior(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
		desc     string
	}{
		// Mixed separator types should output plain decimal
		{
			name:     "comma and underscore mixed",
			expr:     "1,000 + 1_000",
			expected: "2000",
			desc:     "mixed separator types output plain decimal",
		},
		{
			name:     "comma and plain mixed",
			expr:     "1,000 + 1000",
			expected: "2,000",
			desc:     "comma with plain still outputs comma",
		},
		{
			name:     "underscore and plain mixed",
			expr:     "1_000 + 1000",
			expected: "2_000",
			desc:     "underscore with plain still outputs underscore",
		},
		// Hex/octal/binary overrides separator formats
		{
			name:     "hex overrides cs",
			expr:     "0x10 cs",
			expected: "0x10",
			desc:     "hex presence ignores cs specifier",
		},
		{
			name:     "hex overrides us",
			expr:     "0x10 us",
			expected: "0x10",
			desc:     "hex presence ignores us specifier",
		},
		{
			name:     "binary overrides cs",
			expr:     "0b1111 cs",
			expected: "0b1111",
			desc:     "binary presence ignores cs specifier",
		},
		{
			name:     "octal overrides us",
			expr:     "0o77 us",
			expected: "0o77",
			desc:     "octal presence ignores us specifier",
		},
		{
			name:     "hex in expression overrides comma input",
			expr:     "1,000 + 0x10",
			expected: "0x3f8",  // 1000 + 16 = 1016 = 0x3f8
			desc:     "hex in expression outputs hex",
		},
		{
			name:     "binary in expression overrides underscore input",
			expr:     "1_000 + 0b1000",
			expected: "0b1111110000",  // 1000 + 8 = 1008 = 0b1111110000
			desc:     "binary in expression outputs binary",
		},
		// Mixed hex/octal/binary outputs decimal
		{
			name:     "hex and binary mixed",
			expr:     "0x10 + 0b1000",
			expected: "24",
			desc:     "mixed bases output decimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionWithResult(tt.expr)
			if err != nil {
				t.Errorf("EvaluateExpressionWithResult(%q) failed: %v [%s]", tt.expr, err, tt.desc)
				return
			}

			output := result.String()
			if output != tt.expected {
				t.Errorf("Expected %q, got %q [%s]", tt.expected, output, tt.desc)
			}
		})
	}
}

func TestSeparatorFormatting(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		// Test grouping of 3 digits
		{
			name:     "small number no separator",
			expr:     "100 cs",
			expected: "100",
		},
		{
			name:     "four digits with comma",
			expr:     "1000 cs",
			expected: "1,000",
		},
		{
			name:     "seven digits with comma",
			expr:     "1234567 cs",
			expected: "1,234,567",
		},
		{
			name:     "fractional part grouping",
			expr:     "1.234567 cs",
			expected: "1.234,567",
		},
		{
			name:     "both parts grouped",
			expr:     "1234.567891 cs",
			expected: "1,234.567,891",
		},
		// Underscore formatting
		{
			name:     "four digits with underscore",
			expr:     "1000 us",
			expected: "1_000",
		},
		{
			name:     "seven digits with underscore",
			expr:     "1234567 us",
			expected: "1_234_567",
		},
		{
			name:     "fractional part with underscore",
			expr:     "1.234567 us",
			expected: "1.234_567",
		},
		// Negative numbers
		{
			name:     "negative with comma",
			expr:     "-12345 cs",
			expected: "-12,345",
		},
		{
			name:     "negative with underscore",
			expr:     "-12345 us",
			expected: "-12_345",
		},
		// Large numbers from power operations
		{
			name:     "power result with comma formatting",
			expr:     "1,000 ** 2 cs",
			expected: "1,000,000",
		},
		{
			name:     "large power result with comma formatting",
			expr:     "1,000,000 ** 2 cs",
			expected: "1,000,000,000,000",
		},
		{
			name:     "power result with underscore formatting",
			expr:     "1_000 ** 2 us",
			expected: "1_000_000",
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
			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestSeparatorParsing(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected float64
	}{
		// Comma separated parsing
		{
			name:     "simple comma",
			expr:     "1,000",
			expected: 1000,
		},
		{
			name:     "multiple commas",
			expr:     "1,234,567",
			expected: 1234567,
		},
		{
			name:     "comma with decimal",
			expr:     "1,234.56",
			expected: 1234.56,
		},
		// Underscore separated parsing
		{
			name:     "simple underscore",
			expr:     "1_000",
			expected: 1000,
		},
		{
			name:     "multiple underscores",
			expr:     "1_234_567",
			expected: 1234567,
		},
		{
			name:     "underscore with decimal",
			expr:     "1_234.56",
			expected: 1234.56,
		},
		// Calculations with separated numbers
		{
			name:     "addition with commas",
			expr:     "1,000 + 2,000",
			expected: 3000,
		},
		{
			name:     "multiplication with underscores",
			expr:     "1_000 * 2",
			expected: 2000,
		},
		{
			name:     "mixed calculation",
			expr:     "(1,000 + 500) * 2",
			expected: 3000,
		},
		// Power operations with separated numbers
		{
			name:     "power with comma separator",
			expr:     "1,000 ** 2",
			expected: 1000000,
		},
		{
			name:     "large power with comma separator",
			expr:     "1,000,000 ** 2",
			expected: 1000000000000,  // 1 trillion
		},
		{
			name:     "power with underscore separator",
			expr:     "1_000 ** 2",
			expected: 1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expr)
			if err != nil {
				t.Errorf("EvaluateExpression(%q) failed: %v", tt.expr, err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}