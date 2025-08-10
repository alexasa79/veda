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
	FormatComma      // Decimal with comma separators
	FormatUnderscore // Decimal with underscore separators
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

	// If precision is specified, always use decimal format
	// (precision only makes sense for decimal numbers)
	if r.Precision >= 0 {
		// Format with specific precision
		return r.formatWithPrecision()
	}

	// Handle non-decimal formats (only when no precision is specified)
	// Note: Non-decimal formats will truncate floats to integers
	if r.Format == FormatHex || r.Format == FormatBinary || r.Format == FormatOctal {
		return r.formatNonDecimal()
	}
	
	// Handle separator formats (comma or underscore)
	if r.Format == FormatComma || r.Format == FormatUnderscore {
		return r.formatWithSeparator()
	}

	// Default formatting - decimal for floats or when format is decimal
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

	// For reasonable sized numbers, use moderate precision (similar to old 'g' format behavior)
	// For very small or very large numbers, use higher precision to avoid scientific notation
	precision := 10 // Start with 10 decimal places (similar to old behavior)
	
	// Check if the number would round to zero with our default precision
	testStr := r.Value.Text('f', precision)
	if strings.TrimRight(strings.TrimRight(testStr, "0"), ".") == "0" && r.Value.Sign() != 0 {
		// Number is very small, try with higher precision to avoid showing 0
		precision = 100
		testStr = r.Value.Text('f', precision)
		
		// If still zero, try even higher
		if strings.TrimRight(strings.TrimRight(testStr, "0"), ".") == "0" && r.Value.Sign() != 0 {
			precision = 200
		}
	}
	
	// Use Text with 'f' format to avoid scientific notation
	str := r.Value.Text('f', precision)
	
	// Remove trailing zeros and decimal point if not needed
	str = strings.TrimRight(str, "0")
	str = strings.TrimRight(str, ".")
	
	return str
}

// formatWithPrecision formats with specific decimal places
func (r *CalculationResult) formatWithPrecision() string {
	// Use Text with 'f' format for fixed-point notation
	str := r.Value.Text('f', r.Precision)
	return str
}

// formatWithSeparator formats the number with comma or underscore separators
func (r *CalculationResult) formatWithSeparator() string {
	// Get the string representation
	var str string
	if r.isInteger() {
		intVal, _ := r.Value.Int(nil)
		str = intVal.String()
	} else {
		// Use 'f' format to avoid scientific notation, then trim trailing zeros
		precision := 10 // Start with reasonable precision
		str = r.Value.Text('f', precision)
		
		// If we got zero but the value is non-zero, increase precision
		if strings.TrimRight(strings.TrimRight(str, "0"), ".") == "0" && r.Value.Sign() != 0 {
			precision = 100
			str = r.Value.Text('f', precision)
			// If still zero, try even higher
			if strings.TrimRight(strings.TrimRight(str, "0"), ".") == "0" && r.Value.Sign() != 0 {
				precision = 200
				str = r.Value.Text('f', precision)
			}
		}
		
		str = strings.TrimRight(str, "0")
		str = strings.TrimRight(str, ".")
	}
	
	// Determine separator
	sep := ","
	if r.Format == FormatUnderscore {
		sep = "_"
	}
	
	// Split into integer and fractional parts
	parts := strings.Split(str, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}
	
	// Handle negative sign
	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}
	
	// Add separators to integer part (from right to left, every 3 digits)
	formattedInt := addSeparators(intPart, sep, 3, true)
	
	// Add separators to fractional part (from left to right, every 3 digits)
	result := formattedInt
	if fracPart != "" {
		formattedFrac := addSeparators(fracPart, sep, 3, false)
		result = result + "." + formattedFrac
	}
	
	// Add back negative sign
	if negative {
		result = "-" + result
	}
	
	return result
}

// addSeparators adds separators to a string of digits
func addSeparators(s string, sep string, groupSize int, fromRight bool) string {
	if len(s) <= groupSize {
		return s
	}
	
	var result []string
	
	if fromRight {
		// For integer part: group from right to left
		for i := len(s); i > 0; i -= groupSize {
			start := i - groupSize
			if start < 0 {
				start = 0
			}
			result = append([]string{s[start:i]}, result...)
		}
	} else {
		// For fractional part: group from left to right
		for i := 0; i < len(s); i += groupSize {
			end := i + groupSize
			if end > len(s) {
				end = len(s)
			}
			result = append(result, s[i:end])
		}
	}
	
	return strings.Join(result, sep)
}

// formatNonDecimal formats the result as hex, binary, or octal
func (r *CalculationResult) formatNonDecimal() string {
	// Convert to integer (truncate if necessary)
	intVal, _ := r.Value.Int(nil)
	
	// If bit width is specified, mask the value to fit
	if r.BitWidth > 0 {
		// Create a mask for the bit width
		one := big.NewInt(1)
		mask := new(big.Int).Lsh(one, uint(r.BitWidth))
		mask.Sub(mask, one) // mask = (1 << bitWidth) - 1
		
		// Apply mask to get only the bits that fit
		intVal = new(big.Int).And(intVal, mask)
	} else if intVal.Sign() < 0 {
		// Handle negative values - use two's complement representation
		// Auto-determine bit width based on value
		bitWidth := r.autoDetermineBitWidth(intVal)
		// Convert to two's complement
		intVal = r.toTwosComplement(intVal, bitWidth)
	}
	
	// Format based on type
	switch r.Format {
	case FormatHex:
		result := fmt.Sprintf("%x", intVal)
		// Pad to match bit width if specified
		if r.BitWidth > 0 {
			// Calculate hex digits needed for bit width (4 bits per hex digit)
			hexDigits := r.BitWidth / 4
			if len(result) < hexDigits {
				padding := strings.Repeat("0", hexDigits-len(result))
				result = padding + result
			} else if len(result) > hexDigits {
				// Trim if too long
				result = result[len(result)-hexDigits:]
			}
		}
		// Add 0x prefix
		return "0x" + result
	case FormatBinary:
		result := fmt.Sprintf("%b", intVal)
		// Pad or trim to bit width if specified
		if r.BitWidth > 0 {
			if len(result) < r.BitWidth {
				padding := strings.Repeat("0", r.BitWidth-len(result))
				result = padding + result
			} else if len(result) > r.BitWidth {
				// Trim if too long
				result = result[len(result)-r.BitWidth:]
			}
		}
		return "0b" + result
	case FormatOctal:
		result := fmt.Sprintf("%o", intVal)
		// Pad to match bit width if specified
		if r.BitWidth > 0 {
			// Calculate octal digits needed for bit width (3 bits per octal digit)
			// Round up to ensure we have enough digits
			octalDigits := (r.BitWidth + 2) / 3
			if len(result) < octalDigits {
				padding := strings.Repeat("0", octalDigits-len(result))
				result = padding + result
			} else if len(result) > octalDigits {
				// Trim if too long
				result = result[len(result)-octalDigits:]
			}
		}
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