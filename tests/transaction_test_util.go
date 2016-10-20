// Copyright 2015 The go-ethereum Authors
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

package tests

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// Transaction Test JSON Format
type TtTransaction struct {
	Data     string
	GasLimit string
	GasPrice string
	Nonce    string
	R        string
	S        string
	To       string
	V        string
	Value    string
}

type TransactionTest struct {
	Blocknumber string
	Rlp         string
	Sender      string
	Transaction TtTransaction
}

func RunTransactionTestsWithReader(r io.Reader, skipTests []string) error {
	skipTest := make(map[string]bool, len(skipTests))
	for _, name := range skipTests {
		skipTest[name] = true
	}

	bt := make(map[string]TransactionTest)
	if err := readJson(r, &bt); err != nil {
		return err
	}

	for name, test := range bt {
		// if the test should be skipped, return
		if skipTest[name] {
			glog.Infoln("Skipping transaction test", name)
			return nil
		}
		// test the block
		if err := runTransactionTest(test); err != nil {
			return err
		}
		glog.Infoln("Transaction test passed: ", name)

	}
	return nil
}

func RunTransactionTests(file string, skipTests []string) error {
	tests := make(map[string]TransactionTest)
	if err := readJsonFile(file, &tests); err != nil {
		return err
	}

	if err := runTransactionTests(tests, skipTests); err != nil {
		return err
	}
	return nil
}

func runTransactionTests(tests map[string]TransactionTest, skipTests []string) error {
	skipTest := make(map[string]bool, len(skipTests))
	for _, name := range skipTests {
		skipTest[name] = true
	}

	for name, test := range tests {
		// if the test should be skipped, return
		if skipTest[name] {
			glog.Infoln("Skipping transaction test", name)
			return nil
		}

		// test the block
		if err := runTransactionTest(test); err != nil {
			return fmt.Errorf("%s: %v", name, err)
		}
		glog.Infoln("Transaction test passed: ", name)

	}
	return nil
}

func runTransactionTest(txTest TransactionTest) (err error) {
	tx := new(types.Transaction)
	err = rlp.DecodeBytes(mustConvertBytes(txTest.Rlp), tx)

	if err != nil {
		if txTest.Sender == "" {
			// RLP decoding failed and this is expected (test OK)
			return nil
		} else {
			// RLP decoding failed but is expected to succeed (test FAIL)
			return fmt.Errorf("RLP decoding failed when expected to succeed: %s", err)
		}
	}

	validationError := verifyTxFields(txTest, tx)
	if txTest.Sender == "" {
		if validationError != nil {
			// RLP decoding works but validation should fail (test OK)
			return nil
		} else {
			// RLP decoding works but validation should fail (test FAIL)
			// (this should not be possible but added here for completeness)
			return errors.New("Field validations succeeded but should fail")
		}
	}

	if txTest.Sender != "" {
		if validationError == nil {
			// RLP decoding works and validations pass (test OK)
			return nil
		} else {
			// RLP decoding works and validations pass (test FAIL)
			return fmt.Errorf("Field validations failed after RLP decoding: %s", validationError)
		}
	}
	return errors.New("Should not happen: verify RLP decoding and field validation")
}

func verifyTxFields(txTest TransactionTest, decodedTx *types.Transaction) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			buf := make([]byte, 64<<10)
			buf = buf[:runtime.Stack(buf, false)]
			err = fmt.Errorf("%v\n%s", recovered, buf)
		}
	}()

	var (
		decodedSender common.Address
	)

	chainConfig := &params.ChainConfig{HomesteadBlock: params.MainNetHomesteadBlock}
	if chainConfig.IsHomestead(common.String2Big(txTest.Blocknumber)) {
		decodedSender, err = decodedTx.From()
	} else {
		decodedSender, err = decodedTx.FromFrontier()
	}
	if err != nil {
		return err
	}

	expectedSender := mustConvertAddress(txTest.Sender)
	if expectedSender != decodedSender {
		return fmt.Errorf("Sender mismatch: %v %v", expectedSender, decodedSender)
	}

	expectedData := mustConvertBytes(txTest.Transaction.Data)
	if !bytes.Equal(expectedData, decodedTx.Data()) {
		return fmt.Errorf("Tx input data mismatch: %#v %#v", expectedData, decodedTx.Data())
	}

	expectedGasLimit := mustConvertBigInt(txTest.Transaction.GasLimit, 16)
	if expectedGasLimit.Cmp(decodedTx.Gas()) != 0 {
		return fmt.Errorf("GasLimit mismatch: %v %v", expectedGasLimit, decodedTx.Gas())
	}

	expectedGasPrice := mustConvertBigInt(txTest.Transaction.GasPrice, 16)
	if expectedGasPrice.Cmp(decodedTx.GasPrice()) != 0 {
		return fmt.Errorf("GasPrice mismatch: %v %v", expectedGasPrice, decodedTx.GasPrice())
	}

	expectedNonce := mustConvertUint(txTest.Transaction.Nonce, 16)
	if expectedNonce != decodedTx.Nonce() {
		return fmt.Errorf("Nonce mismatch: %v %v", expectedNonce, decodedTx.Nonce())
	}

	v, r, s := decodedTx.SignatureValues()
	expectedR := mustConvertBigInt(txTest.Transaction.R, 16)
	if r.Cmp(expectedR) != 0 {
		return fmt.Errorf("R mismatch: %v %v", expectedR, r)
	}
	expectedS := mustConvertBigInt(txTest.Transaction.S, 16)
	if s.Cmp(expectedS) != 0 {
		return fmt.Errorf("S mismatch: %v %v", expectedS, s)
	}
	expectedV := mustConvertUint(txTest.Transaction.V, 16)
	if uint64(v) != expectedV {
		return fmt.Errorf("V mismatch: %v %v", expectedV, v)
	}

	expectedTo := mustConvertAddress(txTest.Transaction.To)
	if decodedTx.To() == nil {
		if expectedTo != common.BytesToAddress([]byte{}) { // "empty" or "zero" address
			return fmt.Errorf("To mismatch when recipient is nil (contract creation): %v", expectedTo)
		}
	} else {
		if expectedTo != *decodedTx.To() {
			return fmt.Errorf("To mismatch: %v %v", expectedTo, *decodedTx.To())
		}
	}

	expectedValue := mustConvertBigInt(txTest.Transaction.Value, 16)
	if expectedValue.Cmp(decodedTx.Value()) != 0 {
		return fmt.Errorf("Value mismatch: %v %v", expectedValue, decodedTx.Value())
	}

	return nil
}
