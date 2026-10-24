package session

import (
	"fmt"
	"io"
	"net"
	"os"
	"slices"
	"sync"
	"time"
	"veda/internal/events"
	"veda/internal/proxy"
	"veda/internal/ui"
)

// Worker also implements the editor's Screen and terminal clipboard interfaces.
// Rendering is batched once per Show; individual cell writes stay local.
type Worker struct {
	control                       *channel
	commands                      chan Message
	inputs                        chan events.Event
	clipboard                     chan Message
	clipboardMu                   sync.Mutex
	clipboardID                   int
	mu                            sync.Mutex
	width, height                 int
	style                         ui.Style
	cells, previous               []Cell
	previousWidth, previousHeight int
	frame                         Frame
	State                         []byte
	accepted                      bool
}

func OpenWorker() (*Worker, error) {
	file := os.NewFile(4, "veda-session")
	stream, err := net.FileConn(file)
	file.Close()
	if err != nil {
		return nil, err
	}
	return openWorker(stream)
}

func openWorker(stream io.ReadWriteCloser) (*Worker, error) {
	c := newChannel(stream)
	msg, ok := <-c.messages
	if !ok {
		c.Close()
		return nil, fmt.Errorf("session parent disconnected")
	}
	if msg.Kind != "start" || msg.Version != Version || msg.ProxyVersion != proxy.ProxyVersion {
		_ = c.send(Message{Kind: "error", Error: "incompatible session or proxy protocol; restart vd"})
		c.Close()
		return nil, fmt.Errorf("incompatible session or proxy protocol")
	}
	w := &Worker{control: c, commands: make(chan Message, 4), inputs: make(chan events.Event, 128), clipboard: make(chan Message, 1), width: msg.Width, height: msg.Height, State: msg.Data}
	go func() {
		defer close(w.commands)
		for msg := range c.messages {
			switch msg.Kind {
			case "input":
				if msg.Input.Resize != nil {
					w.mu.Lock()
					w.width, w.height = msg.Input.Resize.Width, msg.Input.Resize.Height
					w.mu.Unlock()
				}
				select {
				case w.inputs <- msg.Input.event():
				case <-c.done:
					return
				}
			case "clipboard-result":
				select {
				case w.clipboard <- msg:
				case <-c.done:
					return
				}
			default:
				select {
				case w.commands <- msg:
				case <-c.done:
					return
				}
			}
		}
		// Wake the event loop if the parent dies.
		select {
		case w.inputs <- events.NewCustomEvent("session-disconnected", nil):
		default:
		}
	}()
	return w, nil
}

func (w *Worker) Ready() error {
	if err := w.control.send(Message{Kind: "ready"}); err != nil {
		return err
	}
	msg, ok := <-w.commands
	if !ok || msg.Kind != "activate" {
		return fmt.Errorf("session activation failed")
	}
	return nil
}

// Pause stops the parent from sending more keys. Keys already in flight are
// returned to the parent ahead of newer input, ready for either worker.
func (w *Worker) Pause() error {
	if err := w.control.send(Message{Kind: "pause"}); err != nil {
		return err
	}
	var pending []*Input
	for {
		select {
		case ev := <-w.inputs:
			if in := packInput(ev); in != nil {
				pending = append(pending, in)
			}
		case msg, ok := <-w.commands:
			if !ok || msg.Kind != "paused" {
				return fmt.Errorf("session pause failed")
			}
			for {
				select {
				case ev := <-w.inputs:
					if in := packInput(ev); in != nil {
						pending = append(pending, in)
					}
				default:
					return w.control.send(Message{Kind: "pending-input", Pending: pending})
				}
			}
		}
	}
}

func (w *Worker) CancelReload() { w.previous = nil; _ = w.control.send(Message{Kind: "cancel-reload"}) }

