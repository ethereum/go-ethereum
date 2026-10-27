// Copyright 2026 The go-ethereum Authors
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

package eip7807

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// generatedChain builds six blocks with Geth's own block builder under a
// configuration with every fork active, so each block carries what an
// EIP-7807 block must: blob gas, a beacon root, a requests hash, a block
// access list and a slot number. Each block holds the content one fork era
// introduced, so the six together exercise every element type the container
// hashes. The chain is deterministic: fixed keys, fixed values, the builder's
// own timestamps. Gas limits are generous because Amsterdam's EIP-8037
// charges 183,600 state gas for each account a transfer creates.
func generatedChain(t *testing.T) (*params.ChainConfig, *core.Genesis, []*types.Block) {
	t.Helper()
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = new(uint64)
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		from    = crypto.PubkeyToAddress(key1.PublicKey)
		signer  = types.LatestSigner(&config)
		gwei    = func(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(params.GWei)) }
		addr    = func(b byte) common.Address { return common.Address{0xaa, b} }
		chainID = uint256.MustFromBig(config.ChainID)
	)
	gspec := &core.Genesis{
		Config:   &config,
		GasLimit: 30_000_000,
		Alloc: types.GenesisAlloc{
			from:                             {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))},
			params.BeaconRootsAddress:        {Nonce: 1, Code: params.BeaconRootsCode, Balance: common.Big0},
			params.HistoryStorageAddress:     {Nonce: 1, Code: params.HistoryStorageCode, Balance: common.Big0},
			params.WithdrawalQueueAddress:    {Nonce: 1, Code: params.WithdrawalQueueCode, Balance: common.Big0},
			params.ConsolidationQueueAddress: {Nonce: 1, Code: params.ConsolidationQueueCode, Balance: common.Big0},
			params.BuilderDepositAddress:     {Nonce: 1, Code: params.BuilderDepositCode, Balance: common.Big0},
			params.BuilderExitAddress:        {Nonce: 1, Code: params.BuilderExitCode, Balance: common.Big0},
		},
	}
	nonce := uint64(0)
	send := func(b *core.BlockGen, data types.TxData) {
		b.AddTx(types.MustSignNewTx(key1, signer, data))
		nonce++
	}
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 6, func(i int, b *core.BlockGen) {
		switch i {
		case 0: // Paris: a legacy and an access list transaction.
			send(b, &types.LegacyTx{Nonce: nonce, GasPrice: gwei(5), Gas: 300_000, To: ptr(addr(1)), Value: big.NewInt(1)})
			send(b, &types.AccessListTx{ChainID: config.ChainID, Nonce: nonce, GasPrice: gwei(5), Gas: 300_000, To: ptr(addr(2)), Value: big.NewInt(2),
				AccessList: types.AccessList{{Address: addr(2), StorageKeys: []common.Hash{{0x01}}}}})
		case 1: // Shanghai: fee market transactions and withdrawals.
			for j := 0; j < 2; j++ {
				send(b, &types.DynamicFeeTx{ChainID: config.ChainID, Nonce: nonce, GasTipCap: big.NewInt(2), GasFeeCap: gwei(5), Gas: 300_000, To: ptr(addr(3)), Value: big.NewInt(3)})
			}
			for j := 0; j < 3; j++ {
				b.AddWithdrawal(&types.Withdrawal{Validator: uint64(100 + j), Address: addr(byte(10 + j)), Amount: uint64(1_000_000 * (j + 1))})
			}
		case 2: // Cancun: a blob transaction, so blob gas is used.
			send(b, &types.BlobTx{ChainID: chainID, Nonce: nonce, GasTipCap: uint256.NewInt(2), GasFeeCap: uint256.MustFromBig(gwei(5)), Gas: 300_000, To: addr(4),
				BlobFeeCap: uint256.MustFromBig(gwei(1)), BlobHashes: []common.Hash{{0x01, 0x02}, {0x01, 0x03}}})
		case 3: // Prague: a set code transaction and the request system contracts.
			auth, err := types.SignSetCode(key2, types.SetCodeAuthorization{ChainID: *chainID, Address: addr(5), Nonce: 0})
			if err != nil {
				t.Fatal(err)
			}
			send(b, &types.SetCodeTx{ChainID: chainID, Nonce: nonce, GasTipCap: uint256.NewInt(2), GasFeeCap: uint256.MustFromBig(gwei(5)), Gas: 1_000_000,
				To: crypto.PubkeyToAddress(key2.PublicKey), AuthList: []types.SetCodeAuthorization{auth}})
			// A withdrawal request: 48-byte pubkey and 8-byte amount, one wei of fee.
			send(b, &types.DynamicFeeTx{ChainID: config.ChainID, Nonce: nonce, GasTipCap: big.NewInt(2), GasFeeCap: gwei(5), Gas: 2_000_000,
				To: &params.WithdrawalQueueAddress, Value: gwei(1),
				Data: common.FromHex("b917cfdc0d25b72d55cf94db328e1629b7f4fde2c30cdacf873b664416f76a0c7f7cc50c9f72a3cb84be88144cde91250000000000000d80")})
			// A consolidation request: source and target pubkeys.
			send(b, &types.DynamicFeeTx{ChainID: config.ChainID, Nonce: nonce, GasTipCap: big.NewInt(2), GasFeeCap: gwei(5), Gas: 2_000_000,
				To: &params.ConsolidationQueueAddress, Value: gwei(1),
				Data: append(common.FromHex("b917cfdc0d25b72d55cf94db328e1629b7f4fde2c30cdacf873b664416f76a0c7f7cc50c9f72a3cb84be88144cde9125"),
					common.FromHex("a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90")...)})
		case 4: // Amsterdam: many touched accounts, a wide block access list.
			for j := byte(20); j < 28; j++ {
				send(b, &types.DynamicFeeTx{ChainID: config.ChainID, Nonce: nonce, GasTipCap: big.NewInt(2), GasFeeCap: gwei(5), Gas: 300_000, To: ptr(addr(j)), Value: big.NewInt(int64(j))})
			}
		case 5: // An empty block.
		}
	})
	return &config, gspec, blocks
}

