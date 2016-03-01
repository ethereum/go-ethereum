// Copyright 2016 The go-ethereum Authors
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

package crypto

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Tests that a json key file can be decrypted and encrypted in multiple rounds.
func TestKeyEncryptDecrypt(t *testing.T) {
	address := common.HexToAddress("f626acac23772cbe04dd578bee681b06bdefb9fa")
	keyjson := []byte("{\"address\":\"f626acac23772cbe04dd578bee681b06bdefb9fa\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"1bcf0ab9b14459795ce59f63e63255ffd84dc38d31614a5a78e37144d7e4a17f\",\"cipherparams\":{\"iv\":\"df4c7e225ee2d81adef522013e3fbe24\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":262144,\"p\":1,\"r\":8,\"salt\":\"2909a99dd2bfa7079a4b40991773b1083f8512c0c55b9b63402ab0e3dc8db8b3\"},\"mac\":\"4ecf6a4ad92ae2c016cb7c44abade74799480c3303eb024661270dfefdbc7510\"},\"id\":\"b4718210-9a30-4883-b8a6-dbdd08bd0ceb\",\"version\":3}")
	password := ""

	// Do a few rounds of decryption and encryption
	for i := 0; i < 3; i++ {
		// Try a bad password first
		if _, err := DecryptKey(keyjson, password+"bad"); err == nil {
			t.Error("test %d: json key decrypted with bad password", i)
		}
		// Decrypt with the correct password
		key, err := DecryptKey(keyjson, password)
		if err != nil {
			t.Errorf("test %d: json key failed to decrypt: %v", i, err)
		}
		if key.Address != address {
			t.Errorf("test %d: key address mismatch: have %x, want %x", i, key.Address, address)
		}
		// Recrypt with a new password and start over
		password += "new data appended"
		if keyjson, err = EncryptKey(key, password, LightScryptN, LightScryptP); err != nil {
			t.Errorf("test %d: failed to recrypt key %v", err)
		}
	}
}
