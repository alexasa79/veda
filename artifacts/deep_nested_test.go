package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ComplexDataProcessor is a test struct with deeply nested methods
type ComplexDataProcessor struct {
	data     []string
	config   map[string]interface{}
	counters map[string]int
}

// ProcessBatch processes a batch of items with multiple nested conditions
func (c *ComplexDataProcessor) ProcessBatch(items []string, options map[string]bool) error {
	fmt.Println("Starting batch processing...")

	// First level: validate inputs
	if len(items) == 0 {
		return fmt.Errorf("no items to process")
	}

	// Initialize processing state
	processedCount := 0
	errorCount := 0
	skipCount := 0

	// Main processing loop
	for i, item := range items {
		fmt.Printf("Processing item %d: %s\n", i, item)

		// Check if item should be skipped
		if strings.HasPrefix(item, "skip_") {
			skipCount++
			fmt.Println("  Skipping item due to prefix")
			continue
		}

		// Validate item format
		if len(item) < 3 {
			errorCount++
			fmt.Println("  Error: item too short")

			// Try to recover
			if options["recover_errors"] {
				fmt.Println("    Attempting recovery...")

				// Recovery logic
				if options["pad_short_items"] {
					item = fmt.Sprintf("pad_%s", item)
					fmt.Println("      Padded item:", item)
				} else {
					fmt.Println("      Recovery failed: padding disabled")
					continue
				}
			} else {
				fmt.Println("    Skipping due to error")
				continue
			}
		}

		// Process based on item type
		switch {
		case strings.HasPrefix(item, "data_"):
			fmt.Println("  Processing data item")

			// Extract data
			parts := strings.Split(item, "_")
			if len(parts) >= 2 {
				dataType := parts[1]

				// Handle different data types
				switch dataType {
				case "numeric":
					fmt.Println("    Numeric data detected")

					// Validate numeric data
					if len(parts) >= 3 {
						value := parts[2]

						// Process numeric value
						for j := 0; j < len(value); j++ {
							ch := value[j]

							if ch >= '0' && ch <= '9' {
								fmt.Printf("      Digit: %c\n", ch)

								// Apply transformations
								if options["double_digits"] {
									fmt.Printf("        Doubled: %c%c\n", ch, ch)

									// Check for special cases
									if ch == '7' {
										fmt.Println("          Lucky number detected!")

										// Apply lucky number logic
										if options["enhance_lucky"] {
											fmt.Println("            Enhancing lucky number...")

											for k := 0; k < 3; k++ {
												fmt.Printf("              Enhancement %d\n", k+1)
											}
										}
									}
								}
							} else {
								fmt.Printf("      Non-digit: %c\n", ch)
							}
						}
					}

				case "text":
					fmt.Println("    Text data detected")

					// Process text data
					if len(parts) >= 3 {
						text := strings.Join(parts[2:], "_")

						// Analyze text
						wordCount := len(strings.Fields(text))
						fmt.Printf("      Word count: %d\n", wordCount)

						// Process each word
						words := strings.Fields(text)
						for wi, word := range words {
							fmt.Printf("      Word %d: %s\n", wi, word)

							// Check word properties
							if len(word) > 5 {
								fmt.Println("        Long word detected")

								// Apply long word processing
								if options["capitalize_long"] {
									fmt.Printf("          Capitalized: %s\n", strings.ToUpper(word))

									// Further processing for very long words
									if len(word) > 10 {
										fmt.Println("            Very long word!")

										// Split very long words
										if options["split_very_long"] {
											mid := len(word) / 2
											fmt.Printf("              First half: %s\n", word[:mid])
											fmt.Printf("              Second half: %s\n", word[mid:])
										}
									}
								}
							}
						}
					}

				default:
					fmt.Printf("    Unknown data type: %s\n", dataType)
				}
			}

		case strings.HasPrefix(item, "cmd_"):
			fmt.Println("  Processing command item")

			// Extract command
			cmdParts := strings.Split(item, "_")
			if len(cmdParts) >= 2 {
				command := cmdParts[1]

				// Execute command
				switch command {
				case "process":
					fmt.Println("    Executing process command")

					// Process command logic
					if len(cmdParts) >= 3 {
						target := cmdParts[2]

						// Process target
						for ti := 0; ti < len(target); ti++ {
							fmt.Printf("      Processing target char %d: %c\n", ti, target[ti])

							// Apply transformations
							if target[ti] == 'x' {
								fmt.Println("        Special character 'x' found")

								// Handle special character
								if options["expand_x"] {
									fmt.Println("          Expanding 'x'...")

									// Expansion logic
									for ei := 0; ei < 3; ei++ {
										fmt.Printf("            Expansion %d\n", ei+1)

										// Nested expansion
										if ei == 1 && options["deep_expand"] {
											fmt.Println("              Deep expansion triggered")

											for di := 0; di < 2; di++ {
												fmt.Printf("                Deep level %d\n", di+1)
											}
										}
									}
								}
							}
						}
					}

				case "analyze":
					fmt.Println("    Executing analyze command")

					// Analysis logic
					if len(cmdParts) >= 3 {
						analysisType := cmdParts[2]

						switch analysisType {
						case "deep":
							fmt.Println("      Deep analysis requested")

							// Perform deep analysis
							for ai := 0; ai < 5; ai++ {
								fmt.Printf("        Analysis step %d\n", ai+1)

								// Nested analysis
								if ai%2 == 0 {
									fmt.Println("          Even step - extra processing")

									// Extra processing
									if options["thorough_analysis"] {
										fmt.Println("            Thorough mode enabled")

										for ti := 0; ti < 3; ti++ {
											fmt.Printf("              Thorough check %d\n", ti+1)
										}
									}
								}
							}

						case "quick":
							fmt.Println("      Quick analysis requested")

						default:
							fmt.Printf("      Unknown analysis type: %s\n", analysisType)
						}
					}

				default:
					fmt.Printf("    Unknown command: %s\n", command)
				}
			}

		default:
			fmt.Println("  Processing generic item")

			// Generic processing
			itemLen := len(item)
			fmt.Printf("    Item length: %d\n", itemLen)

			// Length-based processing
			if itemLen > 10 {
				fmt.Println("    Long item detected")

				// Process long items
				segments := itemLen / 5
				for si := 0; si < segments; si++ {
					start := si * 5
					end := start + 5
					if end > itemLen {
						end = itemLen
					}

					segment := item[start:end]
					fmt.Printf("      Segment %d: %s\n", si, segment)

					// Analyze segment
					if strings.Contains(segment, "test") {
						fmt.Println("        Test pattern found in segment")

						// Handle test pattern
						if options["process_tests"] {
							fmt.Println("          Processing test pattern...")

							// Test processing logic
							for pi := 0; pi < len(segment); pi++ {
								if segment[pi] == 't' {
									fmt.Printf("            Found 't' at position %d\n", pi)

									// Check for complete pattern
									if pi+4 <= len(segment) && segment[pi:pi+4] == "test" {
										fmt.Println("              Complete 'test' pattern confirmed")

										// Apply test transformations
										if options["transform_tests"] {
											fmt.Println("                Transforming test pattern...")
										}
									}
								}
							}
						}
					}
				}
			} else if itemLen > 5 {
				fmt.Println("    Medium item detected")

				// Process medium items
				middle := itemLen / 2
				firstHalf := item[:middle]
				secondHalf := item[middle:]

				fmt.Printf("      First half: %s\n", firstHalf)
				fmt.Printf("      Second half: %s\n", secondHalf)

				// Compare halves
				if firstHalf == secondHalf {
					fmt.Println("      Palindrome-like pattern detected!")

					// Handle palindrome
					if options["enhance_palindromes"] {
						fmt.Println("        Enhancing palindrome...")
					}
				}
			} else {
				fmt.Println("    Short item detected")
			}
		}

		processedCount++
		fmt.Printf("  Item %d processing complete\n", i)
	}

	// Final summary
	fmt.Println("\nProcessing Summary:")
	fmt.Printf("  Total items: %d\n", len(items))
	fmt.Printf("  Processed: %d\n", processedCount)
	fmt.Printf("  Errors: %d\n", errorCount)
	fmt.Printf("  Skipped: %d\n", skipCount)

	return nil
}

