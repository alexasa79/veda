package syntax

import (
	"strings"
	"testing"
)

func TestMarkdownConcealRangesForLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "bold", line: "Before **bold** after", want: "Before bold after"},
		{name: "italic", line: "Before *italic* after", want: "Before italic after"},
		{name: "bold italic", line: "Before ***both*** after", want: "Before both after"},
		{name: "inline code", line: "Before `code` after", want: "Before code after"},
		{name: "strikethrough", line: "Before ~~deleted~~ after", want: "Before deleted after"},
		{name: "inline link", line: "Read [the docs](docs.md) now", want: "Read the docs now"},
		{name: "shortcut link", line: "Read [the docs] now", want: "Read the docs now"},
		{name: "image fallback", line: "See ![the diagram](diagram.png)", want: "See the diagram"},
		{name: "heading", line: "## Heading", want: "Heading"},
		{name: "blockquote", line: "> quoted", want: "quoted"},
		{name: "list marker remains", line: "- item", want: "- item"},
		{name: "unchecked task remains", line: "- [ ] task", want: "- [ ] task"},
		{name: "checked task remains", line: "- [x] task", want: "- [x] task"},
		{name: "unmatched delimiter remains", line: "Before **unfinished", want: "Before **unfinished"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHighlighter()
			if err := h.SetLanguage("markdown"); err != nil {
				t.Fatal(err)
			}
			if err := h.ParseFull([]byte(tt.line)); err != nil {
				t.Fatal(err)
			}

			ranges := h.MarkdownConcealRangesForLine(0, len(tt.line))
			var got strings.Builder
			rangeIndex := 0
			for i := 0; i < len(tt.line); i++ {
				for rangeIndex < len(ranges) && i >= ranges[rangeIndex].End {
					rangeIndex++
				}
				if rangeIndex < len(ranges) && i >= ranges[rangeIndex].Start {
					continue
				}
				got.WriteByte(tt.line[i])
			}
			if got.String() != tt.want {
				t.Fatalf("got %q, want %q; ranges=%v", got.String(), tt.want, ranges)
			}
		})
	}
}

func TestMarkdownConcealPreservesFencedCodeContent(t *testing.T) {
	content := "```\n**not bold**\n```"
	h := NewHighlighter()
	if err := h.SetLanguage("markdown"); err != nil {
		t.Fatal(err)
	}
	if err := h.ParseFull([]byte(content)); err != nil {
		t.Fatal(err)
	}

	lineStart := len("```\n")
	line := "**not bold**"
	ranges := h.MarkdownConcealRangesForLine(lineStart, lineStart+len(line))
	if len(ranges) != 0 {
		t.Fatalf("fenced code content has conceal ranges %v", ranges)
	}
}

func TestMarkdownImageForLine(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantPath    string
		wantAlt     string
		wantPresent bool
	}{
		{name: "standalone", content: "![Architecture](images/architecture.png)", wantPath: "images/architecture.png", wantAlt: "Architecture", wantPresent: true},
		{name: "surrounding whitespace", content: "  ![Diagram](diagram.png)\t", wantPath: "diagram.png", wantAlt: "Diagram", wantPresent: true},
		{name: "inline image", content: "Before ![Diagram](diagram.png) after", wantPresent: false},
		{name: "ordinary link", content: "[Diagram](diagram.png)", wantPresent: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHighlighter()
			if err := h.SetLanguage("markdown"); err != nil {
				t.Fatal(err)
			}
			if err := h.ParseFull([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			image, present := h.MarkdownImageForLine(0, len(tt.content))
			if present != tt.wantPresent {
				t.Fatalf("present = %v, want %v; image=%+v", present, tt.wantPresent, image)
			}
			if present && (image.Destination != tt.wantPath || image.Alt != tt.wantAlt) {
				t.Fatalf("image = %+v, want alt %q and path %q", image, tt.wantAlt, tt.wantPath)
			}
		})
	}
}
