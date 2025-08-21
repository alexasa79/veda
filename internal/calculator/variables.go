package calculator

import (
	"fmt"
	"math/big"
	"sync"
)

// Variables stores calculator variables per buffer
type Variables struct {
	mu     sync.RWMutex
	values map[string]*big.Float
}

// NewVariables creates a new variable storage
func NewVariables() *Variables {
	return &Variables{
		values: make(map[string]*big.Float),
	}
}

// Set stores a value in a variable
func (v *Variables) Set(name string, value *big.Float) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Clone the value to avoid mutations
	cloned := new(big.Float).Set(value)
	v.values[name] = cloned
}

// Get retrieves a variable value
func (v *Variables) Get(name string) (*big.Float, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	val, ok := v.values[name]
	if !ok {
		return nil, false
	}

	// Return a clone to avoid mutations
	return new(big.Float).Set(val), true
}

// SetLastResult sets the special $? variable
func (v *Variables) SetLastResult(value *big.Float) {
	v.Set("?", value)
}

// GetLastResult gets the special $? variable
func (v *Variables) GetLastResult() (*big.Float, bool) {
	return v.Get("?")
}

// Clear removes all variables
func (v *Variables) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.values = make(map[string]*big.Float)
}

// GlobalVariables is a map of buffer-specific variable stores
// In a real implementation, this would be managed by the editor
var globalVariables = make(map[string]*Variables)
var globalVarMu sync.RWMutex

// GetVariablesForBuffer gets or creates variables for a specific buffer
func GetVariablesForBuffer(bufferID string) *Variables {
	globalVarMu.Lock()
	defer globalVarMu.Unlock()

	if vars, ok := globalVariables[bufferID]; ok {
		return vars
	}

	vars := NewVariables()
	globalVariables[bufferID] = vars
	return vars
}

// For backwards compatibility, use a default buffer for standalone calculator
var defaultVariables = NewVariables()

// GetDefaultVariables returns the default variable store
func GetDefaultVariables() *Variables {
	return defaultVariables
}

// ParseVariableName checks if a string is a valid variable reference
// Variables must start with $ followed by one or more characters
func ParseVariableName(s string) (string, bool) {
	if len(s) < 2 || s[0] != '$' {
		return "", false
	}

	varName := s[1:]
	if varName == "" {
		return "", false
	}

	// For now, accept any non-empty variable name
	// Special case: $? is the last result
	return varName, true
}

// IsVariableReference checks if a token is a variable reference
func IsVariableReference(token string) bool {
	_, ok := ParseVariableName(token)
	return ok
}

// EvaluateVariable evaluates a variable reference
func EvaluateVariable(varRef string, vars *Variables) (*big.Float, error) {
	varName, ok := ParseVariableName(varRef)
	if !ok {
		return nil, fmt.Errorf("invalid variable reference: %s", varRef)
	}

	value, exists := vars.Get(varName)
	if !exists {
		return nil, fmt.Errorf("undefined variable: $%s", varName)
	}

	return value, nil
}
