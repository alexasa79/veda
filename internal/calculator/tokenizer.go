package calculator

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenNumber TokenType = iota
	TokenPlus
	TokenMinus
	TokenMultiply
	TokenDivide
	TokenPower
	TokenLeftParen
	TokenRightParen
	TokenPreprocessor // "pre" directive
	TokenVariable     // Variable reference like $?
	TokenEOF
	TokenInvalid
)

// NumberBase represents the base of a number token
type NumberBase int

const (
	BaseDecimal NumberBase = iota
	BaseHex
	BaseBinary
	BaseOctal
	BaseDecimalComma      // Decimal with comma separators
	BaseDecimalUnderscore // Decimal with underscore separators
)

// Token represents a single token in the expression
type Token struct {
	Type  TokenType
	Value string
	Pos   int        // Position in the original string
	Base  NumberBase // For number tokens, tracks the input base
}

// Tokenizer handles tokenization of expressions
type Tokenizer struct {
	input string
	pos   int
	start int // Start position in the original string
}

// NewTokenizer creates a new tokenizer for the given input
// start is the starting position in the original string (for substring tokenization)
func NewTokenizer(input string, start int) *Tokenizer {
	return &Tokenizer{
		input: input,
		pos:   0,
		start: start,
	}
}

// NextToken returns the next token from the input
func (t *Tokenizer) NextToken() Token {
	// Skip whitespace
	for t.pos < len(t.input) && unicode.IsSpace(rune(t.input[t.pos])) {
		t.pos++
	}

	// Check for EOF
	if t.pos >= len(t.input) {
		return Token{Type: TokenEOF, Pos: t.start + t.pos}
	}

	// Save the starting position of this token
	tokenStart := t.pos

	// Check for operators and parentheses
	ch := t.input[t.pos]
	switch ch {
	case '+':
		t.pos++
		return Token{Type: TokenPlus, Value: "+", Pos: t.start + tokenStart}
	case '-':
		t.pos++
		return Token{Type: TokenMinus, Value: "-", Pos: t.start + tokenStart}
	case '*':
		// Check for ** (power operator)
		if t.pos+1 < len(t.input) && t.input[t.pos+1] == '*' {
			t.pos += 2
			return Token{Type: TokenPower, Value: "**", Pos: t.start + tokenStart}
		}
		t.pos++
		return Token{Type: TokenMultiply, Value: "*", Pos: t.start + tokenStart}
	case '/':
		t.pos++
		return Token{Type: TokenDivide, Value: "/", Pos: t.start + tokenStart}
	case '(':
		t.pos++
		return Token{Type: TokenLeftParen, Value: "(", Pos: t.start + tokenStart}
	case ')':
		t.pos++
		return Token{Type: TokenRightParen, Value: ")", Pos: t.start + tokenStart}
	}

	// Check for variable references (e.g., $?, $var)
	if ch == '$' {
		return t.scanVariable()
	}

	// Check for numbers
	if unicode.IsDigit(rune(ch)) || (ch == '.' && t.pos+1 < len(t.input) && unicode.IsDigit(rune(t.input[t.pos+1]))) {
		return t.scanNumber()
	}

	// Check for preprocessor keywords: "pre", "hex", "bin", "oct", "dec", "cs", "us"
	if ch == 'p' && t.pos+2 < len(t.input) && t.input[t.pos:t.pos+3] == "pre" {
		// Make sure it's not part of a larger word
		if t.pos+3 >= len(t.input) || !unicode.IsLetter(rune(t.input[t.pos+3])) {
			t.pos += 3
			return Token{Type: TokenPreprocessor, Value: "pre", Pos: t.start + tokenStart}
		}
	}

	if ch == 'h' && t.pos+2 < len(t.input) && t.input[t.pos:t.pos+3] == "hex" {
		// Make sure it's not part of a larger word
		if t.pos+3 >= len(t.input) || !unicode.IsLetter(rune(t.input[t.pos+3])) {
			t.pos += 3
			return Token{Type: TokenPreprocessor, Value: "hex", Pos: t.start + tokenStart}
		}
	}

	if ch == 'b' && t.pos+2 < len(t.input) && t.input[t.pos:t.pos+3] == "bin" {
		// Make sure it's not part of a larger word
		if t.pos+3 >= len(t.input) || !unicode.IsLetter(rune(t.input[t.pos+3])) {
			t.pos += 3
			return Token{Type: TokenPreprocessor, Value: "bin", Pos: t.start + tokenStart}
		}
	}

	if ch == 'o' && t.pos+2 < len(t.input) && t.input[t.pos:t.pos+3] == "oct" {
		// Make sure it's not part of a larger word
		if t.pos+3 >= len(t.input) || !unicode.IsLetter(rune(t.input[t.pos+3])) {
			t.pos += 3
			return Token{Type: TokenPreprocessor, Value: "oct", Pos: t.start + tokenStart}
		}
	}

	if ch == 'd' && t.pos+2 < len(t.input) && t.input[t.pos:t.pos+3] == "dec" {
		// Make sure it's not part of a larger word
		if t.pos+3 >= len(t.input) || !unicode.IsLetter(rune(t.input[t.pos+3])) {
			t.pos += 3
			return Token{Type: TokenPreprocessor, Value: "dec", Pos: t.start + tokenStart}
		}
	}

	if ch == 'c' && t.pos+1 < len(t.input) && t.input[t.pos:t.pos+2] == "cs" {
		// Make sure it's not part of a larger word
		if t.pos+2 >= len(t.input) || !unicode.IsLetter(rune(t.input[t.pos+2])) {
			t.pos += 2
			return Token{Type: TokenPreprocessor, Value: "cs", Pos: t.start + tokenStart}
		}
	}

	if ch == 'u' && t.pos+1 < len(t.input) && t.input[t.pos:t.pos+2] == "us" {
		// Make sure it's not part of a larger word
		if t.pos+2 >= len(t.input) || !unicode.IsLetter(rune(t.input[t.pos+2])) {
			t.pos += 2
			return Token{Type: TokenPreprocessor, Value: "us", Pos: t.start + tokenStart}
		}
	}

	// Invalid character
	t.pos++
	return Token{Type: TokenInvalid, Value: string(ch), Pos: t.start + tokenStart}
}

