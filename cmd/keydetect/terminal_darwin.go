//go:build darwin

package main

import (
	"golang.org/x/sys/unix"
	"os"
)

var originalTermios *unix.Termios

func setRawMode() error {
	fd := int(os.Stdin.Fd())

	// Read current settings (Darwin uses TIOCGETA/TIOCSETA)
	t, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return err
	}
	originalTermios = t

	// Make a copy and update flags
	raw := *t
	raw.Lflag &^= (unix.ICANON | unix.ECHO)
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	// Apply
	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &raw); err != nil {
		return err
	}
	return nil
}

func restoreTerminal() {
	if originalTermios == nil {
		return
	}
	fd := int(os.Stdin.Fd())
	_ = unix.IoctlSetTermios(fd, unix.TIOCSETA, originalTermios)
}
