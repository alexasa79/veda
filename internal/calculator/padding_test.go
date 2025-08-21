package calculator

import (
	"testing"
)

func TestBitWidthPadding(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
	}{
		// Hex padding
		{
			name:     "hex16 padding",
			expr:     "0x10 hex16",
			expected: "0x0010",
		},
		{
			name:     "hex32 padding",
			expr:     "0xFF hex32",
			expected: "0x000000ff",
		},
		{
			name:     "hex8 no padding needed",
			expr:     "0xFF hex8",
			expected: "0xff",
		},

		// Hex trimming
		{
			name:     "hex16 trimming",
			expr:     "0xffffff hex16",
			expected: "0xffff",
		},
		{
			name:     "hex8 trimming",
			expr:     "0x1234 hex8",
			expected: "0x34",
		},

		// Binary padding
		{
			name:     "bin8 padding",
			expr:     "0b101 bin8",
			expected: "0b00000101",
		},
		{
			name:     "bin16 padding",
			expr:     "15 bin16",
			expected: "0b0000000000001111",
		},

		// Binary trimming
		{
			name:     "bin8 trimming",
			expr:     "0xFF bin8",
			expected: "0b11111111",
		},
		{
			name:     "bin4 trimming",
			expr:     "0b11111111 bin8",
			expected: "0b11111111",
		},

		// Octal padding
		{
			name:     "oct padding for 9 bits",
			expr:     "0o7 oct16", // 16 bits = 6 octal digits (rounded up)
			expected: "0o000007",
		},
		{
			name:     "oct trimming",
			expr:     "0o777777 oct8", // 8 bits = 3 octal digits
			expected: "0o377",
		},

		// Calculations with padding
		{
			name:     "addition with hex16",
			expr:     "0x10 + 0x20 hex16",
			expected: "0x0030",
		},
		{
			name:     "multiplication with bin8",
			expr:     "3 * 4 bin8",
			expected: "0b00001100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpressionWithResult(tt.expr)
			if err != nil {
				t.Fatalf("Failed to evaluate %q: %v", tt.expr, err)
			}

			got := result.String()
			if got != tt.expected {
				t.Errorf("Expression %q = %s; want %s", tt.expr, got, tt.expected)
			}
		})
	}
}
