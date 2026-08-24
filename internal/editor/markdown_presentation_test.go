package editor

import (
	"path/filepath"
	"strings"
	"testing"
	"veda/internal/syntax"
)

func newMarkdownPresentationEditor(t *testing.T, width, height int, content string) *TestEditor {
	t.Helper()
	te := NewTestEditorWithContent(width, height, content)
	te.CurrentBuffer().FilePath = "test.md"
	te.CurrentBuffer().Highlighter = syntax.NewHighlighter()
	if err := te.CurrentBuffer().Highlighter.SetLanguage("markdown"); err != nil {
		t.Fatal(err)
	}
	if err := te.CurrentBuffer().Highlighter.ParseFull([]byte(content)); err != nil {
		t.Fatal(err)
	}
	return te
}

func TestMarkdownPresentationDiffersByMode(t *testing.T) {
	content := "Before **bold text** and [link](target.md)."
	te := newMarkdownPresentationEditor(t, 80, 10, content)

	te.renderPane()
	got := GetScreenLine(te.mockScreen.GetContent(), 0)
	if !strings.Contains(got, "Before bold text and link.") {
		t.Fatalf("normal mode rendered %q", got)
	}
	if strings.Contains(got, "**") || strings.Contains(got, "target.md") {
		t.Fatalf("normal mode contains Markdown source syntax: %q", got)
	}

	te.ExecuteCommand("enter-insert-mode")
	te.renderPane()
	got = GetScreenLine(te.mockScreen.GetContent(), 0)
	if !strings.Contains(got, content) {
		t.Fatalf("insert mode rendered %q, want source %q", got, content)
	}
}

func TestMarkdownPresentationControlsWrapping(t *testing.T) {
	te := newMarkdownPresentationEditor(t, 24, 10, "")
	available := te.wrapScreenWidth() - te.CurrentBuffer().LineNumberWidth
	content := "**" + strings.Repeat("x", available) + "**"
	te = newMarkdownPresentationEditor(t, 24, 10, content)

	if got := len(te.CalculateWrappedLines()); got != 1 {
		t.Fatalf("normal mode used %d visual lines, want 1", got)
	}

	te.ExecuteCommand("enter-insert-mode")
	if got := len(te.CalculateWrappedLines()); got <= 1 {
		t.Fatalf("insert mode used %d visual lines, want more than 1", got)
	}
}

func TestMarkdownPresentationMapsCursorToSource(t *testing.T) {
	te := newMarkdownPresentationEditor(t, 40, 10, "**bold**")
	gutter := te.CurrentBuffer().LineNumberWidth

	te.CurrentBuffer().CursorX = 2
	line, column := te.GetCursorVisualPosition()
	if line != 0 || column != gutter {
		t.Fatalf("cursor visual position = (%d,%d), want (0,%d)", line, column, gutter)
	}

	logicalLine, sourceColumn := te.MapVisualToLogical(0, gutter)
	if logicalLine != 0 || sourceColumn != 2 {
		t.Fatalf("visual-to-source position = (%d,%d), want (0,2)", logicalLine, sourceColumn)
	}

	te.CurrentBuffer().CursorX = 0
	te.ExecuteCommand("right")
	if te.CurrentBuffer().CursorX != 3 {
		t.Fatalf("right moved to source column %d, want 3", te.CurrentBuffer().CursorX)
	}
}

func TestMarkdownPresentationNormalizesCursorAfterInsert(t *testing.T) {
	te := newMarkdownPresentationEditor(t, 40, 10, "**bold**")
	te.ExecuteCommand("enter-insert-mode")
	te.CurrentBuffer().CursorX = len("**bold**")
	te.ExecuteCommand("exit-insert-mode")

	if te.CurrentBuffer().CursorX != 5 {
		t.Fatalf("cursor source column = %d, want final visible character at 5", te.CurrentBuffer().CursorX)
	}
}

func TestMarkdownPresentationKeepsSemanticCursorEnteringInsert(t *testing.T) {
	te := newMarkdownPresentationEditor(t, 40, 10, "## Heading")
	te.CurrentBuffer().CursorX = 0
	te.ExecuteCommand("enter-insert-mode")

	if te.CurrentBuffer().CursorX != 3 {
		t.Fatalf("cursor source column = %d, want heading text at 3", te.CurrentBuffer().CursorX)
	}
}

func TestMarkdownPresentationMapsUnicodeCursor(t *testing.T) {
	te := newMarkdownPresentationEditor(t, 40, 10, "**héllo**")
	gutter := te.CurrentBuffer().LineNumberWidth

	te.CurrentBuffer().CursorX = 3
	line, column := te.GetCursorVisualPosition()
	if line != 0 || column != gutter+1 {
		t.Fatalf("cursor visual position = (%d,%d), want (0,%d)", line, column, gutter+1)
	}

	logicalLine, sourceColumn := te.MapVisualToLogical(0, gutter+1)
	if logicalLine != 0 || sourceColumn != 3 {
		t.Fatalf("visual-to-source position = (%d,%d), want (0,3)", logicalLine, sourceColumn)
	}

	te.CurrentBuffer().CursorX = 2
	te.ExecuteCommand("right")
	if te.CurrentBuffer().CursorX != 3 {
		t.Fatalf("right moved to source column %d, want Unicode rune at 3", te.CurrentBuffer().CursorX)
	}
	te.ExecuteCommand("right")
	if te.CurrentBuffer().CursorX != 5 {
		t.Fatalf("right moved to source column %d, want source column 5 after Unicode rune", te.CurrentBuffer().CursorX)
	}
}

