// Package session connects the persistent terminal/proxy owner to replaceable
// editor workers. The protocol contains data, never Go pointers or callbacks.
package session

import (
	"encoding/json"
	"io"
	"sync"
	"time"
	"veda/internal/events"
	"veda/internal/ui"
)

// Bump when a worker can no longer communicate with an older parent.
const Version = 1

type Message struct {
	Kind          string
	Version       int
	ProxyVersion  string
	Data          []byte
	Error         string
	Width, Height int
	Frame         *Frame
	Input         *Input
	Text          string
	Timeout       int64
	RequestID     int
	Pending       []*Input
}

type Cell struct {
	X, Y      int
	Rune      rune
	Combining []rune
	Style     ui.Style
}

type Frame struct {
	Clear            bool
	Cells            []Cell
	CursorX, CursorY int
	CursorStyle      ui.CursorStyle
	Style            ui.Style
}

type Input struct {
	Key    *events.KeyEvent
	Resize *events.ResizeEvent
	Mouse  *events.MouseEvent
	Paste  *events.PasteEvent
}

func packInput(ev events.Event) *Input {
	switch ev := ev.(type) {
	case *events.KeyEvent:
		return &Input{Key: ev}
	case *events.ResizeEvent:
		return &Input{Resize: ev}
	case *events.MouseEvent:
		return &Input{Mouse: ev}
	case *events.PasteEvent:
		return &Input{Paste: ev}
	}
	return nil
}

func (in *Input) event() events.Event {
	switch {
	case in.Key != nil:
		return events.NewKeyEvent(in.Key.Key, in.Key.Rune, in.Key.Modifiers)
	case in.Resize != nil:
		return events.NewResizeEvent(in.Resize.Width, in.Resize.Height)
	case in.Mouse != nil:
		return events.NewMouseEvent(in.Mouse.X, in.Mouse.Y, in.Mouse.Button, in.Mouse.Modifiers)
	case in.Paste != nil:
		return events.NewPasteEvent(in.Paste.Start)
	}
	return nil
}

type channel struct {
	io.ReadWriteCloser
	mu       sync.Mutex
	encoder  *json.Encoder
	messages chan Message
	done     chan struct{}
	once     sync.Once
}

func newChannel(stream io.ReadWriteCloser) *channel {
	c := &channel{ReadWriteCloser: stream, encoder: json.NewEncoder(stream), messages: make(chan Message, 128), done: make(chan struct{})}
	go func() {
		defer close(c.messages)
		decoder := json.NewDecoder(stream)
		for {
			var msg Message
			if decoder.Decode(&msg) != nil {
				return
			}
			select {
			case c.messages <- msg:
			case <-c.done:
				return
			}
		}
	}()
	return c
}

func (c *channel) send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if conn, ok := c.ReadWriteCloser.(interface{ SetWriteDeadline(time.Time) error }); ok {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		defer conn.SetWriteDeadline(time.Time{})
	}
	return c.encoder.Encode(msg)
}

func (c *channel) Close() error {
	c.once.Do(func() { close(c.done) })
	return c.ReadWriteCloser.Close()
}
