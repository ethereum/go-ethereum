// Copyright 2017 The go-ethereum Authors
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

package posw

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// These expected values are the upstream go-ethereum Ethash test vector.
func TestHashimotoReference(t *testing.T) {
	cache := make([]uint32, 1024/4)
	generateCache(cache, 0, make([]byte, 32))
	hash := common.FromHex("0xc9149cc0386e689d789a1c2f3d5d169a61a6218ed30e74414dc736e442ef3d1f")
	digest, result := hashimotoLight(32*1024, cache, hash, 0)
	if !bytes.Equal(digest, common.FromHex("0xe4073cffaef931d37117cefd9afd27ea0f1cad6a981dd2605c4a1ac97c519800")) {
		t.Fatalf("digest: %x", digest)
	}
	if !bytes.Equal(result, common.FromHex("0xd3539235ee2e6f8db665c0a72169f55b7f6c605712330b778ec3944f0eb5a557")) {
		t.Fatalf("result: %x", result)
	}
	if cacheSize(0) != 16776896 || datasetSize(0) != 1073739904 {
		t.Fatal("incorrect Ethash sizes")
	}
	if cacheSize(29999) != cacheSize(0) || datasetSize(29999) != datasetSize(0) ||
		cacheSize(30000) != 16907456 || datasetSize(30000) != 1082130304 {
		t.Fatal("incorrect Ethash epoch boundary sizes")
	}
}
