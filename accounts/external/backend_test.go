// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
// or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser General Public License
// for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package external

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

func TestSignTxArgsSetCodeAuthorizationList(t *testing.T) {
	account := accounts.Account{Address: common.HexToAddress("0x1000000000000000000000000000000000000001")}
	to := common.HexToAddress("0x2000000000000000000000000000000000000002")
	auth := types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(5),
		Address: common.HexToAddress("0x3000000000000000000000000000000000000003"),
		Nonce:   9,
		V:       1,
		R:       *uint256.NewInt(10),
		S:       *uint256.NewInt(11),
	}
	tx := types.NewTx(&types.SetCodeTx{
		ChainID:   uint256.NewInt(5),
		Nonce:     4,
		GasTipCap: uint256.NewInt(3),
		GasFeeCap: uint256.NewInt(30),
		Gas:       21000,
		To:        to,
		Value:     uint256.NewInt(7),
		Data:      []byte{0x01, 0x02},
		AccessList: types.AccessList{{
			Address:     common.HexToAddress("0x4000000000000000000000000000000000000004"),
			StorageKeys: []common.Hash{common.HexToHash("0x01")},
		}},
		AuthList: []types.SetCodeAuthorization{auth},
	})
	args, err := signTxArgs(account, tx, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	if args.ChainID == nil || (*big.Int)(args.ChainID).Cmp(big.NewInt(5)) != 0 {
		t.Fatalf("have chain id %v, want 5", args.ChainID)
	}
	if args.MaxFeePerGas == nil || (*big.Int)(args.MaxFeePerGas).Cmp(big.NewInt(30)) != 0 {
		t.Fatalf("have max fee %v, want 30", args.MaxFeePerGas)
	}
	if args.MaxPriorityFeePerGas == nil || (*big.Int)(args.MaxPriorityFeePerGas).Cmp(big.NewInt(3)) != 0 {
		t.Fatalf("have max priority fee %v, want 3", args.MaxPriorityFeePerGas)
	}
	if !reflect.DeepEqual(args.AuthorizationList, []types.SetCodeAuthorization{auth}) {
		t.Fatalf("have auth list %#v, want %#v", args.AuthorizationList, []types.SetCodeAuthorization{auth})
	}
	blob, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(blob, []byte(`"authorizationList"`)) {
		t.Fatalf("marshaled args missing authorizationList: %s", blob)
	}
}
