## Diff View Implementation Plan (Buffer vs HEAD)

### Scope and UX
- Press `s D` in Normal mode to open a Diff View of the current buffer vs its HEAD version (same path):
  - Left: current buffer (live, unsaved changes included)
  - Right: last committed version (HEAD)
- Single-widget view: we do NOT open or render two editor buffers. The diff is an unusual, read-only view of the same buffer (like a hex view), provided by a dedicated widget attached to the buffer.
- Windowing semantics: the diff view is tied to the active window and cannot be split into separate windows. We prevent moving one “side” elsewhere because there is only one widget rendering both sides.
- Exit: use `:q` to close the buffer (typical behavior). No dedicated `q` keybinding.
- Only buffers backed by a file path are supported. For buffers without a file path or files not tracked by git, show a friendly message.
- The Diff View is read-only, side-by-side, with synchronized vertical scrolling and per-hunk navigation.

### Architecture Overview
- Single internal diff engine (line-based). No external diff tools for diffing.
- Retrieve HEAD content for the current file via a small VCS snapshot helper (production can call git; tests use a stub).
- Represent differences as line operations grouped into hunks and maintain a line mapping for pane sync.
- A `DiffSession` object is attached to the currently viewed buffer while diff mode is active. Rendering, input handling, navigation, and scrolling are delegated to a single `DiffView` widget that paints both sides inside the current window.

### Components
1) internal/diff/
- `types.go`
  - `OpType` ∈ {Equal, Insert, Delete, Replace}
  - `Op { Type OpType; AStart, ALines, BStart, BLines int }`
  - `Hunk { Ops []Op; AStart, AEnd, BStart, BEnd int }`
  - `LineMap { AToB []int; BToA []int }` (−1 for no mapping)
- `line_diff.go`
  - `ComputeLineDiff(a []string, b []string) (hunks []Hunk, lineMap LineMap, err error)`
  - Implementation: straightforward LCS/Myers-style line diff; deterministic; merges adjacent compatible ops into hunks.

2) HEAD content provider
- `internal/vcs/head_content.go` (or `internal/editor/head_snapshot.go`)
  - `type HeadContentProvider interface { GetHeadContent(absPath string) (lines []string, ok bool, err error) }`
  - Default impl executes `git show HEAD:relative/path` and splits into lines.
  - Injection point in editor for tests to replace with a stub (no git required in tests).

3) Diff view widget (single widget attached to buffer)
- `DiffSession` (attached to buffer):
  - References: buffer path, HEAD snapshot metadata
  - Diff data: `[]Hunk`, `LineMap`, cached `[]string` for A/B
  - State: current viewport, current hunk index, sync-scroll enabled
- `DiffView` widget:
  - Draws side-by-side panes within the current window (gutters, line numbers, change markers +/−/~)
  - Enforces read-only behavior while active
  - Navigation: next/prev hunk, jump to counterpart line
  - Sync scroll: uses `LineMap` to align panes while scrolling
  - Theme: uses existing syntax highlighter for both sides; added minimal diff colors for backgrounds

4) Commands / keybindings / help
- Command: `show-diff` (toggle diff mode for current buffer vs HEAD)
- Keybinding: `s D` → `show-diff`
- While in diff mode: close via `:q` (standard quit). No special `q` keybinding.
- Help entry under the “show” category.

### Incremental Delivery
1. Diff algorithm
   - Build `internal/diff` with line-based diff and line mapping.
   - Exhaustive unit tests.
2. HEAD content provider seam
   - Implement provider and editor injection point.
   - Unit tests with stub provider (no git dependency).
3. Single-widget rendering (minimal)
   - Implement `DiffSession` and a basic `DiffView` that renders both sides within one window, read-only.
   - Unit tests for rendering layout and windowing constraints.
4. Key navigation
   - Implement next/prev hunk and jump-to-counterpart commands scoped to diff mode.
   - Unit tests verifying correct movement and state updates.
5. Scrolling synchronization
   - Implement sync scroll using `LineMap` with toggle.
   - Unit tests for scroll behavior and alignment.
6. Syntax highlighting
   - Apply syntax highlighting to both panes with diff overlays.
   - Unit tests verifying tokens/colors are applied consistently.
7. Edge cases, polish, and performance checks.

### Testing Strategy (central, by step)

1) Diff algorithm (`internal/diff/line_diff_test.go`)
- Identity: A vs A → zero hunks; `LineMap` is identity
- Insertions/deletions/replacements at beginning/middle/end
- Mixed edits sequences
- Empty inputs and one-liners
- Reconstruction: applying ops to A reconstructs B (test helper `applyOps`)
- Symmetry: Diff(A,B) vs Diff(B,A) produce dual operations
- `LineMap` integrity: mappings align with Equal ops; −1 where unmapped
- Determinism: repeated runs yield identical outputs
- Randomized small inputs (fixed seed) verifying reconstruction and bounds

2) HEAD provider seam (`internal/editor/head_snapshot_test.go`)
- Use a stub provider returning known “HEAD” content
- Buffer without file path → graceful error message
- Untracked file or missing HEAD → graceful message
- Ensure editor uses injected provider (no real git dependency in tests)

3) Rendering (single widget, windowing) (`internal/editor/diff_view_test.go`)
- One widget paints both sides inside a single window; there is no second buffer
- Attempting to move/split panes does not separate the sides (diff mode remains a single view)
- Left/right text and change markers (+/−/~) render as expected
- Read-only enforcement: edit keys are no-ops

4) Key navigation (`internal/editor/diff_navigation_test.go`)
- `] d`/`[ d` moves to next/previous hunk (viewport and state update)
- `enter` jumps to the counterpart line if mapped; no-op if unmapped
- Status messages (if any) are appropriate

5) Scrolling synchronization (`internal/editor/diff_scroll_test.go`)
- Scrolling in the primary pane aligns the counterpart via `LineMap`
- Toggle sync-scroll on/off; verify independent vs synced behavior
- Large scrolls and page up/down remain aligned

6) Syntax highlighting (`internal/editor/diff_syntax_test.go`)
- Both panes apply the language highlighter based on file type
- Diff overlays (added/removed/changed backgrounds) coexist with syntax colors
- Long lines and clipping preserve highlighting stability

Edge-case tests
- Very long lines: truncation/clip is stable in both panes
- Files with/without trailing newline
- CRLF handling treated consistently as line boundaries

### Acceptance Criteria
- `s D` opens a read-only side-by-side Diff View in a single widget attached to the current buffer: current buffer vs HEAD for the same path.
- Correct change markers and synced scrolling.
- Navigation works: next/prev hunk and jump-to-counterpart.
- All unit tests pass; no reliance on external diff tools; tests do not require git.

### Out of Scope (initial)
- Intra-line highlights
- Ignore-whitespace/case options
- Arbitrary file-to-file diffs
- Staging hunks / patch application
