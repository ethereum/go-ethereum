// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package nat

import (
	"fmt"
	"net"
	"time"

	"github.com/pion/stun"
)

// The code are from erigon p2p/nat/nat_stun.go
// This stun server is part of the mainnet infrastructure.
// The addr are from https://github.com/ethereum/trin/blob/master/portalnet/src/socket.rs
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