// AnalyzePatterns performs pattern analysis with deep nesting
func (c *ComplexDataProcessor) AnalyzePatterns(data [][]string, rules map[string]func(string) bool) map[string]int {
	results := make(map[string]int)

	fmt.Println("Starting pattern analysis...")

	// Iterate through data sets
	for setIdx, dataSet := range data {
		fmt.Printf("\nAnalyzing data set %d\n", setIdx)

		// Check if set is valid
		if len(dataSet) == 0 {
			fmt.Println("  Empty data set, skipping")
			continue
		}

		// Process each item in the set
		for itemIdx, item := range dataSet {
			fmt.Printf("  Item %d: %s\n", itemIdx, item)

			// Apply each rule
			for ruleName, ruleFunc := range rules {
				fmt.Printf("    Applying rule: %s\n", ruleName)

				// Check if rule applies
				if ruleFunc(item) {
					fmt.Println("      Rule matched!")

					// Update results
					results[ruleName]++

					// Apply rule-specific processing
					switch ruleName {
					case "contains_number":
						fmt.Println("        Number pattern detected")

						// Extract numbers
						for i, ch := range item {
							if ch >= '0' && ch <= '9' {
								fmt.Printf("          Digit '%c' at position %d\n", ch, i)

								// Check for sequences
								if i+1 < len(item) && item[i+1] >= '0' && item[i+1] <= '9' {
									fmt.Println("            Part of sequence")

									// Analyze sequence
									seqStart := i
									for j := i; j < len(item) && item[j] >= '0' && item[j] <= '9'; j++ {
										// Sequence processing
										if j-seqStart > 2 {
											fmt.Printf("              Long sequence detected (length: %d)\n", j-seqStart+1)

											// Special handling for long sequences
											if j-seqStart > 4 {
												fmt.Println("                Very long sequence!")
											}
										}
									}
								}
							}
						}

					case "has_uppercase":
						fmt.Println("        Uppercase pattern detected")

						// Count uppercase letters
						upperCount := 0
						for _, ch := range item {
							if ch >= 'A' && ch <= 'Z' {
								upperCount++
							}
						}

						fmt.Printf("          Uppercase count: %d\n", upperCount)

						// Analyze uppercase distribution
						if upperCount > len(item)/2 {
							fmt.Println("          Mostly uppercase")

							// Check for patterns
							if upperCount == len(item) {
								fmt.Println("            All uppercase!")

								// Special processing for all-caps
								if len(item) > 10 {
									fmt.Println("              Long all-caps string")
								}
							}
						}

					case "special_chars":
						fmt.Println("        Special characters detected")

						// Identify special characters
						for i, ch := range item {
							if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
								fmt.Printf("          Special char '%c' at position %d\n", ch, i)

								// Categorize special character
								switch ch {
								case '@', '#', '$', '%':
									fmt.Println("            Symbol character")
								case '!', '?', '.', ',':
									fmt.Println("            Punctuation character")
								case '(', ')', '[', ']', '{', '}':
									fmt.Println("            Bracket character")

									// Check for matching brackets
									if ch == '(' || ch == '[' || ch == '{' {
										fmt.Println("              Opening bracket")

										// Look for closing bracket
										var closingChar rune
										switch ch {
										case '(':
											closingChar = ')'
										case '[':
											closingChar = ']'
										case '{':
											closingChar = '}'
										}

										// Search for closing
										for j := i + 1; j < len(item); j++ {
											if rune(item[j]) == closingChar {
												fmt.Printf("                Matching closing bracket at position %d\n", j)
												break
											}
										}
									}
								default:
									fmt.Println("            Other special character")
								}
							}
						}
					}
				} else {
					fmt.Println("      Rule did not match")
				}
			}
		}
	}

	return results
}

