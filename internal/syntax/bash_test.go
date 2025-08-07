package syntax

import (
	"strings"
	"testing"
	"veda/internal/theme"
)

func TestBashSyntaxHighlighting(t *testing.T) {
	// Create a highlighter
	h := NewHighlighter()
	
	// Set language to bash
	err := h.SetLanguage("bash")
	if err != nil {
		t.Fatalf("Failed to set language to bash: %v", err)
	}
	
	// Test bash script
	bashCode := `#!/bin/bash

# This is a comment
NAME="Hello World"
echo $NAME

if [ $? -eq 0 ]; then
    echo "Success"
fi

function test() {
    local var=$1
    return 0
}

for i in {1..5}; do
    echo "Count: $i"
done`

	// Parse the code
	h.ParseFull([]byte(bashCode))
	
	// Test cases for different token types
	testCases := []struct {
		text      string
		tokenType TokenType
		partial   bool // If true, match substring instead of exact match
	}{
		{"#!/bin/bash", TokenTypeComment, false},
		{"# This is a comment", TokenTypeComment, false},
		{"NAME", TokenTypeVariable, false},
		{"Hello World", TokenTypeString, true}, // String content
		{"echo", TokenTypeFunction, false}, // echo is parsed as a function by tree-sitter
		{"NAME", TokenTypeVariable, true}, // Variable reference (part of $NAME)
		{"if", TokenTypeKeyword, false},
		{"then", TokenTypeKeyword, false},
		{"fi", TokenTypeKeyword, false},
		{"function", TokenTypeKeyword, false},
		{"test", TokenTypeFunction, false},
		{"local", TokenTypeKeyword, false},
		{"return", TokenTypeKeyword, false},
		{"for", TokenTypeKeyword, false},
		{"do", TokenTypeKeyword, false},
		{"done", TokenTypeKeyword, false},
		{"-eq", TokenTypeOperator, false},
		{"?", TokenTypeConstant, true}, // Part of $? - special variable parsed as constant
		{"1", TokenTypeVariable, true}, // Part of $1
	}
	
	// Check each test case
	for _, tc := range testCases {
		found := false
		var foundToken *Token
		
		// Find the token containing our test text
		for lineNum := 0; lineNum < 20; lineNum++ { // Check first 20 lines
			lineStart := 0
			if lineNum > 0 {
				// Calculate line start position
				pos := 0
				for i := 0; i < lineNum; i++ {
					for pos < len(bashCode) && bashCode[pos] != '\n' {
						pos++
					}
					if pos < len(bashCode) {
						pos++ // Skip the newline
					}
				}
				lineStart = pos
			}
			
			// Calculate line end
			lineEnd := lineStart
			for lineEnd < len(bashCode) && bashCode[lineEnd] != '\n' {
				lineEnd++
			}
			
			tokens := h.GetTokensForLine(lineStart, lineEnd)
			for _, token := range tokens {
				tokenText := string(bashCode[lineStart+token.Start : lineStart+token.End])
				if tc.partial {
					// For partial matches, check if the text is contained in the token
					if strings.Contains(tokenText, tc.text) {
						found = true
						foundToken = &token
						break
					}
				} else {
					// For exact matches
					if tokenText == tc.text {
						found = true
						foundToken = &token
						break
					}
				}
			}
			if found {
				break
			}
		}
		
		if !found {
			t.Errorf("Token not found for text: %s", tc.text)
			continue
		}
		
		if foundToken.TokenType != tc.tokenType {
			t.Errorf("Wrong token type for %s: got %v, want %v", tc.text, foundToken.TokenType, tc.tokenType)
		}
		
		// Verify that GetStyleForToken returns the correct style
		testTheme := theme.NewTheme()
		style := GetStyleForToken(foundToken.TokenType, testTheme)
		
		// Verify that tokens get appropriate styles
		switch tc.tokenType {
		case TokenTypeKeyword:
			if style != testTheme.SyntaxKeyword {
				t.Errorf("Token %s should use SyntaxKeyword style", tc.text)
			}
		case TokenTypeVariable:
			if style != testTheme.SyntaxVariable {
				t.Errorf("Token %s should use SyntaxVariable style", tc.text)
			}
		case TokenTypeString:
			if style != testTheme.SyntaxString {
				t.Errorf("Token %s should use SyntaxString style", tc.text)
			}
		case TokenTypeComment:
			if style != testTheme.SyntaxComment {
				t.Errorf("Token %s should use SyntaxComment style", tc.text)
			}
		case TokenTypeFunction:
			if style != testTheme.SyntaxFunction {
				t.Errorf("Token %s should use SyntaxFunction style", tc.text)
			}
		case TokenTypeOperator:
			if style != testTheme.SyntaxOperator {
				t.Errorf("Token %s should use SyntaxOperator style", tc.text)
			}
		case TokenTypeConstant:
			if style != testTheme.SyntaxConstant {
				t.Errorf("Token %s should use SyntaxConstant style", tc.text)
			}
		}
	}
}

func TestBashFileDetection(t *testing.T) {
	h := NewHighlighter()
	
	testCases := []struct {
		filename string
		shouldBeBash bool
	}{
		{"script.sh", true},
		{"script.bash", true},
		{".bashrc", true},
		{".bash_profile", true},
		{"script.py", false},
		{"script.go", false},
		{"README.md", false},
	}
	
	for _, tc := range testCases {
		err := h.SetLanguageFromFile(tc.filename)
		
		if tc.shouldBeBash {
			if err != nil {
				t.Errorf("Failed to set language for %s: %v", tc.filename, err)
				continue
			}
			
			// Check that bash language was loaded by verifying we can parse bash code
			h.ParseFull([]byte("echo test"))
			if h.currentTree == nil {
				t.Errorf("Language not properly set for %s - couldn't parse bash code", tc.filename)
			}
		}
	}
}