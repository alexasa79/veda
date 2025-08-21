package fuzzysearch

import (
	"strings"
)

// Match checks if a line matches the search criteria
// The criteria is split by spaces and all words must be found in the line
func Match(line string, criteria string) bool {
	if criteria == "" {
		return true
	}

	// Convert to lowercase for case-insensitive matching
	lineLower := strings.ToLower(line)
	criteriaLower := strings.ToLower(criteria)

	// Split criteria into words
	words := strings.Fields(criteriaLower)

	// Check if all words are present in the line
	for _, word := range words {
		if !strings.Contains(lineLower, word) {
			return false
		}
	}

	return true
}

// Filter filters a slice of strings based on the search criteria
func Filter(lines []string, criteria string) []string {
	if criteria == "" {
		// Return a copy of all lines
		result := make([]string, len(lines))
		copy(result, lines)
		return result
	}

	var filtered []string
	for _, line := range lines {
		if Match(line, criteria) {
			filtered = append(filtered, line)
		}
	}

	return filtered
}
