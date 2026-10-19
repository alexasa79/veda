package editor

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"veda/internal/proxy"
)

func TestLSPTextObjects(t *testing.T) {
	// Skip if running in CI or short mode
	if testing.Short() {
		t.Skip("Skipping LSP integration test in short mode")
	}

	// Check if gopls is available
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH; skipping test")
	}

	// Create proxy manager
	pm, err := proxy.NewProxyManager()
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}
	defer pm.Close()

	// Copy the Go test project into a temporary directory to avoid mutating repository files
	cwd, _ := os.Getwd()
	projectDir := copyGoTestProject(t, filepath.Join(cwd, "../../artifacts/go_test_project"))
	testFile := filepath.Join(projectDir, "main.go")
	blockTestFile := filepath.Join(projectDir, "block_test.go")
	functionTestFile := filepath.Join(projectDir, "function_test.go")

	t.Logf("Using test file: %s", testFile)

	// Open the file via the proxy to trigger LSP server startup and didOpen
	_, err = pm.GetFile(testFile, true)
	if err != nil {
		t.Fatalf("Failed to open test file via proxy: %v", err)
	}

	// Give gopls time to initialize
	time.Sleep(3 * time.Second)

	t.Run("dab deletes around block", func(t *testing.T) {
		// Create content with a clear if block
		blockContent := `package main

func main() {
	if x > 0 {
		println("positive")
	}
}`
		err := os.WriteFile(blockTestFile, []byte(blockContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write block test file: %v", err)
		}

		// Notify proxy of file change
		pm.NotifyFileChange(blockTestFile, blockContent)
		time.Sleep(500 * time.Millisecond)

		// Open file via proxy to get LSP data
		fileContent, err := pm.GetFile(blockTestFile, true)
		if err != nil {
			t.Fatalf("Failed to open block test file: %v", err)
		}

		te := NewTestEditorWithContent(80, 24, fileContent.Content)

		// Set up proxy manager and file path for LSP text objects
		te.Editor.proxyManager = pm
		te.CurrentBuffer().FilePath = blockTestFile

		// Wait for LSP to populate folding ranges
		time.Sleep(1 * time.Second)

		// Get folding ranges via proxy
		foldingRanges, err := pm.GetFoldingRanges(blockTestFile)
		if err == nil {
			te.CurrentBuffer().FoldingRanges = foldingRanges
			t.Logf("Successfully got %d folding ranges", len(foldingRanges))
			for i, r := range foldingRanges {
				t.Logf("  Range %d: lines %d-%d", i, r.StartLine, r.EndLine)
			}
		} else {
			t.Logf("Failed to get folding ranges: %v", err)
		}

		// Move cursor inside the if block (line 4, inside the if statement)
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")

		t.Logf("Cursor position: line %d, char %d", te.CurrentBuffer().CursorIterator.LineNumber(), te.CurrentBuffer().CursorX)

		// Execute dab
		te.TypeKeys("dab")

		// The if block should be deleted
		expected := `package main

func main() {
}`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("dib deletes inside block", func(t *testing.T) {
		// Update block_test.go to have content for this test
		blockContent := `package main

func main() {
	if x > 0 {
		println("positive")
		return
	}
}`
		err := os.WriteFile(blockTestFile, []byte(blockContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write block test file: %v", err)
		}

		// Notify proxy of file change
		pm.NotifyFileChange(blockTestFile, blockContent)
		time.Sleep(500 * time.Millisecond)

		// Open file via proxy to get LSP data
		fileContent, err := pm.GetFile(blockTestFile, true)
		if err != nil {
			t.Fatalf("Failed to open block test file: %v", err)
		}

		te := NewTestEditorWithContent(80, 24, fileContent.Content)

		// Set up proxy manager and file path for LSP text objects
		te.Editor.proxyManager = pm
		te.CurrentBuffer().FilePath = blockTestFile

		// Get folding ranges via proxy
		foldingRanges, err := pm.GetFoldingRanges(blockTestFile)
		if err == nil {
			te.CurrentBuffer().FoldingRanges = foldingRanges
		}

		// Move cursor inside the if block (accounting for package line)
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")

		// Execute dib
		te.TypeKeys("dib")

		// The content inside the if block should be deleted (use tabs per gofmt)
		expected := `package main

func main() {
	if x > 0 {
	}
}`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("daf deletes around function", func(t *testing.T) {
		// Open file via proxy to get LSP data
		fileContent, err := pm.GetFile(functionTestFile, true)
		if err != nil {
			t.Fatalf("Failed to open function test file: %v", err)
		}

		te := NewTestEditorWithContent(80, 24, fileContent.Content)

		// Set up proxy manager and file path for LSP text objects
		te.Editor.proxyManager = pm
		te.CurrentBuffer().FilePath = functionTestFile

		// Wait for LSP to populate document symbols
		time.Sleep(1 * time.Second)

		// Get document symbols via proxy for debugging
		symbols, err := pm.GetDocumentSymbols(functionTestFile)
		if err == nil {
			t.Logf("Successfully got %d document symbols", len(symbols))
		} else {
			t.Logf("Failed to get document symbols: %v", err)
		}

		// Move cursor inside helper function (line 3, around "return 42")
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")

		// Execute daf
		te.TypeKeys("daf")

		// The calculate function should be deleted from this file
		expected := `package main

`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("dif deletes inside function", func(t *testing.T) {
		// Create a function with body for this test
		functionContent := `package main

func calculate(x, y int) int {
	sum := x + y
	product := x * y
	return sum + product
}`
		err := os.WriteFile(functionTestFile, []byte(functionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write function test file: %v", err)
		}

		// Notify proxy of file change
		pm.NotifyFileChange(functionTestFile, functionContent)
		time.Sleep(500 * time.Millisecond)

		// Open file via proxy to get LSP data
		fileContent, err := pm.GetFile(functionTestFile, true)
		if err != nil {
			t.Fatalf("Failed to open function test file: %v", err)
		}

		te := NewTestEditorWithContent(80, 24, fileContent.Content)

		// Set up proxy manager and file path for LSP text objects
		te.Editor.proxyManager = pm
		te.CurrentBuffer().FilePath = functionTestFile

		// Wait for LSP to populate document symbols
		time.Sleep(1 * time.Second)

		// Move cursor inside function body (accounting for package line)
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")
		te.ExecuteCommand("down")

		// Execute dif
		te.TypeKeys("dif")

		// The function body should be deleted
		expected := `package main

func calculate(x, y int) int {
}`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("daa deletes around argument", func(t *testing.T) {

		content := `result := calculate(10, 20, 30)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to second argument
		for i := 0; i < 24; i++ {
			te.ExecuteCommand("right")
		}

		// Execute daa - this should use the simple parsing (no proxy needed)
		te.TypeKeys("daa")

		// The second argument and surrounding commas should be deleted
		expected := `result := calculate(10, 30)`
		AssertBufferEquals(t, te, expected)
	})

	t.Run("dia deletes inside argument", func(t *testing.T) {

		content := `result := calculate(firstValue, secondValue, thirdValue)`
		te := NewTestEditorWithContent(80, 24, content)

		// Move cursor to second argument
		for i := 0; i < 32; i++ {
			te.ExecuteCommand("right")
		}

		// Execute dia - this should use the simple parsing (no proxy needed)
		te.TypeKeys("dia")

		// Only the argument text should be deleted, not the commas
		expected := `result := calculate(firstValue, , thirdValue)`
		AssertBufferEquals(t, te, expected)
	})
}

func copyGoTestProject(t *testing.T, srcDir string) string {
	t.Helper()
	destDir := t.TempDir()

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		destPath := filepath.Join(destDir, rel)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, info.Mode())
	})
	if err != nil {
		t.Fatalf("Failed to copy go test project: %v", err)
	}

	return destDir
}
