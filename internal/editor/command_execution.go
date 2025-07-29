package editor

import "strings"

// getCommandCompletions provides tab completion for command mode
func (e *Editor) getCommandCompletions(prefix string) []string {
	// Check if this is a variable assignment
	if strings.Contains(prefix, "=") {
		// Complete variable values
		parts := strings.SplitN(prefix, "=", 2)
		if len(parts) == 2 {
			varName := parts[0]

			switch varName {
			case "expandtab":
				values := []string{"true", "false", "yes", "no", "1", "0"}
				var completions []string
				for _, v := range values {
					completion := varName + "=" + v
					if strings.HasPrefix(completion, prefix) {
						completions = append(completions, completion)
					}
				}
				return completions
			case "tabstop", "ts", "shiftwidth", "sw":
				// Suggest common values
				values := []string{"2", "4", "8"}
				var completions []string
				for _, v := range values {
					completion := varName + "=" + v
					if strings.HasPrefix(completion, prefix) {
						completions = append(completions, completion)
					}
				}
				return completions
			case "log":
				// Suggest log levels
				values := []string{"info", "debug", "trace"}
				var completions []string
				for _, v := range values {
					completion := varName + "=" + v
					if strings.HasPrefix(completion, prefix) {
						completions = append(completions, completion)
					}
				}
				return completions
			}
		}
	}

	// Variable names for assignment
	variables := []string{
		"expandtab", "tabstop", "ts", "shiftwidth", "sw",
		"number", "relativenumber", "autoindent", "smartindent",
		"log",
	}

	// Get all registered command names
	// TODO: Add GetCommandNames method to CommandRegistry
	// commands := e.commandRegistry.GetCommandNames()

	// For now, just use variables
	allCompletions := variables

	// Filter based on prefix
	var completions []string
	for _, completion := range allCompletions {
		if strings.HasPrefix(completion, prefix) {
			completions = append(completions, completion)
		}
	}

	return completions
}