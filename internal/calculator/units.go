package calculator

import (
	"regexp"
	"strings"
)

// Unit multipliers for SI and binary prefixes
var unitMultipliers = map[string]string{
	// SI prefixes (case-insensitive, handled by regex)
	"k": "1000",
	"m": "1000000",
	"g": "1000000000",
	"b": "1000000000",
	"t": "1000000000000",
	// Binary prefixes - long form (case-insensitive)
	"kib": "1024",
	"mib": "1048576",
	"gib": "1073741824",
	"tib": "1099511627776",
	"pib": "1125899906842624",
	"eib": "1152921504606846976", // 2^60
	// Binary prefixes - short form (case-insensitive)
	"ki": "1024",
	"mi": "1048576",
	"gi": "1073741824",
	"ti": "1099511627776",
	"pi": "1125899906842624",
	"ei": "1152921504606846976", // 2^60
}

// PreprocessUnits converts unit suffixes in an expression to explicit multiplications.
// Rules:
//   - Number followed by unit: "100k" → "(100*1000)"
//   - Closing paren followed by unit: "(1+2)k" → "(1+2)*1000"
//
// This ensures that "1/1k" becomes "1/(1*1000)" = 0.001, not "(1/1)*1000" = 1000.
func PreprocessUnits(expr string) string {
	if expr == "" {
		return expr
	}

	// Build regex pattern for units (longer units first to match "kib" before "ki" before "k")
	// Order matters: check longer suffixes first
	unitOrder := []string{"eib", "pib", "tib", "gib", "mib", "kib", "ei", "pi", "ti", "gi", "mi", "ki", "k", "m", "g", "b", "t"}

	result := expr

	for _, unit := range unitOrder {
		multiplier := unitMultipliers[unit]

		// Pattern for number followed by unit (case-insensitive)
		// Matches: digits (with optional decimal point and more digits) followed by unit
		// We need to be careful not to match:
		// - Units that are part of other tokens (followed by letters)
		// - "0b" which is a binary prefix, not "0" followed by unit "b"
		// - "0x" which is a hex prefix

		// Pattern 1: Simple decimal number followed by unit
		// e.g., "100k", "3.14m", "1.5g"
		// For "b" unit, we need to exclude "0b" (binary prefix)
		var numberPattern *regexp.Regexp
		if unit == "b" {
			// Special case: don't match "0b" as it's a binary prefix
			// Match digit(s) followed by 'b', but not if the number is just "0"
			// Actually, we need to avoid matching when the preceding char makes it 0b
			// Use negative lookbehind simulation: match [1-9] before b, or match digits ending not in 0b pattern
			numberPattern = regexp.MustCompile(`(?i)([1-9]\d*\.?\d*|0\.\d+)(b)(?:[^a-z]|$)`)
		} else {
			numberPattern = regexp.MustCompile(`(?i)(\d+\.?\d*)(` + unit + `)(?:[^a-z]|$)`)
		}
		result = numberPattern.ReplaceAllStringFunc(result, func(match string) string {
			// Find where the unit starts (case-insensitive)
			lowerMatch := strings.ToLower(match)
			unitIdx := strings.LastIndex(lowerMatch, unit)
			if unitIdx == -1 {
				return match
			}
			number := match[:unitIdx]
			suffix := match[unitIdx+len(unit):]
			return "(" + number + "*" + multiplier + ")" + suffix
		})

		// Pattern 2: Closing paren followed by unit
		// e.g., "(1+2)k", "(100/2)m"
		parenPattern := regexp.MustCompile(`(?i)(\))(` + unit + `)(?:[^a-z]|$)`)
		result = parenPattern.ReplaceAllStringFunc(result, func(match string) string {
			// Find where the unit starts (case-insensitive)
			lowerMatch := strings.ToLower(match)
			unitIdx := strings.Index(lowerMatch, unit)
			if unitIdx == -1 {
				return match
			}
			before := match[:unitIdx]
			suffix := match[unitIdx+len(unit):]
			return before + "*" + multiplier + suffix
		})
	}

	return result
}