func TestMarkdownPresentationRendersStandaloneKittyPNG(t *testing.T) {
	content := "![Architecture](kitty-image-test.png)\nAfter"
	te := newMarkdownPresentationEditor(t, 100, 20, content)
	markdownPath, err := filepath.Abs("../../artifacts/kitty-image-test.md")
	if err != nil {
		t.Fatal(err)
	}
	te.CurrentBuffer().RealPath = markdownPath
	te.mockScreen.EnableKittyImages()

	wrapped := te.CalculateWrappedLines()
	if len(wrapped) <= 2 || wrapped[0].KittyImagePath == "" {
		t.Fatalf("standalone image did not reserve visual rows: %+v", wrapped)
	}

	te.render()
	images := te.mockScreen.GetKittyImages()
	if len(images) != 1 {
		t.Fatalf("rendered %d Kitty images, want 1", len(images))
	}
	wantPath := filepath.Join(filepath.Dir(markdownPath), "kitty-image-test.png")
	if images[0].Path != wantPath {
		t.Fatalf("image path = %q, want %q", images[0].Path, wantPath)
	}
	if images[0].X != te.CurrentBuffer().LineNumberWidth || images[0].Columns != 48 || images[0].Rows != 12 {
		t.Fatalf("image placement = %+v", images[0])
	}
	currentLineBg := te.config.Theme.GetStyle(te.config.Theme.CurrentLine).Bg
	currentLineNumberStyle := te.config.Theme.GetStyle(te.config.Theme.CurrentLineNumber)
	for row := 0; row < images[0].Rows; row++ {
		_, style := te.mockScreen.GetCell(te.CurrentBuffer().LineNumberWidth, row)
		if style.Bg != currentLineBg {
			t.Fatalf("image row %d is missing current-line background", row)
		}
		if row > 0 {
			_, gutterStyle := te.mockScreen.GetCell(0, row)
			if gutterStyle != currentLineNumberStyle {
				t.Fatalf("image gutter row %d is missing current-line-number style", row)
			}
		}
	}

	te.ExecuteCommand("enter-insert-mode")
	te.render()
	if images := te.mockScreen.GetKittyImages(); len(images) != 0 {
		t.Fatalf("insert mode retained %d Kitty images", len(images))
	}
	got := GetScreenLine(te.mockScreen.GetContent(), 0)
	if !strings.Contains(got, "![Architecture](kitty-image-test.png)") {
		t.Fatalf("insert mode rendered %q", got)
	}
}

func TestMarkdownPresentationCachesWrappedViewportAndImageSize(t *testing.T) {
	content := "![Architecture](kitty-image-test.png)\nAfter"
	te := newMarkdownPresentationEditor(t, 100, 20, content)
	markdownPath, err := filepath.Abs("../../artifacts/kitty-image-test.md")
	if err != nil {
		t.Fatal(err)
	}
	te.CurrentBuffer().RealPath = markdownPath
	te.mockScreen.EnableKittyImages()

	first := te.CalculateWrappedLines()
	cache := te.wrappedLinesCache
	if cache == nil || len(first) < 2 {
		t.Fatalf("wrapped viewport was not cached: %+v", first)
	}
	if len(te.markdownImageSizes) != 1 {
		t.Fatalf("cached %d image sizes, want 1", len(te.markdownImageSizes))
	}
	second := te.CalculateWrappedLines()
	if te.wrappedLinesCache != cache || &second[0] != &first[0] {
		t.Fatal("unchanged viewport was recalculated")
	}

	te.ExecuteCommand("down")
	if te.wrappedLinesCache != cache {
		t.Fatal("cursor-only movement invalidated the wrapped viewport")
	}
}

func TestFitMarkdownImageToCellsPreservesAspectRatio(t *testing.T) {
	columns, rows := fitMarkdownImageToCells(1774, 887, 200, 12, 8, 16)
	if columns != 48 || rows != 12 {
		t.Fatalf("image size = %dx%d, want 48x12", columns, rows)
	}

	columns, rows = fitMarkdownImageToCells(160, 80, 200, 12, 8, 16)
	if columns != 20 || rows != 5 {
		t.Fatalf("small image size = %dx%d, want intrinsic 20x5", columns, rows)
	}
}

func TestMarkdownPresentationFallsBackForUnsupportedImage(t *testing.T) {
	te := newMarkdownPresentationEditor(t, 60, 10, "![Photo](photo.jpg)")
	te.CurrentBuffer().RealPath = "/project/readme.md"
	te.mockScreen.EnableKittyImages()

	te.render()
	if images := te.mockScreen.GetKittyImages(); len(images) != 0 {
		t.Fatalf("rendered unsupported image: %+v", images)
	}
	got := GetScreenLine(te.mockScreen.GetContent(), 0)
	if !strings.Contains(got, "Photo") {
		t.Fatalf("fallback rendered %q", got)
	}
}
