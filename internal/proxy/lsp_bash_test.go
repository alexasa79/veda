package proxy

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestBashLSPGoToDefinition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LSP integration test in short mode")
	}
	if _, err := exec.LookPath("bash-language-server"); err != nil {
		t.Skip("bash-language-server not found in PATH; install with: npm i -g bash-language-server")
	}

	pm, err := NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	_, thisFile, _, _ := runtime.Caller(0)
	testFile := filepath.Join(filepath.Dir(thisFile), "../../artifacts/bash_test_project/deploy.sh")
	testFile, err = filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to resolve test file path: %v", err)
	}

	// Read the file through the proxy to trigger LSP server start
	fileResult, err := pm.GetFile(testFile, true)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	t.Logf("Read file successfully, content length: %d", len(fileResult.Content))

	// In deploy.sh, the deploy() function calls check_dependencies on line 31 (0-indexed: 30).
	// "check_dependencies" starts at column 4.
	// The definition of check_dependencies is at line 7 (0-indexed: 6).
	callLine := 30
	callChar := 4

	// Retry GoToDefinition with backoff — the LSP server may need time to index
	var locations []Location
	for attempt := 0; attempt < 10; attempt++ {
		time.Sleep(time.Duration(500+attempt*500) * time.Millisecond)
		locations, err = pm.GoToDefinition(testFile, callLine, callChar)
		if err != nil {
			t.Logf("Attempt %d: GoToDefinition error: %v", attempt+1, err)
			continue
		}
		if len(locations) > 0 {
			break
		}
		t.Logf("Attempt %d: GoToDefinition returned 0 locations, retrying...", attempt+1)
	}

	if err != nil {
		t.Fatalf("GoToDefinition failed after retries: %v", err)
	}
	if len(locations) == 0 {
		t.Fatal("GoToDefinition returned no locations for 'check_dependencies' call")
	}

	// The definition should point to the check_dependencies function (line 7 in file, 0-indexed: 6)
	loc := locations[0]
	defLine := loc.Range.Start.Line
	t.Logf("GoToDefinition returned: line %d, char %d", defLine, loc.Range.Start.Character)

	expectedDefLine := 6
	if defLine != expectedDefLine {
		t.Errorf("Expected definition at line %d, got line %d", expectedDefLine, defLine)
	}
}
