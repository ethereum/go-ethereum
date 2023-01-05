package network

import (
	"errors"
	"fmt"
	"net"
)

const (
	maxPortCheck = 100

	emptyPort = "127.0.0.1:0"
)

var (
	ErrCantFindAPort = errors.New("no available port found")
)

// FindAvailablePort returns the an available port
func FindAvailablePort() (int, net.Listener, error) {
	var (
		listener net.Listener
		err      error
	)

	for i := uint(0); i < maxPortCheck; i++ {
		listener, err = net.Listen("tcp", emptyPort)
		if err != nil {
			continue
		}

		return listener.Addr().(*net.TCPAddr).Port, listener, nil
	}

	return 0, nil, fmt.Errorf("%w: %s", ErrCantFindAPort, err)
}
