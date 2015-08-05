// Copyright 2014 The go-ethereum Authors
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

// Package tests implements execution of Ethereum JSON tests.
package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/core"
)

var (
	baseDir            = filepath.Join(".", "files")
	blockTestDir       = filepath.Join(baseDir, "BlockchainTests")
	stateTestDir       = filepath.Join(baseDir, "StateTests")
	transactionTestDir = filepath.Join(baseDir, "TransactionTests")
	vmTestDir          = filepath.Join(baseDir, "VMTests")
	rlpTestDir         = filepath.Join(baseDir, "RLPTests")

	BlockSkipTests = []string{
		// These tests are not valid, as they are out of scope for RLP and
		// the consensus protocol.
		"BLOCK__RandomByteAtTheEnd",
		"TRANSCT__RandomByteAtTheEnd",
		"BLOCK__ZeroByteAtTheEnd",
		"TRANSCT__ZeroByteAtTheEnd",
	}

	/* Go client does not support transaction (account) nonces above 2^64. This
	technically breaks consensus but is regarded as "reasonable
	engineering constraint" as accounts cannot easily reach such high
	nonce values in practice
	*/
	TransSkipTests = []string{"TransactionWithHihghNonce256"}
	StateSkipTests = []string{}
	VmSkipTests    = []string{}
)

func readJson(reader io.Reader, value interface{}) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("Error reading JSON file", err.Error())
	}

	core.DisableBadBlockReporting = true
	if err = json.Unmarshal(data, &value); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(data, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at line %v: %v", line, err)
		}
		return fmt.Errorf("JSON unmarshal error: %v", err)
	}
	return nil
}

func readJsonHttp(uri string, value interface{}) error {
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = readJson(resp.Body, value)
	if err != nil {
		return err
	}
	return nil
}

func readJsonFile(fn string, value interface{}) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer file.Close()

	err = readJson(file, value)
	if err != nil {
		return fmt.Errorf("%s in file %s", err.Error(), fn)
	}
	return nil
}

// findLine returns the line number for the given offset into data.
func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}
