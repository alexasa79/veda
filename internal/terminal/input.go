package terminal

import (
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// keyCode represents an escape sequence mapping to a key
type keyCode struct {
	seq []byte
	key Key
	mod ModMask
}

// Common escape sequences (borrowing from tcell's extensive mappings)
var keyCodes = []keyCode{
	// Special keys (excluding ESC which we'll put at the end)
	{[]byte{'\x7f'}, KeyBackspace, ModNone},
	{[]byte{'\x08'}, KeyBackspace2, ModNone},
	{[]byte{'\t'}, KeyTab, ModNone},
	{[]byte{'\r'}, KeyEnter, ModNone},
	{[]byte{'\n'}, KeyEnter, ModNone},
	
	// Arrow keys - both normal and application mode
	{[]byte{'\x1b', '[', 'A'}, KeyUp, ModNone},
	{[]byte{'\x1b', '[', 'B'}, KeyDown, ModNone},
	{[]byte{'\x1b', '[', 'C'}, KeyRight, ModNone},
	{[]byte{'\x1b', '[', 'D'}, KeyLeft, ModNone},
	{[]byte{'\x1b', 'O', 'A'}, KeyUp, ModNone},
	{[]byte{'\x1b', 'O', 'B'}, KeyDown, ModNone},
	{[]byte{'\x1b', 'O', 'C'}, KeyRight, ModNone},
	{[]byte{'\x1b', 'O', 'D'}, KeyLeft, ModNone},
	
	// Home/End keys (various encodings)
	{[]byte{'\x1b', '[', 'H'}, KeyHome, ModNone},
	{[]byte{'\x1b', '[', 'F'}, KeyEnd, ModNone},
	{[]byte{'\x1b', '[', '1', '~'}, KeyHome, ModNone},
	{[]byte{'\x1b', '[', '4', '~'}, KeyEnd, ModNone},
	{[]byte{'\x1b', '[', '7', '~'}, KeyHome, ModNone},
	{[]byte{'\x1b', '[', '8', '~'}, KeyEnd, ModNone},
	{[]byte{'\x1b', 'O', 'H'}, KeyHome, ModNone},
	{[]byte{'\x1b', 'O', 'F'}, KeyEnd, ModNone},
	
	// Page Up/Down
	{[]byte{'\x1b', '[', '5', '~'}, KeyPgUp, ModNone},
	{[]byte{'\x1b', '[', '6', '~'}, KeyPgDn, ModNone},
	
	// Insert/Delete
	{[]byte{'\x1b', '[', '2', '~'}, KeyInsert, ModNone},
	{[]byte{'\x1b', '[', '3', '~'}, KeyDelete, ModNone},
	
	// Function keys F1-F12
	{[]byte{'\x1b', 'O', 'P'}, KeyF1, ModNone},
	{[]byte{'\x1b', 'O', 'Q'}, KeyF2, ModNone},
	{[]byte{'\x1b', 'O', 'R'}, KeyF3, ModNone},
	{[]byte{'\x1b', 'O', 'S'}, KeyF4, ModNone},
	{[]byte{'\x1b', '[', '1', '1', '~'}, KeyF1, ModNone},
	{[]byte{'\x1b', '[', '1', '2', '~'}, KeyF2, ModNone},
	{[]byte{'\x1b', '[', '1', '3', '~'}, KeyF3, ModNone},
	{[]byte{'\x1b', '[', '1', '4', '~'}, KeyF4, ModNone},
	{[]byte{'\x1b', '[', '1', '5', '~'}, KeyF5, ModNone},
	{[]byte{'\x1b', '[', '1', '7', '~'}, KeyF6, ModNone},
	{[]byte{'\x1b', '[', '1', '8', '~'}, KeyF7, ModNone},
	{[]byte{'\x1b', '[', '1', '9', '~'}, KeyF8, ModNone},
	{[]byte{'\x1b', '[', '2', '0', '~'}, KeyF9, ModNone},
	{[]byte{'\x1b', '[', '2', '1', '~'}, KeyF10, ModNone},
	{[]byte{'\x1b', '[', '2', '3', '~'}, KeyF11, ModNone},
	{[]byte{'\x1b', '[', '2', '4', '~'}, KeyF12, ModNone},
	
	// Control keys
	{[]byte{'\x01'}, KeyCtrlA, ModNone},
	{[]byte{'\x02'}, KeyCtrlB, ModNone},
	{[]byte{'\x03'}, KeyCtrlC, ModNone},
	{[]byte{'\x04'}, KeyCtrlD, ModNone},
	{[]byte{'\x05'}, KeyCtrlE, ModNone},
	{[]byte{'\x06'}, KeyCtrlF, ModNone},
	{[]byte{'\x07'}, KeyCtrlG, ModNone},
	{[]byte{'\x08'}, KeyCtrlH, ModNone},
	{[]byte{'\x09'}, KeyCtrlI, ModNone},
	{[]byte{'\x0a'}, KeyCtrlJ, ModNone},
	{[]byte{'\x0b'}, KeyCtrlK, ModNone},
	{[]byte{'\x0c'}, KeyCtrlL, ModNone},
	{[]byte{'\x0d'}, KeyCtrlM, ModNone},
	{[]byte{'\x0e'}, KeyCtrlN, ModNone},
	{[]byte{'\x0f'}, KeyCtrlO, ModNone},
	{[]byte{'\x10'}, KeyCtrlP, ModNone},
	{[]byte{'\x11'}, KeyCtrlQ, ModNone},
	{[]byte{'\x12'}, KeyCtrlR, ModNone},
	{[]byte{'\x13'}, KeyCtrlS, ModNone},
	{[]byte{'\x14'}, KeyCtrlT, ModNone},
	{[]byte{'\x15'}, KeyCtrlU, ModNone},
	{[]byte{'\x16'}, KeyCtrlV, ModNone},
	{[]byte{'\x17'}, KeyCtrlW, ModNone},
	{[]byte{'\x18'}, KeyCtrlX, ModNone},
	{[]byte{'\x19'}, KeyCtrlY, ModNone},
	{[]byte{'\x1a'}, KeyCtrlZ, ModNone},
	{[]byte{'\x00'}, KeyCtrlSpace, ModNone},
	
	// ESC key - must be last so escape sequences are checked first
	{[]byte{'\x1b'}, KeyEscape, ModNone},
}

// inputHandler reads from stdin and parses input events
func (t *Terminal) inputHandler() {
	buf := make([]byte, 128)
	var pending []byte
	var lastReadTime time.Time
	
	for {
		select {
		case <-t.stopChan:
			return
		default:
			// Read with timeout
			n, err := t.readWithTimeout(buf, 10*time.Millisecond)
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

// readWithTimeout reads from stdin with a timeout
func (t *Terminal) readWithTimeout(buf []byte, timeout time.Duration) (int, error) {
	// Set non-blocking mode temporarily
	if err := syscall.SetNonblock(int(os.Stdin.Fd()), true); err != nil {
		return 0, err
	}
	defer syscall.SetNonblock(int(os.Stdin.Fd()), false)
	
	// Try to read
	n, err := os.Stdin.Read(buf)
	if err == syscall.EAGAIN {
		// No data available
		time.Sleep(timeout)
		return 0, nil
	}
	
	return n, err
}

// parseInput processes the input buffer and generates events
func (t *Terminal) parseInput(buf *[]byte) {
	for len(*buf) > 0 {
		// Try to match escape sequences first
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
		
		// Generate key event for regular character
		event := &EventKey{
			timestamp: time.Now().UnixNano(),
			Key:       KeyRune,
			Rune:      r,
			Mod:       ModNone,
		}
		
		// Check for Alt modifier (ESC followed by character)
		if len(*buf) > 1 && (*buf)[0] == '\x1b' {
			nextR, nextSize := decodeRune((*buf)[1:])
			if nextR != 0 && nextSize > 0 {
				event.Rune = nextR
				event.Mod = ModAlt
				size = 1 + nextSize
			}
		}
		
		t.inputChan <- event
		*buf = (*buf)[size:]
	}
}

// matchKeyCode tries to match the buffer against known key codes
// Returns (matched, needMoreData)
func (t *Terminal) matchKeyCode(buf *[]byte) (bool, bool) {
	for _, kc := range keyCodes {
		if len(*buf) >= len(kc.seq) && bytesEqual((*buf)[:len(kc.seq)], kc.seq) {
			// Found a match
			event := &EventKey{
				timestamp: time.Now().UnixNano(),
				Key:       kc.key,
				Rune:      0,
				Mod:       kc.mod,
			}
			
			// Special handling for space key
			if kc.key == KeySpace {
				event.Rune = ' '
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