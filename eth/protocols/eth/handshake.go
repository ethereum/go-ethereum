// Copyright 2020 The go-ethereum Authors
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
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	// handshakeTimeout is the maximum allowed time for the `eth` handshake to
	// complete before dropping the connection.= as malicious.
	handshakeTimeout = 5 * time.Second
)

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *Peer) Handshake(networkID uint64, chain *core.BlockChain, rangeMsg BlockRangeUpdatePacket) error {
	switch p.version {
	case ETH69:
		return p.handshake69(networkID, chain, rangeMsg)
	case ETH68:
		return p.handshake68(networkID, chain)
	default:
		return errors.New("unsupported protocol version")
	}
}

func (p *Peer) handshake68(networkID uint64, chain *core.BlockChain) error {
	var (
		genesis    = chain.Genesis()
		latest     = chain.CurrentBlock()
		forkID     = forkid.NewID(chain.Config(), genesis, latest.Number.Uint64(), latest.Time)
		forkFilter = forkid.NewFilter(chain)
	)
	errc := make(chan error, 2)
	go func() {
		pkt := &StatusPacket68{
			ProtocolVersion: uint32(p.version),
			NetworkID:       networkID,
			Head:            latest.Hash(),
			Genesis:         genesis.Hash(),
			ForkID:          forkID,
		}
		errc <- p2p.Send(p.rw, StatusMsg, pkt)
	}()
	var status StatusPacket68 // safe to read after two values have been received from errc
	go func() {
		errc <- p.readStatus68(networkID, &status, genesis.Hash(), forkFilter)
	}()

	return waitForHandshake(errc, p)
}

func (p *Peer) readStatus68(networkID uint64, status *StatusPacket68, genesis common.Hash, forkFilter forkid.Filter) error {
	if err := p.readStatusMsg(status); err != nil {
		return err
	}
	if status.NetworkID != networkID {
		return fmt.Errorf("%w: %d (!= %d)", errNetworkIDMismatch, status.NetworkID, networkID)
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

func (p *Peer) handshake69(networkID uint64, chain *core.BlockChain, rangeMsg BlockRangeUpdatePacket) error {
	var (
		genesis    = chain.Genesis()
		latest     = chain.CurrentBlock()
		forkID     = forkid.NewID(chain.Config(), genesis, latest.Number.Uint64(), latest.Time)
		forkFilter = forkid.NewFilter(chain)
	)

	errc := make(chan error, 2)
	go func() {
		pkt := &StatusPacket69{
			ProtocolVersion: uint32(p.version),
			NetworkID:       networkID,
			Genesis:         genesis.Hash(),
			ForkID:          forkID,
			EarliestBlock:   rangeMsg.EarliestBlock,
			LatestBlock:     rangeMsg.LatestBlock,
			LatestBlockHash: rangeMsg.LatestBlockHash,
		}
		errc <- p2p.Send(p.rw, StatusMsg, pkt)
	}()
	var status StatusPacket69 // safe to read after two values have been received from errc
	go func() {
		errc <- p.readStatus69(networkID, &status, genesis.Hash(), forkFilter)
	}()

	return waitForHandshake(errc, p)
}

func (p *Peer) readStatus69(networkID uint64, status *StatusPacket69, genesis common.Hash, forkFilter forkid.Filter) error {
	if err := p.readStatusMsg(status); err != nil {
		return err
	}
	if status.NetworkID != networkID {
		return fmt.Errorf("%w: %d (!= %d)", errNetworkIDMismatch, status.NetworkID, networkID)
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
	// Handle initial block range.
	initRange := &BlockRangeUpdatePacket{
		EarliestBlock:   status.EarliestBlock,
		LatestBlock:     status.LatestBlock,
		LatestBlockHash: status.LatestBlockHash,
	}
	if err := initRange.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errInvalidBlockRange, err)
	}
	p.lastRange.Store(initRange)
	return nil
}

// readStatusMsg reads the first message on the connection.
func (p *Peer) readStatusMsg(dst any) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return fmt.Errorf("%w: first msg has code %x (!= %x)", errNoStatusMsg, msg.Code, StatusMsg)
	}
	if msg.Size > maxMessageSize {
		return fmt.Errorf("%w: %v > %v", errMsgTooLarge, msg.Size, maxMessageSize)
	}
	if err := msg.Decode(dst); err != nil {
		return err
	}
	return nil
}

func waitForHandshake(errc <-chan error, p *Peer) error {
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for range 2 {
		select {
		case err := <-errc:
			if err != nil {
				markError(p, err)
				return err
			}
		case <-timeout.C:
			markError(p, p2p.DiscReadTimeout)
			return p2p.DiscReadTimeout
		}
	}
	return nil
}

// markError registers the error with the corresponding metric.
func markError(p *Peer, err error) {
	if !metrics.Enabled() {
		return
	}
	m := meters.get(p.Inbound())
	switch errors.Unwrap(err) {
	case errNetworkIDMismatch:
		m.networkIDMismatch.Mark(1)
	case errProtocolVersionMismatch:
		m.protocolVersionMismatch.Mark(1)
	case errGenesisMismatch:
		m.genesisMismatch.Mark(1)
	case errForkIDRejected:
		m.forkidRejected.Mark(1)
	case p2p.DiscReadTimeout:
		m.timeoutError.Mark(1)
	default:
		m.peerError.Mark(1)
	}
}

// Validate checks basic validity of a block range announcement.
func (p *BlockRangeUpdatePacket) Validate() error {
	if p.EarliestBlock > p.LatestBlock {
		return errors.New("earliest > latest")
	}
	if p.LatestBlockHash == (common.Hash{}) {
		return errors.New("zero latest hash")
	}
	return nil
}
