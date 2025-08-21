package editor

import "veda/internal/widgets"

// RenderingMode represents the current mode of the editor's rendering system
type RenderingMode int

const (
	// ModeNormal is the default editing mode
	ModeNormal RenderingMode = iota
	// ModeOverlay is when a modal widget is displayed
	ModeOverlay
)

// RenderingState manages the editor's rendering state
type RenderingState struct {
	mode          RenderingMode
	overlay       Overlay
	overlayResult chan string
}

// NewRenderingState creates a new rendering state
func NewRenderingState() *RenderingState {
	return &RenderingState{
		mode: ModeNormal,
	}
}

// Overlay is a generic interface implemented by modal UI overlays
type Overlay interface {
	Draw(screen widgets.Screen)
	CheckContentUpdates()
	HandleEvent(event interface{}) (bool, string)
	GetCursorPosition() (int, int)
}

// SetOverlay activates overlay mode with the given overlay
func (rs *RenderingState) SetOverlay(overlay Overlay) <-chan string {
	rs.mode = ModeOverlay
	rs.overlay = overlay
	rs.overlayResult = make(chan string, 1)
	return rs.overlayResult
}

// ClearOverlay returns to normal mode
func (rs *RenderingState) ClearOverlay() {
	rs.mode = ModeNormal
	rs.overlay = nil
	if rs.overlayResult != nil {
		close(rs.overlayResult)
		rs.overlayResult = nil
	}
}

// IsOverlayActive returns true if an overlay is currently active
func (rs *RenderingState) IsOverlayActive() bool {
	return rs.mode == ModeOverlay
}

// GetMode returns the current rendering mode
func (rs *RenderingState) GetMode() RenderingMode {
	return rs.mode
}

// GetOverlay returns the current overlay
func (rs *RenderingState) GetOverlay() Overlay {
	return rs.overlay
}

// CompleteOverlay sends the result and clears the overlay
func (rs *RenderingState) CompleteOverlay(result string) {
	if rs.overlayResult != nil {
		rs.overlayResult <- result
	}
	rs.ClearOverlay()
}
