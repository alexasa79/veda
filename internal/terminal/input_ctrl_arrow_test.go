package terminal

import (
	"testing"
	"veda/internal/events"
)

func TestCtrlArrowKeySequences(t *testing.T) {
	tests := []struct {
		name      string
		seq       []byte
		expectKey events.Key
	}{
		{"ctrl+left", []byte{'\x1b', '[', '1', ';', '5', 'D'}, events.KeyLeft},
		{"ctrl+right", []byte{'\x1b', '[', '1', ';', '5', 'C'}, events.KeyRight},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term := &Terminal{inputChan: make(chan events.Event, 1)}
			buf := append([]byte{}, tc.seq...)
			matched, needMore := term.matchKeyCode(&buf)
			if !matched || needMore {
				t.Fatalf("sequence %v was not fully matched", tc.seq)
			}
			if len(buf) != 0 {
				t.Fatalf("expected buffer to be consumed, remaining: %v", buf)
			}
			evt := <-term.inputChan
			keyEvt, ok := evt.(*events.KeyEvent)
			if !ok {
				t.Fatalf("expected KeyEvent, got %T", evt)
			}
			if keyEvt.Key != tc.expectKey {
				t.Fatalf("expected key %v, got %v", tc.expectKey, keyEvt.Key)
			}
			if keyEvt.Modifiers&events.ModCtrl == 0 {
				t.Fatalf("expected ctrl modifier to be set")
			}
		})
	}
}
