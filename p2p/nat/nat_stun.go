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
	"fmt"
	"net"
	"time"

	stunV2 "github.com/pion/stun/v2"
)

var stunDefaultServerList = []string{
	"159.223.0.83:3478",
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun2.l.google.com:19302",
	"stun3.l.google.com:19302",
	"stun4.l.google.com:19302",
	"stun01.sipphone.com",
	"stun.ekiga.net",
	"stun.fwdnet.net",
	"stun.ideasip.com",
	"stun.iptel.org",
	"stun.rixtelecom.se",
	"stun.schlund.de",
	"stunserver.org",
	"stun.softjoys.com",
	"stun.voiparound.com",
	"stun.voipbuster.com",
	"stun.voipstunt.com",
	"stun.voxgratia.org",
	"stun.xten.com",
}

const requestLimit = 3

type stun struct {
	serverList      []string
	activeIndex     int // the server index which return the IP
	pendingRequests int // request in flight
	askedIndex      map[int]struct{}
	replyCh         chan stunResponse
}

func newSTUN(serverAddr string) (Interface, error) {
	serverList := make([]string, 0)
	if serverAddr == "default" {
		serverList = stunDefaultServerList
	} else {
		_, err := net.ResolveUDPAddr("udp4", serverAddr)
		if err != nil {
			return nil, err
		}
		serverList = append(serverList, serverAddr)
	}

	return &stun{
		serverList: serverList,
	}, nil
}

func (s stun) String() string {
	return fmt.Sprintf("STUN(%s)", s.serverList[s.activeIndex])
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

type stunResponse struct {
	ip    net.IP
	err   error
	index int
}

func (s *stun) ExternalIP() (net.IP, error) {
	var err error
	s.replyCh = make(chan stunResponse, requestLimit)
	s.askedIndex = make(map[int]struct{})
	for s.startQueries() {
		response := <-s.replyCh
		s.pendingRequests--
		if response.err != nil {
			err = response.err
			continue
		}
		s.activeIndex = response.index
		return response.ip, nil
	}
	return nil, err
}

func (s *stun) startQueries() bool {
	for i := 0; s.pendingRequests < requestLimit && i < len(s.serverList); i++ {
		_, exist := s.askedIndex[i]
		if exist {
			continue
		}
		s.pendingRequests++
		s.askedIndex[i] = struct{}{}
		go func(index int, server string) {
			ip, err := externalIP(server)
			s.replyCh <- stunResponse{
				ip:    ip,
				index: index,
				err:   err,
			}
		}(i, s.serverList[i])
	}
	return s.pendingRequests > 0
}

func externalIP(server string) (net.IP, error) {
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

	return mappedAddr.IP, nil
}
