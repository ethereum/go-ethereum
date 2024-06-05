// Copyright 2021 The go-ethereum Authors
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

package tracetest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"

	// Force-load live packages, to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/live"
)

type supplyInfoIssuance struct {
	GenesisAlloc *hexutil.Big `json:"genesisAlloc,omitempty"`
	Reward       *hexutil.Big `json:"reward,omitempty"`
	Withdrawals  *hexutil.Big `json:"withdrawals,omitempty"`
}

type supplyInfoBurn struct {
	EIP1559 *hexutil.Big `json:"1559,omitempty"`
	Blob    *hexutil.Big `json:"blob,omitempty"`
	Misc    *hexutil.Big `json:"misc,omitempty"`
}

type supplyInfo struct {
	Issuance *supplyInfoIssuance `json:"issuance,omitempty"`
	Burn     *supplyInfoBurn     `json:"burn,omitempty"`

	// Block info
	Number     uint64      `json:"blockNumber"`
	Hash       common.Hash `json:"hash"`
	ParentHash common.Hash `json:"parentHash"`
}

func emptyBlockGenerationFunc(b *core.BlockGen) {}

func TestSupplyOmittedFields(t *testing.T) {
	var (
		config = *params.MergedTestChainConfig
		gspec  = &core.Genesis{
			Config: &config,
		}
	)

	gspec.Config.TerminalTotalDifficulty = big.NewInt(0)

	out, _, err := testSupplyTracer(t, gspec, func(b *core.BlockGen) {
		b.SetPoS()
	})
	if err != nil {
		t.Fatalf("failed to test supply tracer: %v", err)
	}

	expected := supplyInfo{
		Number:     0,
		Hash:       common.HexToHash("0x52f276d96f0afaaf2c3cb358868bdc2779c4b0cb8de3e7e5302e247c0b66a703"),
		ParentHash: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
	}
	actual := out[expected.Number]

	compareAsJSON(t, expected, actual)
}

func TestSupplyGenesisAlloc(t *testing.T) {
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		eth1    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))

		config = *params.AllEthashProtocolChanges

		gspec = &core.Genesis{
			Config: &config,
			Alloc: types.GenesisAlloc{
				addr1: {Balance: eth1},
				addr2: {Balance: eth1},
			},
		}
	)

	expected := supplyInfo{
		Issuance: &supplyInfoIssuance{
			GenesisAlloc: (*hexutil.Big)(new(big.Int).Mul(common.Big2, big.NewInt(params.Ether))),
		},
		Number:     0,
		Hash:       common.HexToHash("0xbcc9466e9fc6a8b56f4b29ca353a421ff8b51a0c1a58ca4743b427605b08f2ca"),
		ParentHash: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
	}

	out, _, err := testSupplyTracer(t, gspec, emptyBlockGenerationFunc)
	if err != nil {
		t.Fatalf("failed to test supply tracer: %v", err)
	}

	actual := out[expected.Number]

	compareAsJSON(t, expected, actual)
}

func TestSupplyRewards(t *testing.T) {
	var (
		config = *params.AllEthashProtocolChanges

		gspec = &core.Genesis{
			Config: &config,
		}
	)

	expected := supplyInfo{
		Issuance: &supplyInfoIssuance{
			Reward: (*hexutil.Big)(new(big.Int).Mul(common.Big2, big.NewInt(params.Ether))),
		},
		Number:     1,
		Hash:       common.HexToHash("0xcbb08370505be503dafedc4e96d139ea27aba3cbc580148568b8a307b3f51052"),
		ParentHash: common.HexToHash("0xadeda0a83e337b6c073e3f0e9a17531a04009b397a9588c093b628f21b8bc5a3"),
	}

	out, _, err := testSupplyTracer(t, gspec, emptyBlockGenerationFunc)
	if err != nil {
		t.Fatalf("failed to test supply tracer: %v", err)
	}

	actual := out[expected.Number]

	compareAsJSON(t, expected, actual)
}

