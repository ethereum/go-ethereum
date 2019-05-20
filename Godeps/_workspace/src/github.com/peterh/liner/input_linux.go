// +build linux

package liner

import "syscall"

const (
	getTermios = syscall.TCGETS
	setTermios = syscall.TCSETS
)

const (
	icrnl  = syscall.ICRNL
	inpck  = syscall.INPCK
	istrip = syscall.ISTRIP
	ixon   = syscall.IXON
	opost  = syscall.OPOST
	cs8    = syscall.CS8
	isig   = syscall.ISIG
	icanon = syscall.ICANON
	iexten = syscall.IEXTEN
)

type termios struct {
	syscall.Termios
}
