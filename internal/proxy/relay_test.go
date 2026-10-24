package proxy

import (
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestRelayRetainsClientsAcrossWorkers(t *testing.T) {
	var calls atomic.Int32
	client := &Client{forward: func(req JSONRPCRequest, timeout time.Duration) (map[string]interface{}, error) {
		calls.Add(1)
		return map[string]interface{}{"result": map[string]interface{}{"exists": true, "type": "file"}}, nil
	}}
	manager := &ProxyManager{clients: map[string]*Client{"": client, "example": client}}
	relay := NewRelay(manager)
	for i := 0; i < 3; i++ {
		parent, worker := net.Pipe()
		finished := make(chan struct{})
		go func() { relay.Serve(parent); close(finished) }()
		pm, err := newRelayManager(worker)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"local.go", "example:/remote.go"} {
			attr, err := pm.GetFileAttributes(path)
			if err != nil || !attr.Exists {
				t.Fatalf("worker %d: %#v %v", i, attr, err)
			}
		}
		if err := pm.Close(); err != nil {
			t.Fatal(err)
		}
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Fatal("worker relay did not close")
		}
		if manager.clients[""] != client || manager.clients["example"] != client {
			t.Fatal("worker replacement changed parent clients")
		}
	}
	if calls.Load() != 6 {
		t.Fatalf("got %d requests", calls.Load())
	}
}

func TestRelayRoutesConcurrentRequestsAndNotifications(t *testing.T) {
	client := &Client{forward: func(req JSONRPCRequest, timeout time.Duration) (map[string]interface{}, error) {
		return map[string]interface{}{"result": map[string]interface{}{"name": req.Params["path"], "exists": true}}, nil
	}}
	manager := &ProxyManager{clients: map[string]*Client{"": client}}
	relay := NewRelay(manager)
	parent, worker := net.Pipe()
	go relay.Serve(parent)
	pm, _ := newRelayManager(worker)
	defer pm.Close()
	status := make(chan string, 1)
	pm.SetStatusCallback(func(s string) { status <- s })
	diagnostics := make(chan []Diagnostic, 1)
	pm.SetDiagnosticCallback(func(path string, d []Diagnostic) {
		if path == "file.go" {
			diagnostics <- d
		}
	})
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func(i int) {
			path := fmt.Sprintf("file%d.go", i)
			attr, err := pm.GetFileAttributes(path)
			if err == nil && attr.Name != path {
				err = fmt.Errorf("request routed incorrectly: %q != %q", attr.Name, path)
			}
			results <- err
		}(i)
	}
	for i := 0; i < 20; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	manager.reportStatus("connected")
	manager.handleDiagnosticUpdate("file.go", []Diagnostic{{Line: 1, Severity: "error", Message: "test"}})
	select {
	case s := <-status:
		if s != "connected" {
			t.Fatal(s)
		}
	case <-time.After(time.Second):
		t.Fatal("missing status notification")
	}
	select {
	case d := <-diagnostics:
		if len(d) != 1 || d[0].Message != "test" {
			t.Fatalf("bad diagnostics: %#v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("missing diagnostics notification")
	}
}
