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

package utils

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

// PublicWeb3API offers helper utils
type PublicWeb3API struct {
	stack *node.Node
}

// NewPublicWeb3API creates a new Web3Service instance
func NewPublicWeb3API(stack *node.Node) *PublicWeb3API {
	return &PublicWeb3API{stack}
}

// ClientVersion returns the node name
func (s *PublicWeb3API) ClientVersion() string {
	return s.stack.Server().Name
}

// Sha3 applies the ethereum sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *PublicWeb3API) Sha3(input string) string {
	return common.ToHex(crypto.Sha3(common.FromHex(input)))
}

// PublicNetAPI offers network related RPC methods
type PublicNetAPI struct {
	net            *p2p.Server
	networkVersion int
}

// NewPublicNetAPI creates a new net api instance.
func NewPublicNetAPI(net *p2p.Server, networkVersion int) *PublicNetAPI {
	return &PublicNetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetAPI) Listening() bool {
	return true // always listening
}

// Peercount returns the number of connected peers
func (s *PublicNetAPI) PeerCount() *rpc.HexNumber {
	return rpc.NewHexNumber(s.net.PeerCount())
}

// ProtocolVersion returns the current ethereum protocol version.
func (s *PublicNetAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