// RecursiveProcessor demonstrates deep recursive nesting
func (c *ComplexDataProcessor) RecursiveProcessor(level int, maxLevel int, data string) string {
	indent := strings.Repeat("  ", level)
	fmt.Printf("%sLevel %d processing: %s\n", indent, level, data)

	// Base case
	if level >= maxLevel {
		fmt.Printf("%s  Reached max level\n", indent)
		return fmt.Sprintf("DONE_%s", data)
	}

	// Process current level
	result := ""

	// Different processing based on level
	switch level % 3 {
	case 0:
		fmt.Printf("%s  Mode: Transform\n", indent)

		// Transform data
		for i, ch := range data {
			if i%2 == 0 {
				result += strings.ToUpper(string(ch))
			} else {
				result += strings.ToLower(string(ch))
			}

			// Deep nesting for specific characters
			if ch == 'a' || ch == 'A' {
				fmt.Printf("%s    Special character found at position %d\n", indent, i)

				// Nested processing
				if level < maxLevel-2 {
					fmt.Printf("%s      Applying nested transformation\n", indent)

					// More nesting
					for j := 0; j < 3; j++ {
						fmt.Printf("%s        Sub-transformation %d\n", indent, j+1)

						if j == 1 {
							fmt.Printf("%s          Special case\n", indent)
						}
					}
				}
			}
		}

	case 1:
		fmt.Printf("%s  Mode: Reverse\n", indent)

		// Reverse data
		runes := []rune(data)
		for i := 0; i < len(runes)/2; i++ {
			j := len(runes) - 1 - i
			runes[i], runes[j] = runes[j], runes[i]
		}
		result = string(runes)

		// Check for palindrome
		if result == data {
			fmt.Printf("%s    Palindrome detected!\n", indent)

			// Special palindrome processing
			if len(data) > 5 {
				fmt.Printf("%s      Long palindrome\n", indent)

				// Analyze palindrome structure
				mid := len(data) / 2
				fmt.Printf("%s        Center: %s\n", indent, string(data[mid]))
			}
		}

	case 2:
		fmt.Printf("%s  Mode: Duplicate\n", indent)

		// Duplicate with modifications
		for i, ch := range data {
			result += string(ch)

			// Conditional duplication
			if i%3 == 0 {
				result += string(ch)
				fmt.Printf("%s    Duplicated character at position %d\n", indent, i)

				// Nested condition
				if ch >= '0' && ch <= '9' {
					fmt.Printf("%s      Numeric duplication\n", indent)

					// Further nesting
					if ch == '7' {
						fmt.Printf("%s        Lucky number duplication!\n", indent)
					}
				}
			}
		}
	}

	// Recursive call
	fmt.Printf("%s  Recursing to level %d\n", indent, level+1)
	processed := c.RecursiveProcessor(level+1, maxLevel, result)

	fmt.Printf("%s  Returning from level %d\n", indent, level)
	return processed
}

