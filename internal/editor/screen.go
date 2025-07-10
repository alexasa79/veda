package editor

import "github.com/gdamore/tcell/v2"

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
	SetStyle(style tcell.Style)
	
	// Content rendering
	SetContent(x, y int, mainc rune, combc []rune, style tcell.Style)
	ShowCursor(x, y int)
	
	// Event handling
	PollEvent() tcell.Event
}

// TcellScreenProvider is an optional interface that allows access to the underlying tcell.Screen
type TcellScreenProvider interface {
	GetTcellScreen() tcell.Screen
}