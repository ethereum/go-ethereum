// Copyright 2023 The go-ethereum Authors
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

package ethapi

import (
	"encoding/json"
	"hash"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto/keccak"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/require"
)

// testHasher is the helper tool for transaction/receipt list hashing.
// The original hasher is trie, in order to get rid of import cycle,
// use the testing hasher instead.
type testHasher struct {
	hasher hash.Hash
}

func newHasher() *testHasher {
	return &testHasher{hasher: keccak.NewLegacyKeccak256()}
}

func (h *testHasher) Reset() {
	h.hasher.Reset()
}

func (h *testHasher) Update(key, val []byte) error {
	h.hasher.Write(key)
	h.hasher.Write(val)
	return nil
}

func (h *testHasher) Hash() common.Hash {
	return common.BytesToHash(h.hasher.Sum(nil))
}

func TestRPCMarshalBlock(t *testing.T) {
	var (
		txs []*types.Transaction
		to  = common.BytesToAddress([]byte{0x11})
	)
	for i := uint64(1); i <= 4; i++ {
		var tx *types.Transaction
		if i%2 == 0 {
			tx = types.NewTx(&types.LegacyTx{
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		} else {
			tx = types.NewTx(&types.AccessListTx{
				ChainID:  big.NewInt(1337),
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		}
		txs = append(txs, tx)
	}
	block := types.NewBlock(&types.Header{Number: big.NewInt(100)}, &types.Body{Transactions: txs}, nil, newHasher())

	var testSuite = []struct {
		inclTx bool
		fullTx bool
		want   string
	}{
		// without txs
		{
			inclTx: false,
			fullTx: false,
			want: `{
				"difficulty":"0x0",
				"extraData":"0x",
				"gasLimit":"0x0",
				"gasUsed":"0x0",
				"hash":"0x2cb4e4b5b5be5a2520377e87e8d7d2cf83fc0783fa6518d67b9606d3c5317b50",
				"logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				"miner":"0x0000000000000000000000000000000000000000",
				"mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"nonce":"0x0000000000000000",
				"number":"0x64",
				"parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"penalties":"0x",
				"receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
				"sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
				"size":"0x299",
				"stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"timestamp":"0x0",
				"transactionsRoot":"0x661a9febcfa8f1890af549b874faf9fa274aede26ef489d9db0b25daa569450e",
				"uncles":[],
				"validator":"0x",
				"validators":"0x"
			}`,
		},
		// only tx hashes
		{
			inclTx: true,
			fullTx: false,
			want: `{
				"difficulty":"0x0",
				"extraData":"0x",
				"gasLimit":"0x0",
				"gasUsed":"0x0",
				"hash":"0x2cb4e4b5b5be5a2520377e87e8d7d2cf83fc0783fa6518d67b9606d3c5317b50",
				"logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				"miner":"0x0000000000000000000000000000000000000000",
				"mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"nonce":"0x0000000000000000",
				"number":"0x64",
				"parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"penalties":"0x",
				"receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
				"sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
				"size":"0x299",
				"stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"timestamp":"0x0",
				"transactions": [
					"0x7d39df979e34172322c64983a9ad48302c2b889e55bda35324afecf043a77605",
					"0x9bba4c34e57c875ff57ac8d172805a26ae912006985395dc1bdf8f44140a7bf4",
					"0x98909ea1ff040da6be56bc4231d484de1414b3c1dac372d69293a4beb9032cb5",
					"0x12e1f81207b40c3bdcc13c0ee18f5f86af6d31754d57a0ea1b0d4cfef21abef1"
				],
				"transactionsRoot":"0x661a9febcfa8f1890af549b874faf9fa274aede26ef489d9db0b25daa569450e",
				"uncles":[],
				"validator":"0x",
				"validators":"0x"
			}`,
		},

		// full tx details
		{
			inclTx: true,
			fullTx: true,
			want: `{
				"difficulty":"0x0",
				"extraData":"0x",
				"gasLimit":"0x0",
				"gasUsed":"0x0",
				"hash":"0x2cb4e4b5b5be5a2520377e87e8d7d2cf83fc0783fa6518d67b9606d3c5317b50",
				"logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				"miner":"0x0000000000000000000000000000000000000000",
				"mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"nonce":"0x0000000000000000",
				"number":"0x64",
				"parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"penalties":"0x",
				"receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
				"sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
				"size":"0x299",
				"stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000",
				"timestamp":"0x0",
				"transactions": [
					{
						"blockHash":"0x2cb4e4b5b5be5a2520377e87e8d7d2cf83fc0783fa6518d67b9606d3c5317b50",
						"blockNumber":"0x64",
						"from":"0x0000000000000000000000000000000000000000",
						"gas":"0x457",
						"gasPrice":"0x2b67",
						"hash":"0x7d39df979e34172322c64983a9ad48302c2b889e55bda35324afecf043a77605",
						"input":"0x111111",
						"nonce":"0x1",
						"to":"0x0000000000000000000000000000000000000011",
						"transactionIndex":"0x0",
						"value":"0x6f",
						"type":"0x1",
						"accessList":[],
						"chainId":"0x539",
						"v":"0x0",
						"r":"0x0",
						"s":"0x0",
						"yParity":"0x0"
					},
					{
						"blockHash":"0x2cb4e4b5b5be5a2520377e87e8d7d2cf83fc0783fa6518d67b9606d3c5317b50",
						"blockNumber":"0x64",
						"from":"0x0000000000000000000000000000000000000000",
						"gas":"0x457",
						"gasPrice":"0x2b67",
						"hash":"0x9bba4c34e57c875ff57ac8d172805a26ae912006985395dc1bdf8f44140a7bf4",
						"input":"0x111111",
						"nonce":"0x2",
						"to":"0x0000000000000000000000000000000000000011",
						"transactionIndex":"0x1",
						"value":"0x6f",
						"type":"0x0",
						"chainId":"0x7fffffffffffffee",
						"v":"0x0",
						"r":"0x0",
						"s":"0x0"
					},
					{
						"blockHash":"0x2cb4e4b5b5be5a2520377e87e8d7d2cf83fc0783fa6518d67b9606d3c5317b50",
						"blockNumber":"0x64",
						"from":"0x0000000000000000000000000000000000000000",
						"gas":"0x457",
						"gasPrice":"0x2b67",
						"hash":"0x98909ea1ff040da6be56bc4231d484de1414b3c1dac372d69293a4beb9032cb5",
						"input":"0x111111",
						"nonce":"0x3",
						"to":"0x0000000000000000000000000000000000000011",
						"transactionIndex":"0x2",
						"value":"0x6f",
						"type":"0x1",
						"accessList":[],
						"chainId":"0x539",
						"v":"0x0",
						"r":"0x0",
						"s":"0x0",
						"yParity":"0x0"
					},
					{
						"blockHash":"0x2cb4e4b5b5be5a2520377e87e8d7d2cf83fc0783fa6518d67b9606d3c5317b50",
						"blockNumber":"0x64",
						"from":"0x0000000000000000000000000000000000000000",
						"gas":"0x457",
						"gasPrice":"0x2b67",
						"hash":"0x12e1f81207b40c3bdcc13c0ee18f5f86af6d31754d57a0ea1b0d4cfef21abef1",
						"input":"0x111111",
						"nonce":"0x4",
						"to":"0x0000000000000000000000000000000000000011",
						"transactionIndex":"0x3",
						"value":"0x6f",
						"type":"0x0",
						"chainId":"0x7fffffffffffffee",
						"v":"0x0",
						"r":"0x0",
						"s":"0x0"
					}
				],
				"transactionsRoot":"0x661a9febcfa8f1890af549b874faf9fa274aede26ef489d9db0b25daa569450e",
				"uncles":[],
				"validator":"0x",
				"validators":"0x"
			}`,
		},
	}

	for i, tc := range testSuite {
		resp := RPCMarshalBlock(block, tc.inclTx, tc.fullTx, params.MainnetChainConfig)
		out, err := json.Marshal(resp)
		if err != nil {
			t.Errorf("test %d: json marshal error: %v", i, err)
			continue
		}
		require.JSONEqf(t, tc.want, string(out), "test %d", i)
	}
}
