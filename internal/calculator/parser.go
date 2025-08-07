package calculator

import (
	"fmt"
	"math"
	"math/big"
)

// Operator precedence levels (higher value = higher precedence)
const (
	precAddSub = 1
	precMulDiv = 2
	precPower  = 3
	precUnary  = 4
)

// Associativity types
type Associativity int

const (
	leftAssoc Associativity = iota
	rightAssoc
)

// OperatorInfo contains information about an operator
type OperatorInfo struct {
	Precedence    int
	Associativity Associativity
}

// operatorTable maps token types to their operator information
var operatorTable = map[TokenType]OperatorInfo{
	TokenPlus:     {precAddSub, leftAssoc},
	TokenMinus:    {precAddSub, leftAssoc},
	TokenMultiply: {precMulDiv, leftAssoc},
	TokenDivide:   {precMulDiv, leftAssoc},
	TokenPower:    {precPower, rightAssoc},
}

// Parser handles parsing and evaluation of expressions using precedence climbing
type Parser struct {
	tokenizer *Tokenizer
	tokens    []Token
	tokenIdx  int
	current   Token
}

// NewParser creates a new parser for the given input
func NewParser(input string, start int) *Parser {
	p := &Parser{
		tokenizer: NewTokenizer(input, start),
	}
	p.advance() // Load the first token
	return p
}

// NewParserFromTokens creates a new parser from a list of tokens
func NewParserFromTokens(tokens []Token) *Parser {
	p := &Parser{
		tokens:   tokens,
		tokenIdx: 0,
	}
	if len(tokens) > 0 {
		p.current = tokens[0]
	} else {
		p.current = Token{Type: TokenEOF}
	}
	return p
}

// advance moves to the next token
func (p *Parser) advance() {
	if p.tokenizer != nil {
		p.current = p.tokenizer.NextToken()
	} else if p.tokens != nil {
		p.tokenIdx++
		if p.tokenIdx < len(p.tokens) {
			p.current = p.tokens[p.tokenIdx]
		} else {
			p.current = Token{Type: TokenEOF}
		}
	}
}

// Parse parses and evaluates the expression
func (p *Parser) Parse() (*big.Float, error) {
	if p.current.Type == TokenEOF {
		return nil, fmt.Errorf("empty expression")
	}
	
	result, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	
	// We should be at EOF if the entire expression was parsed
	if p.current.Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token at position %d: %s", p.current.Pos, p.current.Value)
	}
	
	return result, nil
}

// parseExpression parses an expression using precedence climbing
// minPrec is the minimum precedence level to handle at this recursion level
func (p *Parser) parseExpression(minPrec int) (*big.Float, error) {
	// Parse the left-hand side (primary expression or unary operator)
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	
	// Process binary operators
	for {
		// Check if current token is a binary operator
		opInfo, isBinaryOp := operatorTable[p.current.Type]
		if !isBinaryOp || opInfo.Precedence < minPrec {
			// Not a binary operator or precedence too low
			break
		}
		
		// Save the operator
		op := p.current.Type
		p.advance()
		
		// Check for invalid consecutive operators like ++
		if op == TokenPlus && p.current.Type == TokenPlus {
			return nil, fmt.Errorf("unexpected + after +")
		}
		if op == TokenMinus && p.current.Type == TokenPlus {
			return nil, fmt.Errorf("unexpected + after -")
		}
		
		// Calculate the minimum precedence for the right side
		nextMinPrec := opInfo.Precedence
		if opInfo.Associativity == leftAssoc {
			nextMinPrec++ // For left-associative operators, use higher precedence
		}
		
		// Parse the right-hand side
		right, err := p.parseExpression(nextMinPrec)
		if err != nil {
			return nil, err
		}
		
		// Apply the operator
		left, err = p.applyBinaryOp(op, left, right)
		if err != nil {
			return nil, err
		}
	}
	
	return left, nil
}

