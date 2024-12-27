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

package backends

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/XinFinOrg/XDPoSChain"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/params"
)

func TestSimulatedBackend_EstimateGas(t *testing.T) {
	/*
		pragma solidity ^0.6.4;
		contract GasEstimation {
		    function PureRevert() public { revert(); }
		    function Revert() public { revert("revert reason");}
		    function OOG() public { for (uint i = 0; ; i++) {}}
		    function Assert() public { assert(false);}
		    function Valid() public {}
		}*/
	const contractAbi = "[{\"inputs\":[],\"name\":\"Assert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OOG\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PureRevert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"Revert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"Valid\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
	const contractBin = "0x60806040523480156100115760006000fd5b50610017565b61016e806100266000396000f3fe60806040523480156100115760006000fd5b506004361061005c5760003560e01c806350f6fe3414610062578063aa8b1d301461006c578063b9b046f914610076578063d8b9839114610080578063e09fface1461008a5761005c565b60006000fd5b61006a610094565b005b6100746100ad565b005b61007e6100b5565b005b6100886100c2565b005b610092610135565b005b6000600090505b5b808060010191505061009b565b505b565b60006000fd5b565b600015156100bf57fe5b5b565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252600d8152602001807f72657665727420726561736f6e0000000000000000000000000000000000000081526020015060200191505060405180910390fd5b565b5b56fea2646970667358221220345bbcbb1a5ecf22b53a78eaebf95f8ee0eceff6d10d4b9643495084d2ec934a64736f6c63430006040033"

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	opts := bind.NewKeyedTransactor(key)

	sim := NewXDCSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(params.Ether)}}, 10000000, &params.ChainConfig{
		ConstantinopleBlock: big.NewInt(0),
		XDPoS: &params.XDPoSConfig{
			Epoch:            900,
			SkipV1Validation: true,
			V2: &params.V2{
				SwitchBlock:   big.NewInt(900),
				CurrentConfig: params.UnitTestV2Configs[0],
			},
		},
	})

	defer sim.Close()

	parsed, _ := abi.JSON(strings.NewReader(contractAbi))
	contractAddr, _, _, _ := bind.DeployContract(opts, parsed, common.FromHex(contractBin), sim)
	sim.Commit()

	var cases = []struct {
		name        string
		message     XDPoSChain.CallMsg
		expect      uint64
		expectError error
	}{
		{"plain transfer(valid)", XDPoSChain.CallMsg{
			From:     addr,
			To:       &addr,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    big.NewInt(1),
			Data:     nil,
		}, params.TxGas, nil},

		{"plain transfer(invalid)", XDPoSChain.CallMsg{
			From:     addr,
			To:       &contractAddr,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    big.NewInt(1),
			Data:     nil,
		}, 0, errors.New("always failing transaction (execution reverted)")},

		{"Revert", XDPoSChain.CallMsg{
			From:     addr,
			To:       &contractAddr,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    nil,
			Data:     common.Hex2Bytes("d8b98391"),
		}, 0, errors.New("always failing transaction (execution reverted) (revert reason)")},

		{"PureRevert", XDPoSChain.CallMsg{
			From:     addr,
			To:       &contractAddr,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    nil,
			Data:     common.Hex2Bytes("aa8b1d30"),
		}, 0, errors.New("always failing transaction (execution reverted)")},

		{"OOG", XDPoSChain.CallMsg{
			From:     addr,
			To:       &contractAddr,
			Gas:      100000,
			GasPrice: big.NewInt(0),
			Value:    nil,
			Data:     common.Hex2Bytes("50f6fe34"),
		}, 0, errors.New("gas required exceeds allowance (100000)")},

		{"Assert", XDPoSChain.CallMsg{
			From:     addr,
			To:       &contractAddr,
			Gas:      100000,
			GasPrice: big.NewInt(0),
			Value:    nil,
			Data:     common.Hex2Bytes("b9b046f9"),
		}, 0, errors.New("always failing transaction (invalid opcode: INVALID)")},

		{"Valid", XDPoSChain.CallMsg{
			From:     addr,
			To:       &contractAddr,
			Gas:      100000,
			GasPrice: big.NewInt(0),
			Value:    nil,
			Data:     common.Hex2Bytes("e09fface"),
		}, 21483, nil},
	}
	for _, c := range cases {
		got, err := sim.EstimateGas(context.Background(), c.message)
		if c.expectError != nil {
			if err == nil {
				t.Fatalf("Expect error, got nil")
			}
			if c.expectError.Error() != err.Error() {
				t.Fatalf("Expect error, want %v, got %v", c.expectError, err)
			}
			continue
		}
		if got != c.expect {
			t.Fatalf("Gas estimation mismatch, want %d, got %d", c.expect, got)
		}
	}
}
