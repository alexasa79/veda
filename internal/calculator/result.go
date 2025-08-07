package calculator

import (
	"fmt"
	"math/big"
	"strings"
)

// FormatType represents the output format for the result
type FormatType int

const (
	FormatDecimal FormatType = iota
	FormatHex
	FormatBinary
	FormatOctal
)

// CalculationResult represents a calculation result with arbitrary precision
type CalculationResult struct {
	Value     *big.Float
	Precision int        // Number of decimal places to display (-1 for default)
	Format    FormatType // Output format (decimal, hex, binary, octal)
	BitWidth  int        // Number of bits for hex/bin/oct output (0 for auto)
}

// NewCalculationResult creates a new result with default precision
func NewCalculationResult(value *big.Float) *CalculationResult {
	return &CalculationResult{
		Value:     value,
		Precision: -1, // -1 means use default formatting
		Format:    FormatDecimal,
		BitWidth:  0, // 0 means auto
	}
}

// SetPrecision sets the display precision
func (r *CalculationResult) SetPrecision(precision int) {
	r.Precision = precision
}

// SetFormat sets the output format and bit width
func (r *CalculationResult) SetFormat(format FormatType, bitWidth int) {
	r.Format = format
	r.BitWidth = bitWidth
}

// String formats the result according to its format and precision settings
func (r *CalculationResult) String() string {
	if r.Value == nil {
		return "0"
	}

	// Handle non-decimal formats
	if r.Format != FormatDecimal {
		return r.formatNonDecimal()
	}

	// Handle decimal format with precision
	if r.Precision >= 0 {
		// Format with specific precision
		return r.formatWithPrecision()
	}

	// Default formatting - try to be smart about it
	return r.formatDefault()
}

// formatDefault formats with automatic precision
func (r *CalculationResult) formatDefault() string {
	// Check if it's an integer
	if r.isInteger() {
		// Format as integer
		intVal, _ := r.Value.Int(nil)
		return intVal.String()
	}

	// Use Text with 'g' format for automatic precision
	// 'g' removes trailing zeros automatically
	str := r.Value.Text('g', 10)
	return str
}

// formatWithPrecision formats with specific decimal places
func (r *CalculationResult) formatWithPrecision() string {
	// Use Text with 'f' format for fixed-point notation
	str := r.Value.Text('f', r.Precision)
	return str
}

// formatNonDecimal formats the result as hex, binary, or octal
func (r *CalculationResult) formatNonDecimal() string {
	// Convert to integer (truncate if necessary)
	intVal, _ := r.Value.Int(nil)
	
	// Handle negative values - use two's complement representation
	if intVal.Sign() < 0 {
		// Determine bit width
		bitWidth := r.BitWidth
		if bitWidth == 0 {
			// Auto-determine bit width based on value
			bitWidth = r.autoDetermineBitWidth(intVal)
		}
		
		// Convert to two's complement
		intVal = r.toTwosComplement(intVal, bitWidth)
	}
	
	// Format based on type
	switch r.Format {
	case FormatHex:
		result := fmt.Sprintf("%x", intVal)
		// Add 0x prefix
		return "0x" + result
	case FormatBinary:
		result := fmt.Sprintf("%b", intVal)
		// Pad to bit width if specified
		if r.BitWidth > 0 && len(result) < r.BitWidth {
			padding := strings.Repeat("0", r.BitWidth-len(result))
			result = padding + result
		}
		return "0b" + result
	case FormatOctal:
		result := fmt.Sprintf("%o", intVal)
		return "0o" + result
	default:
		return r.formatDefault()
	}
}

// autoDetermineBitWidth determines appropriate bit width for a value
func (r *CalculationResult) autoDetermineBitWidth(val *big.Int) int {
	// Get the absolute value for bit count
	absVal := new(big.Int).Abs(val)
	bitLen := absVal.BitLen()
	
	// Round up to nearest power of 2 that's at least 8
	if bitLen <= 8 {
		return 8
	} else if bitLen <= 16 {
		return 16
	} else if bitLen <= 32 {
		return 32
	} else if bitLen <= 64 {
		return 64
	} else if bitLen <= 128 {
		return 128
	}
	// For very large numbers, round up to nearest multiple of 64
	return ((bitLen + 63) / 64) * 64
}

// toTwosComplement converts a negative number to two's complement representation
func (r *CalculationResult) toTwosComplement(val *big.Int, bitWidth int) *big.Int {
	// Create a mask for the bit width
	one := big.NewInt(1)
	mask := new(big.Int).Lsh(one, uint(bitWidth))
	mask.Sub(mask, one) // mask = (1 << bitWidth) - 1
	
	// Two's complement: (2^bitWidth + val) & mask
	result := new(big.Int).Lsh(one, uint(bitWidth))
	result.Add(result, val)
	result.And(result, mask)
	
	return result
}

// isInteger checks if the value is an integer
func (r *CalculationResult) isInteger() bool {
	if r.Value == nil {
		return true
	}

	// Check if the value is an integer by comparing with truncated value
	intPart, _ := r.Value.Int(nil)
	intAsFloat := new(big.Float).SetInt(intPart)
	
	// Compare with original
	diff := new(big.Float).Sub(r.Value, intAsFloat)
	return diff.Sign() == 0
}

// FormatResultWithPrecision formats a float64 with specific precision
func FormatResultWithPrecision(value float64, precision int) string {
	bf := new(big.Float).SetFloat64(value)
	result := NewCalculationResult(bf)
	result.SetPrecision(precision)
	return result.String()
}

// preprocessExpression extracts precision directives and returns cleaned expression
// Returns: (cleaned expression, precision value, found)
func preprocessExpression(expr string) (string, int, bool) {
	// Use the tokenizer to handle this properly
	tokens, directive, hasDirective := TokenizeAndExtractPreprocessors(expr, 0)
	
	// Rebuild the expression from tokens (without the pre directive)
	// The goal is to reconstruct minimal/compact form
	result := strings.Builder{}
	
	for _, token := range tokens {
		if token.Type == TokenEOF {
			break
		}
		result.WriteString(token.Value)
	}
	
	precision := -1
	if hasDirective && directive.Precision >= 0 {
		precision = directive.Precision
	}
	
	return strings.TrimSpace(result.String()), precision, hasDirective && directive.Precision >= 0
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isLetterOrDigit(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}