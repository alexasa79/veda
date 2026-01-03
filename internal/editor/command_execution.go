package editor

import (
	"strings"
	"veda/internal/theme"
)

// getCommandCompletions provides tab completion for command mode
func (e *Editor) getCommandCompletions(prefix string) []string {
	// Check if this is a theme command with argument
	if strings.HasPrefix(prefix, "theme ") {
		themePrefix := strings.TrimPrefix(prefix, "theme ")
		var completions []string
		for _, name := range theme.GetThemeNames() {
			if strings.HasPrefix(name, themePrefix) {
				completions = append(completions, "theme "+name)
			}
		}
		return completions
	}

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
			case "indent":
				// Suggest indent formats
				values := []string{"spaces:2", "spaces:4", "spaces:8", "tabs:2", "tabs:4", "tabs:8"}
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
		"log", "indent",
	}

	// Get all registered command names
	commands := e.commandRegistry.GetCommandNames()

	// Combine commands and variables
	allCompletions := append(commands, variables...)

	// Filter based on prefix
	var completions []string
	for _, completion := range allCompletions {
		if strings.HasPrefix(completion, prefix) {
			completions = append(completions, completion)
		}
	}

	return completions
}
