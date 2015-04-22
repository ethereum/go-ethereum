package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
)

func TestBlockheightInvalidString(t *testing.T) {
	v := "foo"
	var num int64

	str := ExpectInvalidTypeError(blockHeight(v, &num))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockheightEarliest(t *testing.T) {
	v := "earliest"
	e := int64(0)
	var num int64

	err := blockHeight(v, &num)
	if err != nil {
		t.Error(err)
	}

	if num != e {
		t.Errorf("Expected %s but got %s", e, num)
	}
}

func TestBlockheightLatest(t *testing.T) {
	v := "latest"
	e := int64(-1)
	var num int64

	err := blockHeight(v, &num)
	if err != nil {
		t.Error(err)
	}

	if num != e {
		t.Errorf("Expected %s but got %s", e, num)
	}
}

func TestBlockheightPending(t *testing.T) {
	v := "pending"
	e := int64(-2)
	var num int64

	err := blockHeight(v, &num)
	if err != nil {
		t.Error(err)
	}

	if num != e {
		t.Errorf("Expected %s but got %s", e, num)
	}
}

func ExpectValidationError(err error) string {
	var str string
	switch err.(type) {
	case nil:
		str = "Expected error but didn't get one"
	case *ValidationError:
		break
	default:
		str = fmt.Sprintf("Expected *rpc.ValidationError but got %T with message `%s`", err, err.Error())
	}
	return str
}

func ExpectInvalidTypeError(err error) string {
	var str string
	switch err.(type) {
	case nil:
		str = "Expected error but didn't get one"
	case *InvalidTypeError:
		break
	default:
		str = fmt.Sprintf("Expected *rpc.InvalidTypeError but got %T with message `%s`", err, err.Error())
	}
	return str
}

func ExpectInsufficientParamsError(err error) string {
	var str string
	switch err.(type) {
	case nil:
		str = "Expected error but didn't get one"
	case *InsufficientParamsError:
		break
	default:
		str = fmt.Sprintf("Expected *rpc.InsufficientParamsError but got %T with message %s", err, err.Error())
	}
	return str
}

func ExpectDecodeParamError(err error) string {
	var str string
	switch err.(type) {
	case nil:
		str = "Expected error but didn't get one"
	case *DecodeParamError:
		break
	default:
		str = fmt.Sprintf("Expected *rpc.DecodeParamError but got %T with message `%s`", err, err.Error())
	}
	return str
}

func TestSha3(t *testing.T) {
	input := `["0x68656c6c6f20776f726c64"]`
	expected := "0x68656c6c6f20776f726c64"

	args := new(Sha3Args)
	json.Unmarshal([]byte(input), &args)

	if args.Data != expected {
		t.Error("got %s expected %s", input, expected)
	}
}

func TestSha3ArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(Sha3Args)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestSha3ArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(Sha3Args)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}
func TestSha3ArgsDataInvalid(t *testing.T) {
	input := `[4]`

	args := new(Sha3Args)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBalanceArgs(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "0x1f"]`
	expected := new(GetBalanceArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.BlockNumber = 31

	args := new(GetBalanceArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if args.Address != expected.Address {
		t.Errorf("Address should be %v but is %v", expected.Address, args.Address)
	}

	if args.BlockNumber != expected.BlockNumber {
		t.Errorf("BlockNumber should be %v but is %v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetBalanceArgsBlocknumMissing(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1"]`
	expected := new(GetBalanceArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.BlockNumber = -1

	args := new(GetBalanceArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if args.Address != expected.Address {
		t.Errorf("Address should be %v but is %v", expected.Address, args.Address)
	}

	if args.BlockNumber != expected.BlockNumber {
		t.Errorf("BlockNumber should be %v but is %v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetBalanceArgsLatest(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "latest"]`
	expected := new(GetBalanceArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.BlockNumber = -1

	args := new(GetBalanceArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if args.Address != expected.Address {
		t.Errorf("Address should be %v but is %v", expected.Address, args.Address)
	}

	if args.BlockNumber != expected.BlockNumber {
		t.Errorf("BlockNumber should be %v but is %v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetBalanceArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(GetBalanceArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBalanceArgsInvalid(t *testing.T) {
	input := `6`

	args := new(GetBalanceArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBalanceArgsBlockInvalid(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", false]`

	args := new(GetBalanceArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBalanceArgsAddressInvalid(t *testing.T) {
	input := `[-9, "latest"]`

	args := new(GetBalanceArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBlockByHashArgs(t *testing.T) {
	input := `["0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331", true]`
	expected := new(GetBlockByHashArgs)
	expected.BlockHash = "0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331"
	expected.IncludeTxs = true

	args := new(GetBlockByHashArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if args.BlockHash != expected.BlockHash {
		t.Errorf("BlockHash should be %v but is %v", expected.BlockHash, args.BlockHash)
	}

	if args.IncludeTxs != expected.IncludeTxs {
		t.Errorf("IncludeTxs should be %v but is %v", expected.IncludeTxs, args.IncludeTxs)
	}
}

func TestGetBlockByHashArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(GetBlockByHashArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBlockByHashArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(GetBlockByHashArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBlockByHashArgsHashInt(t *testing.T) {
	input := `[8]`

	args := new(GetBlockByHashArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBlockByHashArgsHashBool(t *testing.T) {
	input := `[false, true]`

	args := new(GetBlockByHashArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBlockByNumberArgsBlockNum(t *testing.T) {
	input := `[436, false]`
	expected := new(GetBlockByNumberArgs)
	expected.BlockNumber = 436
	expected.IncludeTxs = false

	args := new(GetBlockByNumberArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if args.BlockNumber != expected.BlockNumber {
		t.Errorf("BlockNumber should be %v but is %v", expected.BlockNumber, args.BlockNumber)
	}

	if args.IncludeTxs != expected.IncludeTxs {
		t.Errorf("IncludeTxs should be %v but is %v", expected.IncludeTxs, args.IncludeTxs)
	}
}

func TestGetBlockByNumberArgsBlockHex(t *testing.T) {
	input := `["0x1b4", false]`
	expected := new(GetBlockByNumberArgs)
	expected.BlockNumber = 436
	expected.IncludeTxs = false

	args := new(GetBlockByNumberArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if args.BlockNumber != expected.BlockNumber {
		t.Errorf("BlockNumber should be %v but is %v", expected.BlockNumber, args.BlockNumber)
	}

	if args.IncludeTxs != expected.IncludeTxs {
		t.Errorf("IncludeTxs should be %v but is %v", expected.IncludeTxs, args.IncludeTxs)
	}
}
func TestGetBlockByNumberArgsWords(t *testing.T) {
	input := `["earliest", true]`
	expected := new(GetBlockByNumberArgs)
	expected.BlockNumber = 0
	expected.IncludeTxs = true

	args := new(GetBlockByNumberArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if args.BlockNumber != expected.BlockNumber {
		t.Errorf("BlockNumber should be %v but is %v", expected.BlockNumber, args.BlockNumber)
	}

	if args.IncludeTxs != expected.IncludeTxs {
		t.Errorf("IncludeTxs should be %v but is %v", expected.IncludeTxs, args.IncludeTxs)
	}
}

func TestGetBlockByNumberEmpty(t *testing.T) {
	input := `[]`

	args := new(GetBlockByNumberArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBlockByNumberShort(t *testing.T) {
	input := `["0xbbb"]`

	args := new(GetBlockByNumberArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetBlockByNumberBool(t *testing.T) {
	input := `[true, true]`

	args := new(GetBlockByNumberArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}
func TestGetBlockByNumberBlockObject(t *testing.T) {
	input := `{}`

	args := new(GetBlockByNumberArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestNewTxArgs(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": "0x9184e72a000",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"},
  "0x10"]`
	expected := new(NewTxArgs)
	expected.From = "0xb60e8dd61c5d32be8058bb8eb970870f07233155"
	expected.To = "0xd46e8dd67c5d32be8058bb8eb970870f072445675"
	expected.Gas = big.NewInt(30400)
	expected.GasPrice = big.NewInt(10000000000000)
	expected.Value = big.NewInt(10000000000000)
	expected.Data = "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
	expected.BlockNumber = big.NewInt(16).Int64()

	args := new(NewTxArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.From != args.From {
		t.Errorf("From shoud be %#v but is %#v", expected.From, args.From)
	}

	if expected.To != args.To {
		t.Errorf("To shoud be %#v but is %#v", expected.To, args.To)
	}

	if bytes.Compare(expected.Gas.Bytes(), args.Gas.Bytes()) != 0 {
		t.Errorf("Gas shoud be %#v but is %#v", expected.Gas.Bytes(), args.Gas.Bytes())
	}

	if bytes.Compare(expected.GasPrice.Bytes(), args.GasPrice.Bytes()) != 0 {
		t.Errorf("GasPrice shoud be %#v but is %#v", expected.GasPrice, args.GasPrice)
	}

	if bytes.Compare(expected.Value.Bytes(), args.Value.Bytes()) != 0 {
		t.Errorf("Value shoud be %#v but is %#v", expected.Value, args.Value)
	}

	if expected.Data != args.Data {
		t.Errorf("Data shoud be %#v but is %#v", expected.Data, args.Data)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestNewTxArgsInt(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": 100,
  "gasPrice": 50,
  "value": 8765456789,
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"},
  5]`
	expected := new(NewTxArgs)
	expected.Gas = big.NewInt(100)
	expected.GasPrice = big.NewInt(50)
	expected.Value = big.NewInt(8765456789)
	expected.BlockNumber = int64(5)

	args := new(NewTxArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if bytes.Compare(expected.Gas.Bytes(), args.Gas.Bytes()) != 0 {
		t.Errorf("Gas shoud be %v but is %v", expected.Gas, args.Gas)
	}

	if bytes.Compare(expected.GasPrice.Bytes(), args.GasPrice.Bytes()) != 0 {
		t.Errorf("GasPrice shoud be %v but is %v", expected.GasPrice, args.GasPrice)
	}

	if bytes.Compare(expected.Value.Bytes(), args.Value.Bytes()) != 0 {
		t.Errorf("Value shoud be %v but is %v", expected.Value, args.Value)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %v but is %v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestNewTxArgsBlockBool(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": "0x9184e72a000",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"},
  false]`

	args := new(NewTxArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestNewTxArgsGasInvalid(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": false,
  "gasPrice": "0x9184e72a000",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
  }]`

	args := new(NewTxArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestNewTxArgsGaspriceInvalid(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": false,
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
  }]`

	args := new(NewTxArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestNewTxArgsValueInvalid(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": "0x9184e72a000",
  "value": false,
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
	}]`

	args := new(NewTxArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestNewTxArgsGasMissing(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gasPrice": "0x9184e72a000",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
  }]`
	expected := new(NewTxArgs)
	expected.Gas = big.NewInt(0)

	args := new(NewTxArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if bytes.Compare(expected.Gas.Bytes(), args.Gas.Bytes()) != 0 {
		t.Errorf("Gas shoud be %v but is %v", expected.Gas, args.Gas)
	}
}

func TestNewTxArgsBlockGaspriceMissing(t *testing.T) {
	input := `[{
	"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
  }]`
	expected := new(NewTxArgs)
	expected.GasPrice = big.NewInt(0)

	args := new(NewTxArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if bytes.Compare(expected.GasPrice.Bytes(), args.GasPrice.Bytes()) != 0 {
		t.Errorf("GasPrice shoud be %v but is %v", expected.GasPrice, args.GasPrice)
	}

}

func TestNewTxArgsValueMissing(t *testing.T) {
	input := `[{
	"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
	}]`
	expected := new(NewTxArgs)
	expected.Value = big.NewInt(0)

	args := new(NewTxArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if bytes.Compare(expected.Value.Bytes(), args.Value.Bytes()) != 0 {
		t.Errorf("Value shoud be %v but is %v", expected.Value, args.Value)
	}

}

func TestNewTxArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(NewTxArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestNewTxArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(NewTxArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}
func TestNewTxArgsNotStrings(t *testing.T) {
	input := `[{"from":6}]`

	args := new(NewTxArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestNewTxArgsFromEmpty(t *testing.T) {
	input := `[{"to": "0xb60e8dd61c5d32be8058bb8eb970870f07233155"}]`

	args := new(NewTxArgs)
	str := ExpectValidationError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCallArgs(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": "0x9184e72a000",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"},
  "0x10"]`
	expected := new(CallArgs)
	expected.From = "0xb60e8dd61c5d32be8058bb8eb970870f07233155"
	expected.To = "0xd46e8dd67c5d32be8058bb8eb970870f072445675"
	expected.Gas = big.NewInt(30400)
	expected.GasPrice = big.NewInt(10000000000000)
	expected.Value = big.NewInt(10000000000000)
	expected.Data = "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
	expected.BlockNumber = big.NewInt(16).Int64()

	args := new(CallArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.To != args.To {
		t.Errorf("To shoud be %#v but is %#v", expected.To, args.To)
	}

	if bytes.Compare(expected.Gas.Bytes(), args.Gas.Bytes()) != 0 {
		t.Errorf("Gas shoud be %#v but is %#v", expected.Gas.Bytes(), args.Gas.Bytes())
	}

	if bytes.Compare(expected.GasPrice.Bytes(), args.GasPrice.Bytes()) != 0 {
		t.Errorf("GasPrice shoud be %#v but is %#v", expected.GasPrice, args.GasPrice)
	}

	if bytes.Compare(expected.Value.Bytes(), args.Value.Bytes()) != 0 {
		t.Errorf("Value shoud be %#v but is %#v", expected.Value, args.Value)
	}

	if expected.Data != args.Data {
		t.Errorf("Data shoud be %#v but is %#v", expected.Data, args.Data)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestCallArgsInt(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": 100,
  "gasPrice": 50,
  "value": 8765456789,
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"},
  5]`
	expected := new(CallArgs)
	expected.Gas = big.NewInt(100)
	expected.GasPrice = big.NewInt(50)
	expected.Value = big.NewInt(8765456789)
	expected.BlockNumber = int64(5)

	args := new(CallArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if bytes.Compare(expected.Gas.Bytes(), args.Gas.Bytes()) != 0 {
		t.Errorf("Gas shoud be %v but is %v", expected.Gas, args.Gas)
	}

	if bytes.Compare(expected.GasPrice.Bytes(), args.GasPrice.Bytes()) != 0 {
		t.Errorf("GasPrice shoud be %v but is %v", expected.GasPrice, args.GasPrice)
	}

	if bytes.Compare(expected.Value.Bytes(), args.Value.Bytes()) != 0 {
		t.Errorf("Value shoud be %v but is %v", expected.Value, args.Value)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %v but is %v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestCallArgsBlockBool(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": "0x9184e72a000",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"},
  false]`

	args := new(CallArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCallArgsGasInvalid(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": false,
  "gasPrice": "0x9184e72a000",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
  }]`

	args := new(CallArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCallArgsGaspriceInvalid(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": false,
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
  }]`

	args := new(CallArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCallArgsValueInvalid(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": "0x9184e72a000",
  "value": false,
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
	}]`

	args := new(CallArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCallArgsGasMissing(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gasPrice": "0x9184e72a000",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
  }]`

	args := new(CallArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	expected := new(CallArgs)
	expected.Gas = big.NewInt(0)

	if bytes.Compare(expected.Gas.Bytes(), args.Gas.Bytes()) != 0 {
		t.Errorf("Gas shoud be %v but is %v", expected.Gas, args.Gas)
	}

}

func TestCallArgsBlockGaspriceMissing(t *testing.T) {
	input := `[{
	"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "value": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
  }]`

	args := new(CallArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	expected := new(CallArgs)
	expected.GasPrice = big.NewInt(0)

	if bytes.Compare(expected.GasPrice.Bytes(), args.GasPrice.Bytes()) != 0 {
		t.Errorf("GasPrice shoud be %v but is %v", expected.GasPrice, args.GasPrice)
	}
}

func TestCallArgsValueMissing(t *testing.T) {
	input := `[{
	"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
  "to": "0xd46e8dd67c5d32be8058bb8eb970870f072445675",
  "gas": "0x76c0",
  "gasPrice": "0x9184e72a000",
  "data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"
	}]`

	args := new(CallArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	expected := new(CallArgs)
	expected.Value = big.NewInt(int64(0))

	if bytes.Compare(expected.Value.Bytes(), args.Value.Bytes()) != 0 {
		t.Errorf("GasPrice shoud be %v but is %v", expected.Value, args.Value)
	}
}

func TestCallArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(CallArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCallArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(CallArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}
func TestCallArgsNotStrings(t *testing.T) {
	input := `[{"from":6}]`

	args := new(CallArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCallArgsToEmpty(t *testing.T) {
	input := `[{"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155"}]`
	args := new(CallArgs)
	str := ExpectValidationError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageArgs(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "latest"]`
	expected := new(GetStorageArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.BlockNumber = -1

	args := new(GetStorageArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Address != args.Address {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetStorageArgsMissingBlocknum(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1"]`
	expected := new(GetStorageArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.BlockNumber = -1

	args := new(GetStorageArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Address != args.Address {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetStorageInvalidArgs(t *testing.T) {
	input := `{}`

	args := new(GetStorageArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageInvalidBlockheight(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", {}]`

	args := new(GetStorageArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageEmptyArgs(t *testing.T) {
	input := `[]`

	args := new(GetStorageArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageAddressInt(t *testing.T) {
	input := `[32456785432456, "latest"]`

	args := new(GetStorageArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageAtArgs(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "0x0", "0x2"]`
	expected := new(GetStorageAtArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.Key = "0x0"
	expected.BlockNumber = 2

	args := new(GetStorageAtArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Address != args.Address {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.Key != args.Key {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetStorageAtArgsMissingBlocknum(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "0x0"]`
	expected := new(GetStorageAtArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.Key = "0x0"
	expected.BlockNumber = -1

	args := new(GetStorageAtArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Address != args.Address {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.Key != args.Key {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetStorageAtEmptyArgs(t *testing.T) {
	input := `[]`

	args := new(GetStorageAtArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageAtArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(GetStorageAtArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageAtArgsAddressNotString(t *testing.T) {
	input := `[true, "0x0", "0x2"]`

	args := new(GetStorageAtArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageAtArgsKeyNotString(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", true, "0x2"]`

	args := new(GetStorageAtArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetStorageAtArgsValueNotString(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "0x1", true]`

	args := new(GetStorageAtArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetTxCountArgs(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "pending"]`
	expected := new(GetTxCountArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.BlockNumber = -2

	args := new(GetTxCountArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Address != args.Address {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetTxCountEmptyArgs(t *testing.T) {
	input := `[]`

	args := new(GetTxCountArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetTxCountEmptyArgsInvalid(t *testing.T) {
	input := `false`

	args := new(GetTxCountArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetTxCountAddressNotString(t *testing.T) {
	input := `[false, "pending"]`

	args := new(GetTxCountArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetTxCountBlockheightMissing(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1"]`
	expected := new(GetTxCountArgs)
	expected.Address = "0x407d73d8a49eeb85d32cf465507dd71d507100c1"
	expected.BlockNumber = -1

	args := new(GetTxCountArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Address != args.Address {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetTxCountBlockheightInvalid(t *testing.T) {
	input := `["0x407d73d8a49eeb85d32cf465507dd71d507100c1", {}]`

	args := new(GetTxCountArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetDataArgs(t *testing.T) {
	input := `["0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8", "latest"]`
	expected := new(GetDataArgs)
	expected.Address = "0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8"
	expected.BlockNumber = -1

	args := new(GetDataArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Address != args.Address {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetDataArgsBlocknumMissing(t *testing.T) {
	input := `["0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8"]`
	expected := new(GetDataArgs)
	expected.Address = "0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8"
	expected.BlockNumber = -1

	args := new(GetDataArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Address != args.Address {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestGetDataArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(GetDataArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetDataArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(GetDataArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetDataArgsAddressNotString(t *testing.T) {
	input := `[12, "latest"]`

	args := new(GetDataArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestGetDataArgsBlocknumberNotString(t *testing.T) {
	input := `["0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8", false]`

	args := new(GetDataArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgs(t *testing.T) {
	input := `[{
  "fromBlock": "0x1",
  "toBlock": "0x2",
  "limit": "0x3",
  "offset": "0x0",
  "address": "0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8",
  "topics":
  [
  	["0xAA", "0xBB"],
  	["0xCC", "0xDD"]
  ]
  }]`

	expected := new(BlockFilterArgs)
	expected.Earliest = 1
	expected.Latest = 2
	expected.Max = 3
	expected.Skip = 0
	expected.Address = []string{"0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8"}
	expected.Topics = [][]string{
		[]string{"0xAA", "0xBB"},
		[]string{"0xCC", "0xDD"},
	}

	args := new(BlockFilterArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Earliest != args.Earliest {
		t.Errorf("Earliest shoud be %#v but is %#v", expected.Earliest, args.Earliest)
	}

	if expected.Latest != args.Latest {
		t.Errorf("Latest shoud be %#v but is %#v", expected.Latest, args.Latest)
	}

	if expected.Max != args.Max {
		t.Errorf("Max shoud be %#v but is %#v", expected.Max, args.Max)
	}

	if expected.Skip != args.Skip {
		t.Errorf("Skip shoud be %#v but is %#v", expected.Skip, args.Skip)
	}

	if expected.Address[0] != args.Address[0] {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.Topics[0][0] != args.Topics[0][0] {
		t.Errorf("Topics shoud be %#v but is %#v", expected.Topics, args.Topics)
	}
	if expected.Topics[0][1] != args.Topics[0][1] {
		t.Errorf("Topics shoud be %#v but is %#v", expected.Topics, args.Topics)
	}
	if expected.Topics[1][0] != args.Topics[1][0] {
		t.Errorf("Topics shoud be %#v but is %#v", expected.Topics, args.Topics)
	}
	if expected.Topics[1][1] != args.Topics[1][1] {
		t.Errorf("Topics shoud be %#v but is %#v", expected.Topics, args.Topics)
	}

}

func TestBlockFilterArgsDefaults(t *testing.T) {
	input := `[{
  "address": ["0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8"],
  "topics": ["0xAA","0xBB"]
  }]`
	expected := new(BlockFilterArgs)
	expected.Earliest = -1
	expected.Latest = -1
	expected.Max = 100
	expected.Skip = 0
	expected.Address = []string{"0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8"}
	expected.Topics = [][]string{[]string{"0xAA"}, []string{"0xBB"}}

	args := new(BlockFilterArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Earliest != args.Earliest {
		t.Errorf("Earliest shoud be %#v but is %#v", expected.Earliest, args.Earliest)
	}

	if expected.Latest != args.Latest {
		t.Errorf("Latest shoud be %#v but is %#v", expected.Latest, args.Latest)
	}

	if expected.Max != args.Max {
		t.Errorf("Max shoud be %#v but is %#v", expected.Max, args.Max)
	}

	if expected.Skip != args.Skip {
		t.Errorf("Skip shoud be %#v but is %#v", expected.Skip, args.Skip)
	}

	if expected.Address[0] != args.Address[0] {
		t.Errorf("Address shoud be %#v but is %#v", expected.Address, args.Address)
	}

	if expected.Topics[0][0] != args.Topics[0][0] {
		t.Errorf("Topics shoud be %#v but is %#v", expected.Topics, args.Topics)
	}

	if expected.Topics[1][0] != args.Topics[1][0] {
		t.Errorf("Topics shoud be %#v but is %#v", expected.Topics, args.Topics)
	}
}

func TestBlockFilterArgsWords(t *testing.T) {
	input := `[{
  "fromBlock": "latest",
  "toBlock": "pending"
  }]`
	expected := new(BlockFilterArgs)
	expected.Earliest = -1
	expected.Latest = -2

	args := new(BlockFilterArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Earliest != args.Earliest {
		t.Errorf("Earliest shoud be %#v but is %#v", expected.Earliest, args.Earliest)
	}

	if expected.Latest != args.Latest {
		t.Errorf("Latest shoud be %#v but is %#v", expected.Latest, args.Latest)
	}
}

func TestBlockFilterArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(BlockFilterArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsFromBool(t *testing.T) {
	input := `[{
  "fromBlock": true,
  "toBlock": "pending"
  }]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsToBool(t *testing.T) {
	input := `[{
  "fromBlock": "pending",
  "toBlock": true
  }]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsEmptyArgs(t *testing.T) {
	input := `[]`

	args := new(BlockFilterArgs)
	err := json.Unmarshal([]byte(input), &args)
	if err == nil {
		t.Error("Expected error but didn't get one")
	}
}

func TestBlockFilterArgsLimitInvalid(t *testing.T) {
	input := `[{"limit": false}]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsOffsetInvalid(t *testing.T) {
	input := `[{"offset": true}]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsAddressInt(t *testing.T) {
	input := `[{
  "address": 1,
  "topics": "0x12341234"}]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsAddressSliceInt(t *testing.T) {
	input := `[{
  "address": [1],
  "topics": "0x12341234"}]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsTopicInt(t *testing.T) {
	input := `[{
  "address": ["0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8"],
  "topics": 1}]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsTopicSliceInt(t *testing.T) {
	input := `[{
  "address": "0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8",
  "topics": [1]}]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsTopicSliceInt2(t *testing.T) {
	input := `[{
  "address": "0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8",
  "topics": ["0xAA", [1]]}]`

	args := new(BlockFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockFilterArgsTopicComplex(t *testing.T) {
	input := `[{
	"address": "0xd5677cf67b5aa051bb40496e68ad359eb97cfbf8",
  "topics": ["0xAA", ["0xBB", "0xCC"]]
  }]`

	args := new(BlockFilterArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
		fmt.Printf("%v\n", args)
		return
	}

	if args.Topics[0][0] != "0xAA" {
		t.Errorf("Topic should be %s but is %s", "0xAA", args.Topics[0][0])
	}

	if args.Topics[1][0] != "0xBB" {
		t.Errorf("Topic should be %s but is %s", "0xBB", args.Topics[0][0])
	}

	if args.Topics[1][1] != "0xCC" {
		t.Errorf("Topic should be %s but is %s", "0xCC", args.Topics[0][0])
	}
}

func TestDbArgs(t *testing.T) {
	input := `["testDB","myKey","0xbeef"]`
	expected := new(DbArgs)
	expected.Database = "testDB"
	expected.Key = "myKey"
	expected.Value = []byte("0xbeef")

	args := new(DbArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if err := args.requirements(); err != nil {
		t.Error(err)
	}

	if expected.Database != args.Database {
		t.Errorf("Database shoud be %#v but is %#v", expected.Database, args.Database)
	}

	if expected.Key != args.Key {
		t.Errorf("Key shoud be %#v but is %#v", expected.Key, args.Key)
	}

	if bytes.Compare(expected.Value, args.Value) != 0 {
		t.Errorf("Value shoud be %#v but is %#v", expected.Value, args.Value)
	}
}

func TestDbArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(DbArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(DbArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbArgsDatabaseType(t *testing.T) {
	input := `[true, "keyval", "valval"]`

	args := new(DbArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbArgsKeyType(t *testing.T) {
	input := `["dbval", 3, "valval"]`

	args := new(DbArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbArgsValType(t *testing.T) {
	input := `["dbval", "keyval", {}]`

	args := new(DbArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbArgsDatabaseEmpty(t *testing.T) {
	input := `["", "keyval", "valval"]`

	args := new(DbArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err.Error())
	}

	str := ExpectValidationError(args.requirements())
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbArgsKeyEmpty(t *testing.T) {
	input := `["dbval", "", "valval"]`

	args := new(DbArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err.Error())
	}

	str := ExpectValidationError(args.requirements())
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbHexArgs(t *testing.T) {
	input := `["testDB","myKey","0xbeef"]`
	expected := new(DbHexArgs)
	expected.Database = "testDB"
	expected.Key = "myKey"
	expected.Value = []byte{0xbe, 0xef}

	args := new(DbHexArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if err := args.requirements(); err != nil {
		t.Error(err)
	}

	if expected.Database != args.Database {
		t.Errorf("Database shoud be %#v but is %#v", expected.Database, args.Database)
	}

	if expected.Key != args.Key {
		t.Errorf("Key shoud be %#v but is %#v", expected.Key, args.Key)
	}

	if bytes.Compare(expected.Value, args.Value) != 0 {
		t.Errorf("Value shoud be %#v but is %#v", expected.Value, args.Value)
	}
}

func TestDbHexArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(DbHexArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbHexArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(DbHexArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbHexArgsDatabaseType(t *testing.T) {
	input := `[true, "keyval", "valval"]`

	args := new(DbHexArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbHexArgsKeyType(t *testing.T) {
	input := `["dbval", 3, "valval"]`

	args := new(DbHexArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbHexArgsValType(t *testing.T) {
	input := `["dbval", "keyval", {}]`

	args := new(DbHexArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbHexArgsDatabaseEmpty(t *testing.T) {
	input := `["", "keyval", "valval"]`

	args := new(DbHexArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err.Error())
	}

	str := ExpectValidationError(args.requirements())
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestDbHexArgsKeyEmpty(t *testing.T) {
	input := `["dbval", "", "valval"]`

	args := new(DbHexArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err.Error())
	}

	str := ExpectValidationError(args.requirements())
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestWhisperMessageArgs(t *testing.T) {
	input := `[{"from":"0xc931d93e97ab07fe42d923478ba2465f2",
  "topics": ["0x68656c6c6f20776f726c64"],
  "payload":"0x68656c6c6f20776f726c64",
  "ttl": "0x64",
  "priority": "0x64"}]`
	expected := new(WhisperMessageArgs)
	expected.From = "0xc931d93e97ab07fe42d923478ba2465f2"
	expected.To = ""
	expected.Payload = "0x68656c6c6f20776f726c64"
	expected.Priority = 100
	expected.Ttl = 100
	// expected.Topics = []string{"0x68656c6c6f20776f726c64"}

	args := new(WhisperMessageArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.From != args.From {
		t.Errorf("From shoud be %#v but is %#v", expected.From, args.From)
	}

	if expected.To != args.To {
		t.Errorf("To shoud be %#v but is %#v", expected.To, args.To)
	}

	if expected.Payload != args.Payload {
		t.Errorf("Value shoud be %#v but is %#v", expected.Payload, args.Payload)
	}

	if expected.Ttl != args.Ttl {
		t.Errorf("Ttl shoud be %#v but is %#v", expected.Ttl, args.Ttl)
	}

	if expected.Priority != args.Priority {
		t.Errorf("Priority shoud be %#v but is %#v", expected.Priority, args.Priority)
	}

	// if expected.Topics != args.Topics {
	// 	t.Errorf("Topic shoud be %#v but is %#v", expected.Topic, args.Topic)
	// }
}

func TestWhisperMessageArgsInt(t *testing.T) {
	input := `[{"from":"0xc931d93e97ab07fe42d923478ba2465f2",
  "topics": ["0x68656c6c6f20776f726c64"],
  "payload":"0x68656c6c6f20776f726c64",
  "ttl": 12,
  "priority": 16}]`
	expected := new(WhisperMessageArgs)
	expected.From = "0xc931d93e97ab07fe42d923478ba2465f2"
	expected.To = ""
	expected.Payload = "0x68656c6c6f20776f726c64"
	expected.Priority = 16
	expected.Ttl = 12
	// expected.Topics = []string{"0x68656c6c6f20776f726c64"}

	args := new(WhisperMessageArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.From != args.From {
		t.Errorf("From shoud be %#v but is %#v", expected.From, args.From)
	}

	if expected.To != args.To {
		t.Errorf("To shoud be %#v but is %#v", expected.To, args.To)
	}

	if expected.Payload != args.Payload {
		t.Errorf("Value shoud be %#v but is %#v", expected.Payload, args.Payload)
	}

	if expected.Ttl != args.Ttl {
		t.Errorf("Ttl shoud be %v but is %v", expected.Ttl, args.Ttl)
	}

	if expected.Priority != args.Priority {
		t.Errorf("Priority shoud be %v but is %v", expected.Priority, args.Priority)
	}

	// if expected.Topics != args.Topics {
	// 	t.Errorf("Topic shoud be %#v but is %#v", expected.Topic, args.Topic)
	// }
}

func TestWhisperMessageArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(WhisperMessageArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestWhisperMessageArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(WhisperMessageArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestWhisperMessageArgsTtlBool(t *testing.T) {
	input := `[{"from":"0xc931d93e97ab07fe42d923478ba2465f2",
  "topics": ["0x68656c6c6f20776f726c64"],
  "payload":"0x68656c6c6f20776f726c64",
  "ttl": true,
  "priority": "0x64"}]`
	args := new(WhisperMessageArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestWhisperMessageArgsPriorityBool(t *testing.T) {
	input := `[{"from":"0xc931d93e97ab07fe42d923478ba2465f2",
  "topics": ["0x68656c6c6f20776f726c64"],
  "payload":"0x68656c6c6f20776f726c64",
  "ttl": "0x12",
  "priority": true}]`
	args := new(WhisperMessageArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestFilterIdArgs(t *testing.T) {
	input := `["0x7"]`
	expected := new(FilterIdArgs)
	expected.Id = 7

	args := new(FilterIdArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Id != args.Id {
		t.Errorf("Id shoud be %#v but is %#v", expected.Id, args.Id)
	}
}

func TestFilterIdArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(FilterIdArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestFilterIdArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(FilterIdArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestFilterIdArgsBool(t *testing.T) {
	input := `[true]`

	args := new(FilterIdArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestWhisperFilterArgs(t *testing.T) {
	input := `[{"topics": ["0x68656c6c6f20776f726c64"], "to": "0x34ag445g3455b34"}]`
	expected := new(WhisperFilterArgs)
	expected.To = "0x34ag445g3455b34"
	expected.Topics = []string{"0x68656c6c6f20776f726c64"}

	args := new(WhisperFilterArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.To != args.To {
		t.Errorf("To shoud be %#v but is %#v", expected.To, args.To)
	}

	// if expected.Topics != args.Topics {
	// 	t.Errorf("Topics shoud be %#v but is %#v", expected.Topics, args.Topics)
	// }
}

func TestWhisperFilterArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(WhisperFilterArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestWhisperFilterArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(WhisperFilterArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestWhisperFilterArgsToInt(t *testing.T) {
	input := `[{"to": 2}]`

	args := new(WhisperFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestWhisperFilterArgsToBool(t *testing.T) {
	input := `[{"topics": ["0x68656c6c6f20776f726c64"], "to": false}]`

	args := new(WhisperFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestWhisperFilterArgsToMissing(t *testing.T) {
	input := `[{"topics": ["0x68656c6c6f20776f726c64"]}]`
	expected := new(WhisperFilterArgs)
	expected.To = ""

	args := new(WhisperFilterArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if args.To != expected.To {
		t.Errorf("To shoud be %v but is %v", expected.To, args.To)
	}
}

func TestWhisperFilterArgsTopicInt(t *testing.T) {
	input := `[{"topics": [6], "to": "0x34ag445g3455b34"}]`

	args := new(WhisperFilterArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCompileArgs(t *testing.T) {
	input := `["contract test { function multiply(uint a) returns(uint d) {   return a * 7;   } }"]`
	expected := new(CompileArgs)
	expected.Source = `contract test { function multiply(uint a) returns(uint d) {   return a * 7;   } }`

	args := new(CompileArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Source != args.Source {
		t.Errorf("Source shoud be %#v but is %#v", expected.Source, args.Source)
	}
}

func TestCompileArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(CompileArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCompileArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(CompileArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestCompileArgsBool(t *testing.T) {
	input := `[false]`

	args := new(CompileArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestFilterStringArgs(t *testing.T) {
	input := `["pending"]`
	expected := new(FilterStringArgs)
	expected.Word = "pending"

	args := new(FilterStringArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Word != args.Word {
		t.Errorf("Word shoud be %#v but is %#v", expected.Word, args.Word)
	}
}

func TestFilterStringEmptyArgs(t *testing.T) {
	input := `[]`

	args := new(FilterStringArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestFilterStringInvalidArgs(t *testing.T) {
	input := `{}`

	args := new(FilterStringArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestFilterStringWordInt(t *testing.T) {
	input := `[7]`

	args := new(FilterStringArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestFilterStringWordWrong(t *testing.T) {
	input := `["foo"]`

	args := new(FilterStringArgs)
	str := ExpectValidationError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestWhisperIdentityArgs(t *testing.T) {
	input := `["0xc931d93e97ab07fe42d923478ba2465f283"]`
	expected := new(WhisperIdentityArgs)
	expected.Identity = "0xc931d93e97ab07fe42d923478ba2465f283"

	args := new(WhisperIdentityArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Identity != args.Identity {
		t.Errorf("Identity shoud be %#v but is %#v", expected.Identity, args.Identity)
	}
}

func TestWhisperIdentityArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(WhisperIdentityArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestWhisperIdentityArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(WhisperIdentityArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestWhisperIdentityArgsInt(t *testing.T) {
	input := `[4]`

	args := new(WhisperIdentityArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Errorf(str)
	}
}

func TestBlockNumArgs(t *testing.T) {
	input := `["0x29a"]`
	expected := new(BlockNumIndexArgs)
	expected.BlockNumber = 666

	args := new(BlockNumArg)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestBlockNumArgsWord(t *testing.T) {
	input := `["pending"]`
	expected := new(BlockNumIndexArgs)
	expected.BlockNumber = -2

	args := new(BlockNumArg)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}
}

func TestBlockNumArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(BlockNumArg)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockNumArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(BlockNumArg)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}
func TestBlockNumArgsBool(t *testing.T) {
	input := `[true]`

	args := new(BlockNumArg)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockNumIndexArgs(t *testing.T) {
	input := `["0x29a", "0x0"]`
	expected := new(BlockNumIndexArgs)
	expected.BlockNumber = 666
	expected.Index = 0

	args := new(BlockNumIndexArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}

	if expected.Index != args.Index {
		t.Errorf("Index shoud be %#v but is %#v", expected.Index, args.Index)
	}
}

func TestBlockNumIndexArgsWord(t *testing.T) {
	input := `["latest", 67]`
	expected := new(BlockNumIndexArgs)
	expected.BlockNumber = -1
	expected.Index = 67

	args := new(BlockNumIndexArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.BlockNumber != args.BlockNumber {
		t.Errorf("BlockNumber shoud be %#v but is %#v", expected.BlockNumber, args.BlockNumber)
	}

	if expected.Index != args.Index {
		t.Errorf("Index shoud be %#v but is %#v", expected.Index, args.Index)
	}
}

func TestBlockNumIndexArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(BlockNumIndexArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockNumIndexArgsInvalid(t *testing.T) {
	input := `"foo"`

	args := new(BlockNumIndexArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockNumIndexArgsBlocknumInvalid(t *testing.T) {
	input := `[{}, "0x1"]`

	args := new(BlockNumIndexArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockNumIndexArgsIndexInvalid(t *testing.T) {
	input := `["0x29a", true]`

	args := new(BlockNumIndexArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestHashIndexArgs(t *testing.T) {
	input := `["0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b", "0x1"]`
	expected := new(HashIndexArgs)
	expected.Hash = "0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b"
	expected.Index = 1

	args := new(HashIndexArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Hash != args.Hash {
		t.Errorf("Hash shoud be %#v but is %#v", expected.Hash, args.Hash)
	}

	if expected.Index != args.Index {
		t.Errorf("Index shoud be %#v but is %#v", expected.Index, args.Index)
	}
}

func TestHashIndexArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(HashIndexArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestHashIndexArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(HashIndexArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestHashIndexArgsInvalidHash(t *testing.T) {
	input := `[7, "0x1"]`

	args := new(HashIndexArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestHashIndexArgsInvalidIndex(t *testing.T) {
	input := `["0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b", false]`

	args := new(HashIndexArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestHashArgs(t *testing.T) {
	input := `["0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b"]`
	expected := new(HashIndexArgs)
	expected.Hash = "0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b"

	args := new(HashArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Hash != args.Hash {
		t.Errorf("Hash shoud be %#v but is %#v", expected.Hash, args.Hash)
	}
}

func TestHashArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(HashArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestHashArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(HashArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestHashArgsInvalidHash(t *testing.T) {
	input := `[7]`

	args := new(HashArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), &args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestSubmitWorkArgs(t *testing.T) {
	input := `["0x0000000000000001", "0x1234567890abcdef1234567890abcdef", "0xD1GE5700000000000000000000000000"]`
	expected := new(SubmitWorkArgs)
	expected.Nonce = 1
	expected.Header = "0x1234567890abcdef1234567890abcdef"
	expected.Digest = "0xD1GE5700000000000000000000000000"

	args := new(SubmitWorkArgs)
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Error(err)
	}

	if expected.Nonce != args.Nonce {
		t.Errorf("Nonce shoud be %d but is %d", expected.Nonce, args.Nonce)
	}

	if expected.Header != args.Header {
		t.Errorf("Header shoud be %#v but is %#v", expected.Header, args.Header)
	}

	if expected.Digest != args.Digest {
		t.Errorf("Digest shoud be %#v but is %#v", expected.Digest, args.Digest)
	}
}

func TestSubmitWorkArgsInvalid(t *testing.T) {
	input := `{}`

	args := new(SubmitWorkArgs)
	str := ExpectDecodeParamError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestSubmitWorkArgsEmpty(t *testing.T) {
	input := `[]`

	args := new(SubmitWorkArgs)
	str := ExpectInsufficientParamsError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestSubmitWorkArgsNonceInt(t *testing.T) {
	input := `[1, "0x1234567890abcdef1234567890abcdef", "0xD1GE5700000000000000000000000000"]`

	args := new(SubmitWorkArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}
func TestSubmitWorkArgsHeaderInt(t *testing.T) {
	input := `["0x0000000000000001", 1, "0xD1GE5700000000000000000000000000"]`

	args := new(SubmitWorkArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}
func TestSubmitWorkArgsDigestInt(t *testing.T) {
	input := `["0x0000000000000001", "0x1234567890abcdef1234567890abcdef", 1]`

	args := new(SubmitWorkArgs)
	str := ExpectInvalidTypeError(json.Unmarshal([]byte(input), args))
	if len(str) > 0 {
		t.Error(str)
	}
}

func TestBlockHeightFromJsonInvalid(t *testing.T) {
	var num int64
	var msg json.RawMessage = []byte(`}{`)
	str := ExpectDecodeParamError(blockHeightFromJson(msg, &num))
	if len(str) > 0 {
		t.Error(str)
	}
}
