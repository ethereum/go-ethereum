// +build darwin freebsd dragonfly netbsd openbsd linux

package reuseport

import (
	"golang.org/x/sys/unix"
	"sync"
	"syscall"
	"time"

	sockaddrnet "github.com/libp2p/go-sockaddr/net"
)

var (
	hasReusePort bool
	didReusePort sync.Once
)

// Available returns whether or not SO_REUSEPORT is available in the OS.
// It does so by attepting to open a tcp socket, setting the option, and
// checking ENOPROTOOPT on error. After checking, the decision is cached
// for the rest of the process run.
func available() bool {
	didReusePort.Do(checkReusePort)
	return hasReusePort
}

func checkReusePort() {
	// there may be fluke reasons to fail to open a socket.
	// so we give it 5 shots. if not, give up and call it not avail.
	for i := 0; i < 5; i++ {
		// try to setup a TCP socket.
		fd, err := socket(sockaddrnet.AF_INET, sockaddrnet.SOCK_STREAM, sockaddrnet.IPPROTO_TCP)
		if err == nil {
			unix.Close(fd)
			hasReusePort = true
			return
		}

		if errno, ok := err.(syscall.Errno); ok && errno == unix.ENOPROTOOPT {
			return // :( that's all folks.
		}

		// not an errno? or not ENOPROTOOPT? retry.
		time.Sleep(20 * time.Millisecond) // wait a bit
	}
}
