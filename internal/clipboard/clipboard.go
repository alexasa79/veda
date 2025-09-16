package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Copy copies text to the system clipboard
func Copy(text string) error {
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

// Paste reads text from the system clipboard
func Paste() (string, error) {
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
