// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"testing"

	"github.com/ethereum/go-ethereum/p2p"
)

func TestDecodeRLPxDisconnect(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    p2p.DiscReason
		wantErr bool
	}{
		{
			name:    "list form (spec-compliant)",
			payload: []byte{0xc1, 0x04}, // [4] = TooManyPeers
			want:    p2p.DiscTooManyPeers,
		},
		{
			name:    "list form with reason zero",
			payload: []byte{0xc1, 0x80}, // [0] = Requested
			want:    p2p.DiscRequested,
		},
		{
			name:    "bare byte form (legacy geth)",
			payload: []byte{0x04}, // 4 = TooManyPeers
			want:    p2p.DiscTooManyPeers,
		},
		{
			name:    "bare byte form zero",
			payload: []byte{0x80}, // 0 = Requested
			want:    p2p.DiscRequested,
		},
		{
			name:    "empty payload",
			payload: []byte{},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeRLPxDisconnect(tc.payload)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got reason=%v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got reason %v, want %v", got, tc.want)
			}
		})
	}
}
