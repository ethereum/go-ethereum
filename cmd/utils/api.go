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

// PublicWeb3Api offers helper utils
type PublicWeb3Api struct {
	stack *node.Node
}

// NewPublicWeb3Api creates a new Web3Service instance
func NewPublicWeb3Api(stack *node.Node) *PublicWeb3Api {
	return &PublicWeb3Api{stack}
}

// ClientVersion returns the node name
func (s *PublicWeb3Api) ClientVersion() string {
	return s.stack.Server().Name
}

// Sha3 applies the ethereum sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *PublicWeb3Api) Sha3(input string) string {
	return common.ToHex(crypto.Sha3(common.FromHex(input)))
}

// NetService offers network related RPC methods
type PublicNetApi struct {
	net            *p2p.Server
	networkVersion int
}

// NewPublicNetApi creates a new net api instance.
func NewPublicNetApi(net *p2p.Server, networkVersion int) *PublicNetApi {
	return &PublicNetApi{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *PublicNetApi) Listening() bool {
	return true // always listening
}

// Peercount returns the number of connected peers
func (s *PublicNetApi) PeerCount() *rpc.HexNumber {
	return rpc.NewHexNumber(s.net.PeerCount())
}

func (s *PublicNetApi) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
