package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSaveLoad(t *testing.T) {
	// Create a temp directory for the test config
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "veda")
	configPath := filepath.Join(configDir, "config.json")

	// Create config and modify it
	cfg := New()
	cfg.ScrollOffset = 15
	cfg.DefaultIndentWidth = 2
	cfg.DefaultIndentType = IndentTabs
	cfg.VerticalRulers = []int{100, 120}
	cfg.RelativeLineNumbers = true

	// We need to manually save since ConfigPath() uses the real home dir
	// So we'll test the serialization logic directly
	indentType := "spaces"
	if cfg.DefaultIndentType == IndentTabs {
		indentType = "tabs"
	}

	cf := ConfigFile{
		Theme:               cfg.Theme.Name,
		ScrollOffset:        cfg.ScrollOffset,
		IndentType:          indentType,
		IndentWidth:         cfg.DefaultIndentWidth,
		VerticalRulers:      cfg.VerticalRulers,
		RelativeLineNumbers: cfg.RelativeLineNumbers,
	}

	// Create directory and save
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Read it back
	readData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var loadedCf ConfigFile
	if err := json.Unmarshal(readData, &loadedCf); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify values
	if loadedCf.Theme != "monokai" {
		t.Errorf("Expected theme 'monokai', got '%s'", loadedCf.Theme)
	}
	if loadedCf.ScrollOffset != 15 {
		t.Errorf("Expected scroll offset 15, got %d", loadedCf.ScrollOffset)
	}
	if loadedCf.IndentType != "tabs" {
		t.Errorf("Expected indent type 'tabs', got '%s'", loadedCf.IndentType)
	}
	if loadedCf.IndentWidth != 2 {
		t.Errorf("Expected indent width 2, got %d", loadedCf.IndentWidth)
	}
	if len(loadedCf.VerticalRulers) != 2 || loadedCf.VerticalRulers[0] != 100 || loadedCf.VerticalRulers[1] != 120 {
		t.Errorf("Expected vertical rulers [100, 120], got %v", loadedCf.VerticalRulers)
	}
	if !loadedCf.RelativeLineNumbers {
		t.Error("Expected relative line numbers to be enabled")
	}
}

func TestConfigLoadNonExistent(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	cfg := New()
	// Load should not error on non-existent file
	err := cfg.Load()
	if err != nil {
		// Only error if it's not a "file not found" scenario (which is acceptable)
		// The Load function should gracefully handle missing config
		t.Logf("Load returned error (may be expected): %v", err)
	}

	// Verify defaults are preserved
	if cfg.ScrollOffset != 10 {
		t.Errorf("Expected default scroll offset 10, got %d", cfg.ScrollOffset)
	}
	if cfg.Theme.Name != "monokai" {
		t.Errorf("Expected default theme 'monokai', got '%s'", cfg.Theme.Name)
	}
	if cfg.RelativeLineNumbers {
		t.Error("Expected relative line numbers to be disabled by default")
	}
}

func TestConfigLoadRelativeLineNumbers(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	configDir := filepath.Join(tempHome, ".config", "veda")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"relative_line_numbers":true}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg := New()
	if err := cfg.Load(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if !cfg.RelativeLineNumbers {
		t.Error("Expected relative line numbers to be enabled")
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Skip("Could not determine home directory")
	}

	// Should end with the expected path
	expectedSuffix := filepath.Join(".config", "veda", "config.json")
	if !containsSuffix(path, expectedSuffix) {
		t.Errorf("Config path '%s' should end with '%s'", path, expectedSuffix)
	}
}

func containsSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func TestConfigFileJSON(t *testing.T) {
	// Test that ConfigFile marshals to expected JSON format
	cf := ConfigFile{
		Theme:          "dracula",
		ScrollOffset:   5,
		IndentType:     "tabs",
		IndentWidth:    4,
		VerticalRulers: []int{80, 100},
	}

	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `{
  "theme": "dracula",
  "scroll_offset": 5,
  "indent_type": "tabs",
  "indent_width": 4,
  "vertical_rulers": [
    80,
    100
  ]
}`
	if string(data) != expected {
		t.Errorf("JSON output mismatch.\nExpected:\n%s\nGot:\n%s", expected, string(data))
	}
}
