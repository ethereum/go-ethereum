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

package core

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

// blackHoleContract is a trivial Ethereum contract that can just store some data
// points in its state trie. It's goal is to have a means to test functionality
// depending on evolving state roots.
//
// contract BlackHole {
//     mapping (int256 => uint) public data;
//
//     function set(int256 key, uint value) {
//         data[key] = value;
//     }
// }
var blackHoleContract = common.Hex2Bytes("6060604052605f8060106000396000f3606060405260e060020a60003504639398e0cd81146024578063a22c554014603b575b005b605560043560006020819052908152604090205481565b600435600090815260208190526040902060243590556022565b6060908152602090f3")
var blackHoleSetter = common.Hex2Bytes("a22c5540")

// Tests that state trie indexes are properly constructed for an entire chain of
// imported blocks, cross referencing between various state roots, as well as
// making sure that state updates do not lose reference entries.
func TestChainIndex(t *testing.T) {
	// Configure the test chain and ensure we have enough funds to play with
	var (
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		key3, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		addr3   = crypto.PubkeyToAddress(key3.PublicKey)
		db, _   = ethdb.NewMemDatabase()
	)
	genesis := WriteGenesisBlockForTesting(db, GenesisAccount{addr1, big.NewInt(10000000000)})

	// Generate a chain will all kinds of events happening in it
	var contract common.Address

	chain, _ := GenerateChain(genesis, db, 8, func(i int, gen *BlockGen) {
		switch i {
		case 0:
			// In block 1, addr1 sends addr2 some ether.
			tx, _ := types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(10000), params.TxGas, nil, nil).SignECDSA(key1)
			gen.AddTx(tx)
		case 1:
			// In block 2, addr1 sends some more ether to addr2.
			// addr2 passes it on to addr3.
			tx1, _ := types.NewTransaction(gen.TxNonce(addr1), addr2, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key1)
			tx2, _ := types.NewTransaction(gen.TxNonce(addr2), addr3, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(key2)
			gen.AddTx(tx1)
			gen.AddTx(tx2)
		case 2:
			// Block 3 is empty but was mined by addr3.
			gen.SetCoinbase(addr3)
			gen.SetExtra([]byte("yeehaw"))
		case 3:
			// Block 4 includes blocks 2 and 3 as uncle headers (with modified extra data).
			b2 := gen.PrevBlock(1).Header()
			b2.Extra = []byte("foo")
			gen.AddUncle(b2)
			b3 := gen.PrevBlock(2).Header()
			b3.Extra = []byte("foo")
			gen.AddUncle(b3)
		case 4:
			// In block 5, we create a simple storage contract to add data entries to
			tx, _ := types.NewContractCreation(gen.TxNonce(addr1), big.NewInt(31415), params.GenesisGasLimit, big.NewInt(1), blackHoleContract).SignECDSA(key1)
			contract = crypto.CreateAddress(addr1, tx.Nonce())
			gen.AddTx(tx)
		case 5:
			// In block 6, we store a single entry into the storage to check single update indexing
			key := common.Hex2BytesFixed("01", 32)
			val := common.Hex2BytesFixed("02", 32)

			tx, _ := types.NewTransaction(gen.TxNonce(addr1), contract, nil, params.GenesisGasLimit, nil, append(append(blackHoleSetter, key...), val...)).SignECDSA(key1)
			gen.AddTx(tx)
		case 6:
			// In block 7, we store a lot of entries into the storage to check multi update indexing
			for i := int64(0); i < 50; i++ {
				key := common.Hex2BytesFixed(common.Bytes2Hex(big.NewInt(i).Bytes()), 32)
				val := common.Hex2BytesFixed(common.Bytes2Hex(big.NewInt(i+2).Bytes()), 32)

				tx, _ := types.NewTransaction(gen.TxNonce(addr1), contract, nil, big.NewInt(45000), nil, append(append(blackHoleSetter, key...), val...)).SignECDSA(key1)
				gen.AddTx(tx)
			}
		}
	})
	// Import the chain. This runs all block validation rules.
	evmux := &event.TypeMux{}

	vm.Debug = true

	blockchain, _ := NewBlockChain(db, FakePow{}, evmux)
	if i, err := blockchain.InsertChain(chain); err != nil {
		t.Fatalf("block %d: insert failed: %v\n", i, err)
		return
	}
	// Iterate over all the blocks and check the state trie indexes
	indexes := make(map[string]struct{})

	for block := uint64(0); block <= blockchain.CurrentBlock().NumberU64(); block++ {
		// Gather all the indexes that should be present in the database
		root := blockchain.GetBlockByNumber(block).Root()

		stateDb, err := state.New(root, db)
		if err != nil {
			t.Fatalf("failed to create state trie at %x: %v", root, err)
		}
		fmt.Println(blockchain.GetBlockByNumber(block).Transactions())
		for it := state.NewNodeIterator(stateDb); it.Next(); {
			if (it.Hash != common.Hash{}) && (it.Parent != common.Hash{}) {
				fmt.Printf("%d: %x -> %x\n", block, it.Parent, it.Hash)
				indexes[string(trie.ParentReferenceIndexKey(it.Parent.Bytes(), it.Hash.Bytes()))] = struct{}{}
			}
		}
	}
	// Cross check the indexes and the database itself
	fmt.Println(len(indexes))
	for index, _ := range indexes {
		if _, err := db.Get([]byte(index)); err != nil {
			t.Errorf("failed to retrieve reported index %x: %v", index, err)
		}
	}
	for _, key := range db.Keys() {
		if bytes.HasPrefix(key, trie.ParentReferenceIndexPrefix) {
			if _, ok := indexes[string(key)]; !ok {
				t.Errorf("index entry not reported %x", key)
			}
		}
	}
}
