package session

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"veda/internal/clipboard"
	"veda/internal/events"
	"veda/internal/log"
	"veda/internal/proxy"
	"veda/internal/ui"

	"golang.org/x/sys/unix"
)

var sessionLog = log.NewLogger("SESSION")

type Screen interface {
	Init() error
	Fini()
	Size() (int, int)
	SetStyle(ui.Style)
	Clear()
	SetContent(int, int, rune, []rune, ui.Style)
	ShowCursor(int, int)
	SetCursorStyle(ui.CursorStyle)
	Show()
	EventChannel() <-chan events.Event
}

type process struct {
	cmd      *exec.Cmd
	control  *channel
	accepted bool
}

type processEvent struct {
	worker  *process
	message Message
	exit    bool
	err     error
}

type workerOutput struct{ bytes.Buffer }

func (w *workerOutput) Write(data []byte) (int, error) {
	if remaining := 64*1024 - w.Len(); remaining > 0 {
		_, _ = w.Buffer.Write(data[:min(len(data), remaining)])
	}
	return len(data), nil
}

func socketPair() (*os.File, *os.File, error) {
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, err
	}
	unix.CloseOnExec(fds[0])
	unix.CloseOnExec(fds[1])
	return os.NewFile(uintptr(fds[0]), "veda-parent"), os.NewFile(uintptr(fds[1]), "veda-worker"), nil
}

func startProcess(executable string, args []string, relay *proxy.Relay, output chan<- processEvent, reloading bool) (*process, error) {
	parentProxy, workerProxy, err := socketPair()
	if err != nil {
		return nil, err
	}
	defer workerProxy.Close()
	parentControl, workerControl, err := socketPair()
	if err != nil {
		parentProxy.Close()
		return nil, err
	}
	defer workerControl.Close()
	cmd := exec.Command(executable, args...)
	cmd.Env = append(os.Environ(), "VEDA_WORKER=1")
	if reloading {
		cmd.Env = append(cmd.Env, "VEDA_RELOAD=1")
	}
	cmd.ExtraFiles = []*os.File{workerProxy, workerControl}
	var diagnostics workerOutput
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, &diagnostics, &diagnostics
	if err := cmd.Start(); err != nil {
		parentProxy.Close()
		parentControl.Close()
		return nil, err
	}
	connection, err := net.FileConn(parentControl)
	parentControl.Close()
	if err != nil {
		cmd.Process.Kill()
		parentProxy.Close()
		return nil, err
	}
	p := &process{cmd: cmd, control: newChannel(connection)}
	go relay.Serve(parentProxy)
	go func() {
		for msg := range p.control.messages {
			output <- processEvent{worker: p, message: msg}
		}
	}()
	go func() {
		err := cmd.Wait()
		if text := strings.TrimSpace(diagnostics.String()); text != "" {
			sessionLog.Debug("Editor worker %d: %s", cmd.Process.Pid, text)
			if err != nil {
				err = fmt.Errorf("%w: %s", err, text)
			}
		}
		output <- processEvent{worker: p, exit: true, err: err}
	}()
	return p, nil
}

func (p *process) stop() {
	_ = p.control.Close()
	_ = p.cmd.Process.Kill()
}