// scanNumber scans a number token (decimal, hex, octal, or binary)
func (t *Tokenizer) scanNumber() Token {
	tokenStart := t.pos
	base := BaseDecimal
	hasSeparator := false

	// Check for hex, octal, or binary prefix
	if t.input[t.pos] == '0' && t.pos+1 < len(t.input) {
		nextChar := t.input[t.pos+1]
		switch nextChar {
		case 'x', 'X':
			// Hexadecimal
			t.pos += 2
			if !t.scanHexDigits() {
				return Token{Type: TokenInvalid, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
			}
			return Token{Type: TokenNumber, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart, Base: BaseHex}
		case 'o', 'O':
			// Octal
			t.pos += 2
			if !t.scanOctalDigits() {
				return Token{Type: TokenInvalid, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
			}
			return Token{Type: TokenNumber, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart, Base: BaseOctal}
		case 'b', 'B':
			// Binary
			t.pos += 2
			if !t.scanBinaryDigits() {
				return Token{Type: TokenInvalid, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
			}
			return Token{Type: TokenNumber, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart, Base: BaseBinary}
		}
	}

	// Scan decimal number (including floating point) with support for separators
	hasDigits := false
	lastWasSeparator := false

	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if unicode.IsDigit(rune(ch)) {
			hasDigits = true
			lastWasSeparator = false
			t.pos++
		} else if (ch == ',' || ch == '_') && hasDigits && !lastWasSeparator {
			// Allow comma or underscore as separator, but not consecutively
			// and not at the beginning
			if t.pos+1 < len(t.input) && unicode.IsDigit(rune(t.input[t.pos+1])) {
				hasSeparator = true
				lastWasSeparator = true
				t.pos++
			} else {
				break // Separator not followed by digit
			}
		} else {
			break
		}
	}

	// Check for decimal point
	if t.pos < len(t.input) && t.input[t.pos] == '.' {
		t.pos++
		lastWasSeparator = false
		// Scan fractional part with separators
		for t.pos < len(t.input) {
			ch := t.input[t.pos]
			if unicode.IsDigit(rune(ch)) {
				hasDigits = true
				lastWasSeparator = false
				t.pos++
			} else if (ch == ',' || ch == '_') && !lastWasSeparator {
				// Allow separator in fractional part too
				if t.pos+1 < len(t.input) && unicode.IsDigit(rune(t.input[t.pos+1])) {
					hasSeparator = true
					lastWasSeparator = true
					t.pos++
				} else {
					break
				}
			} else {
				break
			}
		}
	}

	// Check for scientific notation
	if t.pos < len(t.input) && (t.input[t.pos] == 'e' || t.input[t.pos] == 'E') {
		ePosStart := t.pos
		t.pos++
		// Optional sign
		if t.pos < len(t.input) && (t.input[t.pos] == '+' || t.input[t.pos] == '-') {
			t.pos++
		}
		// Exponent digits (no separators in exponent)
		hasExpDigits := false
		for t.pos < len(t.input) && unicode.IsDigit(rune(t.input[t.pos])) {
			hasExpDigits = true
			t.pos++
		}
		if !hasExpDigits {
			// Invalid scientific notation, backtrack
			t.pos = ePosStart
		}
	}

	if !hasDigits {
		return Token{Type: TokenInvalid, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
	}

	// Determine base based on separator type
	if hasSeparator {
		// Check which separator was used
		value := t.input[tokenStart:t.pos]
		if strings.Contains(value, ",") {
			base = BaseDecimalComma
		} else if strings.Contains(value, "_") {
			base = BaseDecimalUnderscore
		}
	}

	return Token{Type: TokenNumber, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart, Base: base}
}

// scanHexDigits scans hexadecimal digits
func (t *Tokenizer) scanHexDigits() bool {
	start := t.pos
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			break
		}
		t.pos++
	}
	return t.pos > start
}

// scanOctalDigits scans octal digits
func (t *Tokenizer) scanOctalDigits() bool {
	start := t.pos
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if ch < '0' || ch > '7' {
			break
		}
		t.pos++
	}
	return t.pos > start
}

