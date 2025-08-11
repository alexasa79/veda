package terminal

import (
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
	"veda/internal/events"
)

// keyCode represents an escape sequence mapping to a key
type keyCode struct {
	seq []byte
	key events.Key
	mod events.ModMask
}

// macOSOptionKeyMap maps macOS Option+key characters to their Alt+key equivalents
var macOSOptionKeyMap = map[rune]rune{
	// Option+letter (lowercase)
	'å': 'a', // Option+A
	'∫': 'b', // Option+B
	'ç': 'c', // Option+C
	'∂': 'd', // Option+D
	'´': 'e', // Option+E (dead key, may not work)
	'ƒ': 'f', // Option+F
	'©': 'g', // Option+G
	'˙': 'h', // Option+H
	'ˆ': 'i', // Option+I (dead key, may not work)
	'∆': 'j', // Option+J
	'˚': 'k', // Option+K
	'¬': 'l', // Option+L
	'µ': 'm', // Option+M
	'˜': 'n', // Option+N (dead key, may not work)
	'ø': 'o', // Option+O
	'π': 'p', // Option+P
	'œ': 'q', // Option+Q
	'®': 'r', // Option+R
	'ß': 's', // Option+S
	'†': 't', // Option+T
	'¨': 'u', // Option+U (dead key, may not work)
	'√': 'v', // Option+V
	'∑': 'w', // Option+W
	'≈': 'x', // Option+X
	'¥': 'y', // Option+Y
	'Ω': 'z', // Option+Z
}

