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
	"context"
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/swarm/network"
)

// BucketKeyKademlia is the key to be used for storing the kademlia
// instance for particular node, usually inside the ServiceFunc function.
var BucketKeyKademlia BucketKey = "kademlia"

// WaitTillHealthy is blocking until the health of all kademlias is true.
// If error is not nil, a map of kademlia that was found not healthy is returned.
// TODO: Check correctness since change in kademlia depth calculation logic
func (s *Simulation) WaitTillHealthy(ctx context.Context) (ill map[enode.ID]*network.Kademlia, err error) {
	// Prepare PeerPot map for checking Kademlia health
	var ppmap map[string]*network.PeerPot
	kademlias := s.kademlias()
	addrs := make([][]byte, 0, len(kademlias))
	// TODO verify that all kademlias have same params
	for _, k := range kademlias {
		addrs = append(addrs, k.BaseAddr())
	}
	ppmap = network.NewPeerPotMap(s.neighbourhoodSize, addrs)

	// Wait for healthy Kademlia on every node before checking files
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	ill = make(map[enode.ID]*network.Kademlia)
	for {
		select {
		case <-ctx.Done():
			return ill, ctx.Err()
		case <-ticker.C:
			for k := range ill {
				delete(ill, k)
			}
			log.Debug("kademlia health check", "addr count", len(addrs), "kad len", len(kademlias))
			for id, k := range kademlias {
				//PeerPot for this node
				addr := common.Bytes2Hex(k.BaseAddr())
				pp := ppmap[addr]
				//call Healthy RPC
				h := k.GetHealthInfo(pp)
				//print info
				log.Debug(k.String())
				log.Debug("kademlia", "connectNN", h.ConnectNN, "knowNN", h.KnowNN)
				log.Debug("kademlia", "health", h.ConnectNN && h.KnowNN, "addr", hex.EncodeToString(k.BaseAddr()), "node", id)
				log.Debug("kademlia", "ill condition", !h.ConnectNN, "addr", hex.EncodeToString(k.BaseAddr()), "node", id)
				if !h.Healthy() {
					ill[id] = k
				}
			}
			if len(ill) == 0 {
				return nil, nil
			}
		}
	}
}

// kademlias returns all Kademlia instances that are set
// in simulation bucket.
func (s *Simulation) kademlias() (ks map[enode.ID]*network.Kademlia) {
	items := s.UpNodesItems(BucketKeyKademlia)
	log.Debug("kademlia len items", "len", len(items))
	ks = make(map[enode.ID]*network.Kademlia, len(items))
	for id, v := range items {
		k, ok := v.(*network.Kademlia)
		if !ok {
			continue
		}
		ks[id] = k
	}
	return ks
}

// WaitTillSnapshotRecreated is blocking until all the connections specified
// in the snapshot are registered in the kademlia.
// It differs from WaitTillHealthy, which waits only until all the kademlias are
// healthy (it might happen even before all the connections are established).
func (s *Simulation) WaitTillSnapshotRecreated(ctx context.Context, snap *simulations.Snapshot) error {
	expected := getSnapshotConnections(snap.Conns)
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			actual := s.getActualConnections()
			if isAllDeployed(expected, actual) {
				return nil
			}
		}
	}
}

func (s *Simulation) getActualConnections() (res []uint64) {
	kademlias := s.kademlias()
	for base, k := range kademlias {
		k.EachConn(base[:], 256, func(p *network.Peer, _ int) bool {
			res = append(res, getConnectionHash(base, p.ID()))
			return true
		})
	}

	// only list those connections that appear twice (both peers should recognize connection as active)
	res = removeDuplicatesAndSingletons(res)
	return res
}

func getSnapshotConnections(conns []simulations.Conn) (res []uint64) {
	for _, c := range conns {
		res = append(res, getConnectionHash(c.One, c.Other))
	}
	return res
}

// returns an integer connection identifier (similar to 8-byte hash)
func getConnectionHash(a, b enode.ID) uint64 {
	var h [8]byte
	for i := 0; i < 8; i++ {
		h[i] = a[i] ^ b[i]
	}
	res := binary.LittleEndian.Uint64(h[:])
	return res
}

// returns true if all connections in expected are listed in actual
func isAllDeployed(expected []uint64, actual []uint64) bool {
	if len(expected) == 0 {
		return true
	}

	exp := make([]uint64, len(expected))
	copy(exp, expected)
	for _, c := range actual {
		// remove value c from exp
		for i := 0; i < len(exp); i++ {
			if exp[i] == c {
				exp = removeListElement(exp, i)
				if len(exp) == 0 {
					return true
				}
			}
		}
	}
	return len(exp) == 0
}

func removeListElement(arr []uint64, i int) []uint64 {
	last := len(arr) - 1
	arr[i] = arr[last]
	arr = arr[:last]
	return arr
}

func removeDuplicatesAndSingletons(arr []uint64) []uint64 {
	for i := 0; i < len(arr); {
		found := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				arr = removeListElement(arr, j) // remove duplicate
				found = true
				break
			}
		}

		if found {
			i++
		} else {
			arr = removeListElement(arr, i) // remove singleton
		}
	}

	return arr
}
