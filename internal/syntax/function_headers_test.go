package syntax

import (
	"reflect"
	"strings"
	"testing"
)

func TestFunctionHeaderRanges(t *testing.T) {
	tests := []struct {
		name, filename, source string
		headers                []string
	}{
		{"cpp multiline", "sample.cc", "Parcel::Result\nDepot::Load(int count) {\n  return count;\n}\n", []string{"Parcel::Result\nDepot::Load(int count) {"}},
		{"c separate brace", "sample.c", "int measure(\n  int length)\n{\n  return length;\n}\n", []string{"int measure(\n  int length)\n{"}},
		{"empty inline body", "sample.cc", "void reset() {}\n", []string{"void reset() {}"}},
		{"empty constructor", "sample.cc", "Parcel::Parcel(int size) : size{size} {}\n", []string{"Parcel::Parcel(int size) : size{size} {}"}},
		{"empty multiline body", "sample.cc", "void reset() {\n}\n", []string{"void reset() {"}},
		{"comment in body", "sample.cc", "void reset() { /* keep this */ }\n", []string{"void reset() { /* keep this */ }"}},
		{"arrow expression", "sample.js", "const identity = value => value;\n", []string{"value =>"}},
		{"inline body", "sample.cc", "int measure() { return 7; }\n", []string{"int measure() { return 7; }"}},
		{"default initializer", "sample.cc", "void save(Value value = Value{}) {\n  use(value);\n}\n", []string{"void save(Value value = Value{}) {"}},
		{"constructor initializer", "sample.cc", "Entry::Entry() : value{1} {\n  initialize();\n}\n", []string{"Entry::Entry() : value{1} {"}},
		{"template", "sample.cc", "template <typename T>\nT identity(T value) {\n  return value;\n}\n", []string{"template <typename T>\nT identity(T value) {"}},
		{"python decorated", "sample.py", "@cache\ndef measure(\n    length: int,\n) -> int:\n    return length\n", []string{"@cache\ndef measure(\n    length: int,\n) -> int:"}},
		{"python inline", "sample.py", "def measure(): return 7\n", []string{"def measure():"}},
		{"go method", "sample.go", "package sample\nfunc (p *Parcel) Size() int {\n  return 7\n}\n", []string{"func (p *Parcel) Size() int {"}},
		{"javascript arrow", "sample.js", "const twice = (value) => { return value * 2; };\n", []string{"(value) => { return value * 2; }"}},
		{"javascript method", "sample.js", "class Parcel { size() { return 7; } }\n", []string{"size() { return 7; }"}},
		{"nested lambda", "sample.cc", "void run() { auto work = []() { return 7; }; }\n", []string{"void run() { auto work = []() { return 7; }; }"}},
		{"nested lambda multiline parent", "sample.cc", "void run() {\n  auto work = []() { return 7; };\n}\n", []string{"void run() {", "[]() { return 7; }"}},
		{"declaration only", "sample.cc", "int measure();\nint value = measure();\n", nil},
		{"c loop macro", "sample.c", "void visit_items(void)\n{\n  array_each(items, i, item)\n  {\n    consume(item);\n  }\n}\n", []string{"void visit_items(void)\n{"}},
		{"cpp loop macro", "sample.cc", "void visit_items() {\n  array_each(items, i, item) /* each item */\n  {\n    consume(item);\n  }\n}\n", []string{"void visit_items() {"}},
		{"cpp nested loop macros", "sample.cc", "void visit_items() {\n  array_each(items, i, item)\n  {\n    array_each(item, j, value) {\n      consume(value);\n    }\n  }\n}\n", []string{"void visit_items() {"}},
		{"cpp local class", "sample.cc", "void run() {\n  struct Local {\n    Local() {}\n    int size() { return 7; }\n  };\n}\n", []string{"void run() {", "Local() {}", "int size() { return 7; }"}},
		{"comment and string", "sample.cc", "// int measure() { return 7; }\nconst char* text = \"void run() {}\";\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHighlighter()
			if err := h.SetLanguageFromFile(tt.filename); err != nil {
				t.Fatal(err)
			}
			if err := h.ParseFull([]byte(tt.source)); err != nil {
				t.Fatal(err)
			}
			var headers []string
			for _, span := range h.FunctionHeaderRanges(0, len(tt.source)) {
				headers = append(headers, tt.source[span.Start:span.End])
			}
			if !reflect.DeepEqual(headers, tt.headers) {
				t.Fatalf("headers = %q; want %q\ntree: %s", headers, tt.headers, h.currentTree.RootNode())
			}
			for offset := range []byte(tt.source) {
				want := false
				for _, header := range tt.headers {
					start := strings.Index(tt.source, header)
					want = want || offset >= start && offset < start+len(header)
				}
				if got := len(h.FunctionHeaderRanges(offset, offset+1)) > 0; got != want {
					t.Fatalf("byte %d: highlighted = %v; want %v", offset, got, want)
				}
			}
		})
	}
}

func TestFunctionHeaderRangesAfterEdit(t *testing.T) {
	h := NewHighlighter()
	if err := h.SetLanguageFromFile("sample.cc"); err != nil {
		t.Fatal(err)
	}
	if err := h.ParseFull([]byte("int measure() { return 7; }\n")); err != nil {
		t.Fatal(err)
	}
	source := "// \u03c0\nint\nmeasure() {\n  return 7;\n}\n"
	if err := h.ReparseIncremental([]byte(source)); err != nil {
		t.Fatal(err)
	}
	start := strings.Index(source, "measure")
	end := strings.Index(source, "{") + 1
	got := h.FunctionHeaderRanges(start, end)
	want := []ByteRange{{start, end}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranges = %v; want %v", got, want)
	}
	body := strings.Index(source, "return")
	if got := h.FunctionHeaderRanges(body, body+6); len(got) != 0 {
		t.Fatalf("body has header ranges: %v", got)
	}
	if got := NewHighlighter().FunctionHeaderRanges(0, 10); len(got) != 0 {
		t.Fatalf("unparsed source has header ranges: %v", got)
	}
}