// Reload waits while the parent validates a replacement. False means this
// worker should resume because the new executable failed to start or restore.
func (w *Worker) Reload(data []byte) (bool, string) {
	if err := w.control.send(Message{Kind: "reload", Data: data}); err != nil {
		return false, err.Error()
	}
	msg, ok := <-w.commands
	if !ok {
		return false, "session parent disconnected"
	}
	if msg.Kind != "commit" {
		w.previous = nil
	}
	return msg.Kind == "commit", msg.Error
}

func (w *Worker) ReportError(err error) {
	_ = w.control.send(Message{Kind: "error", Error: err.Error()})
}
func (w *Worker) Close() error            { return w.control.Close() }
func (w *Worker) Init() error             { return nil }
func (w *Worker) Fini()                   {}
func (w *Worker) Size() (int, int)        { w.mu.Lock(); defer w.mu.Unlock(); return w.width, w.height }
func (w *Worker) SetStyle(style ui.Style) { w.style = style }
func (w *Worker) Clear() {
	width, height := w.Size()
	if len(w.cells) != width*height {
		w.cells = make([]Cell, width*height)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			w.cells[y*width+x] = Cell{X: x, Y: y, Rune: ' ', Style: w.style}
		}
	}
}
func (w *Worker) SetContent(x, y int, main rune, combining []rune, style ui.Style) {
	width, height := w.Size()
	if x < 0 || y < 0 || x >= width || y >= height || len(w.cells) != width*height {
		return
	}
	w.cells[y*width+x] = Cell{X: x, Y: y, Rune: main, Combining: append([]rune(nil), combining...), Style: style}
}
func (w *Worker) ShowCursor(x, y int)                 { w.frame.CursorX, w.frame.CursorY = x, y }
func (w *Worker) SetCursorStyle(style ui.CursorStyle) { w.frame.CursorStyle = style }
func (w *Worker) PollEvent() events.Event             { return <-w.inputs }
func (w *Worker) EventChannel() <-chan events.Event   { return w.inputs }
func (w *Worker) Show() {
	frame := w.frame
	frame.Style = w.style
	width, height := w.Size()
	frame.Clear = len(w.previous) != len(w.cells) || width != w.previousWidth || height != w.previousHeight
	for i, cell := range w.cells {
		if frame.Clear || cell.Rune != w.previous[i].Rune || cell.Style != w.previous[i].Style || !slices.Equal(cell.Combining, w.previous[i].Combining) {
			frame.Cells = append(frame.Cells, cell)
		}
	}
	w.previous = append(w.previous[:0], w.cells...)
	w.previousWidth, w.previousHeight = width, height
	if err := w.control.send(Message{Kind: "frame", Frame: &frame}); err != nil {
		os.Exit(1)
	}
	if !w.accepted {
		msg, ok := <-w.commands
		if !ok || msg.Kind != "accepted" {
			os.Exit(1)
		}
		w.accepted = true
	}
}

func (w *Worker) clipboardRequest(kind, text string, timeout time.Duration) (string, error) {
	w.clipboardMu.Lock()
	defer w.clipboardMu.Unlock()
	w.clipboardID++
	if err := w.control.send(Message{Kind: kind, Text: text, Timeout: int64(timeout), RequestID: w.clipboardID}); err != nil {
		return "", err
	}
	timer := time.NewTimer(timeout + time.Second)
	defer timer.Stop()
	for {
		select {
		case msg := <-w.clipboard:
			if msg.RequestID != w.clipboardID {
				continue
			}
			if msg.Error != "" {
				return "", fmt.Errorf("%s", msg.Error)
			}
			return msg.Text, nil
		case <-timer.C:
			return "", fmt.Errorf("session clipboard timeout")
		case <-w.control.done:
			return "", fmt.Errorf("session parent disconnected")
		}
	}
}
func (w *Worker) CopyToClipboard(text string) error {
	_, err := w.clipboardRequest("clipboard-copy", text, time.Second)
	return err
}
func (w *Worker) QueryClipboard(timeout time.Duration) (string, error) {
	return w.clipboardRequest("clipboard-query", "", timeout)
}
