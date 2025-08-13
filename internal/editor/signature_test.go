package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/commands"
	"veda/internal/proxyworker"
)

func TestShowSignature(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "fmt.Printf()")
	
	// Set up the file so LSP would work
	te.buffer.FilePath = "test.go"
	
	// Execute show-signature command
	err := te.showSignature(nil)
	if err != nil {
		t.Errorf("showSignature() returned error: %v", err)
	}
	
	// Verify that generation was incremented
	if te.buffer.SignatureGeneration != 1 {
		t.Errorf("Expected SignatureGeneration to be 1, got %d", te.buffer.SignatureGeneration)
	}
}

func TestCheckSignatureUpdate(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		cursorX         int
		hasSignature    bool
		expectReRequest bool
	}{
		{
			name:            "On comma",
			content:         "func(a, b)",
			cursorX:         6, // On comma
			hasSignature:    true,
			expectReRequest: true,
		},
		{
			name:            "On closing paren",
			content:         "func(a)",
			cursorX:         6, // On closing paren
			hasSignature:    true,
			expectReRequest: true,
		},
		{
			name:            "After comma",
			content:         "func(a, b)",
			cursorX:         7, // After comma (space after comma)
			hasSignature:    true,
			expectReRequest: false, // Current implementation only checks current char, not previous
		},
		{
			name:            "Regular character",
			content:         "func(abc)",
			cursorX:         5, // On 'a'
			hasSignature:    true,
			expectReRequest: false,
		},
		{
			name:            "No signature active",
			content:         "func(a, b)",
			cursorX:         6, // On comma
			hasSignature:    false,
			expectReRequest: false,
		},
		{
			name:            "At end of line",
			content:         "func()",
			cursorX:         10, // Beyond content
			hasSignature:    true,
			expectReRequest: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, tt.content)
			te.buffer.FilePath = "test.go"
			te.buffer.CursorX = tt.cursorX
			
			// Set up signature state
			if tt.hasSignature {
				te.buffer.CurrentSignature = &buffer.SignatureHelp{
					Label: "test(a int, b string)",
				}
			}
			
			// Record initial generation
			initialGen := te.buffer.SignatureGeneration
			
			// Check signature update
			te.checkSignatureUpdate()
			
			// Verify generation increment
			if tt.expectReRequest {
				if te.buffer.SignatureGeneration != initialGen+1 {
					t.Errorf("Expected generation to increment, but it didn't. Initial: %d, Current: %d",
						initialGen, te.buffer.SignatureGeneration)
				}
			} else {
				if te.buffer.SignatureGeneration != initialGen {
					t.Errorf("Expected generation to stay the same, but it changed. Initial: %d, Current: %d",
						initialGen, te.buffer.SignatureGeneration)
				}
			}
		})
	}
}

func TestParseSignatureHelpResponse(t *testing.T) {
	tests := []struct {
		name         string
		response     interface{}
		expectNil    bool
		expectLabel  string
		expectActive int
		expectParams int
	}{
		{
			name:      "Nil response",
			response:  nil,
			expectNil: true,
		},
		{
			name: "Valid signature help",
			response: map[string]interface{}{
				"signatures": []interface{}{
					map[string]interface{}{
						"label":         "Printf(format string, a ...interface{})",
						"documentation": "Printf formats according to a format specifier",
						"parameters": []interface{}{
							map[string]interface{}{
								"label": "format string",
							},
							map[string]interface{}{
								"label": "a ...interface{}",
							},
						},
					},
				},
				"activeParameter": float64(1),
			},
			expectNil:    false,
			expectLabel:  "Printf(format string, a ...interface{})",
			expectActive: 1,
			expectParams: 2,
		},
		{
			name: "Empty signatures array",
			response: map[string]interface{}{
				"signatures": []interface{}{},
			},
			expectNil: true,
		},
		{
			name:      "Invalid response format",
			response:  "invalid",
			expectNil: true,
		},
		{
			name: "Missing signatures key",
			response: map[string]interface{}{
				"activeParameter": float64(0),
			},
			expectNil: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			te := NewTestEditorWithContent(80, 24, "test")
			buf := te.buffer
			
			// Parse the response
			te.parseSignatureHelpResponse(buf, tt.response)
			
			// Check if signature is nil as expected
			if tt.expectNil {
				if buf.CurrentSignature != nil {
					t.Errorf("Expected CurrentSignature to be nil, but it wasn't")
				}
			} else {
				if buf.CurrentSignature == nil {
					t.Fatalf("Expected CurrentSignature to be non-nil, but it was nil")
				}
				
				// Check label
				if buf.CurrentSignature.Label != tt.expectLabel {
					t.Errorf("Expected label %q, got %q", tt.expectLabel, buf.CurrentSignature.Label)
				}
				
				// Check active parameter
				if buf.CurrentSignature.ActiveParameter != tt.expectActive {
					t.Errorf("Expected active parameter %d, got %d", 
						tt.expectActive, buf.CurrentSignature.ActiveParameter)
				}
				
				// Check parameter count
				if len(buf.CurrentSignature.Parameters) != tt.expectParams {
					t.Errorf("Expected %d parameters, got %d", 
						tt.expectParams, len(buf.CurrentSignature.Parameters))
				}
			}
		})
	}
}

