package editor

import (
	"fmt"
	"sort"
	"strings"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/events"
)

// keyToString converts a Key to its string representation
func keyToString(key commands.Key) string {
	var parts []string

	// Add modifiers
	if key.Mod&events.ModCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	if key.Mod&events.ModShift != 0 {
		parts = append(parts, "shift")
	}
	if key.Mod&events.ModAlt != 0 {
		parts = append(parts, "alt")
	}

	// Add the key itself
	switch key.Key {
	case events.KeyRune:
		if key.Rune != 0 {
			// Special case for space to make it visible
			if key.Rune == ' ' {
				parts = append(parts, "space")
			} else {
				parts = append(parts, string(key.Rune))
			}
		}
	case events.KeyUp:
		parts = append(parts, "up")
	case events.KeyDown:
		parts = append(parts, "down")
	case events.KeyLeft:
		parts = append(parts, "left")
	case events.KeyRight:
		parts = append(parts, "right")
	case events.KeyHome:
		parts = append(parts, "home")
	case events.KeyEnd:
		parts = append(parts, "end")
	case events.KeyPgUp:
		parts = append(parts, "pageup")
	case events.KeyPgDn:
		parts = append(parts, "pagedown")
	case events.KeyBackspace:
		parts = append(parts, "backspace")
	case events.KeyTab:
		parts = append(parts, "tab")
	case events.KeyEnter:
		parts = append(parts, "enter")
	case events.KeyEscape:
		parts = append(parts, "escape")
	case events.KeySpace:
		parts = append(parts, "space")
	case events.KeyDelete:
		parts = append(parts, "delete")
	case events.KeyInsert:
		parts = append(parts, "insert")
	case events.KeyF1:
		parts = append(parts, "f1")
	case events.KeyF2:
		parts = append(parts, "f2")
	case events.KeyF3:
		parts = append(parts, "f3")
	case events.KeyF4:
		parts = append(parts, "f4")
	case events.KeyF5:
		parts = append(parts, "f5")
	case events.KeyF6:
		parts = append(parts, "f6")
	case events.KeyF7:
		parts = append(parts, "f7")
	case events.KeyF8:
		parts = append(parts, "f8")
	case events.KeyF9:
		parts = append(parts, "f9")
	case events.KeyF10:
		parts = append(parts, "f10")
	case events.KeyF11:
		parts = append(parts, "f11")
	case events.KeyF12:
		parts = append(parts, "f12")
	default:
		parts = append(parts, "unknown")
	}

	return strings.Join(parts, "+")
}

// keySequenceToString converts a KeySequence to its string representation
func keySequenceToString(seq commands.KeySequence) string {
	var keys []string
	for _, key := range seq {
		keys = append(keys, keyToString(key))
	}
	return strings.Join(keys, " ")
}

// modeToString converts a Mode to its string representation
func modeToString(mode commands.Mode) string {
	switch mode {
	case commands.NormalMode:
		return "Normal"
	case commands.InsertMode:
		return "Insert"
	case commands.CommandMode:
		return "Command"
	case commands.SearchMode:
		return "Search"
	default:
		return "Unknown"
	}
}

// showHelpKeys creates a buffer with all key bindings
func (e *Editor) showHelpKeys(args []string) error {
	// Get all bindings
	bindings := e.keyMapper.GetAllBindings()

	// Group bindings by mode
	bindingsByMode := make(map[commands.Mode][]commands.Binding)
	for _, binding := range bindings {
		for _, mode := range binding.Modes {
			bindingsByMode[mode] = append(bindingsByMode[mode], binding)
		}
	}

	// Sort modes for consistent output
	var modes []commands.Mode
	for mode := range bindingsByMode {
		modes = append(modes, mode)
	}
	sort.Slice(modes, func(i, j int) bool {
		return modes[i] < modes[j]
	})

	// Build help text
	var helpText strings.Builder
	helpText.WriteString("=== Veda Key Bindings ===\n\n")
	helpText.WriteString("This list is automatically generated from registered key bindings.\n\n")

	// Add bindings for each mode
	for _, mode := range modes {
		modeBindings := bindingsByMode[mode]

		// Sort bindings by key sequence for consistent output
		sort.Slice(modeBindings, func(i, j int) bool {
			return keySequenceToString(modeBindings[i].Keys) < keySequenceToString(modeBindings[j].Keys)
		})

		helpText.WriteString(fmt.Sprintf("== %s Mode ==\n", modeToString(mode)))

		// Group by command for better readability
		maxKeyWidth := 0
		for _, binding := range modeBindings {
			keyStr := keySequenceToString(binding.Keys)
			if len(keyStr) > maxKeyWidth {
				maxKeyWidth = len(keyStr)
			}
		}

		// Write bindings
		for _, binding := range modeBindings {
			keyStr := keySequenceToString(binding.Keys)
			helpText.WriteString(fmt.Sprintf("%-*s - %s\n",
				maxKeyWidth, keyStr, binding.Command))
		}
		helpText.WriteString("\n")
	}

	helpText.WriteString("== Tips ==\n")
	helpText.WriteString("- Multi-key sequences are shown with spaces (e.g., 'space s f')\n")
	helpText.WriteString("- Modifier keys are shown with + (e.g., 'ctrl+shift+space')\n")
	helpText.WriteString("- Use :help for detailed command descriptions\n")

	// Save current buffer state
	e.saveCurrentBufferState()

	// Check if help-keys buffer already exists
	helpBufferName := "*Key Bindings*"
	if buf, exists := e.buffers[helpBufferName]; exists {
		// Update existing buffer's content
		buf.Rope = buffer.NewRopeFromString(helpText.String())
		buf.CursorX = 0
		buf.ViewIterator = buffer.NewBufferIterator(buf)
		buf.CursorIterator = buffer.NewBufferIterator(buf)
		e.switchToBuffer(buf)
	} else {
		// Create new buffer
		newBuffer := buffer.NewBuffer(helpText.String(), helpBufferName, helpBufferName, e.config.DefaultIndentType, e.config.DefaultIndentWidth)
		newBuffer.IsInternal = true
		newBuffer.RealPath = ""
		e.buffers[helpBufferName] = newBuffer
		e.bufferOrder = append(e.bufferOrder, helpBufferName)
		e.switchToBuffer(newBuffer)
	}

	// Position at the top
	e.CurrentBuffer().CursorIterator = buffer.NewBufferIterator(e.CurrentBuffer())
	e.CurrentBuffer().ViewIterator = buffer.NewBufferIterator(e.CurrentBuffer())
	e.CurrentBuffer().CursorX = 0

	return nil
}
