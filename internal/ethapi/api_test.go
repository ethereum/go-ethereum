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
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func testTransactionMarshal(t *testing.T, tests []txData, config *params.ChainConfig) {
	t.Helper()
	t.Parallel()

	var (
		signer = types.LatestSigner(config)
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	)

	for i, tt := range tests {
		var tx2 types.Transaction

		tx, err := types.SignNewTx(key, signer, tt.Tx)
		if err != nil {
			t.Fatalf("test %d: signing failed: %v", i, err)
		}
		// Regular transaction
		if data, err := json.Marshal(tx); err != nil {
			t.Fatalf("test %d: marshalling failed; %v", i, err)
		} else if err = tx2.UnmarshalJSON(data); err != nil {
			t.Fatalf("test %d: sunmarshal failed: %v", i, err)
		} else if want, have := tx.Hash(), tx2.Hash(); want != have {
			t.Fatalf("test %d: stx changed, want %x have %x", i, want, have)
		}

		// rpcTransaction
		rpcTx := newRPCTransaction(tx, common.Hash{}, 0, 0, big.NewInt(0), config)
		if data, err := json.Marshal(rpcTx); err != nil {
			t.Fatalf("test %d: marshalling failed; %v", i, err)
		} else if err = tx2.UnmarshalJSON(data); err != nil {
			t.Fatalf("test %d: unmarshal failed: %v", i, err)
		} else if want, have := tx.Hash(), tx2.Hash(); want != have {
			t.Fatalf("test %d: tx changed, want %x have %x", i, want, have)
		} else {
			want, have := tt.Want, string(data)
			require.JSONEqf(t, want, have, "test %d: rpc json not match, want %s have %s", i, want, have)
		}
	}
}

type txData struct {
	Tx   types.TxData
	Want string
}

func TestTransaction_RoundTripRpcJSON(t *testing.T) {
	var (
		config = params.AllEthashProtocolChanges
		signer = types.LatestSigner(config)
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		tests  = allTransactionTypes(common.Address{0xde, 0xad}, config)
	)

	testTransactionMarshal(t, tests, config)

	for i, tt := range tests {
		var tx2 types.Transaction

		tx, err := types.SignNewTx(key, signer, tt.Tx)
		if err != nil {
			t.Fatalf("test %d: signing failed: %v", i, err)
		}
		// Regular transaction
		if data, err := json.Marshal(tx); err != nil {
			t.Fatalf("test %d: marshalling failed; %v", i, err)
		} else if err = tx2.UnmarshalJSON(data); err != nil {
			t.Fatalf("test %d: sunmarshal failed: %v", i, err)
		} else if want, have := tx.Hash(), tx2.Hash(); want != have {
			t.Fatalf("test %d: stx changed, want %x have %x", i, want, have)
		}

		//  rpcTransaction
		rpcTx := newRPCTransaction(tx, common.Hash{}, 0, 0, nil, config)
		if data, err := json.Marshal(rpcTx); err != nil {
			t.Fatalf("test %d: marshalling failed; %v", i, err)
		} else if err = tx2.UnmarshalJSON(data); err != nil {
			t.Fatalf("test %d: unmarshal failed: %v", i, err)
		} else if want, have := tx.Hash(), tx2.Hash(); want != have {
			t.Fatalf("test %d: tx changed, want %x have %x", i, want, have)
		}
	}
}

