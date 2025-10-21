package editor

import (
	"os"
	"testing"
)

// TestMain runs before all tests in the package and handles cleanup
func TestMain(m *testing.M) {
	// Run all tests
	code := m.Run()

	// Clean up test fixtures after all tests complete
	CleanupTestFixtures()

	// Exit with the test result code
	os.Exit(code)
}
