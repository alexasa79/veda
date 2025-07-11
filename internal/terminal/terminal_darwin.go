// +build darwin

package terminal

import "golang.org/x/sys/unix"

// Terminal control constants for macOS
const (
	tcGet = unix.TIOCGETA
	tcSet = unix.TIOCSETA
)