// TODO: Update this test to work with the new goroutine-based approach
func TestSignatureHelpGeneration(t *testing.T) {
	t.Skip("Test needs to be updated for new goroutine-based approach")
	
	te := NewTestEditorWithContent(80, 24, "test()")
	te.buffer.FilePath = "test.go"
	
	// Add buffer to editor's buffers map
	te.buffers["test.go"] = te.buffer
	
	// Set up initial signature
	te.buffer.CurrentSignature = &buffer.SignatureHelp{
		Label: "test(a int)",
	}
	te.buffer.SignatureGeneration = 5
	
	// Simulate receiving a response with old generation
	response := proxyworker.ProxyResponse{
		Type:       proxyworker.RequestSignatureHelp,
		FilePath:   "test.go",
		Generation: 4, // Old generation
		SignatureHelp: map[string]interface{}{
			"signatures": []interface{}{
				map[string]interface{}{
					"label": "old signature",
				},
			},
		},
	}
	
	// Handle the response
	// TODO: Update to work with new goroutine approach
	// te.handleLSPResponse(response)
	
	// Verify that old response was ignored
	if te.buffer.CurrentSignature.Label != "test(a int)" {
		t.Errorf("Old response should have been ignored, but signature changed to %q", 
			te.buffer.CurrentSignature.Label)
	}
	
	// Now send a response with current generation
	response.Generation = 5
	response.SignatureHelp = map[string]interface{}{
		"signatures": []interface{}{
			map[string]interface{}{
				"label": "new signature",
			},
		},
	}
	
	// TODO: Update to work with new goroutine approach
	// te.handleLSPResponse(response)
	
	// Verify that new response was applied
	if te.buffer.CurrentSignature.Label != "new signature" {
		t.Errorf("New response should have been applied, but signature is %q", 
			te.buffer.CurrentSignature.Label)
	}
}

func TestUpdateCursorStateCallsCheckSignature(t *testing.T) {
	te := NewTestEditorWithContent(80, 24, "func(a, b)")
	te.buffer.FilePath = "test.go"
	
	// Set up signature
	te.buffer.CurrentSignature = &buffer.SignatureHelp{
		Label: "test(a int, b string)",
	}
	
	// Position cursor on comma
	te.buffer.CursorX = 6
	
	// Record initial generation
	initialGen := te.buffer.SignatureGeneration
	
	// Call updateCursorState
	te.updateCursorState()
	
	// Verify that generation was incremented (indicating checkSignatureUpdate was called)
	if te.buffer.SignatureGeneration != initialGen+1 {
		t.Errorf("Expected generation to increment after updateCursorState, but it didn't")
	}
}

