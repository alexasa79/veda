package editor

import (
    "time"
    "veda/internal/events"
)

// startAsyncOperation registers a new async operation
func (e *Editor) startAsyncOperation(id string, reason string) {
	e.asyncOperations[id] = true
	LL.Debug("Starting spinner for operation '%s': %s", id, reason)
    // Ensure spinner tick goroutine is running
    if !e.spinnerRunning {
        e.spinnerRunning = true
        go func() {
            ticker := time.NewTicker(100 * time.Millisecond)
            defer ticker.Stop()
            for {
                select {
                case <-ticker.C:
                    // poke main loop to redraw spinner
                    e.asyncEvents <- events.NewCustomEvent("spinner-tick", nil)
                case <-e.spinnerStop:
                    return
                }
            }
        }()
    }
}

// endAsyncOperation marks an async operation as complete
func (e *Editor) endAsyncOperation(id string) {
	delete(e.asyncOperations, id)
    // If no more async ops, stop spinner goroutine and wake loop to clear status
    if !e.hasAsyncOperations() && e.spinnerRunning {
        e.spinnerRunning = false
        select {
        case e.spinnerStop <- struct{}{}:
        default:
        }
        e.asyncEvents <- events.NewCustomEvent("spinner-tick", nil)
    }
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
