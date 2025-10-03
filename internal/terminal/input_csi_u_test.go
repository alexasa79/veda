package terminal

import (
	"testing"
	"veda/internal/events"
)

func TestKittyCtrlMSequence(t *testing.T) {
	term := &Terminal{inputChan: make(chan events.Event, 1)}
	seq := []byte{'\x1b', '[', '1', '0', '9', ';', '5', 'u'}
	buf := append([]byte{}, seq...)
	if matched, needMore := term.matchKeyCode(&buf); !matched || needMore {
		t.Fatalf("expected sequence %v to match without needing more data", seq)
	}
	if len(buf) != 0 {
		t.Fatalf("expected buffer to be consumed, remaining: %v", buf)
	}
	evt := <-term.inputChan
	keyEvt, ok := evt.(*events.KeyEvent)
	if !ok {
		t.Fatalf("expected KeyEvent, got %T", evt)
	}
	if keyEvt.Key != events.KeyCtrlM {
		t.Fatalf("expected key Ctrl+M, got %v", keyEvt.Key)
	}
	if keyEvt.Modifiers != events.ModNone {
		t.Fatalf("expected no modifiers, got %v", keyEvt.Modifiers)
	}
}

func TestKittyCtrlShiftMSequence(t *testing.T) {
	term := &Terminal{inputChan: make(chan events.Event, 1)}
	seq := []byte{'\x1b', '[', '7', '7', ';', '6', 'u'}
	buf := append([]byte{}, seq...)
	if matched, needMore := term.matchKeyCode(&buf); !matched || needMore {
		t.Fatalf("expected sequence %v to match without needing more data", seq)
	}
	if len(buf) != 0 {
		t.Fatalf("expected buffer to be consumed, remaining: %v", buf)
	}
	evt := <-term.inputChan
	keyEvt, ok := evt.(*events.KeyEvent)
	if !ok {
		t.Fatalf("expected KeyEvent, got %T", evt)
	}
	if keyEvt.Key != events.KeyCtrlM {
		t.Fatalf("expected key Ctrl+M, got %v", keyEvt.Key)
	}
	if keyEvt.Modifiers&events.ModShift == 0 {
		t.Fatalf("expected shift modifier to be set")
	}
	if keyEvt.Modifiers&events.ModCtrl != 0 {
		t.Fatalf("expected ctrl modifier to be absorbed into key state")
	}
}
