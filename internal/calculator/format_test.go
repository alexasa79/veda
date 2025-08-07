package calculator

import (
	"strings"
	"testing"
)

func TestFormatDirectives(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
		contains []string // For checking partial matches
	}{
		// Dec format tests (explicit decimal)
		{
			name:     "dec format basic integer",
			expr:     "15 dec",
			expected: "15",
		},
		{
			name:     "dec format with float",
			expr:     "3.14159 dec",
			expected: "3.14159",
		},
		{
			name:     "dec format with calculation",
			expr:     "1/3 dec",
			expected: "0.3333333333",
		},
		{
			name:     "dec format prefix notation",
			expr:     "dec 255",
			expected: "255",
		},
		{
			name:     "dec format from hex input",
			expr:     "0xFF dec",
			expected: "255",
		},
		{
			name:     "dec format from binary input",
			expr:     "0b1111 dec",
			expected: "15",
		},
		// Note: "dec16" will leave "16" in the token stream creating "15 16" which is invalid
		// So we remove this test case
		
		// Hex format tests
		{
			name:     "hex format basic",
			expr:     "15 hex",
			expected: "0xf",
		},
		{
			name:     "hex format with bit width 8",
			expr:     "15 hex8",
			expected: "0xf",
		},
		{
			name:     "hex format with bit width 16",
			expr:     "255 hex16",
			expected: "0xff",
		},
		{
			name:     "hex format negative number",
			expr:     "-1 hex8",
			expected: "0xff", // Two's complement for 8-bit
		},
		{
			name:     "hex format negative number 16-bit",
			expr:     "-1 hex16",
			expected: "0xffff",
		},
		{
			name:     "hex format negative number 32-bit",
			expr:     "-1 hex32",
			expected: "0xffffffff",
		},
		{
			name:     "hex format calculation",
			expr:     "10 + 6 hex",
			expected: "0x10",
		},
		{
			name:     "hex format prefix notation",
			expr:     "hex 255",
			expected: "0xff",
		},
		// Note: Invalid bit widths are tricky - "hex10" will try to parse "10" as bit width,
		// fail, then put it back. This creates "15 10" which is invalid.
		// We'll test this scenario separately in bit width validation tests
		
		// Binary format tests
		{
			name:     "binary format basic",
			expr:     "5 bin",
			expected: "0b101",
		},
		{
			name:     "binary format with bit width 8",
			expr:     "5 bin8",
			expected: "0b00000101",
		},
		{
			name:     "binary format with bit width 16",
			expr:     "255 bin16",
			expected: "0b0000000011111111",
		},
		{
			name:     "binary format negative number 8-bit",
			expr:     "-1 bin8",
			expected: "0b11111111",
		},
		{
			name:     "binary format calculation",
			expr:     "2+2 bin",
			expected: "0b100",
		},
		
		// Octal format tests
		{
			name:     "octal format basic",
			expr:     "8 oct",
			expected: "0o10",
		},
		{
			name:     "octal format larger number",
			expr:     "64 oct",
			expected: "0o100",
		},
		{
			name:     "octal format calculation",
			expr:     "7+1 oct",
			expected: "0o10",
		},
		
		// Mixed format tests
		{
			name:     "hex input to binary output",
			expr:     "0xFF bin",
			expected: "0b11111111",
		},
		{
			name:     "binary input to hex output",
			expr:     "0b1111 hex",
			expected: "0xf",
		},
		{
			name:     "octal input to hex output",
			expr:     "0o77 hex",
			expected: "0x3f",
		},
		
		// Floating point truncation
		{
			name:     "float to hex truncates",
			expr:     "15.7 hex",
			expected: "0xf", // Truncates to 15
		},
		{
			name:     "float to binary truncates",
			expr:     "5.9 bin",
			expected: "0b101", // Truncates to 5
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
			
			if tt.expected != "" {
				if output != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, output)
				}
			}
			
			for _, substr := range tt.contains {
				if !strings.Contains(output, substr) {
					t.Errorf("Output %q doesn't contain %q", output, substr)
				}
			}
		})
	}
}

