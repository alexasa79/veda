package calculator

import (
	"testing"
)

func TestAutomaticBaseDetection(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		// Single base auto-detection
		{
			name:     "single hex number",
			expr:     "0xFF",
			expected: "0xff",
		},
		{
			name:     "hex arithmetic",
			expr:     "0xFF + 0x10",
			expected: "0x10f",
		},
		{
			name:     "binary arithmetic",
			expr:     "0b1010 + 0b0101",
			expected: "0b1111",
		},
		{
			name:     "octal arithmetic",
			expr:     "0o77 + 0o1",
			expected: "0o100",
		},
		{
			name:     "decimal stays decimal",
			expr:     "10 + 5",
			expected: "15",
		},

		// Hex with decimal uses hex output (ignore decimal when determining base)
		{
			name:     "hex and decimal",
			expr:     "0xFF + 10",
			expected: "0x109",
		},
		{
			name:     "binary and hex",
			expr:     "0b1111 + 0xFF",
			expected: "270", // Mixed non-decimal bases → decimal
		},
		{
			name:     "octal and decimal",
			expr:     "0o10 + 10",
			expected: "0o22", // Octal with decimal uses octal output
		},
		{
			name:     "all three bases",
			expr:     "0xFF + 0b1111 + 0o10",
			expected: "278",
		},

		// Complex expressions with single base
		{
			name:     "hex multiplication",
			expr:     "0xFF * 0x2",
			expected: "0x1fe",
		},
		{
			name:     "binary with parentheses",
			expr:     "(0b1010 + 0b0101) * 0b10",
			expected: "0b11110",
		},
		{
			name:     "octal division",
			expr:     "0o100 / 0o2",
			expected: "0o40",
		},

		// Explicit directives override auto-detection
		{
			name:     "hex numbers with dec directive",
			expr:     "0xFF + 0x10 dec",
			expected: "271",
		},
		{
			name:     "decimal with hex directive",
			expr:     "255 hex",
			expected: "0xff",
		},
		{
			name:     "binary with oct directive",
			expr:     "0b1111 oct",
			expected: "0o17",
		},
		{
			name:     "mixed bases with bin directive",
			expr:     "0xFF + 10 bin",
			expected: "0b100001001",
		},

		// Decimal numbers with single non-decimal base
		{
			name:     "binary with multiple decimals",
			expr:     "10 + 0b1010 + 5",
			expected: "0b11001", // Binary is the only non-decimal base
		},
		{
			name:     "hex surrounded by decimals",
			expr:     "100 + 0xA + 50",
			expected: "0xa0", // Hex is the only non-decimal base
		},

		// Floating point always decimal (no hex/bin/oct floats)
		{
			name:     "float arithmetic",
			expr:     "3.14 + 2.86",
			expected: "6",
		},
		{
			name:     "hex int with float",
			expr:     "0xFF + 1.5",
			expected: "0x100", // Hex output truncates float result
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionWithResult(tt.expr)
			if err != nil {
				t.Fatalf("Failed to evaluate expression %q: %v", tt.expr, err)
			}

			got := result.String()
			if got != tt.expected {
				t.Errorf("Expression %q = %s; want %s", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestAutoBaseWithPrecision(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		{
			name:     "hex with precision",
			expr:     "0xff / 0x3 pre10",
			expected: "85.0000000000", // Precision forces decimal display
		},
		{
			name:     "binary with precision shows decimal",
			expr:     "0b10 / 0b11 pre10",
			expected: "0.6666666667", // Precision forces decimal
		},
		{
			name:     "mixed bases with precision",
			expr:     "0xFF / 10 pre5",
			expected: "25.50000", // Mixed bases + precision = decimal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionWithResult(tt.expr)
			if err != nil {
				t.Fatalf("Failed to evaluate expression %q: %v", tt.expr, err)
			}

			got := result.String()
			if got != tt.expected {
				t.Errorf("Expression %q = %s; want %s", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestAutoBaseInProcessLineAppend(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		pos      int
		expected string
	}{
		{
			name:     "hex calculation",
			line:     "The value is 0xFF + 0x1",
			pos:      24,
			expected: "The value is 0xFF + 0x1 → 0x100",
		},
		{
			name:     "binary calculation",
			line:     "Binary: 0b1010 + 0b0101",
			pos:      23,
			expected: "Binary: 0b1010 + 0b0101 → 0b1111",
		},
		{
			name:     "mixed bases",
			line:     "Mixed: 0xFF + 10",
			pos:      16,
			expected: "Mixed: 0xFF + 10 → 0x109", // Hex is the only non-decimal base
		},
		// Note: ProcessLineAppend with explicit directives is tested separately
		// in format_test.go since it has specific edge cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessLineAppend(tt.line, tt.pos)
			if err != nil {
				t.Fatalf("ProcessLineAppend failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("ProcessLineAppend(%q, %d) = %q; want %q",
					tt.line, tt.pos, result, tt.expected)
			}
		})
	}
}
