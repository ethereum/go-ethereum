// Copyright 2018 The go-ethereum Authors
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

package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// The validation package contains validation checks for transactions
// - ABI-data validation
// - Transaction semantics validation
// The package provides warnings for typical pitfalls

func (vs *ValidationMessages) crit(msg string) {
	vs.Messages = append(vs.Messages, ValidationInfo{"CRITICAL", msg})
}
func (vs *ValidationMessages) warn(msg string) {
	vs.Messages = append(vs.Messages, ValidationInfo{"WARNING", msg})
}
func (vs *ValidationMessages) info(msg string) {
	vs.Messages = append(vs.Messages, ValidationInfo{"Info", msg})
}

type Validator struct {
	db *AbiDb
}

func NewValidator(db *AbiDb) *Validator {
	return &Validator{db}
}
func testSelector(selector string, data []byte) (*decodedCallData, error) {
	if selector == "" {
		return nil, fmt.Errorf("selector not found")
	}
	abiData, err := MethodSelectorToAbi(selector)
	if err != nil {
		return nil, err
	}
	info, err := parseCallData(data, string(abiData))
	if err != nil {
		return nil, err
	}
	return info, nil

}

// validateCallData checks if the ABI-data + methodselector (if given) can be parsed and seems to match
func (v *Validator) validateCallData(msgs *ValidationMessages, data []byte, methodSelector *string) {
	if len(data) == 0 {
		return
	}
	if len(data) < 4 {
		msgs.warn("Tx contains data which is not valid ABI")
		return
	}
	var (
		info *decodedCallData
		err  error
	)
	// Check the provided one
	if methodSelector != nil {
		info, err = testSelector(*methodSelector, data)
		if err != nil {
			msgs.warn(fmt.Sprintf("Tx contains data, but provided ABI signature could not be matched: %v", err))
		} else {
			msgs.info(info.String())
			//Successfull match. add to db if not there already (ignore errors there)
			v.db.AddSignature(*methodSelector, data[:4])
		}
		return
	}
	// Check the db
	selector, err := v.db.LookupMethodSelector(data[:4])
	if err != nil {
		msgs.warn(fmt.Sprintf("Tx contains data, but the ABI signature could not be found: %v", err))
		return
	}
	info, err = testSelector(selector, data)
	if err != nil {
		msgs.warn(fmt.Sprintf("Tx contains data, but provided ABI signature could not be matched: %v", err))
	} else {
		msgs.info(info.String())
	}
}

// validateSemantics checks if the transactions 'makes sense', and generate warnings for a couple of typical scenarios
func (v *Validator) validate(msgs *ValidationMessages, txargs *SendTxArgs, methodSelector *string) error {
	// Prevent accidental erroneous usage of both 'input' and 'data'
	if txargs.Data != nil && txargs.Input != nil && !bytes.Equal(*txargs.Data, *txargs.Input) {
		// This is a showstopper
		return errors.New(`Ambiguous request: both "data" and "input" are set and are not identical`)
	}
	var (
		data []byte
	)
	// Place data on 'data', and nil 'input'
	if txargs.Input != nil {
		txargs.Data = txargs.Input
		txargs.Input = nil
	}
	if txargs.Data != nil {
		data = *txargs.Data
	}

	if txargs.To == nil {
		//Contract creation should contain sufficient data to deploy a contract
		// A typical error is omitting sender due to some quirk in the javascript call
		// e.g. https://github.com/ethereum/go-ethereum/issues/16106
		if len(data) == 0 {
			if txargs.Value.ToInt().Cmp(big.NewInt(0)) > 0 {
				// Sending ether into black hole
				return errors.New("Tx will create contract with value but empty code!")
			}
			// No value submitted at least
			msgs.crit("Tx will create contract with empty code!")
		} else if len(data) < 40 { //Arbitrary limit
			msgs.warn(fmt.Sprintf("Tx will will create contract, but payload is suspiciously small (%d b)", len(data)))
		}
		// methodSelector should be nil for contract creation
		if methodSelector != nil {
			msgs.warn("Tx will create contract, but method selector supplied; indicating intent to call a method.")
		}

	} else {
		if !txargs.To.ValidChecksum() {
			msgs.warn("Invalid checksum on to-address")
		}
		// Normal transaction
		if bytes.Equal(txargs.To.Address().Bytes(), common.Address{}.Bytes()) {
			// Sending to 0
			msgs.crit("Tx destination is the zero address!")
		}
		// Validate calldata
		v.validateCallData(msgs, data, methodSelector)
	}
	return nil
}

// ValidateTransaction does a number of checks on the supplied transaction, and returns either a list of warnings,
// or an error, indicating that the transaction should be immediately rejected
func (v *Validator) ValidateTransaction(txArgs *SendTxArgs, methodSelector *string) (*ValidationMessages, error) {
	msgs := &ValidationMessages{}
	return msgs, v.validate(msgs, txArgs, methodSelector)
}
