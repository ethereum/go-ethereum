package nat

import (
	"fmt"
	"net"
	"time"

	"github.com/pion/stun"
)

const STUNDefaultServerAddr = "159.223.0.83:3478"

type STUN struct {
	serverAddr string
}

func NewSTUN(serverAddr string) STUN {
	if serverAddr == "" {
		serverAddr = STUNDefaultServerAddr
	}
	return STUN{serverAddr: serverAddr}
}

func (s STUN) String() string {
	return fmt.Sprintf("STUN(%s)", s.serverAddr)
}

func (STUN) SupportsMapping() bool {
	return false
}

func (STUN) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) (uint16, error) {
	return uint16(extport), nil
}

func (STUN) DeleteMapping(string, int, int) error {
	return nil
}

func (s STUN) ExternalIP() (net.IP, error) {
	conn, err := stun.Dial("udp4", s.serverAddr)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	var response *stun.Event
	err = conn.Do(message, func(event stun.Event) {
		response = &event
	})
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, response.Error
	}

	var mappedAddr stun.XORMappedAddress
	if err := mappedAddr.GetFrom(response.Message); err != nil {
		return nil, err
	}

	return mappedAddr.IP, nil
}
