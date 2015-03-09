// +build darwin

package liner

import "syscall"

const (
	getTermios = syscall.TIOCGETA
	setTermios = syscall.TIOCSETA
)

const (
	// Input flags
	inpck  = 0x010
	istrip = 0x020
	icrnl  = 0x100
	ixon   = 0x200

	// Output flags
	opost = 0x1

	// Control flags
	cs8 = 0x300

	// Local flags
	isig   = 0x080
	icanon = 0x100
	iexten = 0x400
)

type termios struct {
	Iflag  uintptr
	Oflag  uintptr
	Cflag  uintptr
	Lflag  uintptr
	Cc     [20]byte
	Ispeed uintptr
	Ospeed uintptr
}
