// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package crypto

import (
	"testing"
)

func TestMnDecode(t *testing.T) {
	words := []string{
		"ink",
		"balance",
		"gain",
		"fear",
		"happen",
		"melt",
		"mom",
		"surface",
		"stir",
		"bottle",
		"unseen",
		"expression",
		"important",
		"curl",
		"grant",
		"fairy",
		"across",
		"back",
		"figure",
		"breast",
		"nobody",
		"scratch",
		"worry",
		"yesterday",
	}
	encode := "c61d43dc5bb7a4e754d111dae8105b6f25356492df5e50ecb33b858d94f8c338"
	result := MnemonicDecode(words)
	if encode != result {
		t.Error("We expected", encode, "got", result, "instead")
	}
}
func TestMnEncode(t *testing.T) {
	encode := "c61d43dc5bb7a4e754d111dae8105b6f25356492df5e50ecb33b858d94f8c338"
	result := []string{
		"ink",
		"balance",
		"gain",
		"fear",
		"happen",
		"melt",
		"mom",
		"surface",
		"stir",
		"bottle",
		"unseen",
		"expression",
		"important",
		"curl",
		"grant",
		"fairy",
		"across",
		"back",
		"figure",
		"breast",
		"nobody",
		"scratch",
		"worry",
		"yesterday",
	}
	words := MnemonicEncode(encode)
	for i, word := range words {
		if word != result[i] {
			t.Error("Mnenonic does not match:", words, result)
		}
	}
}
