package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// Relay keeps the actual proxy clients in the parent process. Closing a worker
// only disconnects its relay; it never closes a proxy.
type Relay struct {
	manager  *ProxyManager
	mu       sync.Mutex
	workers  map[chan map[string]interface{}]io.ReadWriteCloser
	closed   bool
	requests sync.WaitGroup
}

func NewRelay(manager *ProxyManager) *Relay {
	r := &Relay{manager: manager, workers: make(map[chan map[string]interface{}]io.ReadWriteCloser)}
	manager.SetStatusCallback(func(status string) { r.notify("session_status", map[string]interface{}{"status": status}) })
	manager.SetMessageCallback(func(level, message string) {
		r.notify("session_message", map[string]interface{}{"level": level, "message": message})
	})
	manager.SetDiagnosticCallback(func(path string, diagnostics []Diagnostic) {
		r.notify("diagnostics", map[string]interface{}{"path": path, "diagnostics": diagnostics})
	})
	manager.SetLSPStatusCallback(func(language, status, errorMsg string, extensions []string) {
		r.notify("lsp_status", map[string]interface{}{"language": language, "status": status, "error": errorMsg, "extensions": extensions})
	})
	return r
}

func (r *Relay) notify(method string, params interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.workers {
		// A paused worker must not block the proxies or its replacement.
		select {
		case ch <- map[string]interface{}{"method": method, "params": params}:
		default:
		}
	}
}

func newRelayManager(stream io.ReadWriteCloser) (*ProxyManager, error) {
	pm := &ProxyManager{clients: make(map[string]*Client)}
	reader := bufio.NewScanner(stream)
	reader.Buffer(make([]byte, 4096), maxScanTokenSize)
	pm.relay = &Client{stdin: bufio.NewWriter(stream), stdout: reader, manager: pm, transport: stream, pending: make(map[int]chan map[string]interface{}), readerErr: make(chan error, 1)}
	return pm, nil
}

// PauseNotifications prevents callbacks from changing buffers during a snapshot.
func (pm *ProxyManager) PauseNotifications() func() {
	pm.notificationMu.Lock()
	return pm.notificationMu.Unlock
}

func (r *Relay) Serve(stream io.ReadWriteCloser) {
	defer stream.Close()
	notifications := make(chan map[string]interface{}, 128)
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.workers[notifications] = stream
	r.mu.Unlock()
	defer func() { r.mu.Lock(); delete(r.workers, notifications); r.mu.Unlock() }()
	var writeMu sync.Mutex
	enc := json.NewEncoder(stream)
	write := func(msg interface{}) error { writeMu.Lock(); defer writeMu.Unlock(); return enc.Encode(msg) }
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case msg := <-notifications:
				if write(msg) != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()
	dec := json.NewDecoder(stream)
	for {
		var envelope struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params struct {
				Host    string         `json:"host"`
				Request JSONRPCRequest `json:"request"`
				Timeout time.Duration  `json:"timeout"`
			} `json:"params"`
		}
		if dec.Decode(&envelope) != nil {
			return
		}
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			return
		}
		r.requests.Add(1)
		r.mu.Unlock()
		go func() {
			defer r.requests.Done()
			var resp map[string]interface{}
			var err error
			if envelope.Method != "request" {
				err = fmt.Errorf("unknown session request %q", envelope.Method)
			} else {
				path := ""
				if envelope.Params.Host != "" {
					path = envelope.Params.Host + ":/"
				}
				var client *Client
				client, _, err = r.manager.getClient(path)
				if err == nil {
					req := envelope.Params.Request
					req.ID = client.getNextRequestID()
					resp, err = client.sendRequestWithDuration(req, envelope.Params.Timeout)
					if shouldReconnect(err) {
						if client, err = r.manager.reconnect(envelope.Params.Host); err == nil {
							req.ID = client.getNextRequestID()
							resp, err = client.sendRequestWithDuration(req, envelope.Params.Timeout)
						}
					}
				}
			}
			if resp == nil {
				resp = make(map[string]interface{})
			}
			resp["id"] = envelope.ID
			if err != nil {
				resp["error"] = err.Error()
			}
			_ = write(resp)
		}()
	}
}

// Close stops workers and waits for requests before the parent closes its proxy
// manager. In particular, a remote connection cannot be installed after cleanup.
func (r *Relay) Close() {
	r.mu.Lock()
	r.closed = true
	var streams []io.ReadWriteCloser
	for _, stream := range r.workers {
		streams = append(streams, stream)
	}
	r.mu.Unlock()
	for _, stream := range streams {
		_ = stream.Close()
	}
	r.requests.Wait()
}