// Run owns the terminal and proxies until the entire session ends. Workers can
// be replaced without reopening either resource. The old worker remains alive
// until the replacement has restored its state and produced its first frame.
func Run(screen Screen, manager *proxy.ProxyManager, executable string, args []string) (int, error) {
	defer manager.Close()
	if err := screen.Init(); err != nil {
		return 1, err
	}
	defer screen.Fini()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	relay := proxy.NewRelay(manager)
	defer relay.Close()
	output := make(chan processEvent, 128)
	current, err := startProcess(executable, args, relay, output, false)
	if err != nil {
		return 1, err
	}
	workers := map[*process]bool{current: true}
	defer func() {
		for p := range workers {
			p.stop()
		}
	}()
	start := func(p *process, state []byte) error {
		width, height := screen.Size()
		return p.control.send(Message{Kind: "start", Version: Version, ProxyVersion: proxy.ProxyVersion, Width: width, Height: height, Data: state})
	}
	if err := start(current, nil); err != nil {
		return 1, err
	}
	var candidate *process
	paused := true
	var pending []*Input
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	reject := func(reason string) {
		if candidate != nil {
			candidate.stop()
			delete(workers, candidate)
			candidate = nil
		}
		_ = current.control.send(Message{Kind: "resume", Error: reason})
		paused = false
	}
	flush := func() {
		width, height := screen.Size()
		_ = current.control.send(Message{Kind: "input", Input: &Input{Resize: events.NewResizeEvent(width, height)}})
		for _, in := range pending {
			_ = current.control.send(Message{Kind: "input", Input: in})
		}
		pending = nil
	}
	for {
		select {
		case sig := <-signals:
			return 128 + int(sig.(syscall.Signal)), nil
		case ev, ok := <-screen.EventChannel():
			if !ok {
				return 0, nil
			}
			if in := packInput(ev); in != nil {
				if paused {
					pending = append(pending, in)
				} else if err := current.control.send(Message{Kind: "input", Input: in}); err != nil {
					return 1, err
				}
			}
		case <-timer.C:
			if candidate != nil {
				reject("replacement timed out; continuing with the current editor")
				flush()
			} else if paused {
				return 1, fmt.Errorf("editor startup timed out")
			}
		case event := <-output:
			p, msg := event.worker, event.message
			if _, exists := workers[p]; !exists {
				continue
			}
			if event.exit {
				_ = p.control.Close()
				delete(workers, p)
				if p == candidate {
					reject(fmt.Sprintf("replacement exited: %v", event.err))
					flush()
					continue
				}
				if p == current {
					if event.err != nil {
						return p.cmd.ProcessState.ExitCode(), event.err
					}
					return 0, nil
				}
				continue
			}
			if !workers[p] {
				continue
			}
			switch msg.Kind {
			case "pause":
				if p == current {
					paused = true
					_ = p.control.send(Message{Kind: "paused"})
				}
			case "pending-input":
				if p == current {
					pending = append(msg.Pending, pending...)
				}
			case "cancel-reload":
				if p == current && candidate == nil {
					paused = false
					flush()
				}
			case "error":
				if p == candidate {
					reject(msg.Error)
					flush()
				} else {
					return 1, fmt.Errorf("%s", msg.Error)
				}
			case "ready":
				_ = p.control.send(Message{Kind: "activate"})
			case "reload":
				if p != current || candidate != nil {
					continue
				}
				paused = true
				candidate, err = startProcess(executable, args, relay, output, true)
				if err != nil {
					reject(err.Error())
					flush()
					continue
				}
				workers[candidate] = true
				if err = start(candidate, msg.Data); err != nil {
					reject(err.Error())
					flush()
					continue
				}
				timer.Reset(30 * time.Second)
			case "frame":
				if p == current && paused && p.accepted {
					continue
				}
				if p == candidate {
					old := current
					current, candidate = candidate, nil
					workers[old] = false
					_ = old.control.send(Message{Kind: "commit"})
					// Its normal exit closes the worker relay, leaving proxies alive.
				}
				if p != current {
					continue
				}
				needsFlush := paused
				if paused {
					paused = false
					timer.Stop()
				}
				frame := msg.Frame
				screen.SetStyle(frame.Style)
				if frame.Clear {
					screen.Clear()
				}
				for _, cell := range frame.Cells {
					screen.SetContent(cell.X, cell.Y, cell.Rune, cell.Combining, cell.Style)
				}
				screen.ShowCursor(frame.CursorX, frame.CursorY)
				screen.SetCursorStyle(frame.CursorStyle)
				screen.Show()
				if !p.accepted {
					_ = p.control.send(Message{Kind: "accepted"})
					p.accepted = true
				}
				if needsFlush {
					flush()
				}
			case "clipboard-copy", "clipboard-query":
				go func() {
					var text string
					var err error
					if msg.Kind == "clipboard-copy" {
						err = clipboard.OSC52Copy(msg.Text)
					} else {
						text, err = clipboard.OSC52Paste(time.Duration(msg.Timeout))
					}
					response := Message{Kind: "clipboard-result", Text: text, RequestID: msg.RequestID}
					if err != nil {
						response.Error = err.Error()
					}
					_ = p.control.send(response)
				}()
			}
		}
	}
}
