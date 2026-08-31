package syntax

import "testing"

func TestCppFileExtensions(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{filename: "main.cpp", want: "cpp"},
		{filename: "main.cc", want: "cpp"},
		{filename: "main.cxx", want: "cpp"},
		{filename: "main.c++", want: "cpp"},
		{filename: "main.C", want: "cpp"},
		{filename: "main.hpp", want: "cpp"},
		{filename: "main.hh", want: "cpp"},
		{filename: "main.hxx", want: "cpp"},
		{filename: "main.H", want: "cpp"},
		{filename: "main.c", want: "c"},
		{filename: "main.h", want: "c"},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			highlighter := NewHighlighter()
			if err := highlighter.SetLanguageFromFile(test.filename); err != nil {
				t.Fatal(err)
			}
			if got := highlighter.GetCurrentLanguage(); got != test.want {
				t.Fatalf("language = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCppFinalAndOverrideAreKeywords(t *testing.T) {
	code := "void run() final override;"
	highlighter := NewHighlighter()
	if err := highlighter.SetLanguage("cpp"); err != nil {
		t.Fatal(err)
	}

	tokens := highlighter.GetTokensForString(code)
	for _, keyword := range []string{"final", "override"} {
		if !lineHasToken(code, tokens, keyword, TokenTypeKeyword) {
			t.Errorf("%q is not highlighted as a keyword; tokens: %v", keyword, tokens)
		}
	}
}
