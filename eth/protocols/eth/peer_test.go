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

// This file contains some shares testing functionality, common to  multiple
// different files and modules being tested.

package eth

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
)

// testPeer is a simulated peer to allow testing direct network calls.
type testPeer struct {
	*Peer

	net p2p.MsgReadWriter // Network layer reader/writer to simulate remote messaging
	app *p2p.MsgPipeRW    // Application layer reader/writer to simulate the local side
}

// newTestPeer creates a new peer registered at the given data backend.
func newTestPeer(name string, version uint, backend Backend) (*testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Start the peer on a new thread
	var id enode.ID
	rand.Read(id[:])

	peer := NewPeer(version, p2p.NewPeer(id, name, nil), net, backend.TxPool(), backend.BlobPool(), nil)
	errc := make(chan error, 1)
	go func() {
		defer app.Close()

		errc <- backend.RunPeer(peer, func(peer *Peer) error {
			return Handle(backend, peer)
		})
	}()
	return &testPeer{app: app, net: net, Peer: peer}, errc
}

// close terminates the local side of the peer, notifying the remote protocol
// manager of termination.
func (p *testPeer) close() {
	p.Peer.Close()
	p.app.Close()
}

// TestBufferReceiptsNoProgress checks that an incomplete receipt response
// delivering no receipt is rejected, while an empty list at the end of a
// multi-block response is still accepted.
func TestBufferReceiptsNoProgress(t *testing.T) {
	p := &Peer{receiptBuffer: make(map[uint64]*receiptRequest)}

	newRequest := func(id uint64, blocks int) {
		req := new(receiptRequest)
		for i := 0; i < blocks; i++ {
			req.request = append(req.request, common.Hash{byte(i + 1)})
			req.gasUsed = append(req.gasUsed, 1_000_000)
			req.numbers = append(req.numbers, uint64(i+1))
			req.timestamps = append(req.timestamps, 0)
		}
		p.receiptBuffer[id] = req
	}

	// A single empty list with the incomplete flag makes no progress.
	newRequest(1, 1)
	if err := p.bufferReceipts(1, []*ReceiptList{NewReceiptList(nil)}, true); err == nil {
		t.Fatal("expected error for incomplete response without receipts")
	}
	if _, ok := p.receiptBuffer[1]; ok {
		t.Fatal("buffer entry not removed after invalid response")
	}

	// An empty list preceded by other lists advances the request and is valid.
	newRequest(2, 2)
	lists := []*ReceiptList{
		NewReceiptList([]*types.Receipt{{Status: 1, CumulativeGasUsed: 21000}}),
		NewReceiptList(nil),
	}
	if err := p.bufferReceipts(2, lists, true); err != nil {
		t.Fatalf("unexpected error for multi-block incomplete response: %v", err)
	}
	if _, ok := p.receiptBuffer[2]; !ok {
		t.Fatal("buffer entry missing after valid incomplete response")
	}
}

func TestValidateBlockReceiptsTransactionCount(t *testing.T) {
	pragueTime, amsterdamTime := uint64(200), uint64(300)
	config := &params.ChainConfig{
		LondonBlock:   big.NewInt(10),
		PragueTime:    &pragueTime,
		AmsterdamTime: &amsterdamTime,
	}
	tests := []struct {
		name      string
		config    *params.ChainConfig
		number    uint64
		timestamp uint64
		minGas    uint64
	}{
		{"before London", config, 9, 100, 10500},
		{"at London", config, 10, 100, 16800},
		{"before Prague", config, 20, 199, 16800},
		{"at Prague", config, 20, 200, 21000},
		{"before Amsterdam", config, 20, 299, 21000},
		{"at Amsterdam", config, 20, 300, 12000},
	}
	receipts := NewReceiptList([]*types.Receipt{{Status: types.ReceiptStatusSuccessful}})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Peer{chainConfig: tt.config}
			for _, previousTxs := range []int{0, 1} {
				gasUsed := uint64(previousTxs+1) * tt.minGas
				if _, err := p.validateBlockReceipts(receipts, previousTxs, 0, gasUsed, tt.number, tt.timestamp); err != nil {
					t.Fatalf("rejected transaction count at gas bound with %d buffered receipts: %v", previousTxs, err)
				}
				if _, err := p.validateBlockReceipts(receipts, previousTxs, 0, gasUsed-1, tt.number, tt.timestamp); err == nil {
					t.Fatalf("accepted transaction count above gas bound with %d buffered receipts", previousTxs)
				}
			}
		})
	}
}

func TestBufferReceiptsRefundedTransaction(t *testing.T) {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	sender := crypto.PubkeyToAddress(key.PublicKey)
	contract := common.Address{0xaa}
	config := &params.ChainConfig{
		ChainID:        big.NewInt(3),
		HomesteadBlock: big.NewInt(0),
		EIP150Block:    big.NewInt(0),
		EIP155Block:    big.NewInt(0),
		EIP158Block:    big.NewInt(0),
		ByzantiumBlock: big.NewInt(0),
	}
	genesis := &core.Genesis{Config: config, GasLimit: 5_000_000, Alloc: types.GenesisAlloc{
		sender:   {Balance: big.NewInt(1_000_000_000_000_000)},
		contract: {Code: []byte{0x58, 0xff}, Balance: common.Big1}, // PC; SELFDESTRUCT
		{}:       {Balance: common.Big1},
	}}
	db, blocks, receipts := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 1, func(_ int, b *core.BlockGen) {
		b.AddTx(types.MustSignNewTx(key, types.HomesteadSigner{}, &types.LegacyTx{To: &contract, Gas: 100_000, GasPrice: common.Big1}))
	})
	defer db.Close()

	if receipts[0][0].Status != types.ReceiptStatusSuccessful || blocks[0].GasUsed() >= params.TxGas {
		t.Fatalf("expected successful transaction using less than intrinsic gas, got status %d and gas %d", receipts[0][0].Status, blocks[0].GasUsed())
	}
	p := &Peer{
		chainConfig: config,
		receiptBuffer: map[uint64]*receiptRequest{1: {
			request:    []common.Hash{blocks[0].Hash()},
			gasUsed:    []uint64{blocks[0].GasUsed()},
			numbers:    []uint64{blocks[0].NumberU64()},
			timestamps: []uint64{blocks[0].Time()},
		}}}
	if err := p.bufferReceipts(1, []*ReceiptList{NewReceiptList(receipts[0])}, false); err != nil {
		t.Fatalf("valid refunded transaction rejected: %v", err)
	}
}

func TestPeerSet(t *testing.T) {
	size := 5
	s := newKnownCache(size)

	// add 10 items
	for i := 0; i < size*2; i++ {
		s.Add(common.Hash{byte(i)})
	}

	if s.Cardinality() != size {
		t.Fatalf("wrong size, expected %d but found %d", size, s.Cardinality())
	}

	vals := []common.Hash{}
	for i := 10; i < 20; i++ {
		vals = append(vals, common.Hash{byte(i)})
	}

	// add item in batch
	s.Add(vals...)
	if s.Cardinality() < size {
		t.Fatalf("bad size")
	}
}
