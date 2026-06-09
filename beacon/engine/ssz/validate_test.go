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

package ssz

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/karalabe/ssz"
)

func u64p(v uint64) *uint64 { return &v }

func TestExecutionPayloadValidate(t *testing.T) {
	tests := []struct {
		name    string
		fork    ssz.Fork
		mutate  func(*ExecutionPayload)
		wantErr bool
	}{
		{"paris ok", ssz.ForkParis, func(p *ExecutionPayload) {}, false},
		{"paris with withdrawals", ssz.ForkParis, func(p *ExecutionPayload) {
			p.Withdrawals = []*Withdrawal{{}}
		}, true},
		{"paris with blob gas", ssz.ForkParis, func(p *ExecutionPayload) {
			p.BlobGasUsed = u64p(1)
		}, true},
		{"cancun missing blob gas", ssz.ForkDencun, func(p *ExecutionPayload) {
			p.BlobGasUsed, p.ExcessBlobGas = nil, nil
		}, true},
		{"cancun ok", ssz.ForkDencun, func(p *ExecutionPayload) {
			p.BlobGasUsed, p.ExcessBlobGas = u64p(0), u64p(0)
		}, false},
		{"cancun with bal", ssz.ForkDencun, func(p *ExecutionPayload) {
			p.BlobGasUsed, p.ExcessBlobGas = u64p(0), u64p(0)
			p.BlockAccessList = []byte{0x01}
		}, true},
		{"amsterdam missing slot", forkAmsterdam, func(p *ExecutionPayload) {
			p.BlobGasUsed, p.ExcessBlobGas = u64p(0), u64p(0)
			p.SlotNumber = nil
		}, true},
		{"amsterdam ok", forkAmsterdam, func(p *ExecutionPayload) {
			p.BlobGasUsed, p.ExcessBlobGas = u64p(0), u64p(0)
			p.SlotNumber = u64p(1)
			p.BlockAccessList = []byte{0x01}
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &ExecutionPayload{}
			tc.mutate(p)
			err := p.Validate(tc.fork)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate(%d) err=%v wantErr=%v", tc.fork, err, tc.wantErr)
			}
		})
	}
}

func TestPayloadAttributesValidate(t *testing.T) {
	root := common.Hash{0x77}
	tests := []struct {
		name    string
		fork    ssz.Fork
		mutate  func(*PayloadAttributes)
		wantErr bool
	}{
		{"paris ok", ssz.ForkParis, func(a *PayloadAttributes) {}, false},
		{"cancun missing beacon root", ssz.ForkDencun, func(a *PayloadAttributes) {}, true},
		{"cancun ok", ssz.ForkDencun, func(a *PayloadAttributes) {
			a.ParentBeaconBlockRoot = &root
		}, false},
		{"paris with beacon root", ssz.ForkParis, func(a *PayloadAttributes) {
			a.ParentBeaconBlockRoot = &root
		}, true},
		{"amsterdam missing slot/tgl", forkAmsterdam, func(a *PayloadAttributes) {
			a.ParentBeaconBlockRoot = &root
		}, true},
		{"amsterdam ok", forkAmsterdam, func(a *PayloadAttributes) {
			a.ParentBeaconBlockRoot = &root
			a.SlotNumber, a.TargetGasLimit = u64p(1), u64p(2)
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &PayloadAttributes{}
			tc.mutate(a)
			err := a.Validate(tc.fork)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate(%d) err=%v wantErr=%v", tc.fork, err, tc.wantErr)
			}
		})
	}
}

func TestExecutionPayloadBodyValidate(t *testing.T) {
	if err := (&ExecutionPayloadBody{Withdrawals: []*Withdrawal{{}}}).Validate(ssz.ForkParis); err == nil {
		t.Error("expected error for withdrawals at Paris")
	}
	if err := (&ExecutionPayloadBody{BlockAccessList: []byte{1}}).Validate(ssz.ForkDencun); err == nil {
		t.Error("expected error for bal at Cancun")
	}
	if err := (&ExecutionPayloadBody{Withdrawals: []*Withdrawal{{}}}).Validate(forkAmsterdam); err != nil {
		t.Errorf("unexpected error at Amsterdam: %v", err)
	}
}
