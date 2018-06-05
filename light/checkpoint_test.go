// Copyright 2018 The go-ethereum Authors
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

package light

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

var testCheckpoint = &TrustedCheckpoint{
	Name:          "test",
	SectionIdx:    100,
	SectionHead:   common.HexToHash("0xbeef"),
	ChtRoot:       common.HexToHash("0xdead"),
	BloomTrieRoot: common.HexToHash("0xdeadbeef"),
}

func TestRWCheckpoint(t *testing.T) {
	mdb := ethdb.NewMemDatabase()
	WriteTrustedCheckpoint(mdb, testCheckpoint)
	if !assertCheckpointEqual(testCheckpoint, ReadTrustedCheckpoint(mdb)) {
		t.Error("the checkpoint retrieved from database is different")
	}
}

func assertCheckpointEqual(ckp1, ckp2 *TrustedCheckpoint) bool {
	return ckp1.SectionIdx == ckp2.SectionIdx && ckp1.SectionHead == ckp2.SectionHead && ckp1.ChtRoot == ckp2.ChtRoot &&
		ckp1.BloomTrieRoot == ckp2.BloomTrieRoot
}
