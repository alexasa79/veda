package editor

import (
	"testing"
	"veda/internal/buffer"
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

func TestSignatureHelpGeneration(t *testing.T) {
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
	te.handleLSPResponse(response)
	
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
	
	te.handleLSPResponse(response)
	
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