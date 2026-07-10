package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

// ---- Terminal helpers ----

var (
	stdinReader = bufio.NewReader(os.Stdin)
)

func termBold() string {
	if termSupportsColor() {
		return "\033[1m"
	}
	return ""
}

func termDim() string {
	if termSupportsColor() {
		return "\033[2m"
	}
	return ""
}

func termBlue() string {
	if termSupportsColor() {
		return "\033[34m"
	}
	return ""
}

func termGreen() string {
	if termSupportsColor() {
		return "\033[32m"
	}
	return ""
}

func termYellow() string {
	if termSupportsColor() {
		return "\033[33m"
	}
	return ""
}

func termReset() string {
	if termSupportsColor() {
		return "\033[0m"
	}
	return ""
}

func termSupportsColor() bool {
	if !isTerminal() {
		return false
	}
	return true
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func clearScreen() {
	if !isTerminal() {
		return
	}
	fmt.Print("\033[2J\033[3J\033[H")
}

func openBrowser(url string) {
	fmt.Printf("  %s↗ opening%s %s\n", termGreen(), termReset(), url)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		fmt.Printf("  %s⚠ couldn't open browser — visit it manually: %s%s\n", termYellow(), url, termReset())
		return
	}
	_ = cmd.Start()
}

func say(msg string) {
	fmt.Printf("  %s\n", msg)
}

func step(msg string) {
	fmt.Printf("  %s•%s %s\n", termBlue(), termReset(), msg)
}

func note(msg string) {
	fmt.Printf("  %s%s%s\n", termDim(), msg, termReset())
}

func warn(msg string) {
	fmt.Printf("  %s⚠ %s%s\n", termYellow(), msg, termReset())
}

func success(msg string) {
	fmt.Printf("  %s✓ %s%s\n", termGreen(), msg, termReset())
}

func pause(msg string) {
	if msg == "" {
		msg = "Press Enter to continue"
	}
	fmt.Printf("  %s%s%s ", termDim(), msg, termReset())
	_, _ = stdinReader.ReadString('\n')
}

func confirm(question string) bool {
	fmt.Printf("  %s? %s [y/N] %s", termYellow(), question, termReset())
	reply, _ := stdinReader.ReadString('\n')
	reply = strings.TrimSpace(reply)
	return strings.HasPrefix(strings.ToLower(reply), "y")
}

func ask(prompt string) string {
	fmt.Printf("  %s%s%s ", termBold(), prompt, termReset())
	input, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(input)
}

func askWithDefault(prompt, key string) string {
	current := configGet(key)
	if current != "" {
		fmt.Printf("  %s%s%s %s[Enter keeps current]%s ", termBold(), prompt, termReset(), termDim(), termReset())
	} else {
		fmt.Printf("  %s%s%s ", termBold(), prompt, termReset())
	}
	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" && current != "" {
		return current
	}
	return input
}

func askSecret(prompt string) string {
	fmt.Printf("  %s%s%s %s(paste + Enter, hidden)%s ", termBold(), prompt, termReset(), termDim(), termReset())
	oldState, _ := makeStdinHidden()
	input, _ := stdinReader.ReadString('\n')
	restoreStdin(oldState)
	input = strings.TrimSpace(input)
	fmt.Printf("\r  %s✓ received%s\n", termGreen(), termReset())
	return input
}

func askSecretWithDefault(prompt, key string) string {
	current := configGet(key)
	if current != "" {
		fmt.Printf("  %s%s%s %s[Enter keeps current, paste to replace — hidden]%s ", termBold(), prompt, termReset(), termDim(), termReset())
	} else {
		fmt.Printf("  %s%s%s %s(paste + Enter, hidden)%s ", termBold(), prompt, termReset(), termDim(), termReset())
	}
	oldState, _ := makeStdinHidden()
	input, _ := stdinReader.ReadString('\n')
	restoreStdin(oldState)
	input = strings.TrimSpace(input)
	fmt.Printf("\r  %s✓ received%s\n", termGreen(), termReset())
	if input == "" && current != "" {
		return current
	}
	return input
}

func makeStdinHidden() (interface{}, error) {
	fd := int(os.Stdin.Fd())
	var oldState syscall.Termios
	if runtime.GOOS == "windows" {
		return nil, nil
	}
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
