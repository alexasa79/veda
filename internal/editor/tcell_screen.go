package editor

import "github.com/gdamore/tcell/v2"

// TcellScreen wraps tcell.Screen to implement our Screen interface
type TcellScreen struct {
	tcell.Screen
}

// NewTcellScreen creates a new TcellScreen wrapping a tcell.Screen
func NewTcellScreen(s tcell.Screen) Screen {
	return &TcellScreen{Screen: s}
}

// GetTcellScreen returns the underlying tcell.Screen
func (t *TcellScreen) GetTcellScreen() tcell.Screen {
	return t.Screen
}