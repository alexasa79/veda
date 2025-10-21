package syntax

import "testing"

func TestFishSyntaxHighlighting(t *testing.T) {
	h := NewHighlighter()

	if err := h.SetLanguage("fish"); err != nil {
		t.Fatalf("failed to set language to fish: %v", err)
	}

	fishCode := `# Fish greeting
function greet
	set -l name "World"
	if test (count $argv) -gt 0
		set name $argv[1]
	else
		set name "Fallback"
	end

	command echo "Hello $name"
	return 0
end
`

	h.ParseFull([]byte(fishCode))

	type expectation struct {
		text      string
		tokenType TokenType
	}

	tokens := h.GetTokensForString(fishCode)

	expectations := []expectation{
		{"# Fish greeting", TokenTypeComment},
		{"function", TokenTypeKeyword},
		{"greet", TokenTypeFunction},
		{"set", TokenTypeFunction},
		{"$argv[1]", TokenTypeVariable},
		{"\"World\"", TokenTypeString},
		{"if", TokenTypeKeyword},
		{"else", TokenTypeKeyword},
		{"command", TokenTypeFunction},
		{"return", TokenTypeKeyword},
	}

	for _, exp := range expectations {
		found := false
		for _, token := range tokens {
			if token.Start < 0 || token.End > len(fishCode) || token.Start >= token.End {
				continue
			}
			tokenText := fishCode[token.Start:token.End]
			if tokenText == exp.text {
				if token.TokenType != exp.tokenType {
					t.Errorf("wrong token type for %q: got %v, want %v", exp.text, token.TokenType, exp.tokenType)
				}
				found = true
				break
			}
		}
		if !found {
			for _, token := range tokens {
				if token.Start < 0 || token.End > len(fishCode) || token.Start >= token.End {
					continue
				}
				tokenText := fishCode[token.Start:token.End]
				t.Logf("available token %q -> %s", tokenText, token.TokenType)
			}
			t.Errorf("token %q not found", exp.text)
		}
	}
}
