package syntax

import "testing"

func TestLatexSyntaxHighlighting(t *testing.T) {
	h := NewHighlighter()

	if err := h.SetLanguage("latex"); err != nil {
		t.Fatalf("failed to set language to latex: %v", err)
	}

	code := `\documentclass{article}
\usepackage{amsmath}
\title{My Document}
\author{John}

\newcommand{\foo}[1]{bar #1}

\begin{document}

% This is a comment
\section{Introduction}
\subsection{Background}

Hello world. Some \textbf{bold} and \textit{italic} text.

\label{sec:intro}
See \ref{sec:intro} and \cite{knuth1984}.

% Math
$x^2 + y^2 = z^2$

\[ E = mc^2 \]

\begin{equation}
a^2 + b^2 = c^2
\end{equation}

\begin{itemize}
\item First
\item Second
\end{itemize}

\end{document}
`

	h.ParseFull([]byte(code))

	type expectation struct {
		text      string
		tokenType TokenType
	}

	tokens := h.GetTokensForString(code)

	expectations := []expectation{
		// Document structure commands
		{`\documentclass`, TokenTypeKeyword},
		{`\usepackage`, TokenTypeKeyword},
		{`\title`, TokenTypeKeyword},
		{`\author`, TokenTypeKeyword},

		// Paths/package names
		{"article", TokenTypeString},
		{"amsmath", TokenTypeString},

		// Definition
		{`\newcommand`, TokenTypeKeyword},

		// Environment markers
		{`\begin`, TokenTypeKeyword},
		{`\end`, TokenTypeKeyword},

		// Comments
		{"% This is a comment", TokenTypeComment},
		{"% Math", TokenTypeComment},

		// Section commands
		{`\section`, TokenTypeKeyword},
		{`\subsection`, TokenTypeKeyword},

		// Generic commands (textbf, textit)
		{`\textbf`, TokenTypeFunction},
		{`\textit`, TokenTypeFunction},

		// Labels and references
		{`\label`, TokenTypeVariable},
		{"sec:intro", TokenTypeString},
		{`\ref`, TokenTypeVariable},
		{`\cite`, TokenTypeVariable},

		// Item
		{`\item`, TokenTypeKeyword},

		// Punctuation
		{"{", TokenTypePunctuation},
		{"}", TokenTypePunctuation},

		// Math operators
		{"^", TokenTypeOperator},
		{"+", TokenTypeOperator},
		{"=", TokenTypeOperator},
	}

	for _, exp := range expectations {
		found := false
		for _, token := range tokens {
			if token.Start < 0 || token.End > len(code) || token.Start >= token.End {
				continue
			}
			tokenText := code[token.Start:token.End]
			if tokenText == exp.text {
				if token.TokenType != exp.tokenType {
					t.Errorf("wrong token type for %q: got %v, want %v", exp.text, token.TokenType, exp.tokenType)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("token %q not found", exp.text)
		}
	}
}

func TestLatexFileExtensions(t *testing.T) {
	h := NewHighlighter()

	extensions := []string{".tex", ".latex", ".sty", ".cls", ".dtx", ".ltx"}
	for _, ext := range extensions {
		err := h.SetLanguageFromFile("test" + ext)
		if err != nil {
			t.Errorf("SetLanguageFromFile failed for %s: %v", ext, err)
		}
		if h.GetCurrentLanguage() != "latex" {
			t.Errorf("expected language 'latex' for %s, got %q", ext, h.GetCurrentLanguage())
		}
	}
}