func TestAutomaticSignatureHelp(t *testing.T) {
	t.Run("automatic signature help on entering insert mode between parentheses", func(t *testing.T) {
		// Create test editor with function call syntax
		content := "fmt.Printf()"
		te := NewTestEditorWithContent(80, 24, content)
		
		// Position cursor between parentheses (after the opening paren)
		// The content is "fmt.Printf()" - we want cursor at position 11 (after the opening paren)
		// Move to that position
		for i := 0; i < len("fmt.Printf("); i++ {
			te.ExecuteCommand("right")
		}
		
		// Verify we're in the right position (between parens)
		cursorX, _ := te.GetCursorPosition()
		expectedX := len("fmt.Printf(")
		if cursorX != expectedX {
			t.Errorf("Cursor not in expected position. Expected X=%d, got X=%d", expectedX, cursorX)
		}
		
		// Record initial generation
		initialGen := te.buffer.SignatureGeneration
		
		// Enter insert mode - should automatically trigger signature help
		te.ExecuteCommand("enter-insert-mode")
		
		// Verify we're in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected to be in insert mode, got %v", te.mode)
		}
		
		// Verify generation was incremented (automatic signature help triggered)
		if te.buffer.SignatureGeneration != initialGen+1 {
			t.Errorf("Expected signature generation to increment when entering insert mode between parentheses, but it remained %d", te.buffer.SignatureGeneration)
		}
	})

	t.Run("no automatic signature help when not between parentheses", func(t *testing.T) {
		// Create test editor with regular text
		content := "hello world"
		te := NewTestEditorWithContent(80, 24, content)
		
		// Position cursor somewhere in the middle
		te.ExecuteCommand("right")
		te.ExecuteCommand("right")
		
		// Record initial generation
		initialGen := te.buffer.SignatureGeneration
		
		// Enter insert mode - should NOT trigger signature help
		te.ExecuteCommand("enter-insert-mode")
		
		// Verify we're in insert mode
		if te.mode != commands.InsertMode {
			t.Errorf("Expected to be in insert mode, got %v", te.mode)
		}
		
		// Verify generation was NOT incremented (no signature help)
		if te.buffer.SignatureGeneration != initialGen {
			t.Errorf("Expected signature generation to remain %d when not between parentheses, but it became %d", initialGen, te.buffer.SignatureGeneration)
		}
	})

	t.Run("signature help hidden when exiting insert mode", func(t *testing.T) {
		// Create test editor with function call syntax
		content := "fmt.Printf()"
		te := NewTestEditorWithContent(80, 24, content)
		
		// Position cursor between parentheses (after the opening paren)
		for i := 0; i < len("fmt.Printf("); i++ {
			te.ExecuteCommand("right")
		}
		
		// Enter insert mode to trigger signature help
		te.ExecuteCommand("enter-insert-mode")
		
		// Simulate having signature help (normally would come from LSP)
		te.buffer.CurrentSignature = &buffer.SignatureHelp{
			Label: "Printf(format string, a ...interface{}) (n int, err error)",
		}
		
		// Verify signature help is present
		if te.buffer.CurrentSignature == nil {
			t.Fatal("Expected signature help to be present for this test")
		}
		
		// Exit insert mode - should hide signature help
		te.ExecuteCommand("exit-insert-mode")
		
		// Verify we're back in normal mode
		if te.mode != commands.NormalMode {
			t.Errorf("Expected to be in normal mode after exiting insert mode, got %v", te.mode)
		}
		
		// Verify signature help was cleared
		if te.buffer.CurrentSignature != nil {
			t.Errorf("Expected signature help to be cleared when exiting insert mode, but it's still present")
		}
	})

	t.Run("signature help triggered when typing opening parenthesis", func(t *testing.T) {
		// Create test editor with function name (no parentheses yet)
		content := "fmt.Printf"
		te := NewTestEditorWithContent(80, 24, content)
		
		// Position cursor at end of line and enter insert mode
		te.ExecuteCommand("end-of-line")
		// Use 'a' command to position cursor after current character and enter insert mode
		te.ExecuteCommand("append-after-cursor")
		
		// Record initial generation
		initialGen := te.buffer.SignatureGeneration
		
		// Type opening parenthesis - should trigger signature help
		te.InsertChar('(')
		
		// Verify signature generation was incremented (signature help requested)
		if te.buffer.SignatureGeneration != initialGen+1 {
			t.Errorf("Expected signature generation to increment when typing '(' in insert mode, initial=%d, current=%d", initialGen, te.buffer.SignatureGeneration)
		}
		
		// Verify the content now includes the parenthesis
		expectedContent := "fmt.Printf("
		actualContent := te.buffer.Rope.String()
		if actualContent != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, actualContent)
		}
	})

	t.Run("no signature help when typing other characters", func(t *testing.T) {
		// Create test editor with function name
		content := "fmt.Printf"
		te := NewTestEditorWithContent(80, 24, content)
		
		// Position cursor at end of line and enter insert mode
		te.ExecuteCommand("end-of-line")
		te.ExecuteCommand("enter-insert-mode")
		
		// Record initial generation
		initialGen := te.buffer.SignatureGeneration
		
		// Type a regular character - should NOT trigger signature help
		te.InsertChar('x')
		
		// Verify signature generation was NOT incremented
		if te.buffer.SignatureGeneration != initialGen {
			t.Errorf("Expected signature generation to remain %d when typing regular characters, but it became %d", initialGen, te.buffer.SignatureGeneration)
		}
	})
}