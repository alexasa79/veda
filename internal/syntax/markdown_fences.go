package syntax

import "strings"

type markdownCodeBlock struct {
	start       int
	end         int
	highlighter *Highlighter
}

// Markdown fence aliases are deliberately limited to languages Veda already
// highlights. Exact language names accepted by SetLanguage work automatically;
// when adding a language, add only its common fence aliases here.
var markdownFenceAliases = map[string]string{
	"js":       "javascript",
	"jsx":      "javascript",
	"ts":       "typescript",
	"golang":   "go",
	"c++":      "cpp",
	"cc":       "cpp",
	"cxx":      "cpp",
	"sh":       "bash",
	"shell":    "bash",
	"zsh":      "bash",
	"py":       "python",
	"makefile": "make",
	"tla":      "tlaplus",
	"tla+":     "tlaplus",
	"tex":      "latex",
}

func (h *Highlighter) rebuildMarkdownCodeBlocks() error {
	h.markdownCodeBlocks = nil
	if h.currentLang != "markdown" {
		return nil
	}

	source := h.currentSource
	inFence := false
	contentStart := 0
	var embedded *Highlighter

	for lineStart := 0; lineStart <= len(source); {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		line := strings.TrimSpace(string(source[lineStart:lineEnd]))

		if inFence {
			if line == "```" {
				if err := h.addMarkdownCodeBlock(contentStart, lineStart, embedded); err != nil {
					return err
				}
				inFence = false
				embedded = nil
			}
		} else if language, ok := markdownFenceLanguage(line); ok {
			inFence = true
			contentStart = lineEnd
			if contentStart < len(source) && source[contentStart] == '\n' {
				contentStart++
			}

			if language != "" && language != "markdown" {
				candidate := NewHighlighter()
				if candidate.SetLanguage(language) == nil {
					embedded = candidate
				}
			}
		}

		if lineEnd == len(source) {
			break
		}
		lineStart = lineEnd + 1
	}

	if inFence {
		return h.addMarkdownCodeBlock(contentStart, len(source), embedded)
	}
	return nil
}

func (h *Highlighter) addMarkdownCodeBlock(start, end int, embedded *Highlighter) error {
	if embedded == nil {
		return nil
	}
	if err := embedded.ParseFull(h.currentSource[start:end]); err != nil {
		return err
	}
	h.markdownCodeBlocks = append(h.markdownCodeBlocks, markdownCodeBlock{
		start:       start,
		end:         end,
		highlighter: embedded,
	})
	return nil
}

func markdownFenceLanguage(line string) (string, bool) {
	if !strings.HasPrefix(line, "```") || strings.HasPrefix(line, "````") {
		return "", false
	}

	label := strings.TrimSpace(line[3:])
	if label == "" {
		return "", true
	}
	if strings.ContainsAny(label, " \t`") {
		return "", true
	}

	label = strings.ToLower(label)
	if language, ok := markdownFenceAliases[label]; ok {
		return language, true
	}
	return label, true
}
