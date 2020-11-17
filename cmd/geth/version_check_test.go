// Copyright 2020 The go-ethereum Authors
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
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestVerification(t *testing.T) {
	// For this test, the pubkey is in testdata/minisign.pub
	// (the privkey is there aswell, if we want to expand this test. Password 'test' )
	pub := "RWQkliYstQBOKOdtClfgC3IypIPX6TAmoEi7beZ4gyR3wsaezvqOMWsp"
	// Data to verify
	data, err := ioutil.ReadFile("./testdata/vcheck/data.json")
	// Signatures, with and without comments, both trusted and untrusted
	files, err := ioutil.ReadDir("./testdata/vcheck/sigs/")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		sig, err := ioutil.ReadFile(filepath.Join(".", "testdata", "vcheck", "sigs", f.Name()))
		if err != nil {
			t.Fatal(err)
		}
		err = verifySignature(pub, data, sig)
		if err != nil {
			t.Fatal(err)
		}
	}
}