func TestSupplyEip1559Burn(t *testing.T) {
	var (
		config = *params.AllEthashProtocolChanges

		aa = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		// A sender who makes transactions, has some eth1
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		gwei5   = new(big.Int).Mul(big.NewInt(5), big.NewInt(params.GWei))
		eth1    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))

		gspec = &core.Genesis{
			Config:  &config,
			BaseFee: big.NewInt(params.InitialBaseFee),
			Alloc: types.GenesisAlloc{
				addr1: {Balance: eth1},
			},
		}
	)

	signer := types.LatestSigner(gspec.Config)

	eip1559BlockGenerationFunc := func(b *core.BlockGen) {
		txdata := &types.DynamicFeeTx{
			ChainID:   gspec.Config.ChainID,
			Nonce:     0,
			To:        &aa,
			Gas:       21000,
			GasFeeCap: gwei5,
			GasTipCap: big.NewInt(2),
		}
		tx := types.NewTx(txdata)
		tx, _ = types.SignTx(tx, signer, key1)

		b.AddTx(tx)
	}

	out, chain, err := testSupplyTracer(t, gspec, eip1559BlockGenerationFunc)
	if err != nil {
		t.Fatalf("failed to test supply tracer: %v", err)
	}
	var (
		head     = chain.CurrentBlock()
		reward   = new(big.Int).Mul(common.Big2, big.NewInt(params.Ether))
		burn     = new(big.Int).Mul(big.NewInt(21000), head.BaseFee)
		expected = supplyInfo{
			Issuance: &supplyInfoIssuance{
				Reward: (*hexutil.Big)(reward),
			},
			Burn: &supplyInfoBurn{
				EIP1559: (*hexutil.Big)(burn),
			},
			Number:     1,
			Hash:       head.Hash(),
			ParentHash: head.ParentHash,
		}
	)

	actual := out[expected.Number]
	compareAsJSON(t, expected, actual)
}

func TestSupplyWithdrawals(t *testing.T) {
	var (
		config = *params.MergedTestChainConfig
		gspec  = &core.Genesis{
			Config: &config,
		}
	)

	withdrawalsBlockGenerationFunc := func(b *core.BlockGen) {
		b.SetPoS()

		b.AddWithdrawal(&types.Withdrawal{
			Validator: 42,
			Address:   common.Address{0xee},
			Amount:    1337,
		})
	}

	out, chain, err := testSupplyTracer(t, gspec, withdrawalsBlockGenerationFunc)
	if err != nil {
		t.Fatalf("failed to test supply tracer: %v", err)
	}

	var (
		head     = chain.CurrentBlock()
		expected = supplyInfo{
			Issuance: &supplyInfoIssuance{
				Withdrawals: (*hexutil.Big)(big.NewInt(1337000000000)),
			},
			Number:     1,
			Hash:       head.Hash(),
			ParentHash: head.ParentHash,
		}
		actual = out[expected.Number]
	)

	compareAsJSON(t, expected, actual)
}

