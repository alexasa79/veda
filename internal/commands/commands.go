package commands

import (
	"fmt"
	"veda/internal/events"
)

// Mode represents an editor mode
type Mode string

const (
	NormalMode      Mode = "NORMAL"
	CommandMode     Mode = "COMMAND"
	InsertMode      Mode = "INSERT"
	SearchMode      Mode = "SEARCH"
	VisualMode      Mode = "VISUAL"
	VisualLineMode  Mode = "VISUAL LINE"
	CursorMode      Mode = "CURSOR"
)

// Key represents a key with modifiers
type Key struct {
	Key  events.Key
	Rune rune
	Mod  events.ModMask
}

// KeySequence represents a sequence of keys
type KeySequence []Key

// Binding represents a key binding with its associated command and applicable modes
type Binding struct {
	Keys    KeySequence
	Command string
	Modes   []Mode
}

// HandlerFunc is the type for command handler functions
type HandlerFunc func(args []string) error

// CommandInfo contains metadata about a command
type CommandInfo struct {
	Handler HandlerFunc
	ApplyToAllCursors bool // Whether this command should be applied to all cursors
}

// CommandRegistry manages commands and their handlers
type CommandRegistry struct {
	commands map[string]CommandInfo
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]CommandInfo),
	}
}

// Register registers a command handler
func (cr *CommandRegistry) Register(command string, handler HandlerFunc, applyToAllCursors bool) {
	cr.commands[command] = CommandInfo{
		Handler: handler,
		ApplyToAllCursors: applyToAllCursors,
	}
}

// Execute executes a command by name with arguments
func (cr *CommandRegistry) Execute(command string, args []string) error {
	info, exists := cr.commands[command]
	if !exists {
		return fmt.Errorf("unknown command: %s", command)
	}
	return info.Handler(args)
}

// Has checks if a command exists in the registry
func (cr *CommandRegistry) Has(command string) bool {
	_, exists := cr.commands[command]
	return exists
}

// GetAll returns all registered commands
func (cr *CommandRegistry) GetAll() map[string]CommandInfo {
	return cr.commands
}

// GetCommand returns a specific command info by name
func (cr *CommandRegistry) GetCommand(command string) (CommandInfo, bool) {
	info, exists := cr.commands[command]
	return info, exists
}

// GetCommandNames returns all registered command names
func (cr *CommandRegistry) GetCommandNames() []string {
	names := make([]string, 0, len(cr.commands))
	for name := range cr.commands {
		names = append(names, name)
	}
	return names
}

// KeyMapper manages key bindings
type KeyMapper struct {
	bindings []Binding
	pending  KeySequence // Keys pressed so far in a sequence
}

// NewKeyMapper creates a new key mapper
func NewKeyMapper() *KeyMapper {
	return &KeyMapper{
		bindings: []Binding{},
		pending:  KeySequence{},
	}
}

// AddBinding adds a key binding
func (km *KeyMapper) AddBinding(binding Binding) {
	km.bindings = append(km.bindings, binding)
}

// ProcessKey processes a key event and returns the command if a binding matches
func (km *KeyMapper) ProcessKey(ev *events.KeyEvent, currentMode Mode) (string, bool) {
	key := Key{
		Key:  ev.Key,
		Rune: ev.Rune,
		Mod:  ev.Modifiers,
	}

	// Add to pending sequence
	km.pending = append(km.pending, key)

	// Check for matches
	var matchFound bool
	var commandFound string
	var partialMatch bool

	for _, binding := range km.bindings {
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

		// Check if key sequence matches
		if km.sequenceMatches(km.pending, binding.Keys) {
			matchFound = true
			commandFound = binding.Command
		} else if km.isPartialMatch(km.pending, binding.Keys) {
			partialMatch = true
		}
	}

	if matchFound {
		km.pending = KeySequence{} // Reset pending
		return commandFound, true
	}

	if !partialMatch {
		km.pending = KeySequence{} // Reset pending if no partial matches
	}

	return "", false
}

// sequenceMatches checks if two key sequences match exactly
func (km *KeyMapper) sequenceMatches(seq1, seq2 KeySequence) bool {
	if len(seq1) != len(seq2) {
		return false
	}

	for i := range seq1 {
		if !km.keyEquals(seq1[i], seq2[i]) {
			return false
		}
	}

	return true
}

// isPartialMatch checks if seq1 is a prefix of seq2
func (km *KeyMapper) isPartialMatch(seq1, seq2 KeySequence) bool {
	if len(seq1) > len(seq2) {
		return false
	}

	for i := range seq1 {
		if !km.keyEquals(seq1[i], seq2[i]) {
			return false
		}
	}

	return true
}

// keyEquals checks if two keys are equal
func (km *KeyMapper) keyEquals(k1, k2 Key) bool {
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

// ResetPending resets the pending key sequence
func (km *KeyMapper) ResetPending() {
	km.pending = KeySequence{}
}

// HasPending returns true if there are pending keys in a sequence
func (km *KeyMapper) HasPending() bool {
	return len(km.pending) > 0
}

// GetAllBindings returns all registered key bindings
func (km *KeyMapper) GetAllBindings() []Binding {
	return km.bindings
}
