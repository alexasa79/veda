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
	TokenEOF
	TokenInvalid
)

// Token represents a single token in the expression
type Token struct {
	Type  TokenType
	Value string
	Pos   int // Position in the original string
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

	// Check for numbers
	if unicode.IsDigit(rune(ch)) || (ch == '.' && t.pos+1 < len(t.input) && unicode.IsDigit(rune(t.input[t.pos+1]))) {
		return t.scanNumber()
	}

	// Check for "pre" keyword
	if ch == 'p' && t.pos+2 < len(t.input) && t.input[t.pos:t.pos+3] == "pre" {
		// Make sure it's not part of a larger word
		if t.pos+3 >= len(t.input) || !unicode.IsLetter(rune(t.input[t.pos+3])) {
			t.pos += 3
			return Token{Type: TokenPreprocessor, Value: "pre", Pos: t.start + tokenStart}
		}
	}

	// Invalid character
	t.pos++
	return Token{Type: TokenInvalid, Value: string(ch), Pos: t.start + tokenStart}
}

// scanNumber scans a number token (decimal, hex, octal, or binary)
func (t *Tokenizer) scanNumber() Token {
	tokenStart := t.pos
	
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
			return Token{Type: TokenNumber, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
		case 'o', 'O':
			// Octal
			t.pos += 2
			if !t.scanOctalDigits() {
				return Token{Type: TokenInvalid, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
			}
			return Token{Type: TokenNumber, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
		case 'b', 'B':
			// Binary
			t.pos += 2
			if !t.scanBinaryDigits() {
				return Token{Type: TokenInvalid, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
			}
			return Token{Type: TokenNumber, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
		}
	}

	// Scan decimal number (including floating point)
	hasDigits := false
	for t.pos < len(t.input) && unicode.IsDigit(rune(t.input[t.pos])) {
		hasDigits = true
		t.pos++
	}

	// Check for decimal point
	if t.pos < len(t.input) && t.input[t.pos] == '.' {
		t.pos++
		// Scan fractional part
		for t.pos < len(t.input) && unicode.IsDigit(rune(t.input[t.pos])) {
			hasDigits = true
			t.pos++
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
		// Exponent digits
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

	return Token{Type: TokenNumber, Value: t.input[tokenStart:t.pos], Pos: t.start + tokenStart}
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
		// Decimal (including floating point)
		bf := new(big.Float).SetPrec(256) // Use high precision
		bf, _, err := bf.Parse(value, 10)
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

// TokenizeAndExtractPreprocessors tokenizes the expression and extracts preprocessor directives
// Returns the tokens without preprocessor directives and the precision value if found
func TokenizeAndExtractPreprocessors(expr string, startPos int) ([]Token, int, bool) {
	tokenizer := NewTokenizer(expr, startPos)
	var tokens []Token
	precision := -1  // Default to -1 for no precision
	hasPrecision := false
	
	for {
		token := tokenizer.NextToken()
		if token.Type == TokenEOF {
			tokens = append(tokens, token)
			break
		}
		
		if token.Type == TokenPreprocessor && token.Value == "pre" {
			// Look for the next token which should be a number
			nextToken := tokenizer.NextToken()
			if nextToken.Type == TokenNumber {
				// Parse the precision value
				precVal, err := strconv.Atoi(nextToken.Value)
				if err == nil && precVal > 0 {
					precision = precVal
					hasPrecision = true
					// Skip both the "pre" and the number tokens
					continue
				}
			}
			// If not followed by a valid number, include both tokens
			tokens = append(tokens, token)
			if nextToken.Type != TokenEOF {
				tokens = append(tokens, nextToken)
			}
		} else {
			tokens = append(tokens, token)
		}
	}
	
	return tokens, precision, hasPrecision
}