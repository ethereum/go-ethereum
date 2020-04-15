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

package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-colorable"
)

func TestFileStorageAPI(t *testing.T) {
	a := map[string]string{
		"secret":  "value1",
		"secret2": "value2",
	}
	d, err := ioutil.TempDir("", "eth-encrypted-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	stored := &FileStorageAPI{
		filename: fmt.Sprintf("%v/vault.json", d),
	}
	stored.writeStorage(a)
	read := &FileStorageAPI{
		filename: fmt.Sprintf("%v/vault.json", d),
	}
	creds, err := read.readStorage()
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range a {
		if v2, exist := creds[k]; !exist {
			t.Errorf("Missing entry %v", k)
		} else {
			if v != v2 {
				t.Errorf("Wrong ciphertext, expected %x got %x", v, v2)
			}
			if v != v2 {
				t.Errorf("Wrong iv")
			}
		}
	}
}

func TestEnd2End(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(3), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))

	d, err := ioutil.TempDir("", "eth-encrypted-storage-test")
	if err != nil {
		t.Fatal(err)
	}

	filename := fmt.Sprintf("%v/vault.json", d)
	key := []byte("AES256Key-32Characters1234567890")
	fs := NewFileStorage(filename, key)

	fs.Put("bazonk", "foobar")

	// make sure intermediate result is encrypted correctly
	encrypted, err := fs.api.Get("bazonk")
	if err != nil {
		t.Error("Failed to retrieve encrypted credential")
	}
	cred := StoredCredential{}
	if err = json.Unmarshal([]byte(encrypted), &cred); err != nil {
		t.Error("Failed to unmarshal encrypted credential", "err", err)
	}

	// make sure return is correct
	if v, err := fs.Get("bazonk"); v != "foobar" || err != nil {
		t.Errorf("Expected bazonk->foobar (nil error), got '%v' (%v error)", v, err)
	}
}

func TestSwappedKeys(t *testing.T) {
	// It should not be possible to swap the keys/values, so that
	// K1:V1, K2:V2 can be swapped into K1:V2, K2:V1
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(3), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))

	d, err := ioutil.TempDir("", "eth-encrypted-storage-test")
	if err != nil {
		t.Fatal(err)
	}

	filename := fmt.Sprintf("%v/vault.json", d)
	key := []byte("AES256Key-32Characters1234567890")
	s1 := NewFileStorage(filename, key)
	s1.Put("k1", "v1")
	s1.Put("k2", "v2")
	// Now make a modified copy

	creds := make(map[string]string)
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &creds); err != nil {
		t.Fatal(err)
	}
	swap := func() {
		// Turn it into K1:V2, K2:V2
		v1, v2 := creds["k1"], creds["k2"]
		creds["k2"], creds["k1"] = v1, v2
		raw, err = json.Marshal(creds)
		if err != nil {
			t.Fatal(err)
		}
		if err = ioutil.WriteFile(filename, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	swap()
	if v, _ := s1.Get("k1"); v != "" {
		t.Errorf("swapped value should return empty")
	}
	swap()
	if v, _ := s1.Get("k1"); v != "v1" {
		t.Errorf("double-swapped value should work fine")
	}
}