// Common escape sequences (borrowing from tcell's extensive mappings)
var keyCodes = []keyCode{
	// Special keys (excluding ESC which we'll put at the end)
	{[]byte{'\x7f'}, events.KeyBackspace, events.ModNone},
	{[]byte{'\x08'}, events.KeyBackspace2, events.ModNone},
	{[]byte{'\t'}, events.KeyTab, events.ModNone},
	{[]byte{'\r'}, events.KeyEnter, events.ModNone},
	{[]byte{'\n'}, events.KeyEnter, events.ModNone},

	// Arrow keys - both normal and application mode
	{[]byte{'\x1b', '[', 'A'}, events.KeyUp, events.ModNone},
	{[]byte{'\x1b', '[', 'B'}, events.KeyDown, events.ModNone},
	{[]byte{'\x1b', '[', 'C'}, events.KeyRight, events.ModNone},
	{[]byte{'\x1b', '[', 'D'}, events.KeyLeft, events.ModNone},
	{[]byte{'\x1b', 'O', 'A'}, events.KeyUp, events.ModNone},
	{[]byte{'\x1b', 'O', 'B'}, events.KeyDown, events.ModNone},
	{[]byte{'\x1b', 'O', 'C'}, events.KeyRight, events.ModNone},
	{[]byte{'\x1b', 'O', 'D'}, events.KeyLeft, events.ModNone},

	// Home/End keys (various encodings)
	{[]byte{'\x1b', '[', 'H'}, events.KeyHome, events.ModNone},
	{[]byte{'\x1b', '[', 'F'}, events.KeyEnd, events.ModNone},
	{[]byte{'\x1b', '[', '1', '~'}, events.KeyHome, events.ModNone},
	{[]byte{'\x1b', '[', '4', '~'}, events.KeyEnd, events.ModNone},
	{[]byte{'\x1b', '[', '7', '~'}, events.KeyHome, events.ModNone},
	{[]byte{'\x1b', '[', '8', '~'}, events.KeyEnd, events.ModNone},
	{[]byte{'\x1b', 'O', 'H'}, events.KeyHome, events.ModNone},
	{[]byte{'\x1b', 'O', 'F'}, events.KeyEnd, events.ModNone},

	// Page Up/Down
	{[]byte{'\x1b', '[', '5', '~'}, events.KeyPgUp, events.ModNone},
	{[]byte{'\x1b', '[', '6', '~'}, events.KeyPgDn, events.ModNone},

	// Insert/Delete
	{[]byte{'\x1b', '[', '2', '~'}, events.KeyInsert, events.ModNone},
	{[]byte{'\x1b', '[', '3', '~'}, events.KeyDelete, events.ModNone},

	// Function keys F1-F12
	{[]byte{'\x1b', 'O', 'P'}, events.KeyF1, events.ModNone},
	{[]byte{'\x1b', 'O', 'Q'}, events.KeyF2, events.ModNone},
	{[]byte{'\x1b', 'O', 'R'}, events.KeyF3, events.ModNone},
	{[]byte{'\x1b', 'O', 'S'}, events.KeyF4, events.ModNone},
	{[]byte{'\x1b', '[', '1', '1', '~'}, events.KeyF1, events.ModNone},
	{[]byte{'\x1b', '[', '1', '2', '~'}, events.KeyF2, events.ModNone},
	{[]byte{'\x1b', '[', '1', '3', '~'}, events.KeyF3, events.ModNone},
	{[]byte{'\x1b', '[', '1', '4', '~'}, events.KeyF4, events.ModNone},
	{[]byte{'\x1b', '[', '1', '5', '~'}, events.KeyF5, events.ModNone},
	{[]byte{'\x1b', '[', '1', '7', '~'}, events.KeyF6, events.ModNone},
	{[]byte{'\x1b', '[', '1', '8', '~'}, events.KeyF7, events.ModNone},
	{[]byte{'\x1b', '[', '1', '9', '~'}, events.KeyF8, events.ModNone},
	{[]byte{'\x1b', '[', '2', '0', '~'}, events.KeyF9, events.ModNone},
	{[]byte{'\x1b', '[', '2', '1', '~'}, events.KeyF10, events.ModNone},
	{[]byte{'\x1b', '[', '2', '3', '~'}, events.KeyF11, events.ModNone},
	{[]byte{'\x1b', '[', '2', '4', '~'}, events.KeyF12, events.ModNone},

	// Control keys
	{[]byte{'\x01'}, events.KeyCtrlA, events.ModNone},
	{[]byte{'\x02'}, events.KeyCtrlB, events.ModNone},
	{[]byte{'\x03'}, events.KeyCtrlC, events.ModNone},
	{[]byte{'\x04'}, events.KeyCtrlD, events.ModNone},
	{[]byte{'\x05'}, events.KeyCtrlE, events.ModNone},
	{[]byte{'\x06'}, events.KeyCtrlF, events.ModNone},
	{[]byte{'\x07'}, events.KeyCtrlG, events.ModNone},
	{[]byte{'\x08'}, events.KeyCtrlH, events.ModNone},
	{[]byte{'\x09'}, events.KeyCtrlI, events.ModNone},
	{[]byte{'\x0a'}, events.KeyCtrlJ, events.ModNone},
	{[]byte{'\x0b'}, events.KeyCtrlK, events.ModNone},
	{[]byte{'\x0c'}, events.KeyCtrlL, events.ModNone},
	{[]byte{'\x0d'}, events.KeyCtrlM, events.ModNone},
	{[]byte{'\x0e'}, events.KeyCtrlN, events.ModNone},
	{[]byte{'\x0f'}, events.KeyCtrlO, events.ModNone},
	{[]byte{'\x10'}, events.KeyCtrlP, events.ModNone},
	{[]byte{'\x11'}, events.KeyCtrlQ, events.ModNone},
	{[]byte{'\x12'}, events.KeyCtrlR, events.ModNone},
	{[]byte{'\x13'}, events.KeyCtrlS, events.ModNone},
	{[]byte{'\x14'}, events.KeyCtrlT, events.ModNone},
	{[]byte{'\x15'}, events.KeyCtrlU, events.ModNone},
	{[]byte{'\x16'}, events.KeyCtrlV, events.ModNone},
	{[]byte{'\x17'}, events.KeyCtrlW, events.ModNone},
	{[]byte{'\x18'}, events.KeyCtrlX, events.ModNone},
	{[]byte{'\x19'}, events.KeyCtrlY, events.ModNone},
	{[]byte{'\x1a'}, events.KeyCtrlZ, events.ModNone},
	{[]byte{'\x00'}, events.KeyCtrlSpace, events.ModNone},

	// Alt+arrow keys (must be before ESC)
	// Emacs-style keybindings (ESC b for back word, ESC f for forward word)
	{[]byte{'\x1b', 'b'}, events.KeyLeft, events.ModAlt},   // Alt+Left (word backward)
	{[]byte{'\x1b', 'f'}, events.KeyRight, events.ModAlt},  // Alt+Right (word forward)
	
	// These use ESC followed by the arrow sequence
	{[]byte{'\x1b', '\x1b', '[', 'C'}, events.KeyRight, events.ModAlt},  // Alt+Right
	{[]byte{'\x1b', '\x1b', '[', 'D'}, events.KeyLeft, events.ModAlt},   // Alt+Left
	{[]byte{'\x1b', '\x1b', '[', 'A'}, events.KeyUp, events.ModAlt},     // Alt+Up
	{[]byte{'\x1b', '\x1b', '[', 'B'}, events.KeyDown, events.ModAlt},   // Alt+Down
	// Some terminals send different sequences for Alt+arrow
	{[]byte{'\x1b', '[', '1', ';', '3', 'C'}, events.KeyRight, events.ModAlt},  // Alt+Right
	{[]byte{'\x1b', '[', '1', ';', '3', 'D'}, events.KeyLeft, events.ModAlt},   // Alt+Left
	{[]byte{'\x1b', '[', '1', ';', '3', 'A'}, events.KeyUp, events.ModAlt},     // Alt+Up
	{[]byte{'\x1b', '[', '1', ';', '3', 'B'}, events.KeyDown, events.ModAlt},   // Alt+Down

	// Alt+Delete and Alt+Backspace
	{[]byte{'\x1b', 'd'}, events.KeyDelete, events.ModAlt},                     // Alt+d (Emacs-style delete word forward)
	{[]byte{'\x1b', '\x1b', '[', '3', '~'}, events.KeyDelete, events.ModAlt},   // Alt+Delete
	{[]byte{'\x1b', '[', '3', ';', '3', '~'}, events.KeyDelete, events.ModAlt}, // Alt+Delete (alternative)
	{[]byte{'\x1b', '\x7f'}, events.KeyBackspace, events.ModAlt},               // Alt+Backspace
	{[]byte{'\x1b', '\x08'}, events.KeyBackspace, events.ModAlt},               // Alt+Backspace (alternative)

	// ESC key - must be last so escape sequences are checked first
	{[]byte{'\x1b'}, events.KeyEscape, events.ModNone},
}

