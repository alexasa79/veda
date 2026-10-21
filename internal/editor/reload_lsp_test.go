package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"veda/internal/events"
	"veda/internal/proxy"
)

func TestReloadFileUpdatesLSPDefinition(t *testing.T) {
	if testing.Short() {
		t.Skip("requires clangd")
	}
	if _, err := exec.LookPath("clangd"); err != nil {
		t.Skip("clangd not found")
	}
	dir, err := filepath.Abs("../../artifacts/lsp_reload_reproduction")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "main.c")
	original := "int existing(void) { return 1; }\n\nint main(void) { return existing(); }\n"
	modified := original + "\nint added(void) { return 2; }\n\nint caller(void) { return added(); }\n"
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(original)
	te := NewTestEditor(100, 25)
	te.proxyManager.Close()
	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatal(err)
	}
	defer pm.Close()
	te.proxyManager = pm
	file, err := pm.GetFile(path, true)
	if err != nil {
		t.Fatal(err)
	}
	te.finishOpenFile(path, file.Content, &file.Changes, file.Branch, file.Diagnostics)
	AssertBufferEquals(t, te, original)

	// Wait for a real definition response before changing anything on disk.
	deadline := time.Now().Add(15 * time.Second)
	for {
		locations, err := te.proxyManager.GoToDefinition(path, 2, 25)
		if err == nil && len(locations) == 1 && locations[0].Range.Start.Line == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("clangd did not become ready: locations=%+v, error=%v", locations, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	te.SetCursorPosition(25, 2)
	if err := te.ExecuteCommand("go-to-definition"); err != nil {
		t.Fatal(err)
	}
	AssertCursorAt(t, te, 4, 0)
	t.Log("Before external edit: go-to-definition finds existing()")

	// Simulate another process editing the file while it is open in Veda.
	write(modified)
	if err := te.ExecuteCommand("reload-file"); err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
reload:
	for {
		select {
		case event := <-te.asyncEvents:
			if event, ok := event.(*events.CustomEvent); ok && event.Type == "file-reloaded" {
				te.handleFileReloaded(event.Data.(map[string]interface{}))
				break reload
			}
		case <-timer.C:
			t.Fatal("reload did not complete")
		}
	}
	AssertBufferEquals(t, te, modified)
	t.Log("After reload: buffer contains added() and its call site")
	te.SetCursorPosition(26, 6)
	if err := te.ExecuteCommand("go-to-definition"); err != nil {
		t.Fatal(err)
	}
	AssertCursorAt(t, te, 4, 4)
	t.Log("After reload: go-to-definition finds added()")
}
