package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// useOSC52 controls whether OSC 52 fallback is enabled.
var useOSC52 = true

// nativeCopyFailed remembers if native copy tools have failed.
// Once set, we skip native tools and go directly to OSC 52.
var nativeCopyFailed = false

// nativePasteFailed remembers if native paste tools have failed.
// Once set, we skip native tools and go directly to OSC 52.
var nativePasteFailed = false

// SetUseOSC52 enables or disables OSC 52 fallback
func SetUseOSC52(enabled bool) {
	useOSC52 = enabled
}

// ResetNativeClipboardState resets the native clipboard failure flags.
// This can be used if the environment changes (e.g., X11 becomes available).
func ResetNativeClipboardState() {
	nativeCopyFailed = false
	nativePasteFailed = false
}

// Copy copies text to the system clipboard
func Copy(text string) error {
	// If native copy has failed before, go directly to OSC 52
	if nativeCopyFailed && useOSC52 {
		return OSC52Copy(text)
	}

	err := copyNative(text)
	if err != nil {
		// Remember that native copy failed
		nativeCopyFailed = true
		if useOSC52 {
			// Fall back to OSC 52
			return OSC52Copy(text)
		}
	}
	return err
}

// copyNative attempts to copy using native clipboard tools
func copyNative(text string) error {
	switch runtime.GOOS {
	case "darwin":
		// macOS: use pbcopy
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	case "linux":
		// Linux: detect WSL and use Windows clipboard interop if available
		if isWSL() {
			// Use PowerShell exclusively in WSL to avoid clip.exe newline behavior
			if _, err := exec.LookPath("powershell.exe"); err == nil {
				cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", "Set-Clipboard -Value ([Console]::In.ReadToEnd())")
				cmd.Stdin = strings.NewReader(text)
				return cmd.Run()
			}
			return fmt.Errorf("WSL detected but powershell.exe not found")
		}
		// Native Linux: try xclip first, then xsel
		// Try xclip
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command("xclip", "-selection", "clipboard")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}
		// Try xsel
		if _, err := exec.LookPath("xsel"); err == nil {
			cmd := exec.Command("xsel", "--clipboard", "--input")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}
		return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// OSC52PasteTimeout is the timeout for OSC 52 paste operations
const OSC52PasteTimeout = 500 * time.Millisecond

// Paste reads text from the system clipboard
func Paste() (string, error) {
	// If native paste has failed before, go directly to OSC 52
	if nativePasteFailed && useOSC52 {
		return OSC52Paste(OSC52PasteTimeout)
	}

	text, err := pasteNative()
	if err != nil {
		// Remember that native paste failed
		nativePasteFailed = true
		if useOSC52 {
			// Fall back to OSC 52
			return OSC52Paste(OSC52PasteTimeout)
		}
	}
	return text, err
}

// pasteNative attempts to paste using native clipboard tools
func pasteNative() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS: use pbpaste
		cmd = exec.Command("pbpaste")
	case "linux":
		// Linux: detect WSL and use Windows clipboard interop if available
		if isWSL() {
			if _, err := exec.LookPath("powershell.exe"); err == nil {
				// Avoid PowerShell adding a trailing newline by writing directly to Console.Out
				cmd = exec.Command("powershell.exe", "-NoProfile", "-Command", "[Console]::Out.Write((Get-Clipboard -Raw))")
			} else {
				return "", fmt.Errorf("WSL detected but powershell.exe not found")
			}
			break
		}
		// Native Linux: try xclip first, then xsel
		// Try xclip
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-out")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			// Try xsel
			cmd = exec.Command("xsel", "--clipboard", "--output")
		} else {
			return "", fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Normalize newlines: Windows clipboard returns CRLF; convert to LF
	s := string(output)
	if strings.Contains(s, "\r\n") {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}
	return s, nil
}

// isWSL returns true when running under Windows Subsystem for Linux
func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	// Environment variables set by WSL
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	// Check kernel/os release strings
	if data, err := os.ReadFile("/proc/version"); err == nil {
		v := strings.ToLower(string(data))
		if strings.Contains(v, "microsoft") || strings.Contains(v, "wsl") {
			return true
		}
	}
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		v := strings.ToLower(string(data))
		if strings.Contains(v, "microsoft") || strings.Contains(v, "wsl") {
			return true
		}
	}
	return false
}
