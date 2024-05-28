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

package discover

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"
	"slices"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type BucketNode struct {
	Node          *enode.Node `json:"node"`
	AddedToTable  time.Time   `json:"addedToTable"`
	AddedToBucket time.Time   `json:"addedToBucket"`
	Checks        int         `json:"checks"`
	Live          bool        `json:"live"`
}

// tableNode is an entry in Table.
type tableNode struct {
	*enode.Node
	revalList       *revalidationList
	addedToTable    time.Time // first time node was added to bucket or replacement list
	addedToBucket   time.Time // time it was added in the actual bucket
	livenessChecks  uint      // how often liveness was checked
	isValidatedLive bool      // true if existence of node is considered validated right now
}

type encPubkey [64]byte

func encodePubkey(key *ecdsa.PublicKey) encPubkey {
	var e encPubkey
	math.ReadBits(key.X, e[:len(e)/2])
	math.ReadBits(key.Y, e[len(e)/2:])
	return e
}

func decodePubkey(curve elliptic.Curve, e []byte) (*ecdsa.PublicKey, error) {
	if len(e) != len(encPubkey{}) {
		return nil, errors.New("wrong size public key data")
	}
	p := &ecdsa.PublicKey{Curve: curve, X: new(big.Int), Y: new(big.Int)}
	half := len(e) / 2
	p.X.SetBytes(e[:half])
	p.Y.SetBytes(e[half:])
	if !p.Curve.IsOnCurve(p.X, p.Y) {
		return nil, errors.New("invalid curve point")
	}
	return p, nil
}

func (e encPubkey) id() enode.ID {
	return enode.ID(crypto.Keccak256Hash(e[:]))
}

func unwrapNodes(ns []*tableNode) []*enode.Node {
	result := make([]*enode.Node, len(ns))
	for i, n := range ns {
		result[i] = n.Node
	}
	return result
}

func (n *tableNode) String() string {
	return n.Node.String()
}

// nodesByDistance is a list of nodes, ordered by distance to target.
type nodesByDistance struct {
	entries []*enode.Node
	target  enode.ID
}

// push adds the given node to the list, keeping the total size below maxElems.
func (h *nodesByDistance) push(n *enode.Node, maxElems int) {
	ix := sort.Search(len(h.entries), func(i int) bool {
		return enode.DistCmp(h.target, h.entries[i].ID(), n.ID()) > 0
	})

	end := len(h.entries)
	if len(h.entries) < maxElems {
		h.entries = append(h.entries, n)
	}
	if ix < end {
		// Slide existing entries down to make room.
		// This will overwrite the entry we just appended.
		copy(h.entries[ix+1:], h.entries[ix:])
		h.entries[ix] = n
	}
}

type nodeType interface {
	ID() enode.ID
}

// containsID reports whether ns contains a node with the given ID.
func containsID[N nodeType](ns []N, id enode.ID) bool {
	for _, n := range ns {
		if n.ID() == id {
			return true
		}
	}
	return false
}

// deleteNode removes a node from the list.
func deleteNode[N nodeType](list []N, id enode.ID) []N {
	return slices.DeleteFunc(list, func(n N) bool {
		return n.ID() == id
	})
}
