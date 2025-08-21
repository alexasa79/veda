package main

import (
	"syscall"
	"unsafe"
)

// termios structure for macOS/Linux
type termios struct {
	Iflag  uint64
	Oflag  uint64
	Cflag  uint64
	Lflag  uint64
	Cc     [20]uint8
	Ispeed uint64
	Ospeed uint64
}

const (
	TCGETS = 0x40487413
	TCSETS = 0x80487414
	ICANON = 0x100
	ECHO   = 0x8
	ISIG   = 0x1
)

var originalTermios termios

func setRawMode() error {
	fd := int(syscall.Stdin)

	// Get current terminal settings
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), TCGETS, uintptr(unsafe.Pointer(&originalTermios)))
	if errno != 0 {
		return errno
	}

	// Make a copy for modification
	raw := originalTermios

	// Turn off canonical mode and echo. Keep ISIG enabled so Ctrl+C (VINTR)
	// still generates SIGINT, allowing graceful exit.
	raw.Lflag &^= (ICANON | ECHO)

	// Set minimum characters to read and timeout
	raw.Cc[6] = 1 // VMIN
	raw.Cc[5] = 0 // VTIME

	// Apply new settings
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), TCSETS, uintptr(unsafe.Pointer(&raw)))
	if errno != 0 {
		return errno
	}

	return nil
}

func restoreTerminal() {
	fd := int(syscall.Stdin)
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), TCSETS, uintptr(unsafe.Pointer(&originalTermios)))
}
