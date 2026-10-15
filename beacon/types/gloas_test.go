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

package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	coretypes "github.com/ethereum/go-ethereum/core/types"
)

// TestGloasBuilderRequestsEncoding checks the execution request list derived
// from the requests of a Gloas execution payload envelope. Per EIP-7685 a
// request entry is a single request_type byte followed by the concatenation of
// the type's SSZ serialized records. This matches the EIP-8282 system calls,
// which return every record of a type in one call, and the consensus specs'
// get_execution_requests_list.
func TestGloasBuilderRequestsEncoding(t *testing.T) {
	var (
		pubkey1     = common.Hex2Bytes(strings.Repeat("11", 48))
		pubkey2     = common.Hex2Bytes(strings.Repeat("22", 48))
		credentials = common.Hex2Bytes(strings.Repeat("bb", 32))
		signature   = common.Hex2Bytes(strings.Repeat("cc", 96))
		source      = common.Hex2Bytes(strings.Repeat("dd", 20))
	)
	// A BuilderDepositRequest record, as returned by the builder deposit
	// contract: pubkey ++ withdrawal_credentials ++ amount ++ signature.
	deposit := func(pubkey []byte, amount uint64) []byte {
		data := append([]byte{}, pubkey...)
		data = append(data, credentials...)
		data = binary.LittleEndian.AppendUint64(data, amount)
		return append(data, signature...)
	}
	// A BuilderExitRequest record: source_address ++ pubkey.
	exit := func(source, pubkey []byte) []byte {
		return append(append([]byte{}, source...), pubkey...)
	}
	hexStr := func(b []byte) string { return "0x" + common.Bytes2Hex(b) }

	enc := json.RawMessage(`{
		"deposits": [],
		"withdrawals": [],
		"consolidations": [],
		"builder_deposits": [
			{"pubkey": "` + hexStr(pubkey1) + `", "withdrawal_credentials": "` + hexStr(credentials) + `", "amount": "1", "signature": "` + hexStr(signature) + `"},
			{"pubkey": "` + hexStr(pubkey2) + `", "withdrawal_credentials": "` + hexStr(credentials) + `", "amount": "2", "signature": "` + hexStr(signature) + `"}
		],
		"builder_exits": [
			{"source_address": "` + hexStr(source) + `", "pubkey": "` + hexStr(pubkey1) + `"}
		]
	}`)

	requests, err := decodeGloasRequests(enc)
	if err != nil {
		t.Fatalf("failed to decode requests: %v", err)
	}
	want := [][]byte{
		append([]byte{0x03}, append(deposit(pubkey1, 1), deposit(pubkey2, 2)...)...),
		append([]byte{0x04}, exit(source, pubkey1)...),
	}
	if len(requests) != len(want) {
		t.Fatalf("wrong number of request entries: got %d, want %d", len(requests), len(want))
	}
	for i := range want {
		if !bytes.Equal(requests[i], want[i]) {
			t.Errorf("request entry %d mismatch:\n got %x\nwant %x", i, requests[i], want[i])
		}
	}
	// The commitment over the well-formed list must match the hash the
	// execution layer computes while producing the block.
	if have, wantHash := coretypes.CalcRequestsHash(requests), coretypes.CalcRequestsHash(want); have != wantHash {
		t.Errorf("wrong requests hash: got %x, want %x", have, wantHash)
	}
}
