package tests

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	Rlp         string
	Sender      string
	Transaction TtTransaction
}

func RunTransactionTests(file string, notWorking map[string]bool) error {
	bt := make(map[string]TransactionTest)
	if err := LoadJSON(file, &bt); err != nil {
		return err
	}
	for name, in := range bt {
		var err error
		// TODO: remove this, we currently ignore some tests which are broken
		if !notWorking[name] {
			if err = runTest(in); err != nil {
				return fmt.Errorf("bad test %s: %v", name, err)
			}
			fmt.Println("Test passed:", name)
		}
	}
	return nil
}

func runTest(txTest TransactionTest) (err error) {
	tx := new(types.Transaction)
	err = rlp.DecodeBytes(mustConvertBytes(txTest.Rlp), tx)

	if err != nil {
		if txTest.Sender == "" {
			// RLP decoding failed and this is expected (test OK)
			return nil
		} else {
			// RLP decoding failed but is expected to succeed (test FAIL)
			return fmt.Errorf("RLP decoding failed when expected to succeed: ", err)
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
			return fmt.Errorf("Field validations failed after RLP decoding: ", validationError)
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

	decodedSender, err := decodedTx.From()
	if err != nil {
		return err
	}

	expectedSender := mustConvertAddress(txTest.Sender)
	if expectedSender != decodedSender {
		return fmt.Errorf("Sender mismatch: %v %v", expectedSender, decodedSender)
	}

	expectedData := mustConvertBytes(txTest.Transaction.Data)
	if !bytes.Equal(expectedData, decodedTx.Payload) {
		return fmt.Errorf("Tx input data mismatch: %#v %#v", expectedData, decodedTx.Payload)
	}

	expectedGasLimit := mustConvertBigInt(txTest.Transaction.GasLimit, 16)
	if expectedGasLimit.Cmp(decodedTx.GasLimit) != 0 {
		return fmt.Errorf("GasLimit mismatch: %v %v", expectedGasLimit, decodedTx.GasLimit)
	}

	expectedGasPrice := mustConvertBigInt(txTest.Transaction.GasPrice, 16)
	if expectedGasPrice.Cmp(decodedTx.Price) != 0 {
		return fmt.Errorf("GasPrice mismatch: %v %v", expectedGasPrice, decodedTx.Price)
	}

	expectedNonce := mustConvertUint(txTest.Transaction.Nonce, 16)
	if expectedNonce != decodedTx.AccountNonce {
		return fmt.Errorf("Nonce mismatch: %v %v", expectedNonce, decodedTx.AccountNonce)
	}

	expectedR := common.Bytes2Big(mustConvertBytes(txTest.Transaction.R))
	if expectedR.Cmp(decodedTx.R) != 0 {
		return fmt.Errorf("R mismatch: %v %v", expectedR, decodedTx.R)
	}

	expectedS := common.Bytes2Big(mustConvertBytes(txTest.Transaction.S))
	if expectedS.Cmp(decodedTx.S) != 0 {
		return fmt.Errorf("S mismatch: %v %v", expectedS, decodedTx.S)
	}

	expectedV := mustConvertUint(txTest.Transaction.V, 16)
	if expectedV != uint64(decodedTx.V) {
		return fmt.Errorf("V mismatch: %v %v", expectedV, uint64(decodedTx.V))
	}

	expectedTo := mustConvertAddress(txTest.Transaction.To)
	if decodedTx.Recipient == nil {
		if expectedTo != common.BytesToAddress([]byte{}) { // "empty" or "zero" address
			return fmt.Errorf("To mismatch when recipient is nil (contract creation): %v", expectedTo)
		}
	} else {
		if expectedTo != *decodedTx.Recipient {
			return fmt.Errorf("To mismatch: %v %v", expectedTo, *decodedTx.Recipient)
		}
	}

	expectedValue := mustConvertBigInt(txTest.Transaction.Value, 16)
	if expectedValue.Cmp(decodedTx.Amount) != 0 {
		return fmt.Errorf("Value mismatch: %v %v", expectedValue, decodedTx.Amount)
	}

	return nil
}
