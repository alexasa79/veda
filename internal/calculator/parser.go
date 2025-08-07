package calculator

import (
	"fmt"
	"math"
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

// advance moves to the next token
func (p *Parser) advance() {
	p.current = p.tokenizer.NextToken()
}

// Parse parses and evaluates the expression
func (p *Parser) Parse() (float64, error) {
	if p.current.Type == TokenEOF {
		return 0, fmt.Errorf("empty expression")
	}
	
	result, err := p.parseExpression(0)
	if err != nil {
		return 0, err
	}
	
	// We should be at EOF if the entire expression was parsed
	if p.current.Type != TokenEOF {
		return 0, fmt.Errorf("unexpected token at position %d: %s", p.current.Pos, p.current.Value)
	}
	
	return result, nil
}

// parseExpression parses an expression using precedence climbing
// minPrec is the minimum precedence level to handle at this recursion level
func (p *Parser) parseExpression(minPrec int) (float64, error) {
	// Parse the left-hand side (primary expression or unary operator)
	left, err := p.parsePrimary()
	if err != nil {
		return 0, err
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
			return 0, fmt.Errorf("unexpected + after +")
		}
		if op == TokenMinus && p.current.Type == TokenPlus {
			return 0, fmt.Errorf("unexpected + after -")
		}
		
		// Calculate the minimum precedence for the right side
		nextMinPrec := opInfo.Precedence
		if opInfo.Associativity == leftAssoc {
			nextMinPrec++ // For left-associative operators, use higher precedence
		}
		
		// Parse the right-hand side
		right, err := p.parseExpression(nextMinPrec)
		if err != nil {
			return 0, err
		}
		
		// Apply the operator
		left, err = p.applyBinaryOp(op, left, right)
		if err != nil {
			return 0, err
		}
	}
	
	return left, nil
}

// parsePrimary parses a primary expression (number, parenthesized expression, or unary operator)
func (p *Parser) parsePrimary() (float64, error) {
	// Handle unary minus
	if p.current.Type == TokenMinus {
		p.advance()
		// Check that there's something after the unary operator
		if p.current.Type == TokenEOF {
			return 0, fmt.Errorf("unexpected end after unary minus")
		}
		// Parse with high precedence to bind tightly
		value, err := p.parseExpression(precUnary)
		if err != nil {
			return 0, err
		}
		return -value, nil
	}
	
	// Handle unary plus
	if p.current.Type == TokenPlus {
		p.advance()
		// Check that there's something after the unary operator
		if p.current.Type == TokenEOF {
			return 0, fmt.Errorf("unexpected end after unary plus")
		}
		// Check for invalid consecutive operators
		if p.current.Type == TokenPlus || p.current.Type == TokenMultiply || 
		   p.current.Type == TokenDivide || p.current.Type == TokenPower {
			return 0, fmt.Errorf("unexpected operator after unary plus")
		}
		// Parse with high precedence to bind tightly
		return p.parseExpression(precUnary)
	}
	
	// Handle parentheses
	if p.current.Type == TokenLeftParen {
		p.advance()
		value, err := p.parseExpression(0) // Reset precedence inside parentheses
		if err != nil {
			return 0, err
		}
		if p.current.Type != TokenRightParen {
			return 0, fmt.Errorf("expected closing parenthesis at position %d", p.current.Pos)
		}
		p.advance()
		return value, nil
	}
	
	// Handle numbers
	if p.current.Type == TokenNumber {
		value, err := ParseNumber(p.current.Value)
		if err != nil {
			return 0, err
		}
		p.advance()
		return value, nil
	}
	
	// Invalid token
	if p.current.Type == TokenEOF {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	return 0, fmt.Errorf("unexpected token at position %d: %s", p.current.Pos, p.current.Value)
}

// applyBinaryOp applies a binary operator to two operands
func (p *Parser) applyBinaryOp(op TokenType, left, right float64) (float64, error) {
	switch op {
	case TokenPlus:
		return left + right, nil
	case TokenMinus:
		return left - right, nil
	case TokenMultiply:
		return left * right, nil
	case TokenDivide:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	case TokenPower:
		return math.Pow(left, right), nil
	default:
		return 0, fmt.Errorf("unknown operator")
	}
}