func ptr(a common.Address) *common.Address { return &a }

// generatedPins are the keccak hash and the EIP-7807 root of each generated
// block, side by side. The roots were confirmed by the fellowship's Python
// reference, which re-encodes every element from the blocks' JSON-RPC form.
var generatedPins = []struct{ hash, root string }{
	{"0xbc90b243028c69fca8c1f6c13a7817089dcc1a1d3b37406aa3513169f102b4eb", "0x7164298dec5fc663977750f7247174dfdd7a77644a1ee21e2036c238aadbfb22"},
	{"0xd50f84d68815e2317c1d64a4dc872789926c958b19caa709ecc8f399edc04b50", "0x67214d2cbb525d4de4b28a864a9f14f5e4e27bbef7bb2932648b62fb346d5fc2"},
	{"0x5e16f28f0adb0fb0ecd4e45cef70deeaee5b2d10c8c01eb780bfee8ffaefc535", "0xce227d7e1153444976c07d34566e13ad89ca13e51cab4a613587b731697fea27"},
	{"0x0fddc7c42a29f4d40d126c390c3358bd45d9753f1d876e62aed5200f89f3eec5", "0x463dc4d94317bccc6838a6e2e40cc25361692ab59f5108da80a9f3c880a73b13"},
	{"0x7321794d0d9ca65d81cbe341925823fdf814e1753b8037e3dce4480dacfb10ad", "0xec1febd29c338f38209853cde20336c37698690e21df74b6e9fdd5f365f5afd4"},
	{"0x063713ece7bbe937c5e25a0d38548d75fe6609dbb07741ea02790621c0aae0e3", "0x79f38b493c69e04c1392fef7747737e159b33090adeb61e0a5376b00c16947ba"},
}

// TestGeneratedChain imports the generated chain, re-executes each block for
// its receipts and requests (which a block does not carry and Geth does not
// store), maps it, and checks the root against the pins and the summary
// against the root. With EIP7807_REFERENCE_DUMP set to a directory, it also
// writes every block for the Python reference.
func TestGeneratedChain(t *testing.T) {
	config, gspec, blocks := generatedChain(t)
	bc, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), gspec, beacon.New(ethash.NewFaker()), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer bc.Stop()
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	dumpDir := os.Getenv("EIP7807_REFERENCE_DUMP")
	for i, block := range blocks {
		parent := bc.GetHeaderByHash(block.ParentHash())
		statedb, err := bc.StateAt(parent.Root, parent.Number, parent.Time)
		if err != nil {
			t.Fatal(err)
		}
		res, err := core.NewStateProcessor(bc).Process(context.Background(), block, statedb, nil, nil, vm.Config{}, nil)
		if err != nil {
			t.Fatalf("block %d: %v", i, err)
		}
		p, err := PayloadFromBlock(config, block, res.Receipts, res.Requests)
		if err != nil {
			t.Fatalf("block %d: %v", i, err)
		}
		root, err := p.Root()
		if err != nil {
			t.Fatalf("block %d: %v", i, err)
		}
		if summary, err := p.Summary().Root(); err != nil || summary != root {
			t.Errorf("block %d: summary root %x, %v, payload root %x", i, summary, err, root)
		}
		if root == block.Hash() {
			t.Errorf("block %d: payload root equals the keccak hash", i)
		}
		t.Logf("block %d: %d txs %d withdrawals %d requests %d BAL bytes, keccak %x, eip7807 %x",
			i, len(block.Transactions()), len(block.Withdrawals()), len(res.Requests), len(p.BlockAccessList), block.Hash(), root)
		if i < len(generatedPins) {
			if block.Hash() != common.HexToHash(generatedPins[i].hash) {
				t.Errorf("block %d: keccak hash %x, pinned %s", i, block.Hash(), generatedPins[i].hash)
			}
			if root != common.HexToHash(generatedPins[i].root) {
				t.Errorf("block %d: payload root %x, pinned %s", i, root, generatedPins[i].root)
			}
		}
		if dumpDir != "" {
			dump, err := referenceDump(config, block, res.Receipts, res.Requests, root)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dumpDir, fmt.Sprintf("block-%d.json", i)), dump, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}