// parsePrimary parses a primary expression (number, parenthesized expression, or unary operator)
func (p *Parser) parsePrimary() (*big.Float, error) {
	// Handle unary minus
	if p.current.Type == TokenMinus {
		p.advance()
		// Check that there's something after the unary operator
		if p.current.Type == TokenEOF {
			return nil, fmt.Errorf("unexpected end after unary minus")
		}
		// Parse with high precedence to bind tightly
		value, err := p.parseExpression(precUnary)
		if err != nil {
			return nil, err
		}
		return new(big.Float).Neg(value), nil
	}
	
	// Handle unary plus
	if p.current.Type == TokenPlus {
		p.advance()
		// Check that there's something after the unary operator
		if p.current.Type == TokenEOF {
			return nil, fmt.Errorf("unexpected end after unary plus")
		}
		// Check for invalid consecutive operators
		if p.current.Type == TokenPlus || p.current.Type == TokenMultiply || 
		   p.current.Type == TokenDivide || p.current.Type == TokenPower {
			return nil, fmt.Errorf("unexpected operator after unary plus")
		}
		// Parse with high precedence to bind tightly
		return p.parseExpression(precUnary)
	}
	
	// Handle parentheses
	if p.current.Type == TokenLeftParen {
		p.advance()
		value, err := p.parseExpression(0) // Reset precedence inside parentheses
		if err != nil {
			return nil, err
		}
		if p.current.Type != TokenRightParen {
			return nil, fmt.Errorf("expected closing parenthesis at position %d", p.current.Pos)
		}
		p.advance()
		return value, nil
	}
	
	// Handle numbers
	if p.current.Type == TokenNumber {
		value, err := ParseNumberBig(p.current.Value)
		if err != nil {
			return nil, err
		}
		p.advance()
		return value, nil
	}
	
	// Invalid token
	if p.current.Type == TokenEOF {
		return nil, fmt.Errorf("unexpected end of expression")
	}
	return nil, fmt.Errorf("unexpected token at position %d: %s", p.current.Pos, p.current.Value)
}

// applyBinaryOp applies a binary operator to two operands
func (p *Parser) applyBinaryOp(op TokenType, left, right *big.Float) (*big.Float, error) {
	switch op {
	case TokenPlus:
		return new(big.Float).Add(left, right), nil
	case TokenMinus:
		return new(big.Float).Sub(left, right), nil
	case TokenMultiply:
		return new(big.Float).Mul(left, right), nil
	case TokenDivide:
		if right.Sign() == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		// Use high precision for division
		result := new(big.Float).SetPrec(256)
		return result.Quo(left, right), nil
	case TokenPower:
		return bigPow(left, right)
	default:
		return nil, fmt.Errorf("unknown operator")
	}
}

// bigPow computes left^right using big.Float
// This is a simple implementation that works for common cases
func bigPow(base, exp *big.Float) (*big.Float, error) {
	// Handle special cases
	if base.Sign() == 0 {
		if exp.Sign() == 0 {
			return new(big.Float).SetInt64(1), nil // 0^0 = 1 (conventional)
		}
		if exp.Sign() < 0 {
			return nil, fmt.Errorf("division by zero: 0 raised to negative power")
		}
		return new(big.Float).SetInt64(0), nil // 0^positive = 0
	}
	
	// Check if exponent is an integer
	expInt, acc := exp.Int64()
	if acc == big.Exact && expInt >= 0 && expInt < 1000 { // Reasonable limit for integer exponents
		// Integer exponent case - use simple multiplication
		result := new(big.Float).SetInt64(1)
		result.SetPrec(256)
		
		baseCopy := new(big.Float).Copy(base)
		baseCopy.SetPrec(256)
		
		for i := int64(0); i < expInt; i++ {
			result.Mul(result, baseCopy)
		}
		return result, nil
	}
	
	// For non-integer or large exponents, use exp(ln(base) * exp)
	// This requires more complex math that big.Float doesn't provide directly
	// For now, fall back to float64 for these cases
	baseF64, _ := base.Float64()
	expF64, _ := exp.Float64()
	
	if baseF64 <= 0 && expF64 != float64(int64(expF64)) {
		return nil, fmt.Errorf("complex result: negative base with non-integer exponent")
	}
	
	// Use standard library math for complex cases
	result64 := bigPowFloat64(baseF64, expF64)
	return new(big.Float).SetFloat64(result64), nil
}

// bigPowFloat64 is a helper that uses float64 math
func bigPowFloat64(base, exp float64) float64 {
	return math.Pow(base, exp)
}