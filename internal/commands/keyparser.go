package commands

import (
	"strings"

	"github.com/gdamore/tcell/v2"
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
			result.Mod |= tcell.ModCtrl
		case "alt", "meta":
			result.Mod |= tcell.ModAlt
		case "shift":
			result.Mod |= tcell.ModShift
		}
	}

	// Process the key itself (last part)
	keyName := parts[len(parts)-1]

	// Check special key names (convert to lowercase for comparison)
	switch strings.ToLower(keyName) {
	case "space":
		result.Key = tcell.KeyRune
		result.Rune = ' '
	case "tab":
		result.Key = tcell.KeyTab
	case "enter", "return":
		result.Key = tcell.KeyEnter
	case "escape", "esc":
		result.Key = tcell.KeyEscape
	case "backspace", "bs":
		result.Key = tcell.KeyBackspace2
	case "delete", "del":
		result.Key = tcell.KeyDelete
	case "up":
		result.Key = tcell.KeyUp
	case "down":
		result.Key = tcell.KeyDown
	case "left":
		result.Key = tcell.KeyLeft
	case "right":
		result.Key = tcell.KeyRight
	case "home":
		result.Key = tcell.KeyHome
	case "end":
		result.Key = tcell.KeyEnd
	case "pageup", "pgup":
		result.Key = tcell.KeyPgUp
	case "pagedown", "pgdn":
		result.Key = tcell.KeyPgDn
	case "f1":
		result.Key = tcell.KeyF1
	case "f2":
		result.Key = tcell.KeyF2
	case "f3":
		result.Key = tcell.KeyF3
	case "f4":
		result.Key = tcell.KeyF4
	case "f5":
		result.Key = tcell.KeyF5
	case "f6":
		result.Key = tcell.KeyF6
	case "f7":
		result.Key = tcell.KeyF7
	case "f8":
		result.Key = tcell.KeyF8
	case "f9":
		result.Key = tcell.KeyF9
	case "f10":
		result.Key = tcell.KeyF10
	case "f11":
		result.Key = tcell.KeyF11
	case "f12":
		result.Key = tcell.KeyF12
	default:
		// Single character - preserve case
		if len(keyName) == 1 {
			result.Key = tcell.KeyRune
			result.Rune = rune(keyName[0])
		} else {
			// Unknown key
			result.Key = tcell.KeyRune
			result.Rune = '?'
		}
	}

	return result, nil
}