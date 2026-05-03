package terminal

import (
	"testing"
	"veda/internal/events"
)

func TestWindowsTerminalPageKeys(t *testing.T) {
	tests := []struct {
		name string
		seq  []byte
		key  events.Key
	}{
		{
			name: "page up",
			seq:  []byte{'\x1b', '[', '5', '~'},
			key:  events.KeyPgUp,
		},
		{
			name: "page down",
			seq:  []byte{'\x1b', '[', '6', '~'},
			key:  events.KeyPgDn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := &Terminal{inputChan: make(chan events.Event, 1)}
			buf := append([]byte{}, tt.seq...)
			matched, needMore := term.matchKeyCode(&buf)
			if !matched || needMore {
				t.Fatalf("expected sequence %v to match without needing more data", tt.seq)
			}
			if len(buf) != 0 {
				t.Fatalf("expected buffer to be consumed, remaining: %v", buf)
			}

			evt := <-term.inputChan
			keyEvt, ok := evt.(*events.KeyEvent)
			if !ok {
				t.Fatalf("expected KeyEvent, got %T", evt)
			}
			if keyEvt.Key != tt.key {
				t.Fatalf("expected key %v, got %v", tt.key, keyEvt.Key)
			}
			if keyEvt.Modifiers != events.ModNone {
				t.Fatalf("expected no modifiers, got %v", keyEvt.Modifiers)
			}
		})
	}
}
