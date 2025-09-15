//go:build linux

package main

import (
	"golang.org/x/sys/unix"
	"os"
)

var originalTermios *unix.Termios

func setRawMode() error {
	fd := int(os.Stdin.Fd())

	// Read current settings
	t, err := unix.IoctlGetTermios(fd, unix.TCGETS)
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
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return err
	}
	return nil
}

func restoreTerminal() {
	if originalTermios == nil {
		return
	}
	fd := int(os.Stdin.Fd())
	_ = unix.IoctlSetTermios(fd, unix.TCSETS, originalTermios)
}
