// Copyright 2024 The go-ethereum Authors
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
	_ "embed"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	stunV2 "github.com/pion/stun/v2"
)

//go:embed stun-list.txt
var stunDefaultServers string

const requestLimit = 3

type stun struct {
	serverList []string
}

func newSTUN(serverAddr string) (Interface, error) {
	s := new(stun)
	if serverAddr == "default" || serverAddr == "" {
		s.serverList = strings.Split(stunDefaultServers, "\n")
	} else {
		_, err := net.ResolveUDPAddr("udp4", serverAddr)
		if err != nil {
			return nil, err
		}
		s.serverList = []string{serverAddr}
	}
	return s, nil
}

func (s stun) String() string {
	if len(s.serverList) == 1 {
		return fmt.Sprintf("STUN(%s)", s.serverList[0])
	}
	return "STUN"
}

func (stun) SupportsMapping() bool {
	return false
}

func (stun) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) (uint16, error) {
	return uint16(extport), nil
}

func (stun) DeleteMapping(string, int, int) error {
	return nil
}

func (s *stun) ExternalIP() (net.IP, error) {
	for _, server := range s.randomServers(requestLimit) {
		ip, err := s.externalIP(server)
		if err != nil {
			log.Debug("STUN request failed", "server", server, "err", err)
			continue
		}
		return ip, nil
	}
	return nil, errors.New("STUN requests failed")
}

func (s *stun) randomServers(n int) []string {
	n = min(n, len(s.serverList))
	m := make(map[int]struct{}, n)
	list := make([]string, 0, n)
	for i := 0; i < len(s.serverList)*2 && len(list) < n; i++ {
		index := rand.Intn(len(s.serverList))
		if _, alreadyHit := m[index]; alreadyHit {
			continue
		}
		list = append(list, s.serverList[index])
		m[index] = struct{}{}
	}
	return list
}

func (s *stun) externalIP(server string) (net.IP, error) {
	_, _, err := net.SplitHostPort(server)
	if err != nil {
		server += fmt.Sprintf(":%d", stunV2.DefaultPort)
	}

	log.Trace("Attempting STUN binding request", "server", server)
	conn, err := stunV2.Dial("udp4", server)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	message, err := stunV2.Build(stunV2.TransactionID, stunV2.BindingRequest)
	if err != nil {
		return nil, err
	}

	var responseError error
	var mappedAddr stunV2.XORMappedAddress
	err = conn.Do(message, func(event stunV2.Event) {
		if event.Error != nil {
			responseError = event.Error
			return
		}
		if err := mappedAddr.GetFrom(event.Message); err != nil {
			responseError = err
		}
	})
	if err != nil {
		return nil, err
	}
	if responseError != nil {
		return nil, responseError
	}
	log.Trace("STUN returned IP", "server", server, "ip", mappedAddr.IP)
	return mappedAddr.IP, nil
}