// inputHandler reads from stdin and parses input events
func (t *Terminal) inputHandler() {
	buf := make([]byte, 128)
	var pending []byte
	var lastReadTime time.Time

	for {
		// Use a goroutine to handle blocking read with cancellation
		readDone := make(chan struct{})
		var n int
		var err error
		
		go func() {
			n, err = os.Stdin.Read(buf)
			close(readDone)
		}()
		
		select {
		case <-t.stopChan:
			return
		case <-readDone:
			if err != nil && err != io.EOF {
				continue
			}

			if n > 0 {
				pending = append(pending, buf[:n]...)
				lastReadTime = time.Now()
				t.parseInput(&pending)
			} else if len(pending) > 0 {
				// No new data, but we have pending bytes
				// If it's been more than 50ms since last read, process what we have
				if time.Since(lastReadTime) > 50*time.Millisecond {
					t.parseInput(&pending)
				}
			}
		}
	}
}


// parseInput processes the input buffer and generates events
func (t *Terminal) parseInput(buf *[]byte) {
	for len(*buf) > 0 {
		// Special handling for ESC that might be part of Alt+key
		if len(*buf) > 0 && (*buf)[0] == '\x1b' && len(*buf) > 1 {
			// Check if ESC is followed by a letter or common Alt+key character
			// but NOT '[' or 'O' which start escape sequences
			nextChar := (*buf)[1]
			if nextChar != '[' && nextChar != 'O' && // Exclude escape sequence starters
			   ((nextChar >= 'a' && nextChar <= 'z') || 
			   (nextChar >= 'A' && nextChar <= 'Z') ||
			   (nextChar >= '0' && nextChar <= '9') ||
			   nextChar == ' ' || nextChar == '-' || nextChar == '=' ||
			   nextChar == ',' || nextChar == '.' || nextChar == '/' ||
			   nextChar == ';' || nextChar == '\'' || nextChar == '\\' ||
			   nextChar == '`' || nextChar == '\x7f') { // backspace
				// It's ESC followed by a regular character - treat as Alt+char
				r, size := decodeRune((*buf)[1:])
				if r != 0 && size > 0 {
					event := events.NewKeyEvent(events.KeyRune, r, events.ModAlt)
					t.inputChan <- event
					*buf = (*buf)[1+size:] // Skip ESC + character
					continue
				}
			}
			
			// Try to match Alt+key sequences by checking if ESC + remaining buffer matches a key
			altBuf := (*buf)[1:]
			matched, consumedBytes := t.matchKeyCodeWithAlt(&altBuf)
			if matched {
				*buf = (*buf)[1+consumedBytes:] // Skip ESC + consumed bytes
				continue
			}
		}

		// Check for bracketed paste sequences first
		pasteMatched, pasteNeedMore := t.matchBracketedPaste(buf)
		if pasteMatched {
			continue
		}
		if pasteNeedMore {
			return
		}

		// Try to match escape sequences
		matched, needMoreData := t.matchKeyCode(buf)
		if matched {
			continue
		}
		if needMoreData {
			// Partial match - wait for more data unless it's just ESC
			// The timeout in inputHandler will ensure we don't wait forever
			return
		}

		// Handle UTF-8 runes
		r, size := decodeRune(*buf)
		if r == 0 && size == 0 {
			// Need more bytes for complete rune
			return
		}

		// Check if this is a macOS Option+key character
		if mappedRune, ok := macOSOptionKeyMap[r]; ok {
			// Convert macOS Option+key character to Alt+key event
			event := events.NewKeyEvent(events.KeyRune, mappedRune, events.ModAlt)
			t.inputChan <- event
			*buf = (*buf)[size:]
			continue
		}

		// Generate key event for regular character
		event := events.NewKeyEvent(events.KeyRune, r, events.ModNone)

		// Check for Alt modifier (ESC followed by character)
		if len(*buf) > 1 && (*buf)[0] == '\x1b' {
			nextR, nextSize := decodeRune((*buf)[1:])
			if nextR != 0 && nextSize > 0 {
				event = events.NewKeyEvent(events.KeyRune, nextR, events.ModAlt)
				size = 1 + nextSize
			}
		}

		t.inputChan <- event
		*buf = (*buf)[size:]
	}
}

