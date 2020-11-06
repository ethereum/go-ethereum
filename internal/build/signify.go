// Copyright 2020 The go-ethereum Authors
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

// signFile reads the contents of an input file and signs it (in armored format)
// with the key provided, placing the signature into the output file.

package build

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"crypto/ed25519"
)

var (
	errInvalidKeyHeader = errors.New("Incorrect key header")
	errInvalidKeyLength = errors.New("invalid, key length != 42")
)

func readSKey(key []byte) (ed25519.PrivateKey, error) {
	if len(key) != 104 {
		return nil, errInvalidKeyLength
	}

	if string(key[:2]) != "Ed" {
		return nil, errInvalidKeyHeader
	}

	return ed25519.PrivateKey(key[40:]), nil

}

// SignifySignFile creates a signature of the input file.
func SignifySignFile(input string, output string, key string) error {
	in, err := os.Open(input)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	keydata, err := base64.StdEncoding.DecodeString(key)
	skey, err := readSKey(keydata)
	if err != nil {
		return nil
	}

	filedata, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	out.WriteString(fmt.Sprintf("untrusted comment: verify with geth.pub\n%s\n", base64.StdEncoding.EncodeToString(ed25519.Sign(skey, filedata))))
	return nil
}
