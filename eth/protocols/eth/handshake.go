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

package eth

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	// handshakeTimeout is the maximum allowed time for the `eth` handshake to
	// complete before dropping the connection.= as malicious.
	handshakeTimeout = 5 * time.Second
)

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *Peer) Handshake(network uint64, td *big.Int, head common.Hash, genesis common.Hash, forkID forkid.ID, forkFilter forkid.Filter) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)

	var (
		status63 statusData63 // safe to read after two values have been received from errc
		status   statusData   // safe to read after two values have been received from errc
	)
	go func() {
		switch {
		case p.version == ETH63:
			errc <- p2p.Send(p.rw, statusMsg, &statusData63{
				ProtocolVersion: uint32(p.version),
				NetworkId:       network,
				TD:              td,
				CurrentBlock:    head,
				GenesisBlock:    genesis,
			})
		case p.version >= ETH64:
			errc <- p2p.Send(p.rw, statusMsg, &statusData{
				ProtocolVersion: uint32(p.version),
				NetworkID:       network,
				TD:              td,
				Head:            head,
				Genesis:         genesis,
				ForkID:          forkID,
			})
		default:
			panic(fmt.Sprintf("unsupported eth protocol version: %d", p.version))
		}
	}()
	go func() {
		switch {
		case p.version == ETH63:
			errc <- p.readStatusLegacy(network, &status63, genesis)
		case p.version >= ETH64:
			errc <- p.readStatus(network, &status, genesis, forkFilter)
		default:
			panic(fmt.Sprintf("unsupported eth protocol version: %d", p.version))
		}
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	switch {
	case p.version == ETH63:
		p.td, p.head = status63.TD, status63.CurrentBlock
	case p.version >= ETH64:
		p.td, p.head = status.TD, status.Head
	default:
		panic(fmt.Sprintf("unsupported eth protocol version: %d", p.version))
	}
	// TD at mainnet block #7753254 is 76 bits. If it becomes 100 million times
	// larger, it will still fit within 100 bits
	if tdlen := p.td.BitLen(); tdlen > 100 {
		return fmt.Errorf("too large block TD: bitlen %d", tdlen)
	}
	return nil
}

// readStatusLegacy reads the remote handshake message for eth/63 and older.
func (p *Peer) readStatusLegacy(network uint64, status *statusData63, genesis common.Hash) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != statusMsg {
		return fmt.Errorf("%w: first msg has code %x (!= %x)", errNoStatusMsg, msg.Code, statusMsg)
	}
	if msg.Size > maxMessageSize {
		return fmt.Errorf("%w: %v > %v", errMsgTooLarge, msg.Size, maxMessageSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	if status.GenesisBlock != genesis {
		return fmt.Errorf("%w: %x (!= %x)", errGenesisMismatch, status.GenesisBlock[:8], genesis[:8])
	}
	if status.NetworkId != network {
		return fmt.Errorf("%w: %d (!= %d)", errNetworkIDMismatch, status.NetworkId, network)
	}
	if uint(status.ProtocolVersion) != p.version {
		return fmt.Errorf("%w: %d (!= %d)", errProtocolVersionMismatch, status.ProtocolVersion, p.version)
	}
	return nil
}

// readStatus reads the remote handshake message for eth/64 and newer.
func (p *Peer) readStatus(network uint64, status *statusData, genesis common.Hash, forkFilter forkid.Filter) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != statusMsg {
		return fmt.Errorf("%w: first msg has code %x (!= %x)", errNoStatusMsg, msg.Code, statusMsg)
	}
	if msg.Size > maxMessageSize {
		return fmt.Errorf("%w: %v > %v", errMsgTooLarge, msg.Size, maxMessageSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	if status.NetworkID != network {
		return fmt.Errorf("%w: %d (!= %d)", errNetworkIDMismatch, status.NetworkID, network)
	}
	if uint(status.ProtocolVersion) != p.version {
		return fmt.Errorf("%w: %d (!= %d)", errProtocolVersionMismatch, status.ProtocolVersion, p.version)
	}
	if status.Genesis != genesis {
		return fmt.Errorf("%w: %x (!= %x)", errGenesisMismatch, status.Genesis, genesis)
	}
	if err := forkFilter(status.ForkID); err != nil {
		return fmt.Errorf("%w: %v", errForkIDRejected, err)
	}
	return nil
}
