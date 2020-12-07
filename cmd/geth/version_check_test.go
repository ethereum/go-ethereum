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
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestVerification(t *testing.T) {
	// Signatures generated with `minisign`
	t.Run("minisig", func(t *testing.T) {
		// For this test, the pubkey is in testdata/minisign.pub
		// (the privkey is `minisign.sec`, if we want to expand this test. Password 'test' )
		pub := "RWQkliYstQBOKOdtClfgC3IypIPX6TAmoEi7beZ4gyR3wsaezvqOMWsp"
		testVerification(t, pub, "./testdata/vcheck/minisig-sigs/")
	})
	// Signatures generated with `signify-openbsd`
	t.Run("signify-openbsd", func(t *testing.T) {
		t.Skip("This currently fails, minisign expects 4 lines of data, signify provides only 2")
		// For this test, the pubkey is in testdata/signifykey.pub
		// (the privkey is `signifykey.sec`, if we want to expand this test. Password 'test' )
		pub := "RWSKLNhZb0KdATtRT7mZC/bybI3t3+Hv/O2i3ye04Dq9fnT9slpZ1a2/"
		testVerification(t, pub, "./testdata/vcheck/signify-sigs/")
	})
}

func testVerification(t *testing.T, pubkey, sigdir string) {
	// Data to verify
	data, err := ioutil.ReadFile("./testdata/vcheck/data.json")
	if err != nil {
		t.Fatal(err)
	}
	// Signatures, with and without comments, both trusted and untrusted
	files, err := ioutil.ReadDir(sigdir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		sig, err := ioutil.ReadFile(filepath.Join(sigdir, f.Name()))
		if err != nil {
			t.Fatal(err)
		}
		err = verifySignature([]string{pubkey}, data, sig)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestJson(t *testing.T) {
	data, _ := ioutil.ReadFile("./testdata/vcheck/data2.json")
	var vulns []vulnJson
	if err := json.Unmarshal(data, &vulns); err != nil {
		t.Fatal(err)
	}
	if len(vulns) == 0 {
		t.Fatal("expected data, got none")
	}
	if have, want := vulns[0].CVE, "correct"; have != want {
		t.Errorf("have %v, want %v", have, want)
	}
}
