package TableWriter

import (
	"syscall"
	"unsafe"
)

type dividers struct {
	HLine  string
	VLine  string
	TL     string
	TR     string
	BL     string
	BR     string
	Cross  string
	TUp    string
	TDown  string
	TRight string
	TLeft  string
	VLeft  string
	VRight string
}

// Winsize is the structure used for ioctl calls, to obtain the terminal size.
type winsize struct {
	Row    uint16 // Rows number
	Col    uint16 // Columns number (width)
	Xpixel uint16 // Pixel's width
	Ypixel uint16 // Pixel's Height
}

// getTerminalSize retrieves the terminal's size associated to the given file descriptor
func getTerminalSize(fd uintptr) (cols, rows int, err error) {
	ws := &winsize{}

	// TIOCGWINSZ is the constant that tells the kernel to retrieve the TTY size.
	// Using the TIOCGWINSZ syscall is tailored to Linux/macOS.
	ret, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)

	if int(ret) == -1 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}
