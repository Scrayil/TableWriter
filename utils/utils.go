package utils

import (
	"syscall"
	"unsafe"
)

type Dividers struct {
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

// Winsize è la struttura usata dalle chiamate ioctl per ottenere le dimensioni del terminale.
type Winsize struct {
	Row    uint16 // Numero di righe
	Col    uint16 // Numero di colonne (larghezza)
	Xpixel uint16 // Larghezza in pixel (spesso 0)
	Ypixel uint16 // Altezza in pixel (spesso 0)
}

// GetTerminalSize ottiene le dimensioni del terminale associato al file descriptor (fd).
func GetTerminalSize(fd uintptr) (cols, rows int, err error) {
	ws := &Winsize{}

	// TIOCGWINSZ è la costante che dice al kernel di ottenere la dimensione della finestra TTY.
	// L'uso di syscall.TIOCGWINSZ è specifico per Linux/macOS.
	ret, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		fd,
		uintptr(syscall.TIOCGWINSZ), // La richiesta specifica
		uintptr(unsafe.Pointer(ws)),
	)

	if int(ret) == -1 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}
