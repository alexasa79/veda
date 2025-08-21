package widgets

import (
	"fmt"
	"sort"
	"strings"
	"veda/internal/commands"
	"veda/internal/events"
	"veda/internal/theme"
)

// KeyHint represents a single key hint with its description
type KeyHint struct {
	Key         string
	Description string
}

// KeyHintsPopup represents a popup window showing available key combinations
type KeyHintsPopup struct {
	x, y          int
	width, height int
	theme         *theme.Theme
	hints         []KeyHint
	visible       bool
}

// NewKeyHintsPopup creates a new key hints popup
func NewKeyHintsPopup(theme *theme.Theme) *KeyHintsPopup {
	return &KeyHintsPopup{
		theme:   theme,
		hints:   []KeyHint{},
		visible: false,
	}
}

// SetPosition sets the position of the popup
func (khp *KeyHintsPopup) SetPosition(x, y int) {
	khp.x = x
	khp.y = y
}

// SetHints sets the key hints to display
func (khp *KeyHintsPopup) SetHints(hints []KeyHint) {
	khp.hints = make([]KeyHint, len(hints))
	copy(khp.hints, hints)

	// Calculate optimal size based on content
	khp.calculateSize()
}

// calculateSize calculates the optimal size for the popup based on content
func (khp *KeyHintsPopup) calculateSize() {
	if len(khp.hints) == 0 {
		khp.width = 20
		khp.height = 3
		return
	}

	// Calculate width: longest hint + padding + borders
	maxWidth := 0
	for _, hint := range khp.hints {
		hintWidth := len(hint.Key) + 3 + len(hint.Description) // key + " - " + description
		if hintWidth > maxWidth {
			maxWidth = hintWidth
		}
	}

	// Add borders and padding
	khp.width = maxWidth + 4 // 2 for borders + 2 for padding
	if khp.width < 20 {
		khp.width = 20
	}
	if khp.width > 50 {
		khp.width = 50 // Cap maximum width
	}

	// Height: number of hints + borders
	khp.height = len(khp.hints) + 2 // +2 for top and bottom borders
	// Increase cap to allow more entries before truncation
	if khp.height > 30 {
		khp.height = 30 // Cap maximum height
	}
}

// Show makes the popup visible
func (khp *KeyHintsPopup) Show() {
	khp.visible = true
}

// Hide makes the popup invisible
func (khp *KeyHintsPopup) Hide() {
	khp.visible = false
}

// IsVisible returns whether the popup is currently visible
func (khp *KeyHintsPopup) IsVisible() bool {
	return khp.visible
}

// ShouldAutoHide returns true if the popup should automatically hide due to timeout
func (khp *KeyHintsPopup) ShouldAutoHide() bool {
	// Never auto-hide - let the user dismiss manually or by completing commands
	return false
}

// Draw renders the key hints popup on the screen
func (khp *KeyHintsPopup) Draw(screen Screen) {
	if !khp.visible || len(khp.hints) == 0 {
		return
	}

	// Get screen dimensions to ensure popup fits
	screenWidth, screenHeight := screen.Size()

	// Adjust position if popup would go off screen
	adjustedX := khp.x
	adjustedY := khp.y

	if adjustedX+khp.width > screenWidth {
		adjustedX = screenWidth - khp.width
	}
	if adjustedX < 0 {
		adjustedX = 0
	}

	if adjustedY+khp.height > screenHeight {
		adjustedY = screenHeight - khp.height
	}
	if adjustedY < 0 {
		adjustedY = 0
	}

	// Draw popup background and borders
	khp.drawBackground(screen, adjustedX, adjustedY)

	// Draw hints
	khp.drawHints(screen, adjustedX, adjustedY)
}

