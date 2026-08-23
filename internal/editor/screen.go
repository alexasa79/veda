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

// KittyImageScreen is implemented by screens that can place PNG images using
// the Kitty graphics protocol. Image frames are rebuilt on every editor render.
type KittyImageScreen interface {
	SupportsKittyImages() bool
	CellPixelSize() (width, height int)
	BeginKittyImageFrame()
	DrawKittyImage(key, path string, x, y, columns, rows int)
}