// scanBinaryDigits scans binary digits
func (t *Tokenizer) scanBinaryDigits() bool {
	start := t.pos
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if ch != '0' && ch != '1' {
			break
		}
		t.pos++
	}
	return t.pos > start
}

// scanVariable scans a variable reference starting with $
func (t *Tokenizer) scanVariable() Token {
	tokenStart := t.pos
	t.pos++ // Skip the $

	// Special case for $? (last result)
	if t.pos < len(t.input) && t.input[t.pos] == '?' {
		t.pos++
		return Token{Type: TokenVariable, Value: "$?", Pos: t.start + tokenStart}
	}

	// Scan alphanumeric variable name
	varStart := t.pos
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if !unicode.IsLetter(rune(ch)) && !unicode.IsDigit(rune(ch)) && ch != '_' {
			break
		}
		t.pos++
	}

	// Check if we got a valid variable name
	if t.pos > varStart {
		return Token{Type: TokenVariable, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
	}

	// Invalid variable reference
	return Token{Type: TokenInvalid, Value: "$", Pos: t.start + tokenStart}
}

// ParseNumberBig parses a number token value into a big.Float
func ParseNumberBig(value string) (*big.Float, error) {
	// Handle different number bases
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		// Hexadecimal
		n, err := strconv.ParseInt(value[2:], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid hexadecimal number: %s", value)
		}
		return new(big.Float).SetInt64(n), nil
	} else if strings.HasPrefix(value, "0o") || strings.HasPrefix(value, "0O") {
		// Octal
		n, err := strconv.ParseInt(value[2:], 8, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid octal number: %s", value)
		}
		return new(big.Float).SetInt64(n), nil
	} else if strings.HasPrefix(value, "0b") || strings.HasPrefix(value, "0B") {
		// Binary
		n, err := strconv.ParseInt(value[2:], 2, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid binary number: %s", value)
		}
		return new(big.Float).SetInt64(n), nil
	} else {
		// Decimal (including floating point), possibly with separators
		// Remove any comma or underscore separators
		cleanValue := strings.ReplaceAll(value, ",", "")
		cleanValue = strings.ReplaceAll(cleanValue, "_", "")

		bf := new(big.Float).SetPrec(256) // Use high precision
		bf, _, err := bf.Parse(cleanValue, 10)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", value)
		}
		return bf, nil
	}
}

