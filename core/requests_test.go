// TODO: move back into core/blockchain_test.go when ready to merge.

package core

import (
	"encoding/binary"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/params"
)

// TestRequests verifies that Prague requests are processed correctly.
func TestRequests(t *testing.T) {
	var (
		engine = beacon.NewFaker()

		// A sender who makes transactions, has some funds
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		funds  = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
		config = *params.AllEthashProtocolChanges
		gspec  = &Genesis{
			Config: &config,
			Alloc: types.GenesisAlloc{
				addr:                                {Balance: funds},
				params.WithdrawalRequestsAddress:    {Code: common.FromHex("3373fffffffffffffffffffffffffffffffffffffffe146090573615156028575f545f5260205ff35b366038141561012e5760115f54600182026001905f5b5f82111560595781019083028483029004916001019190603e565b90939004341061012e57600154600101600155600354806003026004013381556001015f3581556001016020359055600101600355005b6003546002548082038060101160a4575060105b5f5b81811460dd5780604c02838201600302600401805490600101805490600101549160601b83528260140152906034015260010160a6565b910180921460ed579060025560f8565b90505f6002555f6003555b5f548061049d141561010757505f5b60015460028282011161011c5750505f610122565b01600290035b5f555f600155604c025ff35b5f5ffd")},
				params.ConsolidationRequestsAddress: {Code: common.FromHex("3373fffffffffffffffffffffffffffffffffffffffe146098573615156028575f545f5260205ff35b36606014156101445760115f54600182026001905f5b5f82111560595781019083028483029004916001019190603e565b90939004341061014457600154600101600155600354806004026004013381556001015f35815560010160203581556001016040359055600101600355005b6003546002548082038060011160ac575060015b5f5b81811460f15780607402838201600402600401805490600101805490600101805490600101549260601b84529083601401528260340152906054015260010160ae565b9101809214610103579060025561010e565b90505f6002555f6003555b5f548061049d141561011d57505f5b6001546001828201116101325750505f610138565b01600190035b5f555f6001556074025ff35b5f5ffd")},
			},
		}
	)
	gspec.Config.BerlinBlock = common.Big0
	gspec.Config.LondonBlock = common.Big0
	gspec.Config.TerminalTotalDifficulty = common.Big0
	gspec.Config.TerminalTotalDifficultyPassed = true
	gspec.Config.ShanghaiTime = u64(0)
	gspec.Config.CancunTime = u64(0)
	gspec.Config.PragueTime = u64(0)
	signer := types.LatestSigner(gspec.Config)

	// Withdrawal requests to send.
	wxs := types.WithdrawalRequests{
		{
			Source:    addr,
			PublicKey: [48]byte{42},
			Amount:    42,
		},
		{
			Source:    addr,
			PublicKey: [48]byte{13, 37},
			Amount:    1337,
		},
	}
	cxs := types.ConsolidationRequests{
		{
			Source:          addr,
			SourcePublicKey: [48]byte{13, 37},
			TargetPublicKey: [48]byte{11, 11},
		},
		{
			Source:          addr,
			SourcePublicKey: [48]byte{42, 42},
			TargetPublicKey: [48]byte{11, 11},
		},
	}

	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 3, func(i int, b *BlockGen) {
		switch i {
		case 0:
			// Block 1: submit withdrawal requests
			for _, wx := range wxs {
				data := make([]byte, 56)
				copy(data, wx.PublicKey[:])
				binary.BigEndian.PutUint64(data[48:], wx.Amount)
				txdata := &types.DynamicFeeTx{
					ChainID:    gspec.Config.ChainID,
					Nonce:      b.TxNonce(addr),
					To:         &params.WithdrawalRequestsAddress,
					Value:      big.NewInt(1),
					Gas:        500000,
					GasFeeCap:  newGwei(5),
					GasTipCap:  big.NewInt(2),
					AccessList: nil,
					Data:       data,
				}
				tx := types.NewTx(txdata)
				tx, _ = types.SignTx(tx, signer, key)
				b.AddTx(tx)
			}
		case 1:
			// Block 2: submit consolidation requests
			for _, cx := range cxs {
				data := make([]byte, 96)
				copy(data, cx.SourcePublicKey[:])
				copy(data[48:], cx.TargetPublicKey[:])
				txdata := &types.DynamicFeeTx{
					ChainID:    gspec.Config.ChainID,
					Nonce:      b.TxNonce(addr),
					To:         &params.ConsolidationRequestsAddress,
					Value:      big.NewInt(1),
					Gas:        500000,
					GasFeeCap:  newGwei(5),
					GasTipCap:  big.NewInt(2),
					AccessList: nil,
					Data:       data,
				}
				tx := types.NewTx(txdata)
				tx, _ = types.SignTx(tx, signer, key)
				b.AddTx(tx)
			}
		}
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), nil, gspec, nil, engine, vm.Config{Tracer: logger.NewMarkdownLogger(&logger.Config{}, os.Stderr).Hooks()}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	defer chain.Stop()
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	// Verify the withdrawal requests match.
	block := chain.GetBlockByNumber(1)
	if block == nil {
		t.Fatalf("failed to retrieve block 1")
	}

	// Verify the withdrawal requests match.
	got := block.Requests()
	if len(got) != 2 {
		t.Fatalf("wrong number of withdrawal requests: wanted 2, got %d", len(got))
	}
	for i, want := range wxs {
		got, ok := got[i].Inner().(*types.WithdrawalRequest)
		if !ok {
			t.Fatalf("expected withdrawal request")
		}
		if want.Source != got.Source {
			t.Fatalf("wrong source address: want %s, got %s", want.Source, got.Source)
		}
		if want.PublicKey != got.PublicKey {
			t.Fatalf("wrong public key: want %s, got %s", common.Bytes2Hex(want.PublicKey[:]), common.Bytes2Hex(got.PublicKey[:]))
		}
		if want.Amount != got.Amount {
			t.Fatalf("wrong amount: want %d, got %d", want.Amount, got.Amount)
		}
	}

	// Verify the consolidation requests match. Even though both requests are sent
	// in block two, only one is dequeued at a time.
	for i, want := range cxs {
		block := chain.GetBlockByNumber(uint64(i + 2))
		if block == nil {
			t.Fatalf("failed to retrieve block")
		}
		requests := block.Requests()
		if len(requests) != 1 {
			t.Fatalf("wrong number of consolidation requests: wanted 1, got %d", len(got))
		}
		got, ok := requests[0].Inner().(*types.ConsolidationRequest)
		if !ok {
			t.Fatalf("expected consolidation request")
		}
		if want.Source != got.Source {
			t.Fatalf("wrong source address: want %s, got %s", want.Source, got.Source)
		}
		if want.SourcePublicKey != got.SourcePublicKey {
			t.Fatalf("wrong source public key: want %s, got %s", common.Bytes2Hex(want.SourcePublicKey[:]), common.Bytes2Hex(got.SourcePublicKey[:]))
		}
		if want.TargetPublicKey != got.TargetPublicKey {
			t.Fatalf("wrong target public key: want %s, got %s", common.Bytes2Hex(want.TargetPublicKey[:]), common.Bytes2Hex(got.TargetPublicKey[:]))
		}
	}
}
