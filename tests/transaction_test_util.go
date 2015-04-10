package tests

import (
	"bytes"
	"fmt"
	"math/big"
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

func RunTransactionTests(file string, notWorking  map[string]bool) error {
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
	expectedSender, expectedTo, expectedData, rlpBytes, expectedGasLimit, expectedGasPrice, expectedValue, expectedR, expectedS, expectedNonce, expectedV, err := convertTestTypes(txTest)

	if err != nil {
		if txTest.Sender == "" { // tx is invalid and this is expected (test OK)
			return nil
		} else {
			return err // tx is invalid and this is NOT expected (test FAIL)
		}
	}
	tx := new(types.Transaction)
	rlp.DecodeBytes(rlpBytes, tx)

	sender, err := tx.From()
	if err != nil {
		return err
	}

	if expectedSender != sender {
		return fmt.Errorf("Sender mismatch: %v %v", expectedSender, sender)
	}
	if !bytes.Equal(expectedData, tx.Payload) {
		return fmt.Errorf("Tx input data mismatch: %#v %#v", expectedData, tx.Payload)
	}
	if expectedGasLimit.Cmp(tx.GasLimit) != 0 {
		return fmt.Errorf("GasLimit mismatch: %v %v", expectedGasLimit, tx.GasLimit)
	}
	if expectedGasPrice.Cmp(tx.Price) != 0 {
		return fmt.Errorf("GasPrice mismatch: %v %v", expectedGasPrice, tx.Price)
	}
	if expectedNonce != tx.AccountNonce {
		return fmt.Errorf("Nonce mismatch: %v %v", expectedNonce, tx.AccountNonce)
	}
	if expectedR.Cmp(tx.R) != 0 {
		return fmt.Errorf("R mismatch: %v %v", expectedR, tx.R)
	}
	if expectedS.Cmp(tx.S) != 0 {
		return fmt.Errorf("S mismatch: %v %v", expectedS, tx.S)
	}
	if expectedV != uint64(tx.V) {
		return fmt.Errorf("V mismatch: %v %v", expectedV, uint64(tx.V))
	}
	if expectedTo != *tx.Recipient {
		return fmt.Errorf("To mismatch: %v %v", expectedTo, *tx.Recipient)
	}
	if expectedValue.Cmp(tx.Amount) != 0 {
		return fmt.Errorf("Value mismatch: %v %v", expectedValue, tx.Amount)
	}

	return nil
}

func convertTestTypes(txTest TransactionTest) (sender, to common.Address,
	txInputData, rlpBytes []byte,
	gasLimit, gasPrice, value, r, s *big.Int,
	nonce, v uint64,
	err error) {

	defer func() {
		if recovered := recover(); recovered != nil {
			buf := make([]byte, 64<<10)
			buf = buf[:runtime.Stack(buf, false)]
			err = fmt.Errorf("%v\n%s", recovered, buf)
		}
	}()

	sender = mustConvertAddress(txTest.Sender)
	to = mustConvertAddress(txTest.Transaction.To)

	txInputData = mustConvertBytes(txTest.Transaction.Data)
	rlpBytes = mustConvertBytes(txTest.Rlp)

	gasLimit = mustConvertBigIntHex(txTest.Transaction.GasLimit)
	gasPrice = mustConvertBigIntHex(txTest.Transaction.GasPrice)
	value = mustConvertBigIntHex(txTest.Transaction.Value)

	r = common.Bytes2Big(mustConvertBytes(txTest.Transaction.R))
	s = common.Bytes2Big(mustConvertBytes(txTest.Transaction.S))

	nonce = mustConvertUintHex(txTest.Transaction.Nonce)
	v = mustConvertUintHex(txTest.Transaction.V)

	return sender, to, txInputData, rlpBytes, gasLimit, gasPrice, value, r, s, nonce, v, nil
}
