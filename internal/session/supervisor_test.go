package session

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
	"veda/internal/editor"
	"veda/internal/events"
	"veda/internal/proxy"
	"veda/internal/ui"
)

type observedScreen struct {
	*editor.MockScreen
	input        chan events.Event
	frames       chan string
	inits, finis int
}

func (s *observedScreen) Init() error                       { s.inits++; return nil }
func (s *observedScreen) Fini()                             { s.finis++ }
func (s *observedScreen) EventChannel() <-chan events.Event { return s.input }
func (s *observedScreen) Show() {
	s.MockScreen.Show()
	select {
	case s.frames <- strings.TrimSpace(s.GetLine(0)):
	default:
	}
}

func TestSupervisorReplacesWorkerAndRetainsProxyTerminal(t *testing.T) { testSupervisor(t, false) }
func TestSupervisorResumesOldWorkerAfterRejectedRestore(t *testing.T)  { testSupervisor(t, true) }

func testSupervisor(t *testing.T, reject bool) {
	t.Helper()
	if reject {
		t.Setenv("VEDA_TEST_REJECT_RELOAD", "1")
	}
	screen := &observedScreen{MockScreen: editor.NewMockScreen(40, 8), input: make(chan events.Event, 8), frames: make(chan string, 8)}
	manager, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := Run(screen, manager, executable, []string{"-test.run=^TestSessionWorkerProcess$"})
		done <- err
	}()
	waitFrame := func(want string) {
		t.Helper()
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		for {
			select {
			case frame := <-screen.frames:
				if frame == want {
					return
				}
			case err := <-done:
				t.Fatalf("session exited before %q: %v", want, err)
			case <-timer.C:
				t.Fatalf("timed out waiting for %q", want)
			}
		}
	}
	waitFrame("initial")
	screen.input <- events.NewKeyEvent(events.KeyRune, 'r', events.ModNone)
	if reject {
		waitFrame("resumed")
	} else {
		waitFrame("reloaded")
	}
	screen.input <- events.NewKeyEvent(events.KeyRune, 'q', events.ModNone)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("session failed to exit")
	}
	if screen.inits != 1 || screen.finis != 1 {
		t.Fatalf("terminal lifecycle changed across reload: init=%d fini=%d", screen.inits, screen.finis)
	}
}

// This is a noninteractive worker running in a subprocess of the tests above.
// Its live PTY ID is transferred to the replacement, proving the same Python
// proxy survives while the Go worker PID changes.
func TestSessionWorkerProcess(t *testing.T) {
	if os.Getenv("VEDA_WORKER") != "1" {
		return
	}
	worker, err := OpenWorker()
	if err != nil {
		t.Fatal(err)
	}
	manager, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		SessionID, Directory string
		PID                  int
	}
	if len(worker.State) != 0 {
		if os.Getenv("VEDA_TEST_REJECT_RELOAD") == "1" {
			worker.ReportError(os.ErrInvalid)
			os.Exit(0)
		}
		if err := json.Unmarshal(worker.State, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.PID == os.Getpid() {
			t.Fatal("worker PID did not change")
		}
	} else {
		saved.Directory, _ = os.Getwd()
		saved.SessionID, _, err = manager.StartTerminal(saved.Directory, []string{"/bin/sh", "-c", "read line; printf 'retained:%s' \"$line\""}, nil, 40, 8)
		if err != nil {
			t.Fatal(err)
		}
		saved.PID = os.Getpid()
	}
	if err := worker.Ready(); err != nil {
		t.Fatal(err)
	}
	draw := func(text string) {
		worker.Clear()
		for x, ch := range text {
			worker.SetContent(x, 0, ch, nil, ui.StyleDefault)
		}
		worker.Show()
	}
	if len(worker.State) == 0 {
		draw("initial")
	} else {
		if err := manager.WriteTerminal(saved.Directory, saved.SessionID, []byte("proof\n")); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(3 * time.Second)
		var output strings.Builder
		for !strings.Contains(output.String(), "retained:proof") {
			data, _, _, err := manager.ReadTerminal(saved.Directory, saved.SessionID, 65536)
			if err != nil {
				t.Fatal(err)
			}
			output.Write(data)
			if time.Now().After(deadline) {
				t.Fatalf("lost proxy terminal after worker replacement: %q", output.String())
			}
			time.Sleep(10 * time.Millisecond)
		}
		draw("reloaded")
	}
	for {
		ev := worker.PollEvent()
		key, ok := ev.(*events.KeyEvent)
		if !ok {
			continue
		}
		switch key.Rune {
		case 'r':
			if err := worker.Pause(); err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(saved)
			committed, reason := worker.Reload(data)
			if committed {
				_ = manager.Close()
				_ = worker.Close()
				os.Exit(0)
			}
			if reason == "" {
				t.Fatal("failed reload has no explanation")
			}
			draw("resumed")
		case 'q':
			_ = manager.Close()
			_ = worker.Close()
			os.Exit(0)
		}
	}
}
