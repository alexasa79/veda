# Terminal Mode Design and Proxy Integration Plan

## Goals
- Terminal mode similar to Vim: all keystrokes go to a PTY process; `Esc` returns to Normal mode on the same buffer.
- Terminal buffer is read-only for editor edits; normal-mode motions, visual selection, yank work.
- Scrollback supported; output continues while in Normal mode.
- Seamless local/remote: terminal I/O always goes through the proxy so remote workspaces open a remote shell.

## Core decisions
- All terminal I/O routed via the Python proxy (stdin/stdout JSON-RPC). No direct PTY from the Go editor.
- No server push; Go polls the proxy for new output (non-blocking) with backoff.
- Use `working_dir` to pick locality. For `host:/path`, the proxy is already remote via SSH, so the PTY launches on that host.

## Proxy API (JSON-RPC)
- Sessions multiplexed by `session_id` (string). Data transported as base64 for binary safety.
- Methods:
  - `start_terminal(working_dir: str, command?: List[str], env?: Dict[str,str], cols: int, rows: int)` → `{ session_id: str, pid: int }`
  - `write_terminal(session_id: str, data_b64: str)` → `{ success: bool }`
  - `read_terminal(session_id: str, max_bytes: int = 65536)` → `{ data_b64: str, closed: bool, exit_code?: int }`
  - `resize_terminal(session_id: str, cols: int, rows: int)` → `{ success: bool }`
  - `close_terminal(session_id: str)` → `{ success: bool }`
- Defaults: `command` falls back to user shell (`$SHELL`); set `TERM=xterm-256color`.

## Proxy implementation (Python)
- Add `TerminalManager` to `veda-proxy.py`:
  - `pty.openpty()`; spawn process with `subprocess.Popen` on slave; master fd non-blocking.
  - Reader thread per session: `select` + `os.read` → append to a thread-safe buffer (e.g., `collections.deque` of chunks).
  - Track `closed`, `exit_code`; clean up fds/threads on exit or `close_terminal`.
  - `read_terminal` drains up to `max_bytes`; non-blocking; cheap when empty.
  - `write_terminal` decodes base64 and writes bytes to master.
  - `resize_terminal` uses `fcntl.ioctl(TIOCSWINSZ)`.
- Register methods in `VedaProxy.methods`.
- Increment `PROXY_VERSION` and keep in sync with Go.

## Go proxy client additions (`internal/proxy/proxy.go`)
- Client methods:
  - `StartTerminal(workingDir string, command []string, env map[string]string, cols, rows int) (sessionID string, pid int, err error)`
  - `WriteTerminal(sessionID string, data []byte) error`
  - `ReadTerminal(sessionID string, maxBytes int) (data []byte, closed bool, exitCode *int, err error)`
  - `ResizeTerminal(sessionID string, cols, rows int) error`
  - `CloseTerminal(sessionID string) error`
- Manager helpers:
  - Resolve correct `*Client` from `workingDir` using existing `getClient` (handles `host:/path`).
  - Do not hold the manager-wide mutex across streaming calls; sessions keep a reference to the `*Client`.

## Editor integration (Go)
- Directory: `internal/terminal/`.
- `TerminalSession`:
  - Holds `sessionID`, pointer to proxy `Client`, shell dimensions, polling goroutine.
  - Poll cadence: ~20–30 Hz when active; backoff to 2–5 Hz when idle; coalesce updates.
  - Optional VT emulation phase: M1 can append plain text; M3 adds ANSI attributes.
- `TerminalBuffer`:
  - Read-only; mirrors emulator screen + scrollback; appends new output.
  - When viewport at bottom, auto-follow; otherwise keep position.
- Modes:
  - `ModeTerminal`: forward all keys to PTY; `Esc` switches to Normal.
  - In Normal mode on terminal buffer: motions/search/visual/yank only; no edits.
- Commands/keys:
  - `:term [cmd…]` opens terminal in editor `workingDirectory`. Remote paths open on remote host.
  - Optional keybinding (e.g., `space s t`).
- Resizing:
  - On layout changes, call `ResizeTerminal`.
- Status bar:
  - Show “TERMINAL” vs “NORMAL (terminal)”, PID, and exit status when closed.

## Read-only enforcement
- Add/ensure `IsReadOnly()` to the buffer interface; guard insert/delete in `edit_operations.go`.
- Terminal buffers set `readOnly=true`; undo/redo not tracked for terminal output.

## Remote session behavior
- Host selection follows `workingDirectory`:
  - Local path → local proxy → local shell.
  - `host:/path` → remote proxy via SSH → remote shell started on `host`.
- `working_dir` passed to proxy determines initial cwd of the shell.

## Key mapping and paste
- Map tcell keys to xterm sequences (arrows, Home/End, PgUp/PgDn, Tab, Enter, Backspace, Ctrl-keys).
- Paste in terminal mode: send clipboard content in chunks to `WriteTerminal`.

## Performance & concurrency
- Coalesce output bursts; minimize redraw churn.
- Avoid global locks on streaming calls; `Client` is concurrency-safe via request routing.

## Testing
- Python (proxy):
  - `start_terminal` with `['/bin/sh','-lc','printf hi; exit 0']` → `read_terminal` returns `hi`, `closed=true`, `exit_code=0`.
  - `resize_terminal` and `close_terminal` sanity checks.
- Go client:
  - Start, write `echo hi\n`, read back.
- Editor:
  - Read-only enforcement for terminal buffers.
  - `Esc` switches to Normal; key forwarding stops.
  - Scrollback browsing + bottom anchor behavior.
  - Resize propagates.
- Manual remote:
  - On remote workspace, `:term` then `uname -a` shows remote kernel/hostname.

## Milestones
- M1: Proxy terminal API + Go client + text-only terminal buffer, `:term`, Esc toggling, read-only enforcement.
- M2: Full key mapping, resize, better polling/backoff, robust scrollback.
- M3: VT/ANSI attributes and terminal cursor rendering.
- M4: Paste, status details, help docs, optional host override.

## Acceptance criteria
- `:term` launches user shell; interactive programs (`top`, `less`, `vim`) work.
- `Esc` switches to Normal on same buffer; buffer is read-only to editor edits; motions/visual/yank work.
- Output continues while in Normal mode; bottom anchoring works as expected.
- Resizing updates PTY size.
- Remote workspace opens a remote shell automatically (via proxy).

## Versioning
- Bump both: `ProxyVersion` in `internal/proxy/proxy.go` and `PROXY_VERSION` in `internal/proxy/veda-proxy.py` when adding terminal RPCs.
