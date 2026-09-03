# Veda

Veda is a terminal-based text editor written in Go, with vim-style modal
editing and a set of modern IDE features built in. The binary is called `vd`.

## TL;DR — what it does

- **Vim-like modal editing**: normal, insert, visual, visual-line, and command
  modes, with operators, text objects, counts, undo/redo, registers, and `.`
  repeat.
- **Multiple cursors**: place cursors anywhere and edit all of them at once.
- **LSP support**: go to definition, find references, diagnostics, completion,
  signature help, and symbol search for Go, C/C++, Python,
  JavaScript/TypeScript, and Bash.
- **Git built in**: stage or revert individual lines and hunks, blame, show
  commits, and a project-changes overview — without leaving the editor.
- **Remote editing over SSH**: open `host:path` and edit files on another
  machine as if they were local. The remote side needs only `python3`.
- **Embedded terminal**: run a shell inside the editor (works on remote hosts
  too).
- **Side-by-side diff view**: compare your buffer against git HEAD with
  syntax-highlighted, word-level diffs.
- **Syntax highlighting** via tree-sitter: Go, C, C++, Python, JavaScript,
  TypeScript, Bash, Fish, Markdown, Make, LaTeX, TLA+.
- **Extras**: multi-window layout, fuzzy file/buffer pickers, markdown
  rendering with clickable links, an inline calculator, LLM-based spellcheck
  (via Ollama), themes, and zen mode.

## Getting started

Requirements: Go 1.24+, a C compiler (for the tree-sitter grammars), and
`python3` at runtime.

```bash
make        # builds the 'vd' binary
./vd file.txt
./vd myserver:src/app.py   # edit a remote file over SSH
make test   # run all tests (can take a while)
```

Useful flags: `-d` (debug logging), `-log <path>` (log location),
`-cpuprofile <file>`. Configuration lives in `~/.config/veda/config.json`
(theme, indent settings, and `relative_line_numbers`, which defaults to `false`).

## Editing basics

Veda follows vim conventions with a few deliberate differences.

- **Movement**: `h j k l` or arrows, `w`/`b` by word, `f<char>`/`t<char>` on
  the line, `0`/`$` line start/end, `gg`/`G` top/bottom.
- **Editing**: operators `d`, `c`, `y` combine with motions and text objects;
  `dd`/`cc`/`yy` for lines, `p`/`P` paste, `x`, `J` join, `>>`/`<<` indent,
  `Ctrl+/` toggle comment, `.` repeat.
- **Undo/redo**: `u` undoes, `U` redoes. Everything you do in one insert-mode
  session undoes as a single unit.
- **Words are identifiers**: `w` treats `snake_case_name` as one word. `W`/`B`
  treat the underscore as a boundary, so they move through the sub-words —
  the opposite of vim's "bigger WORD".
- **Whole-word operations**: `dw` deletes the entire word under the cursor,
  no matter where in the word the cursor sits.
- **Text objects**: brackets (`di(`, `ca{`), arguments (`daa`/`dia`), and
  LSP-powered blocks (`dib`) and functions (`dif`). `ds` operates on the last
  search match.
- **Search**: `/` forward, `?` backward, `*` for the word under the cursor,
  `n`/`N` to repeat.
- **Registers**: prefix with `'` — `'ay` yanks into register a, `'ap` pastes
  from it. `'d` holds the last delete, `'0` is the system clipboard. `:reg`
  lists them.
- **Multiple cursors**: press `m` to enter cursor mode and drop cursors
  (Enter places one and moves down), then edit — commands apply at every
  cursor. `Alt+A` aligns cursors into a column.

Command mode (`:`) supports the usual suspects — `:w`, `:q`, `:wq`, `:qa`,
`:e <path>`, a bare `:42` to jump to a line — plus settings like
`:indent=spaces:4`, `:tabstop=8`, and `:theme <name>` (monokai, dracula,
nord, onedark, tender, koda, tokyonight, vscode). The command line has
history and tab completion.

## Windows and buffers

Each window can hold several buffers. `space v` splits vertically,
`space 1..9` jumps to a window, `space tab` cycles, `space w left/right/c`
moves the current buffer between windows, `space w f` toggles fullscreen, and
`Ctrl+M` toggles zen mode. `space space` opens the buffer picker (which shows
each buffer's window), and `space s f` opens the fuzzy file browser.

## Code intelligence (LSP)

Language servers start automatically when you open a matching file and stop
when the editor exits. Key bindings: `g d` go to definition (`g D` in a new
window), `g r` find references, `g e` diagnostics, `g s` signature help,
`s d`/`s w` document/project symbols, `g ,` to jump back, `Ctrl+Space` for
completion in insert mode.

Servers used: `gopls` (Go), `clangd` (C/C++), `pylsp` (Python),
`typescript-language-server` (JS/TS), `bash-language-server` (Bash). Install
whichever you need; Veda tells you if one is missing.

## Git

- `g h` stages or unstages the hunk under the cursor; `g space` does the same
  for a single line.
- `g j J` / `g j j` revert (drop) a hunk or line.
- `g b b` shows blame for the current line; `g b s` opens the commit that
  last touched it.
- `s C` opens a *Project Changes* buffer summarizing what changed across the
  repo; `s c` opens a side-by-side diff of the current buffer against git
  HEAD.

## Remote editing

Open a file as `host:path` and Veda starts its helper process on that machine
over SSH. The helper is a single dependency-free Python script embedded in
the binary — if it's missing or out of date on the remote host, Veda copies
it over automatically. All file, git, LSP, and terminal features work through
it, so the embedded terminal and git commands run on the remote machine too.

## Odds and ends

- **Markdown**: internal buffers (help, diagnostics, references, project
  changes) render as markdown. Links show as plain text; press Enter on one
  to open the target, including `file.go#L42`-style line references.
- **Calculator**: `- -` evaluates the expression on the line and appends the
  result; `- =` replaces it. Understands hex/octal/binary numbers and unit
  suffixes like `4ki` or `2m`.
- **Spellcheck**: `Alt+S` checks the selection or buffer using a local Ollama
  model.
- **Terminal**: `space t` (or `:term`) opens a shell in a buffer, with
  scrollback, search, and selection.

## Architecture

- **Go core** (`internal/editor/`): the editor itself — modes, commands,
  rendering, windows.
- **Custom terminal layer** (`internal/terminal/`): raw-mode TTY handling,
  double-buffered drawing, and input parsing, with no external UI library.
- **Rope buffer** (`internal/buffer/`): a balanced rope holds text, so edits
  and line lookups stay fast even in very large files. Undo history lives
  here too.
- **Python proxy** (`internal/proxy/`): a companion process that does all
  file access, git operations, LSP server management, and terminal PTYs,
  talking to the editor over JSON-RPC. Running it over SSH is what makes
  remote editing work.
- **Syntax** (`internal/syntax/`): tree-sitter grammars, some vendored and
  compiled in-tree.

## Testing

`make test` runs the full suite. Editor behavior is tested against a mock
screen, so nearly everything — cursor movement, editing, rendering — is
covered by plain `go test`. Some text-object tests need `gopls` or `clangd`
installed.
