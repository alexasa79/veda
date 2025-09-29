package indent

import (
	"strings"
	"testing"
)

func TestCLanguageTokenizer_ProducesCommentTokens(t *testing.T) {
	code := "int main() {\n    /* Comment heading\n       next word */\n    return 0;\n}\n"

	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	var seenStart, seenWord, seenEnd bool
	for _, tok := range tokens {
		switch tok.Kind {
		case TokenCommentStart:
			seenStart = true
			if tok.Start.Line != 1 {
				t.Fatalf("expected block comment start on line 1, got %d", tok.Start.Line)
			}
		case TokenCommentWord:
			seenWord = true
		case TokenCommentEnd:
			seenEnd = true
		}
	}

	if !seenStart || !seenWord || !seenEnd {
		t.Fatalf("expected comment tokens start=%v word=%v end=%v", seenStart, seenWord, seenEnd)
	}
}

func TestAnalyzer_IndentLevelsForBlocks(t *testing.T) {
	code := "void f() {\n    if (foo) {\n        bar();\n    }\n}\n"

	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	annotated, err := NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	got := make(map[int]int)
	for _, at := range annotated {
		if at.Token.Kind == TokenWhitespace || at.Token.Kind == TokenNewline {
			continue
		}
		line := at.Token.Start.Line
		if _, ok := got[line]; !ok {
			got[line] = at.IndentLevel
		}
	}

	want := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 1,
		4: 0,
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected number of lines; got %v", got)
	}

	for line, exp := range want {
		if got[line] != exp {
			t.Fatalf("line %d indent mismatch: got %d want %d", line, got[line], exp)
		}
	}
}

func TestAnalyzer_ContinuationLevels(t *testing.T) {
	code := "int sum = add(\n    a,\n    b);\n"

	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	annotated, err := NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	lineContinuations := make(map[int]int)
	closingLevel := -1
	for _, at := range annotated {
		if at.Token.Kind == TokenWhitespace || at.Token.Kind == TokenNewline {
			continue
		}
		line := at.Token.Start.Line
		if _, ok := lineContinuations[line]; !ok {
			lineContinuations[line] = at.ContinuationLevel
		}
		if at.Token.Kind == TokenCloseParen {
			closingLevel = at.ContinuationLevel
		}
	}

	if lineContinuations[1] != 1 {
		t.Fatalf("expected continuation level 1 on line 1, got %d", lineContinuations[1])
	}
	if lineContinuations[2] != 1 {
		t.Fatalf("expected continuation level 1 for continued arguments on line 2, got %d", lineContinuations[2])
	}
	if closingLevel != 0 {
		t.Fatalf("expected closing parenthesis continuation level 0, got %d", closingLevel)
	}
}

func TestAnalyzer_UnmatchedAnchorError(t *testing.T) {
	code := "int main() {\n"

	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	_, err = NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err == nil {
		t.Fatalf("expected unmatched anchor error")
	}
}

func TestCLanguageTokenizer_PreprocessorDirective(t *testing.T) {
	code := "#define MAX_VALUE 10\nint value = MAX_VALUE;\n"
	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	first := firstNonWhitespace(tokens)
	if first == nil {
		t.Fatalf("expected non-whitespace tokens")
	}
	if first.Kind != TokenPreprocessor {
		t.Fatalf("expected first token to be preprocessor, got %v", first.Kind)
	}
	if first.Lexeme != "#define MAX_VALUE 10" {
		t.Fatalf("unexpected preprocessor lexeme: %q", first.Lexeme)
	}
}

func TestCLanguageTokenizer_TernaryQuestionAndColon(t *testing.T) {
	code := "int x = cond ? left : right;\n"
	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	var seenQuestion, seenColon bool
	for _, tok := range tokens {
		if tok.Kind == TokenQuestion {
			seenQuestion = true
		}
		if tok.Kind == TokenColon {
			seenColon = true
		}
	}
	if !seenQuestion || !seenColon {
		t.Fatalf("expected to see ternary question and colon tokens, got question=%v colon=%v", seenQuestion, seenColon)
	}
}

