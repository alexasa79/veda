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

func TestHeaderLanguageFromContent(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{"empty", "", "c"},
		{"shared declarations", "struct Entry { int value; };\nint read_entry(void);", "c"},
		{"C keywords as identifiers", "struct Entry { int class; int namespace; int using; int new; };", "c"},
		{"C tag and typedef", "typedef struct class { int template; } namespace;", "c"},
		{"C function names", "int using(int namespace);\nint new(void);", "c"},
		{"C bitwise expression", "int mask = FIRST & SECOND;", "c"},
		{"comments", "// namespace example {\n/* template <class T> class Example {}; */\nint value;", "c"},
		{"strings", "const char *text = \"using Value = int; namespace example {}\";", "c"},
		{"macro body", "#define DESCRIPTION namespace example {\nint value;", "c"},
		{"macro continuation", "#define DESCRIPTION \\\nusing Value = int;\nint value;", "c"},
		{"C++ compatibility guard", "#ifdef __cplusplus\nextern \"C\" {\n#endif\nint read_entry(void);\n#ifdef __cplusplus\n}\n#endif\n", "c"},
		{"class", "class Entry { int value; };", "cpp"},
		{"forward class", "class Entry;", "cpp"},
		{"namespace", "namespace example { int value; }", "cpp"},
		{"namespace alias", "namespace ex = example;", "cpp"},
		{"template", "template <typename T> struct Holder { T value; };", "cpp"},
		{"alias", "using Value = unsigned long;", "cpp"},
		{"using declaration", "using example::value;", "cpp"},
		{"using namespace", "using namespace example;", "cpp"},
		{"scoped enum", "enum class State { Ready, Done };", "cpp"},
		{"scoped struct enum", "enum struct State { Ready, Done };", "cpp"},
		{"qualified type", "std::string label;", "cpp"},
		{"reference", "int &get_value();", "cpp"},
		{"access specifier", "struct Entry { public: int value; };", "cpp"},
		{"include guard", "#ifndef EXAMPLE_H\n#define EXAMPLE_H\nusing Value = int;\n#endif\n", "cpp"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := NewHighlighter()
			if err := h.SetLanguageFromFileAndContent("example.h", []byte(test.code)); err != nil {
				t.Fatal(err)
			}
			if got := h.GetCurrentLanguage(); got != test.want {
				t.Fatalf("language = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCppHeaderUsingIsHighlighted(t *testing.T) {
	code := "using Index = unsigned long;"
	h := NewHighlighter()
	if err := h.SetLanguageFromFileAndContent("example.h", []byte(code)); err != nil {
		t.Fatal(err)
	}
	if err := h.ParseFull([]byte(code)); err != nil {
		t.Fatal(err)
	}
	if tokens := h.GetTokensForLine(0, len(code)); !lineHasToken(code, tokens, "using", TokenTypeKeyword) {
		t.Fatalf("using is not highlighted as a keyword: %v", tokens)
	}
}
