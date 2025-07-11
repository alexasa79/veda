package commands

import (
	"strings"

	"veda/internal/ui"
)

// ParseKeySequence parses a string like "ctrl+w" or "space f" into a KeySequence
func ParseKeySequence(s string) (KeySequence, error) {
	// Split by spaces to get individual key combinations
	parts := strings.Fields(s)
	sequence := make(KeySequence, 0, len(parts))

	for _, part := range parts {
		key, err := ParseKey(part)
		if err != nil {
			return nil, err
		}
		sequence = append(sequence, key)
	}

	return sequence, nil
}

// ParseKey parses a single key combination like "ctrl+w" or "f" or "f1"
func ParseKey(s string) (Key, error) {
	var result Key

	// Split by + to handle modifiers
	parts := strings.Split(s, "+")

	// Process modifiers (convert to lowercase for comparison)
	for i := 0; i < len(parts)-1; i++ {
		switch strings.ToLower(parts[i]) {
		case "ctrl", "control":
			result.Mod |= ui.ModCtrl
		case "alt", "meta":
			result.Mod |= ui.ModAlt
		case "shift":
			result.Mod |= ui.ModShift
		}
	}

	// Process the key itself (last part)
	keyName := parts[len(parts)-1]

	// Check special key names (convert to lowercase for comparison)
	switch strings.ToLower(keyName) {
	case "space":
		result.Key = ui.KeyRune
		result.Rune = ' '
	case "tab":
		result.Key = ui.KeyTab
	case "enter", "return":
		result.Key = ui.KeyEnter
	case "escape", "esc":
		result.Key = ui.KeyEscape
	case "backspace", "bs":
		result.Key = ui.KeyBackspace2
	case "delete", "del":
		result.Key = ui.KeyDelete
	case "up":
		result.Key = ui.KeyUp
	case "down":
		result.Key = ui.KeyDown
	case "left":
		result.Key = ui.KeyLeft
	case "right":
		result.Key = ui.KeyRight
	case "home":
		result.Key = ui.KeyHome
	case "end":
		result.Key = ui.KeyEnd
	case "pageup", "pgup":
		result.Key = ui.KeyPgUp
	case "pagedown", "pgdn":
		result.Key = ui.KeyPgDn
	case "f1":
		result.Key = ui.KeyF1
	case "f2":
		result.Key = ui.KeyF2
	case "f3":
		result.Key = ui.KeyF3
	case "f4":
		result.Key = ui.KeyF4
	case "f5":
		result.Key = ui.KeyF5
	case "f6":
		result.Key = ui.KeyF6
	case "f7":
		result.Key = ui.KeyF7
	case "f8":
		result.Key = ui.KeyF8
	case "f9":
		result.Key = ui.KeyF9
	case "f10":
		result.Key = ui.KeyF10
	case "f11":
		result.Key = ui.KeyF11
	case "f12":
		result.Key = ui.KeyF12
	default:
		// Single character - preserve case
		if len(keyName) == 1 {
			result.Key = ui.KeyRune
			result.Rune = rune(keyName[0])
		} else {
			// Unknown key
			result.Key = ui.KeyRune
			result.Rune = '?'
		}
	}

	return result, nil
}