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

package framepool

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestClassifyPrefixShapes(t *testing.T) {
	sender := common.HexToAddress("0x1234")
	sponsor := common.HexToAddress("0x5678")
	deploy := types.Frame{Mode: types.ModeDefault}
	self := types.Frame{Mode: types.ModeVerify, Flags: types.ApproveExecutionAndPayment}
	only := types.Frame{Mode: types.ModeVerify, Flags: types.ApproveExecution, Target: &sender}
	pay := types.Frame{Mode: types.ModeVerify, Flags: types.ApprovePayment, Target: &sponsor}
	expiry := types.Frame{Mode: types.ModeVerify, Target: &params.FrameTxExpiryVerifier, Data: make([]byte, 8)}
	binary.BigEndian.PutUint64(expiry.Data, 0x123456789abcdef0)
	shapes := []struct {
		frames              []types.Frame
		deploy, verify, pay int
	}{
		{[]types.Frame{self}, -1, 0, -1},
		{[]types.Frame{deploy, self}, 0, 1, -1},
		{[]types.Frame{only, pay}, -1, 0, 1},
		{[]types.Frame{deploy, only, pay}, 0, 1, 2},
	}
	for i, shape := range shapes {
		for _, withExpiry := range []bool{false, true} {
			t.Run(fmt.Sprintf("shape%d/expiry%t", i, withExpiry), func(t *testing.T) {
				frames := append([]types.Frame{}, shape.frames...)
				offset, expiryIndex := 0, -1
				if withExpiry {
					frames = append([]types.Frame{expiry}, frames...)
					offset, expiryIndex = 1, 0
				}
				for j := range frames {
					frames[j].GasLimits = types.Limits{Execution: 100, State: 200}
				}
				end := len(frames) - 1
				// Post-ops are not a second deploy, and do not count toward the budget.
				frames = append(frames, types.Frame{Mode: types.ModeDefault, GasLimits: types.Limits{Execution: math.MaxUint64}})
				got, err := ClassifyPrefix(frames, sender, nil)
				if err != nil {
					t.Fatal(err)
				}
				shifted := func(index int) int {
					if index < 0 {
						return -1
					}
					return index + offset
				}
				payer := sender
				if shape.pay >= 0 {
					payer = sponsor
				}
				want := Prefix{ExpiryFrame: expiryIndex, DeployFrame: shifted(shape.deploy), VerifyFrame: shifted(shape.verify), PayFrame: shifted(shape.pay), End: end, Payer: payer, ExecutionGas: uint64(end+1) * 100, StateGas: uint64(end+1) * 200}
				if withExpiry {
					want.ExpiryDeadline = 0x123456789abcdef0
				}
				if got != want {
					t.Fatalf("got %+v, want %+v", got, want)
				}
			})
		}
	}
	cases := []struct {
		name   string
		frames []types.Frame
	}{
		{"empty", nil}, {"expiry only", []types.Frame{expiry}},
		{"expiry late", []types.Frame{self, expiry}},
		{"expiry twice", []types.Frame{expiry, expiry, self}},
		{"deploy not first", []types.Frame{only, deploy, pay}},
		{"two deploys", []types.Frame{deploy, deploy, self}},
		{"verify after prefix", []types.Frame{self, only}},
		{"missing pay", []types.Frame{only}},
		{"wrong sender target", []types.Frame{{Mode: types.ModeVerify, Flags: 3, Target: &sponsor}}},
		{"wrong only target", []types.Frame{{Mode: types.ModeVerify, Flags: 2, Target: &sponsor}, pay}},
		{"wrong sender mode", []types.Frame{{Mode: types.ModeSender, Flags: 3}}},
		{"atomic deploy", []types.Frame{{Mode: types.ModeDefault, Flags: types.AtomicBatchFlag}, self}},
		{"expiry data short", []types.Frame{{Mode: types.ModeVerify, Target: &params.FrameTxExpiryVerifier, Data: []byte{1}}, self}},
	}
	for flags := uint64(0); flags < 8; flags++ {
		if flags != 2 && flags != 3 {
			cases = append(cases, struct {
				name   string
				frames []types.Frame
			}{fmt.Sprintf("sender flags %d", flags), []types.Frame{{Mode: types.ModeVerify, Flags: flags}, pay}})
		}
		if flags != 1 {
			cases = append(cases, struct {
				name   string
				frames []types.Frame
			}{fmt.Sprintf("pay flags %d", flags), []types.Frame{only, {Mode: types.ModeVerify, Flags: flags}}})
		}
		if flags != 0 {
			cases = append(cases, struct {
				name   string
				frames []types.Frame
			}{fmt.Sprintf("expiry flags %d", flags), []types.Frame{{Mode: types.ModeVerify, Flags: flags, Target: &params.FrameTxExpiryVerifier, Data: make([]byte, 8)}, self}})
		}
	}
	cases = append(cases, struct {
		name   string
		frames []types.Frame
	}{"pay wrong mode", []types.Frame{only, {Mode: types.ModeDefault, Flags: 1}}})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ClassifyPrefix(tc.frames, sender, nil); !errors.Is(err, ErrInvalidPrefix) {
				t.Fatalf("got %v, want ErrInvalidPrefix", err)
			}
		})
	}
	// Explicit sender self-verification and sender payment are valid targets too.
	self.Target = &sender
	pay.Target = nil
	for _, frames := range [][]types.Frame{{self}, {only, pay}} {
		if _, err := ClassifyPrefix(frames, sender, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestClassifyPrefixBudgets(t *testing.T) {
	sigs := types.SignatureList{{Scheme: types.FrameTxSchemeSecp256k1}, {Scheme: types.FrameTxSchemeP256}, {Scheme: types.FrameTxSchemeArbitrary}}
	sigGas := uint64(0)
	for i := range sigs {
		sigGas += types.FrameTxSignatureGas(&sigs[i])
	}
	cases := []struct {
		name             string
		execution, state uint64
		want             error
	}{
		{"execution cap", MaxVerifyGas - sigGas, 0, nil},
		{"execution cap plus one", MaxVerifyGas - sigGas + 1, 0, ErrPrefixGasLimit},
		{"state cap", 0, MaxVerifyStateGas, nil},
		{"state cap plus one", 0, MaxVerifyStateGas + 1, ErrPrefixGasLimit},
		{"execution overflow", math.MaxUint64, 0, ErrPrefixGasLimit},
		{"state overflow", 0, math.MaxUint64, ErrPrefixGasLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Include the leading expiry budget, with state growth budgeted
			// in the deploy frame and execution spread across frames.
			expiryGas := min(tc.execution, 100)
			frames := []types.Frame{
				{Mode: types.ModeVerify, Target: &params.FrameTxExpiryVerifier, Data: make([]byte, 8), GasLimits: types.Limits{Execution: expiryGas}},
				{Mode: types.ModeDefault, GasLimits: types.Limits{State: tc.state}},
				{Mode: types.ModeVerify, Flags: 3, GasLimits: types.Limits{Execution: tc.execution - expiryGas}},
			}
			got, err := ClassifyPrefix(frames, common.Address{}, sigs)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if err == nil && (got.ExecutionGas != tc.execution+sigGas || got.StateGas != tc.state) {
				t.Fatalf("wrong budgets: %+v", got)
			}
		})
	}
	// Cross-frame summation cannot wrap, nor can many signatures exceed the cap.
	frames := []types.Frame{{Mode: types.ModeDefault, GasLimits: types.Limits{Execution: 1}}, {Mode: types.ModeVerify, Flags: 3, GasLimits: types.Limits{Execution: math.MaxUint64}}}
	if _, err := ClassifyPrefix(frames, common.Address{}, nil); !errors.Is(err, ErrPrefixGasLimit) {
		t.Fatal(err)
	}
	sigs = make(types.SignatureList, MaxVerifyGas/params.FrameTxArbitrarySigGas+1)
	if _, err := ClassifyPrefix([]types.Frame{{Mode: types.ModeVerify, Flags: 3}}, common.Address{}, sigs); !errors.Is(err, ErrPrefixGasLimit) {
		t.Fatal(err)
	}
}
