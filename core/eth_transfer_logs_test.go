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

package core

import (
	"encoding/binary"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var ethTransferTestCode = common.FromHex("6080604052600436106100345760003560e01c8063574ffc311461003957806366e41cb714610090578063f8a8fd6d1461009a575b600080fd5b34801561004557600080fd5b5061004e6100a4565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6100986100ac565b005b6100a26100f5565b005b63deadbeef81565b7f38e80b5c85ba49b7280ccc8f22548faa62ae30d5a008a1b168fba5f47f5d1ee560405160405180910390a1631234567873ffffffffffffffffffffffffffffffffffffffff16ff5b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405160405180910390a163deadbeef73ffffffffffffffffffffffffffffffffffffffff166002348161014657fe5b046040516024016040516020818303038152906040527f66e41cb7000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff83818316178352505050506040518082805190602001908083835b602083106101fd57805182526020820191506020810190506020830392506101da565b6001836020036101000a03801982511681845116808217855250505050505090500191505060006040518083038185875af1925050503d806000811461025f576040519150601f19603f3d011682016040523d82523d6000602084013e610264565b606091505b50505056fea265627a7a723158202cce817a434785d8560c200762f972d453ccd30694481be7545f9035a512826364736f6c63430005100032")

/*
pragma solidity >=0.4.22 <0.6.0;

contract TestLogs {

  address public constant target_contract = 0x00000000000000000000000000000000DeaDBeef;
  address payable constant selfdestruct_addr = 0x0000000000000000000000000000000012345678;

  event Response(bool success, bytes data);
    event TestEvent();
    event TestEvent2();

    function test() public payable {
       emit TestEvent();
        target_contract.call.value(msg.value/2)(abi.encodeWithSignature("test2()"));
    }
    function test2() public payable {
       emit TestEvent2();
       selfdestruct(selfdestruct_addr);
    }
}
*/

// TestEthTransferLogs tests EIP-7708 ETH transfer log output by simulating a
// scenario including transaction, CALL and SELFDESTRUCT value transfers, and
// also "ordinary" logs emitted. The same scenario is also tested with no value
// transferred.
func TestEthTransferLogs(t *testing.T) {
	testEthTransferLogs(t, 1_000_000_000)
	testEthTransferLogs(t, 0)
}

func testEthTransferLogs(t *testing.T, value uint64) {
	var (
		key1, _    = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1      = crypto.PubkeyToAddress(key1.PublicKey)
		addr2      = common.HexToAddress("cafebabe") // caller
		addr3      = common.HexToAddress("deadbeef") // callee
		addr4      = common.HexToAddress("12345678") // selfdestruct target
		testEvent  = crypto.Keccak256Hash([]byte("TestEvent()"))
		testEvent2 = crypto.Keccak256Hash([]byte("TestEvent2()"))
		config     = *params.MergedTestChainConfig
		signer     = types.LatestSigner(&config)
		engine     = beacon.New(ethash.NewFaker())
	)

	//TODO remove this hacky config initialization when final Amsterdam config is available
	config.AmsterdamTime = new(uint64)
	blobConfig := *config.BlobScheduleConfig
	blobConfig.Amsterdam = blobConfig.Osaka
	config.BlobScheduleConfig = &blobConfig

	gspec := &Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			addr1: {Balance: newGwei(1000000000)},
			addr2: {Code: ethTransferTestCode},
			addr3: {Code: ethTransferTestCode},
		},
	}
	_, blocks, receipts := GenerateChainWithGenesis(gspec, engine, 1, func(i int, b *BlockGen) {
		tx := types.MustSignNewTx(key1, signer, &types.DynamicFeeTx{
			ChainID:   gspec.Config.ChainID,
			Nonce:     0,
			To:        &addr2,
			Gas:       500_000,
			GasFeeCap: newGwei(5),
			GasTipCap: newGwei(5),
			Value:     big.NewInt(int64(value)),
			Data:      common.FromHex("f8a8fd6d"),
		})
		b.AddTx(tx)
	})

	blockHash := blocks[0].Hash()
	txHash := blocks[0].Transactions()[0].Hash()
	addr2hash := func(addr common.Address) (hash common.Hash) {
		copy(hash[12:], addr[:])
		return
	}
	u256 := func(amount uint64) []byte {
		data := make([]byte, 32)
		binary.BigEndian.PutUint64(data[24:], amount)
		return data
	}

	var expLogs = []*types.Log{
		{
			Address: params.SystemAddress,
			Topics:  []common.Hash{params.EthTransferLogEvent, addr2hash(addr1), addr2hash(addr2)},
			Data:    u256(value),
		},
		{
			Address: addr2,
			Topics:  []common.Hash{testEvent},
			Data:    nil,
		},
		{
			Address: params.SystemAddress,
			Topics:  []common.Hash{params.EthTransferLogEvent, addr2hash(addr2), addr2hash(addr3)},
			Data:    u256(value / 2),
		},
		{
			Address: addr3,
			Topics:  []common.Hash{testEvent2},
			Data:    nil,
		},
		{
			Address: params.SystemAddress,
			Topics:  []common.Hash{params.EthTransferLogEvent, addr2hash(addr3), addr2hash(addr4)},
			Data:    u256(value / 2),
		},
	}
	if value == 0 {
		// no ETH transfer logs expected with zero value
		expLogs = []*types.Log{expLogs[1], expLogs[3]}
	}
	for i, log := range expLogs {
		log.BlockNumber = 1
		log.BlockHash = blockHash
		log.BlockTimestamp = 10
		log.TxIndex = 0
		log.TxHash = txHash
		log.Index = uint(i)
	}

	if len(expLogs) != len(receipts[0][0].Logs) {
		t.Fatalf("Incorrect number of logs (expected: %d, got: %d)", len(expLogs), len(receipts[0][0].Logs))
	}
	for i, log := range receipts[0][0].Logs {
		if !reflect.DeepEqual(expLogs[i], log) {
			t.Fatalf("Incorrect log at index %d (expected: %v, got: %v)", i, expLogs[i], log)
		}
	}
}