// ParseNumber parses a number token value into a float64
func ParseNumber(value string) (float64, error) {
	// Handle different number bases
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		// Hexadecimal
		n, err := strconv.ParseInt(value[2:], 16, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hexadecimal number: %s", value)
		}
		return float64(n), nil
	} else if strings.HasPrefix(value, "0o") || strings.HasPrefix(value, "0O") {
		// Octal
		n, err := strconv.ParseInt(value[2:], 8, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid octal number: %s", value)
		}
		return float64(n), nil
	} else if strings.HasPrefix(value, "0b") || strings.HasPrefix(value, "0B") {
		// Binary
		n, err := strconv.ParseInt(value[2:], 2, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid binary number: %s", value)
		}
		return float64(n), nil
	} else {
		// Decimal (including floating point)
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", value)
		}
		return n, nil
	}
}

// PreprocessorDirective holds information about preprocessor directives
type PreprocessorDirective struct {
	Precision int        // For "pre" directive
	Format    FormatType // For "hex", "bin", "oct" directives
	BitWidth  int        // For "hexNN", "binNN", "octNN" directives
}

// isPowerOfTwo checks if a number is a power of 2
func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// TokenizeAndExtractPreprocessors tokenizes the expression and extracts preprocessor directives
// Returns the tokens without preprocessor directives and the directive information
func TokenizeAndExtractPreprocessors(expr string, startPos int) ([]Token, *PreprocessorDirective, bool) {
	tokenizer := NewTokenizer(expr, startPos)
	var tokens []Token
	directive := &PreprocessorDirective{
		Precision: -1,
		Format:    FormatDecimal,
		BitWidth:  0,
	}
	hasDirective := false

	// Track the bases of all numbers in the expression for auto-detection
	var numberBases []NumberBase
	hasExplicitDirective := false

	for {
		token := tokenizer.NextToken()
		if token.Type == TokenEOF {
			tokens = append(tokens, token)
			break
		}

		if token.Type == TokenNumber {
			// Track the base of this number
			numberBases = append(numberBases, token.Base)
			tokens = append(tokens, token)
		} else if token.Type == TokenPreprocessor {
			switch token.Value {
			case "pre":
				// Look for the next token which should be a number
				nextToken := tokenizer.NextToken()
				if nextToken.Type == TokenNumber {
					// Parse the precision value
					precVal, err := strconv.Atoi(nextToken.Value)
					if err == nil && precVal > 0 {
						directive.Precision = precVal
						hasDirective = true
						// Skip both tokens
						continue
					}
				}
				// If not followed by a valid number, include both tokens
				tokens = append(tokens, token)
				if nextToken.Type != TokenEOF {
					if nextToken.Type == TokenNumber {
						numberBases = append(numberBases, nextToken.Base)
					}
					tokens = append(tokens, nextToken)
				}

			case "hex", "bin", "oct":
				// Set the format
				switch token.Value {
				case "hex":
					directive.Format = FormatHex
				case "bin":
					directive.Format = FormatBinary
				case "oct":
					directive.Format = FormatOctal
				}
				hasDirective = true
				hasExplicitDirective = true

				// Check if followed by a number (bit width)
				nextToken := tokenizer.NextToken()
				if nextToken.Type == TokenNumber {
					bitWidth, err := strconv.Atoi(nextToken.Value)
					// Only consume the number if it's a valid bit width
					// This allows "hex 255" to work (255 is not consumed as bit width)
					if err == nil && isPowerOfTwo(bitWidth) && bitWidth >= 8 && bitWidth <= 128 {
						directive.BitWidth = bitWidth
						// Skip both tokens
						continue
					}
					// Not a valid bit width, put the number back for the expression
					numberBases = append(numberBases, nextToken.Base)
					tokens = append(tokens, nextToken)
				} else if nextToken.Type != TokenEOF {
					// Put the non-number token back
					tokens = append(tokens, nextToken)
				}
				// Skip only the format directive token

			case "dec":
				// Set format to decimal (explicit decimal output)
				directive.Format = FormatDecimal
				hasDirective = true
				hasExplicitDirective = true
				// "dec" doesn't support bit width, so check the next token
				// and put it back into the stream for parsing
				nextToken := tokenizer.NextToken()
				if nextToken.Type != TokenEOF {
					if nextToken.Type == TokenNumber {
						numberBases = append(numberBases, nextToken.Base)
					}
					tokens = append(tokens, nextToken)
				}
				// Skip only the "dec" directive token

			case "cs":
				// Set format to comma-separated decimal (may be overridden later)
				directive.Format = FormatComma
				hasDirective = true
				// Note: don't set hasExplicitDirective for cs/us - they can be overridden
				// Check the next token and put it back
				nextToken := tokenizer.NextToken()
				if nextToken.Type != TokenEOF {
					if nextToken.Type == TokenNumber {
						numberBases = append(numberBases, nextToken.Base)
					}
					tokens = append(tokens, nextToken)
				}
				// Skip only the "cs" directive token

			case "us":
				// Set format to underscore-separated decimal (may be overridden later)
				directive.Format = FormatUnderscore
				hasDirective = true
				// Note: don't set hasExplicitDirective for cs/us - they can be overridden
				// Check the next token and put it back
				nextToken := tokenizer.NextToken()
				if nextToken.Type != TokenEOF {
					if nextToken.Type == TokenNumber {
						numberBases = append(numberBases, nextToken.Base)
					}
					tokens = append(tokens, nextToken)
				}
				// Skip only the "us" directive token

			default:
				// Unknown preprocessor, include it
				tokens = append(tokens, token)
			}
		} else {
			tokens = append(tokens, token)
		}
	}

	// Check if we need to override cs/us directives when hex/octal/binary is present
	if (directive.Format == FormatComma || directive.Format == FormatUnderscore) && len(numberBases) > 0 {
		// Check if any number has hex/octal/binary base
		for _, base := range numberBases {
			if base == BaseHex || base == BaseOctal || base == BaseBinary {
				// Override cs/us with appropriate base format
				directive.Format = FormatDecimal // Reset to decimal first
				hasExplicitDirective = false     // Allow auto-detection to take over
				break
			}
		}
	}

	// Apply automatic base detection if no explicit directive was given
	if !hasExplicitDirective && len(numberBases) > 0 {
		// First check for hex/octal/binary - these override separator formats
		hasHexOctBin := false
		for _, base := range numberBases {
			if base == BaseHex || base == BaseOctal || base == BaseBinary {
				hasHexOctBin = true
				break
			}
		}

		// If we have hex/octal/binary, ignore separator formats and apply normal base detection
		if hasHexOctBin {
			// Collect all non-decimal bases (hex, octal, binary)
			var nonDecimalBases []NumberBase
			for _, base := range numberBases {
				if base == BaseHex || base == BaseOctal || base == BaseBinary {
					nonDecimalBases = append(nonDecimalBases, base)
				}
			}

			// If all non-decimal numbers have the same base, use that base for output
			if len(nonDecimalBases) > 0 {
				firstNonDecimal := nonDecimalBases[0]
				allSameBase := true
				for _, base := range nonDecimalBases[1:] {
					if base != firstNonDecimal {
						allSameBase = false
						break
					}
				}

				if allSameBase {
					switch firstNonDecimal {
					case BaseHex:
						directive.Format = FormatHex
						hasDirective = true
					case BaseBinary:
						directive.Format = FormatBinary
						hasDirective = true
					case BaseOctal:
						directive.Format = FormatOctal
						hasDirective = true
					}
				}
				// If mixed non-decimal bases, keep default (decimal)
			}
		} else {
			// No hex/octal/binary - check for separator formats
			var separatorBases []NumberBase
			for _, base := range numberBases {
				if base == BaseDecimalComma || base == BaseDecimalUnderscore {
					separatorBases = append(separatorBases, base)
				}
			}

			// If all separator numbers have the same format, use that format for output
			if len(separatorBases) > 0 {
				firstSeparator := separatorBases[0]
				allSameSeparator := true
				for _, base := range separatorBases[1:] {
					if base != firstSeparator {
						allSameSeparator = false
						break
					}
				}

				if allSameSeparator {
					switch firstSeparator {
					case BaseDecimalComma:
						directive.Format = FormatComma
						hasDirective = true
					case BaseDecimalUnderscore:
						directive.Format = FormatUnderscore
						hasDirective = true
					}
				}
				// If mixed separator types, keep default (plain decimal)
			}
		}
	}

	return tokens, directive, hasDirective
}