func TestCLanguageTokenizer_LineCommentWords(t *testing.T) {
	code := "// first comment line\nint x;\n"
	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	var wordCount int
	for _, tok := range tokens {
		if tok.Kind == TokenCommentWord {
			wordCount++
		}
	}
	if wordCount == 0 {
		t.Fatalf("expected comment word tokens, got 0")
	}
}

func firstNonWhitespace(tokens []Token) *Token {
	for i := range tokens {
		switch tokens[i].Kind {
		case TokenWhitespace, TokenNewline:
			continue
		default:
			return &tokens[i]
		}
	}
	return nil
}

func TestAnalyzer_BlockExitIndent(t *testing.T) {
	code := strings.Join([]string{
		"if (foo) {",
		"    baz();",
		"}",
		"    bar();",
	}, "\n") + "\n"

	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	annotated, err := NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	lineIndent := make(map[int]int)
	for _, at := range annotated {
		if at.Token.Kind == TokenWhitespace || at.Token.Kind == TokenNewline {
			continue
		}
		if _, ok := lineIndent[at.Token.Start.Line]; !ok {
			lineIndent[at.Token.Start.Line] = at.IndentLevel
		}
	}

	val, ok := lineIndent[3]
	if !ok {
		t.Fatalf("expected analyzer to report indent for line 3")
	}
	if val != 0 {
		t.Fatalf("expected line 3 indent level 0, got %d", val)
	}
}

func TestAnalyzer_MultilineConditionIndent(t *testing.T) {
	code := "void\nqux_override(void)\n{\n    if (FLAG_ZERO ||\n        mode_kind == MODE_PRIMARY ||\n        mode_kind == MODE_TRANSITION)\n    {\n        if (ctx.link_active) {\n            trace(LOG_INFO,\n                  \"placeholder\",\n                  mode_label(mode_kind));\n            ctx.link_active = FLAG_ZERO;\n        }\n    }\n}\n"

	lang := NewCLanguage()
	tokens, err := lang.Tokenize([]rune(code))
	if err != nil {
		t.Fatalf("tokenize failed: %v", err)
	}

	annotated, err := NewAnalyzer(tokens, lang.Rules()).Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	indentPerLine := make(map[int]int)
	contPerLine := make(map[int]int)
	for _, at := range annotated {
		if at.Token.Kind == TokenWhitespace || at.Token.Kind == TokenNewline {
			continue
		}
		line := at.Token.Start.Line
		if _, ok := indentPerLine[line]; !ok {
			indentPerLine[line] = at.IndentLevel
			contPerLine[line] = at.ContinuationLevel
		}
	}

	expectIndent := map[int]int{
		0:  0, // void
		1:  0, // qux_override(void)
		2:  0, // {
		3:  1, // if line
		4:  1, // continuation condition
		5:  1, // final condition + closing paren
		6:  1, // opening brace aligned with if
		7:  2, // inner if
		8:  3, // trace call line inside inner block
		9:  3, // continuation of trace
		10: 3, // continuation of trace
		11: 3, // assignment inside inner if
		12: 2, // closing inner brace
		13: 1, // closing outer brace
		14: 0, // closing function brace
	}

	for line, want := range expectIndent {
		if got := indentPerLine[line]; got != want {
			t.Fatalf("line %d indent mismatch: got %d want %d", line, got, want)
		}
	}

	if contPerLine[4] != 1 || contPerLine[5] != 1 {
		t.Fatalf("expected continuation level 1 on multiline condition lines, got %d and %d", contPerLine[4], contPerLine[5])
	}
	if contPerLine[6] != 0 {
		t.Fatalf("expected continuation to reset when brace line encountered, got %d", contPerLine[6])
	}
}
