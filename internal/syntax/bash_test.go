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
		{"#!/bin/bash", TokenTypeBashComment, false},
		{"# This is a comment", TokenTypeBashComment, false},
		{"NAME", TokenTypeBashVariable, false},
		{"Hello World", TokenTypeBashString, true}, // String content
		{"echo", TokenTypeBashBuiltin, false},
		{"NAME", TokenTypeBashVariable, true}, // Variable reference (part of $NAME)
		{"if", TokenTypeBashKeyword, false},
		{"then", TokenTypeBashKeyword, false},
		{"fi", TokenTypeBashKeyword, false},
		{"function", TokenTypeBashKeyword, false},
		{"test", TokenTypeBashFunction, false},
		{"local", TokenTypeBashKeyword, false},
		{"return", TokenTypeBashKeyword, false},
		{"for", TokenTypeBashKeyword, false},
		{"do", TokenTypeBashKeyword, false},
		{"done", TokenTypeBashKeyword, false},
		{"-eq", TokenTypeBashOperator, false},
		{"?", TokenTypeBashSpecial, true}, // Part of $?
		{"1", TokenTypeBashVariable, true}, // Part of $1
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
		
		// Verify that bash-specific tokens get bash-specific styles
		switch tc.tokenType {
		case TokenTypeBashKeyword:
			if style != testTheme.BashKeyword {
				t.Errorf("Token %s should use BashKeyword style", tc.text)
			}
		case TokenTypeBashBuiltin:
			if style != testTheme.BashBuiltin {
				t.Errorf("Token %s should use BashBuiltin style", tc.text)
			}
		case TokenTypeBashVariable:
			if style != testTheme.BashVariable {
				t.Errorf("Token %s should use BashVariable style", tc.text)
			}
		case TokenTypeBashString:
			if style != testTheme.BashString {
				t.Errorf("Token %s should use BashString style", tc.text)
			}
		case TokenTypeBashComment:
			if style != testTheme.BashComment {
				t.Errorf("Token %s should use BashComment style", tc.text)
			}
		case TokenTypeBashFunction:
			if style != testTheme.BashFunction {
				t.Errorf("Token %s should use BashFunction style", tc.text)
			}
		case TokenTypeBashSpecial:
			if style != testTheme.BashSpecial {
				t.Errorf("Token %s should use BashSpecial style", tc.text)
			}
		case TokenTypeBashOperator:
			if style != testTheme.BashOperator {
				t.Errorf("Token %s should use BashOperator style", tc.text)
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