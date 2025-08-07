package calculator

import (
	"strings"
	"testing"
)

func TestPrecisionIntegration(t *testing.T) {
	// Test that precision works in practical scenarios
	tests := []struct {
		name        string
		line        string
		cursorPos   int
		wantContains string
		minDecimals int
	}{
		{
			name:        "Calculate pi approximation with 50 digits",
			line:        "355/113 pre50",
			cursorPos:   14,
			wantContains: "3.14159292035398230088495575221238938053097345132",
			minDecimals: 50,
		},
		{
			name:        "Calculate 1/3 with 100 digits",
			line:        "1/3 pre100",
			cursorPos:   10,
			wantContains: "0.33333333333333333333333333333333333333333333333333333333333333333333333333333",
			minDecimals: 100,
		},
		{
			name:        "Calculate golden ratio approximation",
			line:        "(1+2.2360679774997896964091736687312762/2) pre30",
			cursorPos:   50,
			wantContains: "2.11803398874989",
			minDecimals: 30,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessLineAppend(tt.line, tt.cursorPos)
			if err != nil {
				t.Errorf("ProcessLineAppend failed: %v", err)
				return
			}
			
			// Extract the calculated part
			t.Logf("Full result: %q", result)
			parts := strings.Split(result, "→")
			if len(parts) != 2 {
				t.Errorf("Result should contain arrow: %s", result)
				return
			}
			
			calculated := strings.TrimSpace(parts[1])
			t.Logf("Calculated part: %q", calculated)
			
			// Check if result contains expected value
			if !strings.HasPrefix(calculated, tt.wantContains) {
				t.Errorf("Result %q doesn't start with %q", calculated, tt.wantContains)
			}
			
			// Check decimal places
			decParts := strings.Split(calculated, ".")
			if len(decParts) == 2 {
				decimals := len(decParts[1])
				if decimals < tt.minDecimals {
					t.Errorf("Result has %d decimal places, expected at least %d", decimals, tt.minDecimals)
				}
			}
		})
	}
}