func TestBitWidthValidation(t *testing.T) {
	tests := []struct {
		name         string
		expr         string
		shouldWork   bool
		shouldParse  bool
		description  string
	}{
		// Test that dec doesn't consume bit width
		{
			name:         "dec with number doesn't consume as bit width",
			expr:         "15 dec8",
			shouldWork:   false,  // No bit width for dec
			shouldParse:  false,  // Creates "15 8" which is invalid
			description:  "dec doesn't support bit width, creates invalid expression",
		},
		{
			name:         "dec with power of 2 still doesn't consume",
			expr:         "15 dec16",
			shouldWork:   false,
			shouldParse:  false,  // Creates "15 16" which is invalid
			description:  "dec doesn't support bit width even if power of 2",
		},
		{
			name:         "valid bit width 8",
			expr:         "15 hex8",
			shouldWork:   true,
			shouldParse:  true,
			description:  "8 is power of 2",
		},
		{
			name:         "valid bit width 16",
			expr:         "15 hex16",
			shouldWork:   true,
			shouldParse:  true,
			description:  "16 is power of 2",
		},
		{
			name:         "valid bit width 32",
			expr:         "15 hex32",
			shouldWork:   true,
			shouldParse:  true,
			description:  "32 is power of 2",
		},
		{
			name:         "valid bit width 64",
			expr:         "15 hex64",
			shouldWork:   true,
			shouldParse:  true,
			description:  "64 is power of 2",
		},
		{
			name:         "invalid bit width 10",
			expr:         "15 hex10",
			shouldWork:   false,
			shouldParse:  false, // Invalid bit width creates "15 10" which is invalid
			description:  "10 is not power of 2, creates invalid expression",
		},
		{
			name:         "invalid bit width 24",
			expr:         "15 bin24",
			shouldWork:   false,
			shouldParse:  false, // Invalid bit width creates "15 24" which is invalid
			description:  "24 is not power of 2, creates invalid expression",
		},
		{
			name:         "invalid bit width 7",
			expr:         "15 bin7",
			shouldWork:   false,
			shouldParse:  false, // Invalid bit width creates "15 7" which is invalid
			description:  "7 is not power of 2 and less than 8, creates invalid expression",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, directive, _ := TokenizeAndExtractPreprocessors(tt.expr, 0)
			
			// Check if bit width was accepted
			if tt.shouldWork {
				if directive.BitWidth == 0 {
					t.Errorf("Expected bit width to be set for %q (%s)", tt.expr, tt.description)
				}
			} else {
				if directive.BitWidth != 0 {
					t.Errorf("Expected bit width to be ignored for %q (%s), got %d", 
						tt.expr, tt.description, directive.BitWidth)
				}
			}
			
			// Check if expression parses
			parser := NewParserFromTokens(tokens)
			_, err := parser.Parse()
			if tt.shouldParse {
				if err != nil {
					t.Errorf("Expression %q should be parseable: %v", tt.expr, err)
				}
			} else {
				if err == nil {
					t.Errorf("Expression %q should not be parseable (invalid bit width)", tt.expr)
				}
			}
		})
	}
}

func TestIsPowerOfTwo(t *testing.T) {
	tests := []struct {
		n        int
		expected bool
	}{
		{1, true},
		{2, true},
		{4, true},
		{8, true},
		{16, true},
		{32, true},
		{64, true},
		{128, true},
		{256, true},
		{3, false},
		{6, false},
		{10, false},
		{15, false},
		{24, false},
		{0, false},
		{-1, false},
		{-8, false},
	}
	
	for _, tt := range tests {
		t.Run(string(rune(tt.n)), func(t *testing.T) {
			result := isPowerOfTwo(tt.n)
			if result != tt.expected {
				t.Errorf("isPowerOfTwo(%d) = %v, expected %v", tt.n, result, tt.expected)
			}
		})
	}
}

func TestProcessLineWithFormats(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		cursorPos int
		expected  string
	}{
		{
			name:      "hex format append",
			line:      "15 hex",
			cursorPos: 2,  // Cursor at end of "15"
			expected:  "15 → 0xf hex",
		},
		{
			name:      "binary format append",
			line:      "Value: 7 bin",
			cursorPos: 8,  // Cursor at end of "7"
			expected:  "Value: 7 → 0b111 bin",
		},
		{
			name:      "octal format with calculation",
			line:      "Result = 8+8 oct",
			cursorPos: 12,  // Cursor at end of "8+8"
			expected:  "Result = 8+8 → 0o20 oct",
		},
		{
			name:      "hex with bit width",
			line:      "Status: 255 hex16",
			cursorPos: 11,  // Cursor at end of "255"
			expected:  "Status: 255 → 0xff hex16",
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
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}