// Main function to demonstrate the nested structures
func main() {
	rand.Seed(time.Now().UnixNano())

	processor := &ComplexDataProcessor{
		data:     []string{},
		config:   make(map[string]interface{}),
		counters: make(map[string]int),
	}

	// Test data
	items := []string{
		"data_numeric_12345",
		"data_text_hello_world_test",
		"cmd_process_xyz",
		"cmd_analyze_deep",
		"skip_this_one",
		"data_numeric_777",
		"test_pattern_here",
		"ALLCAPS_STRING_HERE",
		"mixed_Case_String_123",
		"special!@#$%chars",
		"nested(brackets[here{deep}])",
		"short",
		"a_very_long_string_that_will_trigger_different_processing_logic",
	}

	// Options
	options := map[string]bool{
		"recover_errors":      true,
		"pad_short_items":     true,
		"double_digits":       true,
		"enhance_lucky":       true,
		"capitalize_long":     true,
		"split_very_long":     true,
		"expand_x":            true,
		"deep_expand":         true,
		"thorough_analysis":   true,
		"process_tests":       true,
		"transform_tests":     true,
		"enhance_palindromes": true,
	}

	// Process batch
	fmt.Println("=== BATCH PROCESSING ===")
	err := processor.ProcessBatch(items, options)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Pattern analysis
	fmt.Println("\n\n=== PATTERN ANALYSIS ===")
	dataSets := [][]string{
		{"abc123", "DEF456", "test@example.com"},
		{"UPPERCASE", "lowercase", "MiXeD"},
		{"special!!", "(nested)", "plain"},
	}

	rules := map[string]func(string) bool{
		"contains_number": func(s string) bool {
			for _, ch := range s {
				if ch >= '0' && ch <= '9' {
					return true
				}
			}
			return false
		},
		"has_uppercase": func(s string) bool {
			for _, ch := range s {
				if ch >= 'A' && ch <= 'Z' {
					return true
				}
			}
			return false
		},
		"special_chars": func(s string) bool {
			for _, ch := range s {
				if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
					return true
				}
			}
			return false
		},
	}

	results := processor.AnalyzePatterns(dataSets, rules)
	fmt.Println("\nPattern Results:")
	for rule, count := range results {
		fmt.Printf("  %s: %d matches\n", rule, count)
	}

	// Recursive processing
	fmt.Println("\n\n=== RECURSIVE PROCESSING ===")
	finalResult := processor.RecursiveProcessor(0, 5, "TestData123")
	fmt.Printf("\nFinal result: %s\n", finalResult)
}
