package terminal

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"
	"veda/internal/events"
)

func newKeyboardTestTerminal(t *testing.T) (*Terminal, *os.File) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reader.Close(); writer.Close() })
	return &Terminal{
		tty: writer, inputChan: make(chan events.Event, 20), stopChan: make(chan struct{}),
	}, reader
}

func assertKeyboardOutput(t *testing.T, term *Terminal, reader *os.File, want string) {
	t.Helper()
	term.tty.Close()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("terminal output = %q, want %q", got, want)
	}
}

func TestKeyboardProtocolDetection(t *testing.T) {
	tests := []struct {
		name, reply string
		state       keyboardProtocolState
		output      string
	}{
		{"supported", "\x1b[?0u\x1b[?1;2c", keyboardProtocolEnabled, "\x1b[>1u"},
		{"Kitty trailing separator", "\x1b[?0u\x1b[?62;52;c", keyboardProtocolEnabled, "\x1b[>1u"},
		{"Kitty without clipboard", "\x1b[?0u\x1b[?62;c", keyboardProtocolEnabled, "\x1b[>1u"},
		{"already enabled", "\x1b[?31u\x1b[?1;2c", keyboardProtocolEnabled, "\x1b[>1u"},
		{"unsupported", "\x1b[?1;2c", keyboardProtocolDisabled, ""},
		{"unsupported trailing separator", "\x1b[?62;52;c", keyboardProtocolDisabled, ""},
		{"no response", "", keyboardProtocolQuerying, ""},
		{"late response", "\x1b[?1;2c\x1b[?0u", keyboardProtocolDisabled, ""},
		{"duplicate response", "\x1b[?0u\x1b[?0u", keyboardProtocolEnabled, "\x1b[>1u"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term, reader := newKeyboardTestTerminal(t)
			term.queryKeyboardProtocol()
			// Replies and a user key can arrive in the same input read.
			buf := []byte(tc.reply + "x")
			term.parseInput(&buf)
			if len(buf) != 0 || term.keyboardProtocol != tc.state {
				t.Fatalf("remaining = %q, protocol state = %v, want %v", buf, term.keyboardProtocol, tc.state)
			}
			if len(term.inputChan) != 1 {
				t.Fatalf("terminal replies generated key events: %d events", len(term.inputChan))
			}
			key := (<-term.inputChan).(*events.KeyEvent)
			if key.Key != events.KeyRune || key.Rune != 'x' {
				t.Fatalf("user key = %+v", key)
			}
			assertKeyboardOutput(t, term, reader, "\x1b[?u\x1b[c"+tc.output)
		})
	}
}

func TestKeyboardProtocolFragmentedReply(t *testing.T) {
	term, reader := newKeyboardTestTerminal(t)
	term.queryKeyboardProtocol()
	var buf []byte
	for _, b := range []byte("\x1b[?0u\x1b[?62;52;c") {
		buf = append(buf, b)
		term.parseInput(&buf)
	}
	if len(buf) != 0 || len(term.inputChan) != 0 || term.keyboardProtocol != keyboardProtocolEnabled {
		t.Fatalf("fragmented reply: remaining = %q, events = %d, state = %v", buf, len(term.inputChan), term.keyboardProtocol)
	}
	assertKeyboardOutput(t, term, reader, "\x1b[?u\x1b[c\x1b[>1u")
}

func TestKittyStartupRepliesDoNotGenerateEscape(t *testing.T) {
	const reply = "\x1b[?0u\x1b[?62;52;c"
	// A terminal response must never dismiss an open picker, regardless of
	// which read contains the end of the keyboard-protocol response.
	for split := 0; split <= len(reply); split++ {
		t.Run(fmt.Sprint(split), func(t *testing.T) {
			term, reader := newKeyboardTestTerminal(t)
			term.queryKeyboardProtocol()
			buf := []byte(reply[:split])
			term.parseInput(&buf)
			buf = append(buf, reply[split:]...)
			term.parseInput(&buf)
			if len(buf) != 0 || len(term.inputChan) != 0 {
				t.Fatalf("startup reply leaked into keyboard input: remaining = %q, events = %d", buf, len(term.inputChan))
			}
			if term.keyboardProtocol != keyboardProtocolEnabled {
				t.Fatal("keyboard protocol was not enabled")
			}
			assertKeyboardOutput(t, term, reader, "\x1b[?u\x1b[c\x1b[>1u")
		})
	}
}

func TestKeyboardProtocolCloseRestoresMode(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "unsupported", true: "supported"}[enabled], func(t *testing.T) {
			term, reader := newKeyboardTestTerminal(t)
			term.queryKeyboardProtocol()
			want := "\x1b[?u\x1b[c"
			if enabled {
				buf := []byte("\x1b[?0u")
				term.parseInput(&buf)
				want += "\x1b[>1u\x1b[<u"
			}
			term.Close()
			if term.keyboardProtocol != keyboardProtocolDisabled {
				t.Fatal("protocol still active after Close")
			}
			// Pop must precede leaving the alternate screen.
			assertKeyboardOutput(t, term, reader, want+"\x1b[?2004l\x1b[?25h\x1b[?1049l\x1b[2J\x1b[H")
		})
	}
}

func TestKeyboardProtocolReplyAfterRestore(t *testing.T) {
	term, reader := newKeyboardTestTerminal(t)
	term.queryKeyboardProtocol()
	term.restoreKeyboardProtocol()
	buf := []byte("\x1b[?0u")
	term.parseInput(&buf)
	if term.keyboardProtocol != keyboardProtocolDisabled || len(term.inputChan) != 0 {
		t.Fatal("late reply re-enabled protocol or became a key")
	}
	assertKeyboardOutput(t, term, reader, "\x1b[?u\x1b[c")
}

func TestLegacyEscapeTimeout(t *testing.T) {
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	term := &Terminal{inputChan: make(chan events.Event, 2), stopChan: make(chan struct{})}
	done := make(chan struct{})
	go func() { term.readInput(input); close(done) }()
	defer func() {
		close(term.stopChan)
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("input handler did not stop")
		}
	}()
	if _, err := writer.Write([]byte{'\x1b'}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-term.inputChan:
		if key := event.(*events.KeyEvent); key.Key != events.KeyEscape {
			t.Fatalf("expected Escape, got %+v", key)
		}
	case <-time.After(time.Second):
		t.Fatal("legacy Escape was left buffered")
	}
	if _, err := writer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-term.inputChan:
		if key := event.(*events.KeyEvent); key.Key != events.KeyRune || key.Rune != 'x' || key.Modifiers != events.ModNone {
			t.Fatalf("key after Escape = %+v", key)
		}
	case <-time.After(time.Second):
		t.Fatal("input stalled after Escape timeout")
	}
}
