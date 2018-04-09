package eventfd

/*
 * eventfd wrapper for go
 * Provides a ReadWriteCloser interface for handling eventfd()'s
 * Eventfd provides a simple filedescriptor with very low overhead.
 * It stores a bitfield of 64 bits which are added when written to
 * the fd.
 *
 * For more information on eventfd() see `man eventfd`.
 */

import (
	"fmt"
	"syscall"

	"github.com/gxed/GoEndian"
)

type EventFD struct {
	fd    int
	valid bool
}

/* Create a new EventFD. */
func New() (*EventFD, error) {
	fd, _, err := syscall.Syscall(syscall.SYS_EVENTFD2, 0, uintptr(syscall.O_CLOEXEC), 0)
	if err != 0 {
		return nil, err
	}

	e := &EventFD{
		fd:    int(fd),
		valid: true,
	}
	return e, nil
}

/* Read events from Eventfd. p should be at max 8 bytes.
 * Returns the number of read bytes or 0 and error is set.
 */
func (e *EventFD) Read(p []byte) (int, error) {
	n, err := syscall.Read(e.fd, p[:])
	if err != nil {
		return 0, err
	}
	return n, nil
}

/* Read events into a uint64 and return it. Returns 0 and error
 * if an error occured
 */
func (e *EventFD) ReadEvents() (uint64, error) {
	buf := make([]byte, 8)
	n, err := syscall.Read(e.fd, buf[:])
	if err != nil {
		return 0, err
	}
	if n != 8 {
		return 0, fmt.Errorf("could not read for eventfd")
	}

	val := endian.Endian.Uint64(buf)
	return val, nil
}

/* Write bytes to eventfd. Will be added to the current
 * value of the internal uint64 of the eventfd().
 */
func (e *EventFD) Write(p []byte) (int, error) {
	n, err := syscall.Write(e.fd, p[:])
	if err != nil {
		return 0, err
	}
	return n, nil
}

/* Write a uint64 to eventfd. Value will be added to current value
 * of the eventfd
 */
func (e *EventFD) WriteEvents(val uint64) error {
	buf := make([]byte, 8)
	endian.Endian.PutUint64(buf, val)

	n, err := syscall.Write(e.fd, buf[:])
	if err != nil {
		return err
	}
	if n != 8 {
		return fmt.Errorf("could not write to eventfd")
	}

	return nil
}

/* Returns the filedescriptor which is internally used */
func (e *EventFD) Fd() int {
	return e.fd
}

/* Close the eventfd */
func (e *EventFD) Close() error {
	if e.valid == false {
		return nil
	}
	e.valid = false
	return syscall.Close(e.fd)
}
