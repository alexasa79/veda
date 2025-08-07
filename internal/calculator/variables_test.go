package calculator

import (
	"math/big"
	"testing"
)

func TestVariableStorage(t *testing.T) {
	vars := NewVariables()
	
	// Test setting and getting a variable
	value := new(big.Float).SetFloat64(42.5)
	vars.Set("x", value)
	
	retrieved, ok := vars.Get("x")
	if !ok {
		t.Error("Failed to retrieve variable 'x'")
	}
	
	if retrieved.Cmp(value) != 0 {
		t.Errorf("Expected %v, got %v", value, retrieved)
	}
	
	// Test getting non-existent variable
	_, ok = vars.Get("y")
	if ok {
		t.Error("Should not find non-existent variable 'y'")
	}
}

func TestLastResultVariable(t *testing.T) {
	vars := NewVariables()
	
	// Set last result
	value := new(big.Float).SetFloat64(100)
	vars.SetLastResult(value)
	
	// Get using GetLastResult
	retrieved, ok := vars.GetLastResult()
	if !ok {
		t.Error("Failed to retrieve last result")
	}
	
	if retrieved.Cmp(value) != 0 {
		t.Errorf("Expected %v, got %v", value, retrieved)
	}
	
	// Should also be accessible as "?"
	retrieved2, ok := vars.Get("?")
	if !ok {
		t.Error("Failed to retrieve '?' variable")
	}
	
	if retrieved2.Cmp(value) != 0 {
		t.Errorf("Expected %v, got %v", value, retrieved2)
	}
}

func TestVariableInExpression(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() // Setup expression to store in $?
		expr     string
		expected float64
	}{
		{
			name: "use last result",
			setup: func() {
				EvaluateExpression("10 + 5")
			},
			expr:     "$?",
			expected: 15,
		},
		{
			name: "use last result in calculation",
			setup: func() {
				EvaluateExpression("100")
			},
			expr:     "$? * 2",
			expected: 200,
		},
		{
			name: "chain calculations with $?",
			setup: func() {
				EvaluateExpression("5")
			},
			expr:     "$? + $?",
			expected: 10,
		},
		{
			name: "complex expression with $?",
			setup: func() {
				EvaluateExpression("10")
			},
			expr:     "($? + 5) * $?",
			expected: 150, // (10 + 5) * 10
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear variables before each test
			GetDefaultVariables().Clear()
			
			// Setup
			if tt.setup != nil {
				tt.setup()
			}
			
			// Evaluate expression
			result, err := EvaluateExpression(tt.expr)
			if err != nil {
				t.Errorf("Failed to evaluate %q: %v", tt.expr, err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestVariableTokenization(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
		values   []string
	}{
		{
			input:    "$?",
			expected: []TokenType{TokenVariable, TokenEOF},
			values:   []string{"$?", ""},
		},
		{
			input:    "$? + 10",
			expected: []TokenType{TokenVariable, TokenPlus, TokenNumber, TokenEOF},
			values:   []string{"$?", "+", "10", ""},
		},
		{
			input:    "5 * $?",
			expected: []TokenType{TokenNumber, TokenMultiply, TokenVariable, TokenEOF},
			values:   []string{"5", "*", "$?", ""},
		},
		{
			input:    "$var1 + $var2",
			expected: []TokenType{TokenVariable, TokenPlus, TokenVariable, TokenEOF},
			values:   []string{"$var1", "+", "$var2", ""},
		},
		{
			input:    "$",
			expected: []TokenType{TokenInvalid, TokenEOF},
			values:   []string{"$", ""},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input, 0)
			
			for i, expectedType := range tt.expected {
				token := tokenizer.NextToken()
				if token.Type != expectedType {
					t.Errorf("Token %d: expected type %v, got %v", i, expectedType, token.Type)
				}
				if token.Value != tt.values[i] {
					t.Errorf("Token %d: expected value %q, got %q", i, tt.values[i], token.Value)
				}
			}
		})
	}
}

func TestUndefinedVariable(t *testing.T) {
	// Clear variables
	GetDefaultVariables().Clear()
	
	// Try to use undefined variable
	_, err := EvaluateExpression("$undefined")
	if err == nil {
		t.Error("Expected error for undefined variable")
	}
	
	expectedError := "undefined variable: $undefined"
	if err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err.Error())
	}
}

func TestVariableWithFormatting(t *testing.T) {
	tests := []struct {
		name     string
		setup    string
		expr     string
		expected string
	}{
		{
			name:     "last result with hex format",
			setup:    "255",
			expr:     "$? hex",
			expected: "0xff",
		},
		{
			name:     "last result with precision",
			setup:    "1/3",
			expr:     "$? pre3",
			expected: "0.333",
		},
		{
			name:     "last result with comma separator",
			setup:    "1000000",
			expr:     "$? cs",
			expected: "1,000,000",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear variables
			GetDefaultVariables().Clear()
			
			// Setup
			_, err := EvaluateExpression(tt.setup)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			
			// Evaluate with formatting
			result, err := EvaluateExpressionWithResult(tt.expr)
			if err != nil {
				t.Errorf("Failed to evaluate %q: %v", tt.expr, err)
				return
			}
			
			output := result.String()
			if output != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestParseVariableName(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantOk   bool
	}{
		{"$?", "?", true},
		{"$var", "var", true},
		{"$var123", "var123", true},
		{"$_test", "_test", true},
		{"$", "", false},
		{"var", "", false},
		{"", "", false},
		{"$$", "$", true}, // Second $ is part of name
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, ok := ParseVariableName(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ParseVariableName(%q): got ok=%v, want %v", tt.input, ok, tt.wantOk)
			}
			if name != tt.wantName {
				t.Errorf("ParseVariableName(%q): got name=%q, want %q", tt.input, name, tt.wantName)
			}
		})
	}
}