// Tests fund retrieval after contract's selfdestruct.
// Contract A calls contract B which selfdestructs, but B receives eth1
// after the selfdestruct opcode executes from Contract A.
// Because Contract B is removed only at the end of the transaction
// the ether sent in between is burnt before Cancun hard fork.
func TestSupplySelfdestruct(t *testing.T) {
	var (
		config = *params.TestChainConfig

		aa      = common.HexToAddress("0x1111111111111111111111111111111111111111")
		bb      = common.HexToAddress("0x2222222222222222222222222222222222222222")
		dad     = common.HexToAddress("0x0000000000000000000000000000000000000dad")
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		gwei5   = new(big.Int).Mul(big.NewInt(5), big.NewInt(params.GWei))
		eth1    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))

		gspec = &core.Genesis{
			Config:  &config,
			BaseFee: big.NewInt(params.InitialBaseFee),
			Alloc: types.GenesisAlloc{
				addr1: {Balance: eth1},
				aa: {
					Code: common.FromHex("0x61face60f01b6000527322222222222222222222222222222222222222226000806002600080855af160008103603457600080fd5b60008060008034865af1905060008103604c57600080fd5b5050"),
					// Nonce:   0,
					Balance: big.NewInt(0),
				},
				bb: {
					Code:    common.FromHex("0x6000357fface000000000000000000000000000000000000000000000000000000000000808203602f57610dad80ff5b5050"),
					Nonce:   0,
					Balance: eth1,
				},
			},
		}
	)

	gspec.Config.TerminalTotalDifficulty = big.NewInt(0)

	signer := types.LatestSigner(gspec.Config)

	testBlockGenerationFunc := func(b *core.BlockGen) {
		b.SetPoS()

		txdata := &types.LegacyTx{
			Nonce:    0,
			To:       &aa,
			Value:    gwei5,
			Gas:      150000,
			GasPrice: gwei5,
			Data:     []byte{},
		}

		tx := types.NewTx(txdata)
		tx, _ = types.SignTx(tx, signer, key1)

		b.AddTx(tx)
	}

	// 1. Test pre Cancun
	preCancunOutput, preCancunChain, err := testSupplyTracer(t, gspec, testBlockGenerationFunc)
	if err != nil {
		t.Fatalf("Pre-cancun failed to test supply tracer: %v", err)
	}

	// Check balance at state:
	// 1. 0x0000...000dad has 1 ether
	// 2. A has 0 ether
	// 3. B has 0 ether
	statedb, _ := preCancunChain.State()
	if got, exp := statedb.GetBalance(dad), eth1; got.CmpBig(exp) != 0 {
		t.Fatalf("Pre-cancun address \"%v\" balance, got %v exp %v\n", dad, got, exp)
	}
	if got, exp := statedb.GetBalance(aa), big.NewInt(0); got.CmpBig(exp) != 0 {
		t.Fatalf("Pre-cancun address \"%v\" balance, got %v exp %v\n", aa, got, exp)
	}
	if got, exp := statedb.GetBalance(bb), big.NewInt(0); got.CmpBig(exp) != 0 {
		t.Fatalf("Pre-cancun address \"%v\" balance, got %v exp %v\n", bb, got, exp)
	}

	head := preCancunChain.CurrentBlock()
	// Check live trace output
	expected := supplyInfo{
		Burn: &supplyInfoBurn{
			EIP1559: (*hexutil.Big)(big.NewInt(55289500000000)),
			Misc:    (*hexutil.Big)(big.NewInt(5000000000)),
		},
		Number:     1,
		Hash:       head.Hash(),
		ParentHash: head.ParentHash,
	}

	actual := preCancunOutput[expected.Number]

	compareAsJSON(t, expected, actual)

	// 2. Test post Cancun
	cancunTime := uint64(0)
	gspec.Config.ShanghaiTime = &cancunTime
	gspec.Config.CancunTime = &cancunTime

	postCancunOutput, postCancunChain, err := testSupplyTracer(t, gspec, testBlockGenerationFunc)
	if err != nil {
		t.Fatalf("Post-cancun failed to test supply tracer: %v", err)
	}

	// Check balance at state:
	// 1. 0x0000...000dad has 1 ether
	// 3. A has 0 ether
	// 3. B has 5 gwei
	statedb, _ = postCancunChain.State()
	if got, exp := statedb.GetBalance(dad), eth1; got.CmpBig(exp) != 0 {
		t.Fatalf("Post-shanghai address \"%v\" balance, got %v exp %v\n", dad, got, exp)
	}
	if got, exp := statedb.GetBalance(aa), big.NewInt(0); got.CmpBig(exp) != 0 {
		t.Fatalf("Post-shanghai address \"%v\" balance, got %v exp %v\n", aa, got, exp)
	}
	if got, exp := statedb.GetBalance(bb), gwei5; got.CmpBig(exp) != 0 {
		t.Fatalf("Post-shanghai address \"%v\" balance, got %v exp %v\n", bb, got, exp)
	}

	// Check live trace output
	head = postCancunChain.CurrentBlock()
	expected = supplyInfo{
		Burn: &supplyInfoBurn{
			EIP1559: (*hexutil.Big)(big.NewInt(55289500000000)),
		},
		Number:     1,
		Hash:       head.Hash(),
		ParentHash: head.ParentHash,
	}

	actual = postCancunOutput[expected.Number]

	compareAsJSON(t, expected, actual)
}