// matchKeyCodeWithAlt tries to match the buffer against known key codes and sends Alt version
// Returns true if matched
func (t *Terminal) matchKeyCodeWithAlt(buf *[]byte) (bool, int) {
	for _, kc := range keyCodes {
		if len(*buf) >= len(kc.seq) && bytesEqual((*buf)[:len(kc.seq)], kc.seq) {
			// Found a match - send it with Alt modifier
			var event *events.KeyEvent
			
			// Skip if it's already an Alt combination or ESC key
			if kc.mod&events.ModAlt != 0 || kc.key == events.KeyEscape {
				return false, 0
			}
			
			// Special handling for space key
			if kc.key == events.KeySpace {
				event = events.NewKeyEvent(kc.key, ' ', kc.mod|events.ModAlt)
			} else {
				event = events.NewKeyEvent(kc.key, 0, kc.mod|events.ModAlt)
			}
			
			t.inputChan <- event
			*buf = (*buf)[len(kc.seq):]
			return true, len(kc.seq)
		}
	}
	return false, 0
}

// matchKeyCode tries to match the buffer against known key codes
// Returns (matched, needMoreData)
func (t *Terminal) matchKeyCode(buf *[]byte) (bool, bool) {
	for _, kc := range keyCodes {
		if len(*buf) >= len(kc.seq) && bytesEqual((*buf)[:len(kc.seq)], kc.seq) {
			// Found a match
			var event *events.KeyEvent

			// Special handling for space key
			if kc.key == events.KeySpace {
				event = events.NewKeyEvent(kc.key, ' ', kc.mod)
			} else {
				event = events.NewKeyEvent(kc.key, 0, kc.mod)
			}

			t.inputChan <- event
			*buf = (*buf)[len(kc.seq):]
			return true, false
		}
	}

	// Check for partial matches (need more data)

	for _, kc := range keyCodes {
		if len(*buf) < len(kc.seq) && bytesEqual(*buf, kc.seq[:len(*buf)]) {
			// Partial match - wait for more data
			return false, true
		}
	}

	return false, false
}

