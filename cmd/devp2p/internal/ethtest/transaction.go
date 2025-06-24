// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/internal/utesting"
)

// sendTxs sends the given transactions to the node and
// expects the node to accept and propagate them.
func (s *Suite) sendTxs(t *utesting.T, txs []*types.Transaction) error {
	// Open sending conn.
	sendConn, err := s.dial()
	if err != nil {
		return err
	}
	defer sendConn.Close()
	if err = sendConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}

	// Open receiving conn.
	recvConn, err := s.dial()
	if err != nil {
		return err
	}
	defer recvConn.Close()
	if err = recvConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}

	if err = sendConn.Write(ethProto, eth.TransactionsMsg, eth.TransactionsPacket(txs)); err != nil {
		return fmt.Errorf("failed to write message to connection: %v", err)
	}

	var (
		got = make(map[common.Hash]bool)
		end = time.Now().Add(timeout)
	)

	// Wait for the transaction announcements, make sure all txs ar propagated.
	for time.Now().Before(end) {
		msg, err := recvConn.ReadEth()
		if err != nil {
			return fmt.Errorf("failed to read from connection: %w", err)
		}
		switch msg := msg.(type) {
		case *eth.TransactionsPacket:
			for _, tx := range *msg {
				got[tx.Hash()] = true
			}
		case *eth.NewPooledTransactionHashesPacket:
			for _, hash := range msg.Hashes {
				got[hash] = true
			}
		case *eth.GetBlockHeadersPacket:
			headers, err := s.chain.GetHeaders(msg)
			if err != nil {
				t.Logf("invalid GetBlockHeaders request: %v", err)
			}
			recvConn.Write(ethProto, eth.BlockHeadersMsg, &eth.BlockHeadersPacket{
				RequestId:           msg.RequestId,
				BlockHeadersRequest: headers,
			})
		default:
			return fmt.Errorf("unexpected eth wire msg: %s", pretty.Sdump(msg))
		}

		// Check if all txs received.
		allReceived := func() bool {
			for _, tx := range txs {
				if !got[tx.Hash()] {
					return false
				}
			}
			return true
		}
		if allReceived() {
			return nil
		}
	}

	return errors.New("timed out waiting for txs")
}

func (s *Suite) sendInvalidTxs(t *utesting.T, txs []*types.Transaction) error {
	// Open sending conn.
	sendConn, err := s.dial()
	if err != nil {
		return err
	}
	defer sendConn.Close()
	if err = sendConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	sendConn.SetDeadline(time.Now().Add(timeout))

	// Open receiving conn.
	recvConn, err := s.dial()
	if err != nil {
		return err
	}
	defer recvConn.Close()
	if err = recvConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	recvConn.SetDeadline(time.Now().Add(timeout))

	if err = sendConn.Write(ethProto, eth.TransactionsMsg, txs); err != nil {
		return fmt.Errorf("failed to write message to connection: %w", err)
	}

	// Make map of invalid txs.
	invalids := make(map[common.Hash]struct{})
	for _, tx := range txs {
		invalids[tx.Hash()] = struct{}{}
	}

	// Get responses.
	recvConn.SetReadDeadline(time.Now().Add(timeout))
	for {
		msg, err := recvConn.ReadEth()
		if errors.Is(err, os.ErrDeadlineExceeded) {
			// Successful if no invalid txs are propagated before timeout.
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read from connection: %w", err)
		}

		switch msg := msg.(type) {
		case *eth.TransactionsPacket:
			for _, tx := range txs {
				if _, ok := invalids[tx.Hash()]; ok {
					return fmt.Errorf("received bad tx: %s", tx.Hash())
				}
			}
		case *eth.NewPooledTransactionHashesPacket:
			for _, hash := range msg.Hashes {
				if _, ok := invalids[hash]; ok {
					return fmt.Errorf("received bad tx: %s", hash)
				}
			}
		case *eth.GetBlockHeadersPacket:
			headers, err := s.chain.GetHeaders(msg)
			if err != nil {
				t.Logf("invalid GetBlockHeaders request: %v", err)
			}
			recvConn.Write(ethProto, eth.BlockHeadersMsg, &eth.BlockHeadersPacket{
				RequestId:           msg.RequestId,
				BlockHeadersRequest: headers,
			})
		default:
			return fmt.Errorf("unexpected eth message: %v", pretty.Sdump(msg))
		}
	}
}
