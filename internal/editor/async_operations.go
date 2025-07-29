package editor

import "time"

// startAsyncOperation registers a new async operation
func (e *Editor) startAsyncOperation(id string) {
	e.asyncOperations[id] = true
}

// endAsyncOperation marks an async operation as complete
func (e *Editor) endAsyncOperation(id string) {
	delete(e.asyncOperations, id)
}

// hasAsyncOperations returns true if any async operations are in progress
func (e *Editor) hasAsyncOperations() bool {
	return len(e.asyncOperations) > 0
}

// getSpinnerChar returns the current spinner character and advances the animation
func (e *Editor) getSpinnerChar() string {
	// Unicode spinner characters: ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	// Update spinner every 100ms
	now := time.Now()
	if now.Sub(e.lastSpinnerTime) > 100*time.Millisecond {
		e.spinnerIndex = (e.spinnerIndex + 1) % len(spinnerChars)
		e.lastSpinnerTime = now
	}

	return spinnerChars[e.spinnerIndex]
}