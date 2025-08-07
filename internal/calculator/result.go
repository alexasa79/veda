package calculator

import (
	"math/big"
	"strings"
)

// CalculationResult represents a calculation result with arbitrary precision
type CalculationResult struct {
	Value     *big.Float
	Precision int // Number of decimal places to display (-1 for default)
}

// NewCalculationResult creates a new result with default precision
func NewCalculationResult(value *big.Float) *CalculationResult {
	return &CalculationResult{
		Value:     value,
		Precision: -1, // -1 means use default formatting
	}
}

// SetPrecision sets the display precision
func (r *CalculationResult) SetPrecision(precision int) {
	r.Precision = precision
}

// String formats the result according to its precision setting
func (r *CalculationResult) String() string {
	if r.Value == nil {
		return "0"
	}

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
	tokens, precision, hasPrecision := TokenizeAndExtractPreprocessors(expr, 0)
	
	// Rebuild the expression from tokens (without the pre directive)
	// The goal is to reconstruct minimal/compact form
	result := strings.Builder{}
	
	for _, token := range tokens {
		if token.Type == TokenEOF {
			break
		}
		result.WriteString(token.Value)
	}
	
	return strings.TrimSpace(result.String()), precision, hasPrecision
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isLetterOrDigit(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}