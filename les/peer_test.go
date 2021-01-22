// Copyright 2019 The go-ethereum Authors
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

package les

import (
	"crypto/rand"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
)

type fakeChain struct{}

func (f *fakeChain) Config() *params.ChainConfig { return params.MainnetChainConfig }
func (f *fakeChain) Genesis() *types.Block {
	return core.DefaultGenesisBlock().ToBlock(rawdb.NewMemoryDatabase())
}
func (f *fakeChain) CurrentHeader() *types.Header { return &types.Header{Number: big.NewInt(10000000)} }

func TestHandshake(t *testing.T) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id enode.ID
	rand.Read(id[:])

	peer1 := newClientPeer(2, NetworkId, p2p.NewPeer(id, "name", nil), net)
	peer2 := newServerPeer(2, NetworkId, true, p2p.NewPeer(id, "name", nil), app)

	var (
		errCh1 = make(chan error, 1)
		errCh2 = make(chan error, 1)

		td      = big.NewInt(100)
		head    = common.HexToHash("deadbeef")
		headNum = uint64(10)
		genesis = common.HexToHash("cafebabe")

		chain1, chain2   = &fakeChain{}, &fakeChain{}
		forkID1          = forkid.NewID(chain1.Config(), chain1.Genesis().Hash(), chain1.CurrentHeader().Number.Uint64())
		forkID2          = forkid.NewID(chain2.Config(), chain2.Genesis().Hash(), chain2.CurrentHeader().Number.Uint64())
		filter1, filter2 = forkid.NewFilter(chain1), forkid.NewFilter(chain2)
	)

	go func() {
		errCh1 <- peer1.handshake(td, head, headNum, genesis, forkID1, filter1, func(list *keyValueList) {
			var announceType uint64 = announceTypeSigned
			*list = (*list).add("announceType", announceType)
		}, nil)
	}()
	go func() {
		errCh2 <- peer2.handshake(td, head, headNum, genesis, forkID2, filter2, nil, func(recv keyValueMap) error {
			var reqType uint64
			err := recv.get("announceType", &reqType)
			if err != nil {
				return err
			}
			if reqType != announceTypeSigned {
				return errors.New("Expected announceTypeSigned")
			}
			return nil
		})
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh1:
			if err != nil {
				t.Fatalf("handshake failed, %v", err)
			}
		case err := <-errCh2:
			if err != nil {
				t.Fatalf("handshake failed, %v", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}