func allTransactionTypes(addr common.Address, config *params.ChainConfig) []txData {
	return []txData{
		{
			Tx: &types.LegacyTx{
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       &addr,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				V:        big.NewInt(9),
				R:        big.NewInt(10),
				S:        big.NewInt(11),
			},
			Want: `{
				"blockHash": null,
				"blockNumber": null,
				"from": "0x71562b71999873db5b286df957af199ec94617f7",
				"gas": "0x7",
				"gasPrice": "0x6",
				"hash": "0x5f3240454cd09a5d8b1c5d651eefae7a339262875bcd2d0e6676f3d989967008",
				"input": "0x0001020304",
				"nonce": "0x5",
				"to": "0xdead000000000000000000000000000000000000",
				"transactionIndex": null,
				"value": "0x8",
				"type": "0x0",
				"chainId": "0x539",
				"v": "0xa96",
				"r": "0xbc85e96592b95f7160825d837abb407f009df9ebe8f1b9158a4b8dd093377f75",
				"s": "0x1b55ea3af5574c536967b039ba6999ef6c89cf22fc04bcb296e0e8b0b9b576f5"
			}`,
		}, {
			Tx: &types.LegacyTx{
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       nil,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				V:        big.NewInt(32),
				R:        big.NewInt(10),
				S:        big.NewInt(11),
			},
			Want: `{
				"blockHash": null,
				"blockNumber": null,
				"from": "0x71562b71999873db5b286df957af199ec94617f7",
				"gas": "0x7",
				"gasPrice": "0x6",
				"hash": "0x806e97f9d712b6cb7e781122001380a2837531b0fc1e5f5d78174ad4cb699873",
				"input": "0x0001020304",
				"nonce": "0x5",
				"to": null,
				"transactionIndex": null,
				"value": "0x8",
				"type": "0x0",
				"chainId": "0x539",
				"v": "0xa96",
				"r": "0x9dc28b267b6ad4e4af6fe9289668f9305c2eb7a3241567860699e478af06835a",
				"s": "0xa0b51a071aa9bed2cd70aedea859779dff039e3630ea38497d95202e9b1fec7"
			}`,
		},
		{
			Tx: &types.AccessListTx{
				ChainID:  config.ChainID,
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       &addr,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{
					types.AccessTuple{
						Address:     common.Address{0x2},
						StorageKeys: []common.Hash{types.EmptyRootHash},
					},
				},
				V: big.NewInt(32),
				R: big.NewInt(10),
				S: big.NewInt(11),
			},
			Want: `{
				"blockHash": null,
				"blockNumber": null,
				"from": "0x71562b71999873db5b286df957af199ec94617f7",
				"gas": "0x7",
				"gasPrice": "0x6",
				"hash": "0x121347468ee5fe0a29f02b49b4ffd1c8342bc4255146bb686cd07117f79e7129",
				"input": "0x0001020304",
				"nonce": "0x5",
				"to": "0xdead000000000000000000000000000000000000",
				"transactionIndex": null,
				"value": "0x8",
				"type": "0x1",
				"accessList": [
					{
						"address": "0x0200000000000000000000000000000000000000",
						"storageKeys": [
							"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
						]
					}
				],
				"chainId": "0x539",
				"v": "0x0",
				"r": "0xf372ad499239ae11d91d34c559ffc5dab4daffc0069e03afcabdcdf231a0c16b",
				"s": "0x28573161d1f9472fa0fd4752533609e72f06414f7ab5588699a7141f65d2abf"
			}`,
		}, {
			Tx: &types.AccessListTx{
				ChainID:  config.ChainID,
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       nil,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{
					types.AccessTuple{
						Address:     common.Address{0x2},
						StorageKeys: []common.Hash{types.EmptyRootHash},
					},
				},
				V: big.NewInt(32),
				R: big.NewInt(10),
				S: big.NewInt(11),
			},
			Want: `{
				"blockHash": null,
				"blockNumber": null,
				"from": "0x71562b71999873db5b286df957af199ec94617f7",
				"gas": "0x7",
				"gasPrice": "0x6",
				"hash": "0x067c3baebede8027b0f828a9d933be545f7caaec623b00684ac0659726e2055b",
				"input": "0x0001020304",
				"nonce": "0x5",
				"to": null,
				"transactionIndex": null,
				"value": "0x8",
				"type": "0x1",
				"accessList": [
					{
						"address": "0x0200000000000000000000000000000000000000",
						"storageKeys": [
							"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
						]
					}
				],
				"chainId": "0x539",
				"v": "0x1",
				"r": "0x542981b5130d4613897fbab144796cb36d3cb3d7807d47d9c7f89ca7745b085c",
				"s": "0x7425b9dd6c5deaa42e4ede35d0c4570c4624f68c28d812c10d806ffdf86ce63"
			}`,
		}, {
			Tx: &types.DynamicFeeTx{
				ChainID:   config.ChainID,
				Nonce:     5,
				GasTipCap: big.NewInt(6),
				GasFeeCap: big.NewInt(9),
				Gas:       7,
				To:        &addr,
				Value:     big.NewInt(8),
				Data:      []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{
					types.AccessTuple{
						Address:     common.Address{0x2},
						StorageKeys: []common.Hash{types.EmptyRootHash},
					},
				},
				V: big.NewInt(32),
				R: big.NewInt(10),
				S: big.NewInt(11),
			},
			Want: `{
				"blockHash": null,
				"blockNumber": null,
				"from": "0x71562b71999873db5b286df957af199ec94617f7",
				"gas": "0x7",
				"gasPrice": "0x9",
				"maxFeePerGas": "0x9",
				"maxPriorityFeePerGas": "0x6",
				"hash": "0xb63e0b146b34c3e9cb7fbabb5b3c081254a7ded6f1b65324b5898cc0545d79ff",
				"input": "0x0001020304",
				"nonce": "0x5",
				"to": "0xdead000000000000000000000000000000000000",
				"transactionIndex": null,
				"value": "0x8",
				"type": "0x2",
				"accessList": [
					{
						"address": "0x0200000000000000000000000000000000000000",
						"storageKeys": [
							"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
						]
					}
				],
				"chainId": "0x539",
				"v": "0x1",
				"r": "0x3b167e05418a8932cd53d7578711fe1a76b9b96c48642402bb94978b7a107e80",
				"s": "0x22f98a332d15ea2cc80386c1ebaa31b0afebfa79ebc7d039a1e0074418301fef"
			}`,
		}, {
			Tx: &types.DynamicFeeTx{
				ChainID:    config.ChainID,
				Nonce:      5,
				GasTipCap:  big.NewInt(6),
				GasFeeCap:  big.NewInt(9),
				Gas:        7,
				To:         nil,
				Value:      big.NewInt(8),
				Data:       []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{},
				V:          big.NewInt(32),
				R:          big.NewInt(10),
				S:          big.NewInt(11),
			},
			Want: `{
				"blockHash": null,
				"blockNumber": null,
				"from": "0x71562b71999873db5b286df957af199ec94617f7",
				"gas": "0x7",
				"gasPrice": "0x9",
				"maxFeePerGas": "0x9",
				"maxPriorityFeePerGas": "0x6",
				"hash": "0xcbab17ee031a9d5b5a09dff909f0a28aedb9b295ac0635d8710d11c7b806ec68",
				"input": "0x0001020304",
				"nonce": "0x5",
				"to": null,
				"transactionIndex": null,
				"value": "0x8",
				"type": "0x2",
				"accessList": [],
				"chainId": "0x539",
				"v": "0x0",
				"r": "0x6446b8a682db7e619fc6b4f6d1f708f6a17351a41c7fbd63665f469bc78b41b9",
				"s": "0x7626abc15834f391a117c63450047309dbf84c5ce3e8e609b607062641e2de43"
			}`,
		},
	}
}
