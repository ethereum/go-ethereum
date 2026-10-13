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
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestQKCExtraEncoding(t *testing.T) {
	want := &QKCExtra{
		Version:        1,
		ShardID:        0x00010003,
		PrevRootBlock:  common.BytesToHash(bytes.Repeat([]byte{0xaa}, 32)),
		Cursor:         XShardCursor{RootBlockHeight: 1, MinorBlockIndex: 2, XShardDepositIndex: 3},
		XShardGasUsed:  21000,
		XShardGasLimit: 1_000_000,
		Nonce:          EncodeNonce(0x0102030405060708),
		MixHash:        common.BytesToHash(bytes.Repeat([]byte{0xbb}, 32)),
	}
	wantExtra := common.FromHex("0x" +
		"00000001" + // version
		"00010003" + // shardId
		strings.Repeat("aa", 32) + // prevRootBlock
		"0000000000000001" + "0000000000000002" + "0000000000000003" + // cursor
		"0000000000005208" + // xShardGasUsed
		"00000000000f4240" + // xShardGasLimit
		"0102030405060708" + // nonce
		strings.Repeat("bb", 32)) // mixHash

	extra := want.Encode()
	if !bytes.Equal(extra, wantExtra) {
		t.Fatalf("encoding mismatch:\nhave %x\nwant %x", extra, wantExtra)
	}
	header := &Header{Extra: extra}
	have, err := header.QKCExtra()
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !reflect.DeepEqual(have, want) {
		t.Fatalf("decoding mismatch:\nhave %+v\nwant %+v", have, want)
	}
}

func TestQKCExtraInvalidLength(t *testing.T) {
	for _, size := range []int{0, 32, QKCExtraSize - 1, QKCExtraSize + 1} {
		if _, err := DecodeQKCExtra(make([]byte, size)); err == nil {
			t.Errorf("size %d: expected error", size)
		}
	}
}

func TestQKCDifficulty(t *testing.T) {
	diff := big.NewInt(1_000_000)
	header := &Header{Difficulty: common.Big0, MixDigest: common.BigToHash(diff)}
	if have := header.QKCDifficulty(); have.Cmp(diff) != 0 {
		t.Fatalf("difficulty mismatch: have %v, want %v", have, diff)
	}
}
