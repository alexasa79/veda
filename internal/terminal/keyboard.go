package terminal

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
	"veda/internal/events"
)

type keyboardProtocolState int

const (
	keyboardProtocolDisabled keyboardProtocolState = iota
	keyboardProtocolQuerying
	keyboardProtocolEnabled
)

// Called with mu held, after entering the alternate screen. The device
// attributes reply marks the end of detection on terminals without support.
func (t *Terminal) queryKeyboardProtocol() {
	t.keyboardProtocol = keyboardProtocolQuerying
	if _, err := fmt.Fprint(t.tty, "\033[?u\033[c"); err != nil {
		t.keyboardProtocol = keyboardProtocolDisabled
	}
}

// Called with mu held, before leaving the alternate screen.
func (t *Terminal) restoreKeyboardProtocol() {
	if t.keyboardProtocol == keyboardProtocolEnabled {
		fmt.Fprint(t.tty, "\033[<u")
	}
	t.keyboardProtocol = keyboardProtocolDisabled
}

func (t *Terminal) matchKeyboardResponse(buf *[]byte) (bool, bool) {
	params, final, size, needMore := readCSI(*buf)
	if size == 0 || !strings.HasPrefix(params, "?") || (final != 'u' && final != 'c') {
		return false, needMore
	}
	fields := strings.Split(params[1:], ";")
	for _, field := range fields {
		// Device attributes allow omitted parameters, including the trailing
		// empty parameter in Kitty's reply. Keyboard flags must be explicit.
		if field == "" && final == 'c' {
			continue
		}
		if n, err := strconv.Atoi(field); err != nil || n < 0 {
			return false, false
		}
	}
	if final == 'u' && len(fields) != 1 {
		return false, false
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.keyboardProtocol == keyboardProtocolQuerying {
		if final == 'u' {
			// Push exactly flag 1; repeat/release and text reporting stay off.
			if _, err := fmt.Fprint(t.tty, "\033[>1u"); err == nil {
				t.keyboardProtocol = keyboardProtocolEnabled
			}
		} else {
			t.keyboardProtocol = keyboardProtocolDisabled
		}
	}
	*buf = (*buf)[size:]
	return true, false
}

// readCSI leaves incomplete sequences buffered and stops at the first final
// byte, so a terminal reply and the following key may arrive in one read.
func readCSI(data []byte) (params string, final byte, size int, needMore bool) {
	if len(data) < 2 || data[0] != '\x1b' || data[1] != '[' {
		return "", 0, 0, false
	}
	for i := 2; i < len(data); i++ {
		if data[i] >= 0x40 && data[i] <= 0x7e {
			return string(data[2:i]), data[i], i + 1, false
		}
		if data[i] < 0x20 || data[i] > 0x3f {
			return "", 0, 0, false
		}
	}
	return "", 0, 0, true
}

// Decode the three key encodings used by Kitty's disambiguation mode:
// CSI code[;mod]u, CSI code[;mod]~, and CSI [1;mod]letter.
func (t *Terminal) matchCSIKeySequence(buf *[]byte) (bool, bool) {
	params, final, size, needMore := readCSI(*buf)
	if size == 0 {
		return false, needMore
	}
	if !strings.ContainsRune("u~ABCDEFHPQS", rune(final)) {
		return false, false
	}
	fields := strings.Split(params, ";")
	if len(fields) > 2 {
		return false, false
	}
	code := 1
	var err error
	if fields[0] != "" {
		code, err = strconv.Atoi(fields[0])
		if err != nil || code < 0 {
			return false, false
		}
	}
	modCode := 1
	if len(fields) == 2 && fields[1] != "" {
		modCode, err = strconv.Atoi(fields[1])
		if err != nil || modCode < 1 {
			return false, false
		}
	}
	flags := modCode - 1
	var key events.Key
	var ch rune
	switch final {
	case 'u':
		key, ch = kittyKey(code)
	case '~':
		// Preserve the legacy Alt+Backspace encodings accepted by Veda.
		if flags == 2 && (code == 8 || code == 127) {
			key = events.KeyBackspace
			break
		}
		key = map[int]events.Key{
			1: events.KeyHome, 2: events.KeyInsert, 3: events.KeyDelete,
			4: events.KeyEnd, 5: events.KeyPgUp, 6: events.KeyPgDn,
			7: events.KeyHome, 8: events.KeyEnd,
			11: events.KeyF1, 12: events.KeyF2, 13: events.KeyF3, 14: events.KeyF4,
			15: events.KeyF5, 17: events.KeyF6, 18: events.KeyF7, 19: events.KeyF8,
			20: events.KeyF9, 21: events.KeyF10, 23: events.KeyF11, 24: events.KeyF12,
		}[code]
	default:
		if code != 1 {
			return false, false
		}
		key = map[byte]events.Key{
			'A': events.KeyUp, 'B': events.KeyDown, 'C': events.KeyRight,
			'D': events.KeyLeft, 'F': events.KeyEnd, 'H': events.KeyHome,
			'P': events.KeyF1, 'Q': events.KeyF2, 'S': events.KeyF4,
		}[final]
	}

	// Consume unsupported functional keys without inserting private-use runes.
	*buf = (*buf)[size:]
	if key == events.KeyRune && ch == 0 {
		return true, false
	}
	if flags&4 != 0 {
		if key == events.KeyRune {
			lower := unicode.ToLower(ch)
			switch {
			case lower >= 'a' && lower <= 'z':
				key = events.KeyCtrlA + events.Key(lower-'a')
			case ch == ' ':
				key = events.KeyCtrlSpace
			case ch == '[':
				key = events.KeyEscape
			case ch == '/' || ch == '_':
				key = events.KeyCtrlUnderscore
			}
			if key != events.KeyRune {
				ch = 0
				flags &^= 4
			}
		} else if key == events.KeyEnter {
			// Preserve the existing Ctrl+Enter alias for Ctrl+M.
			key = events.KeyCtrlM
			flags &^= 4
		}
	}
	var mod events.ModMask
	if flags&1 != 0 {
		mod |= events.ModShift
	}
	if flags&2 != 0 {
		mod |= events.ModAlt
	}
	if flags&4 != 0 {
		mod |= events.ModCtrl
	}
	if flags&(8|16|32) != 0 {
		mod |= events.ModMeta
	}
	t.inputChan <- events.NewKeyEvent(key, ch, mod)
	return true, false
}

func kittyKey(code int) (events.Key, rune) {
	switch code {
	case 9:
		return events.KeyTab, 0
	case 13, 57414:
		return events.KeyEnter, 0
	case 27:
		return events.KeyEscape, 0
	case 127:
		return events.KeyBackspace, 0
	}
	if code >= 57376 && code <= 57387 {
		return events.KeyF13 + events.Key(code-57376), 0
	}
	if code >= 57399 && code <= 57416 {
		return events.KeyRune, rune("0123456789./*-+\r=,"[code-57399])
	}
	if code >= 57417 && code <= 57426 {
		return [...]events.Key{
			events.KeyLeft, events.KeyRight, events.KeyUp, events.KeyDown,
			events.KeyPgUp, events.KeyPgDn, events.KeyHome, events.KeyEnd,
			events.KeyInsert, events.KeyDelete,
		}[code-57417], 0
	}
	if code < 32 || (code >= 57344 && code <= 63743) || !utf8.ValidRune(rune(code)) || code > utf8.MaxRune {
		return events.KeyRune, 0
	}
	return events.KeyRune, rune(code)
}
