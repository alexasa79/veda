//go:build linux
// +build linux

package terminal

import "golang.org/x/sys/unix"

// Terminal control constants for Linux
const (
	tcGet = unix.TCGETS
	tcSet = unix.TCSETS
)