// Tests selfdestructing contract to send its balance to itself (burn).
// It tests both cases of selfdestructing succeeding and being reverted.
//   - Contract A calls B and D.
//   - Contract B selfdestructs and sends the eth1 to itself (Burn amount to be counted).
//   - Contract C selfdestructs and sends the eth1 to itself.
//   - Contract D calls C and reverts (Burn amount of C
//     has to be reverted as well).
func TestSupplySelfdestructItselfAndRevert(t *testing.T) {
	var (
		config = *params.TestChainConfig

		aa      = common.HexToAddress("0x1111111111111111111111111111111111111111")
		bb      = common.HexToAddress("0x2222222222222222222222222222222222222222")
		cc      = common.HexToAddress("0x3333333333333333333333333333333333333333")
		dd      = common.HexToAddress("0x4444444444444444444444444444444444444444")
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		gwei5   = new(big.Int).Mul(big.NewInt(5), big.NewInt(params.GWei))
		eth1    = new(big.Int).Mul(common.Big1, big.NewInt(params.Ether))
		eth2    = new(big.Int).Mul(common.Big2, big.NewInt(params.Ether))
		eth5    = new(big.Int).Mul(big.NewInt(5), big.NewInt(params.Ether))

		gspec = &core.Genesis{
			Config: &config,
			// BaseFee: big.NewInt(params.InitialBaseFee),
			Alloc: types.GenesisAlloc{
				addr1: {Balance: eth1},
				aa: {
					// Contract code in YUL:
					//
					// object "ContractA" {
					// 	code {
					// 			let B := 0x2222222222222222222222222222222222222222
					// 			let D := 0x4444444444444444444444444444444444444444

					// 			// Call to Contract B
					// 			let resB:= call(gas(), B, 0, 0x0, 0x0, 0, 0)

					// 			// Call to Contract D
					// 			let resD := call(gas(), D, 0, 0x0, 0x0, 0, 0)
					// 	}
					// }
					Code:    common.FromHex("0x73222222222222222222222222222222222222222273444444444444444444444444444444444444444460006000600060006000865af160006000600060006000865af150505050"),
					Balance: common.Big0,
				},
				bb: {
					// Contract code in YUL:
					//
					// object "ContractB" {
					// 	code {
					// 			let self := address()
					// 			selfdestruct(self)
					// 	}
					// }
					Code:    common.FromHex("0x3080ff50"),
					Balance: eth5,
				},
				cc: {
					Code:    common.FromHex("0x3080ff50"),
					Balance: eth1,
				},
				dd: {
					// Contract code in YUL:
					//
					// object "ContractD" {
					// 	code {
					// 			let C := 0x3333333333333333333333333333333333333333

					// 			// Call to Contract C
					// 			let resC := call(gas(), C, 0, 0x0, 0x0, 0, 0)

					// 			// Revert
					// 			revert(0, 0)
					// 	}
					// }
					Code:    common.FromHex("0x73333333333333333333333333333333333333333360006000600060006000855af160006000fd5050"),
					Balance: eth2,
				},
			},
		}
	)

	gspec.Config.TerminalTotalDifficulty = big.NewInt(0)

	signer := types.LatestSigner(gspec.Config)

	testBlockGenerationFunc := func(b *core.BlockGen) {
		b.SetPoS()

		txdata := &types.LegacyTx{
			Nonce:    0,
			To:       &aa,
			Value:    common.Big0,
			Gas:      150000,
			GasPrice: gwei5,
			Data:     []byte{},
		}

		tx := types.NewTx(txdata)
		tx, _ = types.SignTx(tx, signer, key1)

		b.AddTx(tx)
	}

	output, chain, err := testSupplyTracer(t, gspec, testBlockGenerationFunc)
	if err != nil {
		t.Fatalf("failed to test supply tracer: %v", err)
	}

	// Check balance at state:
	// 1. A has 0 ether
	// 2. B has 0 ether, burned
	// 3. C has 2 ether, selfdestructed but parent D reverted
	// 4. D has 1 ether, reverted
	statedb, _ := chain.State()
	if got, exp := statedb.GetBalance(aa), common.Big0; got.CmpBig(exp) != 0 {
		t.Fatalf("address \"%v\" balance, got %v exp %v\n", aa, got, exp)
	}
	if got, exp := statedb.GetBalance(bb), common.Big0; got.CmpBig(exp) != 0 {
		t.Fatalf("address \"%v\" balance, got %v exp %v\n", bb, got, exp)
	}
	if got, exp := statedb.GetBalance(cc), eth1; got.CmpBig(exp) != 0 {
		t.Fatalf("address \"%v\" balance, got %v exp %v\n", bb, got, exp)
	}
	if got, exp := statedb.GetBalance(dd), eth2; got.CmpBig(exp) != 0 {
		t.Fatalf("address \"%v\" balance, got %v exp %v\n", bb, got, exp)
	}

	// Check live trace output
	block := chain.GetBlockByNumber(1)

	expected := supplyInfo{
		Burn: &supplyInfoBurn{
			EIP1559: (*hexutil.Big)(new(big.Int).Mul(block.BaseFee(), big.NewInt(int64(block.GasUsed())))),
			Misc:    (*hexutil.Big)(eth5), // 5ETH burned from contract B
		},
		Number:     1,
		Hash:       block.Hash(),
		ParentHash: block.ParentHash(),
	}

	actual := output[expected.Number]

	compareAsJSON(t, expected, actual)
}

