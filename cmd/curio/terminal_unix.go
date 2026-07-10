//go:build !windows

package main

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

func makeStdinHidden() (interface{}, error) {
	fd := int(os.Stdin.Fd())
	var oldState syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(getTermiosIoctl()), uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); err != 0 {
		return nil, err
	}
	newState := oldState
	newState.Lflag &^= syscall.ECHO
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(setTermiosIoctl()), uintptr(unsafe.Pointer(&newState)), 0, 0, 0); err != 0 {
		return nil, err
	}
	return oldState, nil
}

func restoreStdin(state interface{}) {
	if state == nil {
		return
	}
	fd := int(os.Stdin.Fd())
	if oldState, ok := state.(syscall.Termios); ok {
		_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(setTermiosIoctl()), uintptr(unsafe.Pointer(&oldState)), 0, 0, 0)
	}
}

func getTermiosIoctl() uintptr {
	if runtime.GOOS == "darwin" {
		return 0x40487413 // TIOCGETA on macOS
	}
	return 0x5401 // TCGETS on Linux
}

func setTermiosIoctl() uintptr {
	if runtime.GOOS == "darwin" {
		return 0x80487414 // TIOCSETA on macOS
	}
	return 0x5402 // TCSETS on Linux
}
