// Copyright 2018 The go-ethereum Authors
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

package simulation

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

// Service returns a single Service by name on a particular node
// with provided id.
func (s *Simulation) Service(name string, id discover.NodeID) node.Service {
	simNode, ok := s.Net.GetNode(id).Node.(*adapters.SimNode)
	if !ok {
		return nil
	}
	services := simNode.ServiceMap()
	if len(services) == 0 {
		return nil
	}
	return services[name]
}

// RandomService returns a single Service by name on a
// randomly chosen node that is up.
func (s *Simulation) RandomService(name string) node.Service {
	n := s.RandomUpNode()
	if n == nil {
		return nil
	}
	return n.Service(name)
}

// Services returns all services with a provided name
// from nodes that are up.
func (s *Simulation) Services(name string) (services map[discover.NodeID]node.Service) {
	nodes := s.Net.GetNodes()
	services = make(map[discover.NodeID]node.Service)
	for _, node := range nodes {
		if !node.Up {
			continue
		}
		simNode, ok := node.Node.(*adapters.SimNode)
		if !ok {
			continue
		}
		services[node.ID()] = simNode.Service(name)
	}
	return services
}
