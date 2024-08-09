package core

import (
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
	"github.com/holiman/uint256"
)

func TestEIP7702(t *testing.T) {
	var (
		aa     = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		bb     = common.HexToAddress("0x000000000000000000000000000000000000bbbb")
		engine = beacon.NewFaker()

		// A sender who makes transactions, has some funds
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		funds   = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
		config  = *params.AllEthashProtocolChanges
		gspec   = &Genesis{
			Config: &config,
			Alloc: types.GenesisAlloc{
				addr1: {Balance: funds},
				addr2: {Balance: funds},
				// The address 0xAAAA sstores 1 into slot 2.
				aa: {
					Code: []byte{
						byte(vm.PC),          // [0]
						byte(vm.DUP1),        // [0,0]
						byte(vm.DUP1),        // [0,0,0]
						byte(vm.DUP1),        // [0,0,0,0]
						byte(vm.PUSH1), 0x01, // [0,0,0,0,1] (value)
						byte(vm.PUSH20), addr2[0], addr2[1], addr2[2], addr2[3], addr2[4], addr2[5], addr2[6], addr2[7], addr2[8], addr2[9], addr2[10], addr2[11], addr2[12], addr2[13], addr2[14], addr2[15], addr2[16], addr2[17], addr2[18], addr2[19],
						byte(vm.GAS),
						byte(vm.CALL),
						byte(vm.STOP),
					},
					Nonce:   0,
					Balance: big.NewInt(0),
				},
				// The address 0xBBBB sstores 42 into slot 42.
				bb: {
					Code: []byte{
						byte(vm.PUSH1), 0x42,
						byte(vm.DUP1),
						byte(vm.SSTORE),
						byte(vm.STOP),
					},
					Nonce:   0,
					Balance: big.NewInt(0),
				},
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

	_, blocks, _ := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		b.SetCoinbase(aa)
		// One transaction to Coinbase

		auth1, _ := types.SignAuth(&types.Authorization{
			ChainID: new(big.Int).Set(gspec.Config.ChainID),
			Address: aa,
			Nonce:   nil,
		}, key1)

		auth2, _ := types.SignAuth(&types.Authorization{
			ChainID: new(big.Int).Set(gspec.Config.ChainID),
			Address: bb,
			Nonce:   []uint64{0},
		}, key2)

		txdata := &types.SetCodeTx{
			ChainID:   uint256.MustFromBig(gspec.Config.ChainID),
			Nonce:     0,
			To:        &addr1,
			Gas:       500000,
			GasFeeCap: uint256.MustFromBig(newGwei(5)),
			GasTipCap: uint256.NewInt(2),
			AuthList:  []*types.Authorization{auth1, auth2},
		}
		tx := types.NewTx(txdata)
		tx, err := types.SignTx(tx, signer, key1)
		if err != nil {
			t.Fatalf("%s", err)
		}
		b.AddTx(tx)
	})
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), nil, gspec, nil, engine, vm.Config{Tracer: logger.NewMarkdownLogger(&logger.Config{}, os.Stderr).Hooks()}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	defer chain.Stop()
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}

	var (
		state, _ = chain.State()
		fortyTwo = common.BytesToHash([]byte{0x42})
		actual   = state.GetState(addr2, fortyTwo)
	)
	if actual.Cmp(fortyTwo) != 0 {
		t.Fatalf("addr2 storage wrong: expected %d, got %d", fortyTwo, actual)
	}
	if code := state.GetCode(addr1); code != nil {
		t.Fatalf("addr1 code not cleared: got %s", common.Bytes2Hex(code))
	}
	if code := state.GetCode(addr2); code != nil {
		t.Fatalf("addr2 code not cleared: got %s", common.Bytes2Hex(code))
	}
}
