package terminal

import (
	"fmt"
	"testing"
	"veda/internal/events"
)

func TestKittyDisambiguatedKeys(t *testing.T) {
	tests := []struct {
		name, seq string
		key       events.Key
		ch        rune
		mod       events.ModMask
	}{
		{"Escape", "\x1b[27u", events.KeyEscape, 0, events.ModNone},
		{"explicit modifier", "\x1b[27;1u", events.KeyEscape, 0, events.ModNone},
		{"empty modifier", "\x1b[27;u", events.KeyEscape, 0, events.ModNone},
		{"Alt Escape", "\x1b[27;3u", events.KeyEscape, 0, events.ModAlt},
		{"Enter", "\r", events.KeyEnter, 0, events.ModNone},
		{"CSI Enter", "\x1b[13u", events.KeyEnter, 0, events.ModNone},
		{"Shift Enter", "\x1b[13;2u", events.KeyEnter, 0, events.ModShift},
		{"Ctrl Enter alias", "\x1b[13;5u", events.KeyCtrlM, 0, events.ModNone},
		{"Ctrl M", "\x1b[109;5u", events.KeyCtrlM, 0, events.ModNone},
		{"Ctrl I", "\x1b[105;5u", events.KeyCtrlI, 0, events.ModNone},
		{"Ctrl J", "\x1b[106;5u", events.KeyCtrlJ, 0, events.ModNone},
		{"Ctrl H", "\x1b[104;5u", events.KeyCtrlH, 0, events.ModNone},
		{"Ctrl Alt M", "\x1b[109;7u", events.KeyCtrlM, 0, events.ModAlt},
		{"Ctrl Shift M", "\x1b[109;6u", events.KeyCtrlM, 0, events.ModShift},
		{"Ctrl Space", "\x1b[32;5u", events.KeyCtrlSpace, 0, events.ModNone},
		{"Ctrl bracket Escape alias", "\x1b[91;5u", events.KeyEscape, 0, events.ModNone},
		{"Ctrl slash", "\x1b[47;5u", events.KeyCtrlUnderscore, 0, events.ModNone},
		{"Ctrl underscore", "\x1b[95;5u", events.KeyCtrlUnderscore, 0, events.ModNone},
		{"Tab", "\t", events.KeyTab, 0, events.ModNone},
		{"Shift Tab", "\x1b[9;2u", events.KeyTab, 0, events.ModShift},
		{"Backspace", "\x7f", events.KeyBackspace, 0, events.ModNone},
		{"Alt Backspace", "\x1b[127;3u", events.KeyBackspace, 0, events.ModAlt},
		{"legacy Alt Backspace", "\x1b[127;3~", events.KeyBackspace, 0, events.ModAlt},
		{"legacy Alt Backspace variant", "\x1b[8;3~", events.KeyBackspace, 0, events.ModAlt},
		{"Alt letter", "\x1b[113;3u", events.KeyRune, 'q', events.ModAlt},
		{"Unicode case", "\x1b[937u", events.KeyRune, 'Ω', events.ModNone},
		{"Shift Alt letter", "\x1b[113;4u", events.KeyRune, 'q', events.ModShift | events.ModAlt},
		{"Meta key", "\x1b[97;9u", events.KeyRune, 'a', events.ModMeta},
		{"lock modifiers", "\x1b[109;197u", events.KeyCtrlM, 0, events.ModNone},
		{"up", "\x1b[A", events.KeyUp, 0, events.ModNone},
		{"Shift Ctrl left", "\x1b[1;6D", events.KeyLeft, 0, events.ModShift | events.ModCtrl},
		{"Ctrl Home", "\x1b[1;5H", events.KeyHome, 0, events.ModCtrl},
		{"Alt End", "\x1b[1;3F", events.KeyEnd, 0, events.ModAlt},
		{"Alt Delete", "\x1b[3;3~", events.KeyDelete, 0, events.ModAlt},
		{"Ctrl PageUp", "\x1b[5;5~", events.KeyPgUp, 0, events.ModCtrl},
		{"Shift F1", "\x1b[1;2P", events.KeyF1, 0, events.ModShift},
		{"F3", "\x1b[13~", events.KeyF3, 0, events.ModNone},
		{"Ctrl F12", "\x1b[24;5~", events.KeyF12, 0, events.ModCtrl},
		{"F13", "\x1b[57376u", events.KeyF13, 0, events.ModNone},
		{"F24", "\x1b[57387u", events.KeyF24, 0, events.ModNone},
		{"keypad zero", "\x1b[57399u", events.KeyRune, '0', events.ModNone},
		{"keypad nine", "\x1b[57408u", events.KeyRune, '9', events.ModNone},
		{"keypad decimal", "\x1b[57409u", events.KeyRune, '.', events.ModNone},
		{"keypad plus", "\x1b[57413u", events.KeyRune, '+', events.ModNone},
		{"keypad Enter", "\x1b[57414u", events.KeyEnter, 0, events.ModNone},
		{"keypad equals", "\x1b[57415u", events.KeyRune, '=', events.ModNone},
		{"keypad Ctrl left", "\x1b[57417;5u", events.KeyLeft, 0, events.ModCtrl},
		{"keypad Delete", "\x1b[57426u", events.KeyDelete, 0, events.ModNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term := &Terminal{inputChan: make(chan events.Event, 4)}
			// Exercise the full input parser, including every possible read boundary.
			var buf []byte
			for _, b := range []byte(tc.seq) {
				buf = append(buf, b)
				term.parseInput(&buf)
			}
			if len(buf) != 0 || len(term.inputChan) != 1 {
				t.Fatalf("remaining = %q, events = %d", buf, len(term.inputChan))
			}
			got := (<-term.inputChan).(*events.KeyEvent)
			if got.Key != tc.key || got.Rune != tc.ch || got.Modifiers != tc.mod {
				t.Fatalf("got %v %q %v, want %v %q %v", got.Key, got.Rune, got.Modifiers, tc.key, tc.ch, tc.mod)
			}
		})
	}
}

func TestKittyUnsupportedKeysAreNotInserted(t *testing.T) {
	for _, code := range []int{0, 57358, 57398, 57428, 57441, 0xd800, 0x110000} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			term := &Terminal{inputChan: make(chan events.Event, 4)}
			buf := []byte(fmt.Sprintf("\x1b[%du", code))
			term.parseInput(&buf)
			if len(buf) != 0 || len(term.inputChan) != 0 {
				t.Fatalf("unsupported key: remaining = %q, events = %d", buf, len(term.inputChan))
			}
		})
	}
}

func TestKittyKeysDoNotConsumeFollowingInput(t *testing.T) {
	term := &Terminal{inputChan: make(chan events.Event, 8)}
	buf := []byte("\x1b[109;5u\r\x1b[27u\x1b[200~x\x1b[201~")
	term.parseInput(&buf)
	if len(buf) != 0 || len(term.inputChan) != 6 {
		t.Fatalf("remaining = %q, events = %d", buf, len(term.inputChan))
	}
	for _, want := range []events.Key{events.KeyCtrlM, events.KeyEnter, events.KeyEscape} {
		if got := (<-term.inputChan).(*events.KeyEvent).Key; got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if !(<-term.inputChan).(*events.PasteEvent).Start {
		t.Fatal("missing paste start")
	}
	if key := (<-term.inputChan).(*events.KeyEvent); key.Rune != 'x' {
		t.Fatalf("pasted key = %+v", key)
	}
	if (<-term.inputChan).(*events.PasteEvent).Start {
		t.Fatal("missing paste end")
	}
}

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