// matchBracketedPaste handles bracketed paste sequences (\033[200~ ... \033[201~)
func (t *Terminal) matchBracketedPaste(buf *[]byte) (bool, bool) {
	// Check for paste start sequence: \033[200~
	pasteStart := []byte{'\x1b', '[', '2', '0', '0', '~'}
	pasteEnd := []byte{'\x1b', '[', '2', '0', '1', '~'}
	
	if len(*buf) >= len(pasteStart) && bytesEqual((*buf)[:len(pasteStart)], pasteStart) {
		// Found paste start - send start event and consume the sequence
		LL.Debug("BRACKETED PASTE START detected!")
		event := events.NewPasteEvent("", true)
		t.inputChan <- event
		*buf = (*buf)[len(pasteStart):]
		return true, false
	}
	
	if len(*buf) >= len(pasteEnd) && bytesEqual((*buf)[:len(pasteEnd)], pasteEnd) {
		// Found paste end - send end event and consume the sequence
		LL.Debug("BRACKETED PASTE END detected!")
		event := events.NewPasteEvent("", false)
		t.inputChan <- event
		*buf = (*buf)[len(pasteEnd):]
		return true, false
	}
	
	// Check for partial matches
	if len(*buf) < len(pasteStart) && bytesEqual(*buf, pasteStart[:len(*buf)]) {
		return false, true // Need more data for start sequence
	}
	
	if len(*buf) < len(pasteEnd) && bytesEqual(*buf, pasteEnd[:len(*buf)]) {
		return false, true // Need more data for end sequence
	}
	
	return false, false
}

// resizeHandler watches for terminal resize events
func (t *Terminal) resizeHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)

	for {
		select {
		case <-t.stopChan:
			signal.Stop(sigChan)
			return
		case <-sigChan:
			t.handleResize()
		}
	}
}

// Helper functions

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// decodeRune safely decodes a UTF-8 rune from the buffer
func decodeRune(buf []byte) (rune, int) {
	if len(buf) == 0 {
		return 0, 0
	}

	// ASCII fast path
	if buf[0] < 0x80 {
		return rune(buf[0]), 1
	}

	// Multi-byte UTF-8
	if buf[0]&0xe0 == 0xc0 && len(buf) >= 2 {
		r := rune(buf[0]&0x1f)<<6 | rune(buf[1]&0x3f)
		return r, 2
	}
	if buf[0]&0xf0 == 0xe0 && len(buf) >= 3 {
		r := rune(buf[0]&0x0f)<<12 | rune(buf[1]&0x3f)<<6 | rune(buf[2]&0x3f)
		return r, 3
	}
	if buf[0]&0xf8 == 0xf0 && len(buf) >= 4 {
		r := rune(buf[0]&0x07)<<18 | rune(buf[1]&0x3f)<<12 | rune(buf[2]&0x3f)<<6 | rune(buf[3]&0x3f)
		return r, 4
	}

	// Invalid or incomplete UTF-8
	return 0, 0
}
