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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestTransaction(t *testing.T) {
	t.Parallel()

	txt := new(testMatcher)
	// We don't allow more than uint64 in gas amount
	// This is a pseudo-consensus vulnerability, but not in practice
	// because of the gas limit
	txt.skipLoad("^ttGasLimit/TransactionWithGasLimitxPriceOverflow.json")
	// We _do_ allow more than uint64 in gas price, as opposed to the tests
	// This is also not a concern, as long as tx.Cost() uses big.Int for
	// calculating the final cost
	txt.skipLoad("^ttGasPrice/TransactionWithGasPriceOverflow.json")

	// The maximum value of nonce is 2^64 - 1
	txt.skipLoad("^ttNonce/TransactionWithHighNonce64Minus1.json")

	// The value is larger than uint64, which according to the test is invalid.
	// Geth accepts it, which is not a consensus issue since we use big.Int's
	// internally to calculate the cost
	txt.skipLoad("^ttValue/TransactionWithHighValueOverflow.json")

	// The size of a create tx's initcode is only checked during the state
	// transition
	txt.skipLoad("^ttEIP3860/DataTestInitCodeTooBig.json")

	// The following tests require the tx precheck to be performed
	// TODO(s1na): expose stateTransition.precheck publicly to be able to run these tests
	txt.skipLoad("^ttEIP1559/maxPriorityFeePerGass32BytesValue.json")
	txt.skipLoad("^ttEIP1559/maxPriorityFeePerGasOverflow.json")
	txt.skipLoad("^ttEIP1559/maxFeePerGas32BytesValue.json")
	txt.skipLoad("^ttEIP1559/maxFeePerGasOverflow.json")
	txt.skipLoad("^ttEIP1559/GasLimitPriceProductPlusOneOverflow.json")
	txt.skipLoad("^ttEIP1559/GasLimitPriceProductOverflow.json")

	txt.walk(t, transactionTestDir, func(t *testing.T, name string, test *TransactionTest) {
		if err := txt.checkFailure(t, test.Run()); err != nil {
			t.Error(err)
		}
	})
}

func TestExecutionSpecTransaction(t *testing.T) {
	if !common.FileExist(executionSpecStateTestDir) {
		t.Skipf("directory %s does not exist", executionSpecStateTestDir)
	}
	st := new(testMatcher)

	// Emptiness of authorization list is only validated during the tx precheck
	st.skipLoad("^prague/eip7702_set_code_tx/invalid_tx/empty_authorization_list.json")

	st.walk(t, executionSpecTransactionTestDir, func(t *testing.T, name string, test *TransactionTest) {
		if err := st.checkFailure(t, test.Run()); err != nil {
			t.Error(err)
		}
	})
}
