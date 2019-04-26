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

package testhelpers

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// MakeChain creates a chain of n blocks starting at and including parent.
// the returned hash chain is ordered head->parent. In addition, every 3rd block
// contains a transaction and every 5th an uncle to allow testing correct block
// reassembly.
func MakeChain(n int, parent *types.Block) ([]common.Hash, map[common.Hash]*types.Block, *core.BlockChain) {
	blocks, _ := core.GenerateChain(params.TestChainConfig, parent, ethash.NewFaker(), Testdb, n, testChainGen)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	chain, _ := core.NewBlockChain(Testdb, nil, params.TestChainConfig, ethash.NewFaker(), vm.Config{}, nil)

	hashes := make([]common.Hash, n+1)
	hashes[len(hashes)-1] = parent.Hash()
	blockm := make(map[common.Hash]*types.Block, n+1)
	blockm[parent.Hash()] = parent
	for i, b := range blocks {
		hashes[len(hashes)-i-2] = b.Hash()
		blockm[b.Hash()] = b
	}
	return hashes, blockm, chain
}

func testChainGen(i int, block *core.BlockGen) {
	signer := types.HomesteadSigner{}
	switch i {
	case 0:
		// In block 1, the test bank sends account #1 some ether.
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(TestBankAddress), Account1Addr, big.NewInt(10000), params.TxGas, nil, nil), signer, TestBankKey)
		block.AddTx(tx)
	case 1:
		// In block 2, the test bank sends some more ether to account #1.
		// account1Addr passes it on to account #2.
		// account1Addr creates a test contract.
		tx1, _ := types.SignTx(types.NewTransaction(block.TxNonce(TestBankAddress), Account1Addr, big.NewInt(1000), params.TxGas, nil, nil), signer, TestBankKey)
		nonce := block.TxNonce(Account1Addr)
		tx2, _ := types.SignTx(types.NewTransaction(nonce, Account2Addr, big.NewInt(1000), params.TxGas, nil, nil), signer, Account1Key)
		nonce++
		tx3, _ := types.SignTx(types.NewContractCreation(nonce, big.NewInt(0), 1000000, big.NewInt(0), ContractCode), signer, Account1Key)
		ContractAddr = crypto.CreateAddress(Account1Addr, nonce) //0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476592
		block.AddTx(tx1)
		block.AddTx(tx2)
		block.AddTx(tx3)
	case 2:
		// Block 3 is empty but was mined by account #2.
		block.SetCoinbase(Account2Addr)
		//get function: 60cd2685
		//put function: c16431b9
		data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003")
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(TestBankAddress), ContractAddr, big.NewInt(0), 100000, nil, data), signer, TestBankKey)
		block.AddTx(tx)
	}
}
