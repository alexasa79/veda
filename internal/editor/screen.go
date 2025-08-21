package editor

import (
	"veda/internal/events"
	"veda/internal/ui"
)

// Screen is the minimal interface for screen operations used by the editor
type Screen interface {
	// Initialization and lifecycle
	Init() error
	Fini()

	// Screen operations
	Clear()
	Show()

	// Screen properties
	Size() (width, height int)
	SetStyle(style ui.Style)

	// Content rendering
	SetContent(x, y int, mainc rune, combc []rune, style ui.Style)
	ShowCursor(x, y int)
	SetCursorStyle(style ui.CursorStyle)

	// Event handling
	PollEvent() events.Event
	EventChannel() <-chan events.Event // Returns the underlying event channel
}