// drawBackground draws the popup background and borders
func (khp *KeyHintsPopup) drawBackground(screen Screen, x, y int) {
	borderStyle := khp.theme.GetStyle(khp.theme.FuzzyListBorder)
	contentStyle := khp.theme.GetStyle(khp.theme.FuzzyListContent)

	// Draw top border with rounded corners
	screen.SetContent(x, y, '╭', nil, borderStyle)
	for i := 1; i < khp.width-1; i++ {
		screen.SetContent(x+i, y, '─', nil, borderStyle)
	}
	screen.SetContent(x+khp.width-1, y, '╮', nil, borderStyle)

	// Draw content area with side borders
	for row := 1; row < khp.height-1; row++ {
		// Left border
		screen.SetContent(x, y+row, '│', nil, borderStyle)

		// Content area
		for col := 1; col < khp.width-1; col++ {
			screen.SetContent(x+col, y+row, ' ', nil, contentStyle)
		}

		// Right border
		screen.SetContent(x+khp.width-1, y+row, '│', nil, borderStyle)
	}

	// Draw bottom border with rounded corners
	screen.SetContent(x, y+khp.height-1, '╰', nil, borderStyle)
	for i := 1; i < khp.width-1; i++ {
		screen.SetContent(x+i, y+khp.height-1, '─', nil, borderStyle)
	}
	screen.SetContent(x+khp.width-1, y+khp.height-1, '╯', nil, borderStyle)
}

// drawHints draws the key hints content
func (khp *KeyHintsPopup) drawHints(screen Screen, x, y int) {
	contentStyle := khp.theme.GetStyle(khp.theme.FuzzyListContent)
	contentWidth := khp.width - 2 // Exclude borders

	// Limit to visible hints
	maxHints := khp.height - 2 // Exclude borders
	visibleHints := khp.hints
	if len(visibleHints) > maxHints {
		visibleHints = khp.hints[:maxHints]
	}

	for i, hint := range visibleHints {
		row := y + 1 + i
		col := x + 1

		// Format hint as "key - description"
		hintText := fmt.Sprintf("%s - %s", hint.Key, hint.Description)

		// Truncate if too long
		runes := []rune(hintText)
		if len(runes) > contentWidth {
			runes = runes[:contentWidth]
		}

		// Draw hint text
		for j, r := range runes {
			if j < contentWidth {
				screen.SetContent(col+j, row, r, nil, contentStyle)
			}
		}
	}
}

// GetAvailableKeyHints returns available key hints for the given partial key sequence
func GetAvailableKeyHints(keyMapper *commands.KeyMapper, partialSequence commands.KeySequence, currentMode commands.Mode, commandDescriptions map[string]string, categoryDescriptions map[string]string) []KeyHint {
	var hints []KeyHint
	seen := make(map[string]bool)

	// No hints for empty sequence
	if len(partialSequence) == 0 {
		return hints
	}

	// Get all bindings
	bindings := keyMapper.GetAllBindings()

	for _, binding := range bindings {
		// Check if mode matches
		modeMatches := false
		for _, mode := range binding.Modes {
			if mode == currentMode {
				modeMatches = true
				break
			}
		}
		if !modeMatches {
			continue
		}

		// Check if this binding starts with our partial sequence
		if len(binding.Keys) <= len(partialSequence) {
			continue
		}

		// Check if partial sequence matches the beginning of this binding
		matches := true
		for i, key := range partialSequence {
			if !keyEquals(key, binding.Keys[i]) {
				matches = false
				break
			}
		}

		if matches {
			// Get the next key in the sequence
			nextKey := binding.Keys[len(partialSequence)]
			nextKeyStr := formatKey(nextKey)

			// Avoid duplicates
			if seen[nextKeyStr] {
				continue
			}
			seen[nextKeyStr] = true

			// Get description for the command
			description := binding.Command
			if desc, exists := commandDescriptions[binding.Command]; exists {
				description = desc
			}

			hints = append(hints, KeyHint{
				Key:         nextKeyStr,
				Description: description,
			})
		}
	}

	// Apply category descriptions where available
	for i, hint := range hints {
		partialWithNext := make(commands.KeySequence, len(partialSequence)+1)
		copy(partialWithNext, partialSequence)

		// Parse the hint key back to a Key object
		hintKey, err := parseKeyFromString(hint.Key)
		if err == nil {
			partialWithNext[len(partialSequence)] = hintKey
			categoryKey := formatKeySequence(partialWithNext)
			if categoryDesc, exists := categoryDescriptions[categoryKey]; exists {
				hints[i].Description = categoryDesc
			}
		}
	}

	// Sort hints by key for consistent display
	sort.Slice(hints, func(i, j int) bool {
		return hints[i].Key < hints[j].Key
	})

	return hints
}

