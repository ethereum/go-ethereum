// Copyright 2019 The Swarm Authors
// This file is part of the Swarm library.
//
// The Swarm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Swarm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Swarm library. If not, see <http://www.gnu.org/licenses/>.

package testutil

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/network"
)

// NodeConfigAtPo brute forces a node config to create a node that has an overlay address at the provided po in relation to the given baseaddr
func NodeConfigAtPo(t *testing.T, baseaddr []byte, po int) *adapters.NodeConfig {
	foundPo := -1
	var conf *adapters.NodeConfig
	for foundPo != po {
		conf = adapters.RandomNodeConfig()
		ip := net.IPv4(127, 0, 0, 1)
		enrIP := enr.IP(ip)
		conf.Record.Set(&enrIP)
		enrTCPPort := enr.TCP(conf.Port)
		conf.Record.Set(&enrTCPPort)
		enrUDPPort := enr.UDP(0)
		conf.Record.Set(&enrUDPPort)

		err := enode.SignV4(&conf.Record, conf.PrivateKey)
		if err != nil {
			t.Fatalf("unable to generate ENR: %v", err)
		}
		nod, err := enode.New(enode.V4ID{}, &conf.Record)
		if err != nil {
			t.Fatalf("unable to create enode: %v", err)
		}

		n := network.NewAddr(nod)
		foundPo = chunk.Proximity(baseaddr, n.Over())
	}

	return conf
}
