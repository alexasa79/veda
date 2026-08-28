package editor

import (
	"testing"
	"veda/internal/buffer"
	"veda/internal/config"
	"veda/internal/syntax"
)

func TestCppContextNodes(t *testing.T) {
	content := `namespace demo {
template <typename T>
class Worker {
public:
	void run(const T &items) {
		auto task = [this]() {
			for (const auto &item : items) {
				try {
					use(item);
				} catch (...) {
					recover();
				}
			}
		};
		task();
	}
private:
	int state;
};
}`

	buf := buffer.NewBuffer(content, "test.cpp", "test.cpp", config.IndentTabs, 4)
	highlighter := syntax.NewHighlighter()
	if err := highlighter.SetLanguageFromFile("test.cpp"); err != nil {
		t.Fatal(err)
	}
	if err := highlighter.ParseFull([]byte(content)); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		cursorLine int
		viewStart  int
		viewEnd    int
		want       []int
	}{
		{
			name:       "nested C++ constructs",
			cursorLine: 10,
			viewStart:  10,
			viewEnd:    12,
			want:       []int{0, 1, 2, 4, 5, 6, 7, 9},
		},
		{
			name:       "class field",
			cursorLine: 17,
			viewStart:  17,
			viewEnd:    19,
			want:       []int{0, 1, 2},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := getTreeSitterContext(buf, highlighter, test.cursorLine, test.viewStart, test.viewEnd, 10)
			assertContextLines(t, got, test.want)
		})
	}
}

func TestCppTemplateAndStructUseSeparateContextLines(t *testing.T) {
	content := `template<class T>
struct red_per_lcore_t {
	T value;
};`

	buf := buffer.NewBuffer(content, "test.cpp", "test.cpp", config.IndentTabs, 4)
	highlighter := syntax.NewHighlighter()
	if err := highlighter.SetLanguageFromFile("test.cpp"); err != nil {
		t.Fatal(err)
	}
	if err := highlighter.ParseFull([]byte(content)); err != nil {
		t.Fatal(err)
	}

	got := getTreeSitterContext(buf, highlighter, 2, 2, 3, 10)
	assertContextLines(t, got, []int{0, 1})
}

func TestCppContextNodesDoNotAffectJavaScript(t *testing.T) {
	content := `function example() {
	try {
		work();
	} catch (error) {
		recover(error);
	}
}`

	buf := buffer.NewBuffer(content, "test.js", "test.js", config.IndentTabs, 4)
	highlighter := syntax.NewHighlighter()
	if err := highlighter.SetLanguageFromFile("test.js"); err != nil {
		t.Fatal(err)
	}
	if err := highlighter.ParseFull([]byte(content)); err != nil {
		t.Fatal(err)
	}

	got := getTreeSitterContext(buf, highlighter, 4, 4, 5, 10)
	assertContextLines(t, got, []int{0})
}

func assertContextLines(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("context lines = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("context lines = %v, want %v", got, want)
		}
	}
}
