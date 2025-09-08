package commands

import (
	"strings"

	"veda/internal/events"
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
			result.Mod |= events.ModCtrl
		case "alt", "meta":
			result.Mod |= events.ModAlt
		case "shift":
			result.Mod |= events.ModShift
		}
	}

	// Process the key itself (last part)
	keyName := parts[len(parts)-1]

	// Check special key names (convert to lowercase for comparison)
	switch strings.ToLower(keyName) {
	case "o":
		// Check if this is Ctrl+O
		if result.Mod&events.ModCtrl != 0 {
			result.Key = events.KeyCtrlO
			// Clear Ctrl since it's part of the key
			result.Mod = result.Mod &^ events.ModCtrl
		} else {
			result.Key = events.KeyRune
			result.Rune = rune(keyName[0])
		}
	case "space":
		// Check if this is Ctrl+Space (with or without Shift)
		if result.Mod&events.ModCtrl != 0 {
			result.Key = events.KeyCtrlSpace
			// Preserve Shift modifier if present, clear Ctrl since it's part of the key
			result.Mod = result.Mod &^ events.ModCtrl
		} else {
			result.Key = events.KeyRune
			result.Rune = ' '
		}
	case "tab":
		result.Key = events.KeyTab
	case "enter", "return":
		result.Key = events.KeyEnter
	case "escape", "esc":
		result.Key = events.KeyEscape
	case "backspace", "bs":
		result.Key = events.KeyBackspace
	case "backspace2", "bs2":
		result.Key = events.KeyBackspace2
	case "delete", "del":
		result.Key = events.KeyDelete
	case "up":
		result.Key = events.KeyUp
	case "down":
		result.Key = events.KeyDown
	case "left":
		result.Key = events.KeyLeft
	case "right":
		result.Key = events.KeyRight
	case "home":
		result.Key = events.KeyHome
	case "end":
		result.Key = events.KeyEnd
	case "pageup", "pgup":
		result.Key = events.KeyPgUp
	case "pagedown", "pgdn":
		result.Key = events.KeyPgDn
	case "f1":
		result.Key = events.KeyF1
	case "f2":
		result.Key = events.KeyF2
	case "f3":
		result.Key = events.KeyF3
	case "f4":
		result.Key = events.KeyF4
	case "f5":
		result.Key = events.KeyF5
	case "f6":
		result.Key = events.KeyF6
	case "f7":
		result.Key = events.KeyF7
	case "f8":
		result.Key = events.KeyF8
	case "f9":
		result.Key = events.KeyF9
	case "f10":
		result.Key = events.KeyF10
	case "f11":
		result.Key = events.KeyF11
	case "f12":
		result.Key = events.KeyF12
	case "/", "_":
		// Many terminals send Ctrl+/ as 0x1F (Unit Separator), same as Ctrl+_
		if result.Mod&events.ModCtrl != 0 {
			result.Key = events.KeyCtrlUnderscore
			result.Mod = result.Mod &^ events.ModCtrl
			return result, nil
		}
		result.Key = events.KeyRune
		result.Rune = rune(keyName[0])
	default:
		// Single character - preserve case
		if len(keyName) == 1 {
			// If Ctrl modifier is present with a letter, map to KeyCtrl[A-Z]
			if result.Mod&events.ModCtrl != 0 {
				lower := strings.ToLower(keyName)
				ch := rune(lower[0])
				if ch >= 'a' && ch <= 'z' {
					offset := int(ch - 'a')
					result.Key = events.Key(int(events.KeyCtrlA) + offset)
					// Clear Ctrl since it is encoded in the KeyCtrl* value
					result.Mod = result.Mod &^ events.ModCtrl
					return result, nil
				}
			}
			result.Key = events.KeyRune
			result.Rune = rune(keyName[0])
		} else {
			// Unknown key
			result.Key = events.KeyRune
			result.Rune = '?'
		}
	}

	return result, nil
}