func testSupplyTracer(t *testing.T, genesis *core.Genesis, gen func(*core.BlockGen)) ([]supplyInfo, *core.BlockChain, error) {
	var (
		engine = beacon.New(ethash.NewFaker())
	)

	traceOutputPath := filepath.ToSlash(t.TempDir())
	traceOutputFilename := path.Join(traceOutputPath, "supply.jsonl")

	// Load supply tracer
	tracer, err := tracers.LiveDirectory.New("supply", json.RawMessage(fmt.Sprintf(`{"path":"%s"}`, traceOutputPath)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create call tracer: %v", err)
	}

	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), core.DefaultCacheConfigWithScheme(rawdb.PathScheme), genesis, nil, engine, vm.Config{Tracer: tracer}, nil, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tester chain: %v", err)
	}
	defer chain.Stop()

	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 1, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{1})
		gen(b)
	})

	if n, err := chain.InsertChain(blocks); err != nil {
		return nil, chain, fmt.Errorf("block %d: failed to insert into chain: %v", n, err)
	}

	// Check and compare the results
	file, err := os.OpenFile(traceOutputFilename, os.O_RDONLY, 0666)
	if err != nil {
		return nil, chain, fmt.Errorf("failed to open output file: %v", err)
	}
	defer file.Close()

	var output []supplyInfo
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		blockBytes := scanner.Bytes()

		var info supplyInfo
		if err := json.Unmarshal(blockBytes, &info); err != nil {
			return nil, chain, fmt.Errorf("failed to unmarshal result: %v", err)
		}

		output = append(output, info)
	}

	return output, chain, nil
}

func compareAsJSON(t *testing.T, expected interface{}, actual interface{}) {
	want, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("failed to marshal expected value to JSON: %v", err)
	}

	have, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("failed to marshal actual value to JSON: %v", err)
	}

	if !bytes.Equal(want, have) {
		t.Fatalf("incorrect supply info: expected %s, got %s", string(want), string(have))
	}
}