// formatKey formats a key for display
func formatKey(key commands.Key) string {
	var parts []string

	// Add modifiers
	if key.Mod&events.ModAlt != 0 {
		parts = append(parts, "alt")
	}
	if key.Mod&events.ModCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	if key.Mod&events.ModShift != 0 {
		parts = append(parts, "shift")
	}

	// Add key name
	if key.Key == events.KeyRune {
		if key.Rune == ' ' {
			parts = append(parts, "space")
		} else {
			parts = append(parts, string(key.Rune))
		}
	} else {
		parts = append(parts, formatSpecialKey(key.Key))
	}

	return strings.Join(parts, "+")
}

// formatSpecialKey formats special keys for display
func formatSpecialKey(key events.Key) string {
	switch key {
	case events.KeyUp:
		return "up"
	case events.KeyDown:
		return "down"
	case events.KeyLeft:
		return "left"
	case events.KeyRight:
		return "right"
	case events.KeyEnter:
		return "enter"
	case events.KeyEscape:
		return "escape"
	case events.KeyBackspace:
		return "backspace"
	case events.KeyDelete:
		return "delete"
	case events.KeyTab:
		return "tab"
	case events.KeySpace:
		return "space"
	case events.KeyHome:
		return "home"
	case events.KeyEnd:
		return "end"
	case events.KeyPgUp:
		return "pageup"
	case events.KeyPgDn:
		return "pagedown"
	default:
		return "unknown"
	}
}

// parseKeyFromString parses a formatted key string back to a Key object
// This is a reverse of formatKey function
func parseKeyFromString(keyStr string) (commands.Key, error) {
	// Handle simple rune keys
	if len(keyStr) == 1 {
		return commands.Key{Key: events.KeyRune, Rune: rune(keyStr[0])}, nil
	}

	// Handle special keys
	switch keyStr {
	case "space":
		return commands.Key{Key: events.KeyRune, Rune: ' '}, nil
	case "left":
		return commands.Key{Key: events.KeyLeft}, nil
	case "right":
		return commands.Key{Key: events.KeyRight}, nil
	case "up":
		return commands.Key{Key: events.KeyUp}, nil
	case "down":
		return commands.Key{Key: events.KeyDown}, nil
	case "enter":
		return commands.Key{Key: events.KeyEnter}, nil
	case "escape":
		return commands.Key{Key: events.KeyEscape}, nil
	case "tab":
		return commands.Key{Key: events.KeyTab}, nil
	case "backspace":
		return commands.Key{Key: events.KeyBackspace}, nil
	case "delete":
		return commands.Key{Key: events.KeyDelete}, nil
	case "home":
		return commands.Key{Key: events.KeyHome}, nil
	case "end":
		return commands.Key{Key: events.KeyEnd}, nil
	case "pageup":
		return commands.Key{Key: events.KeyPgUp}, nil
	case "pagedown":
		return commands.Key{Key: events.KeyPgDn}, nil
	default:
		// For unsupported keys, return an error
		return commands.Key{}, fmt.Errorf("unsupported key string: %s", keyStr)
	}
}

// formatKeySequence formats a key sequence for use as a map key
func formatKeySequence(seq commands.KeySequence) string {
	var parts []string
	for _, key := range seq {
		parts = append(parts, formatKey(key))
	}
	return strings.Join(parts, " ")
}

// keyEquals checks if two keys are equal (copied from commands package)
func keyEquals(k1, k2 commands.Key) bool {
	if k1.Mod != k2.Mod {
		return false
	}

	if k1.Key != events.KeyRune && k2.Key != events.KeyRune {
		return k1.Key == k2.Key
	}

	if k1.Key == events.KeyRune && k2.Key == events.KeyRune {
		return k1.Rune == k2.Rune
	}

	return false
}
