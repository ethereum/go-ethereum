// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package convertedv1bindtests

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// TODO: convert this type to value type after everything works.
// OverloadMetaData contains all meta data concerning the Overload contract.
var OverloadMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":false,\"inputs\":[{\"name\":\"i\",\"type\":\"uint256\"},{\"name\":\"j\",\"type\":\"uint256\"}],\"name\":\"foo\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"i\",\"type\":\"uint256\"}],\"name\":\"foo\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"i\",\"type\":\"uint256\"}],\"name\":\"bar\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"i\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"j\",\"type\":\"uint256\"}],\"name\":\"bar\",\"type\":\"event\"}]",
	Pattern: "f49f0ff7ed407de5c37214f49309072aec",
	Bin:     "0x608060405234801561001057600080fd5b50610153806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806304bc52f81461003b5780632fbebd3814610073575b600080fd5b6100716004803603604081101561005157600080fd5b8101908080359060200190929190803590602001909291905050506100a1565b005b61009f6004803603602081101561008957600080fd5b81019080803590602001909291905050506100e4565b005b7fae42e9514233792a47a1e4554624e83fe852228e1503f63cd383e8a431f4f46d8282604051808381526020018281526020019250505060405180910390a15050565b7f0423a1321222a0a8716c22b92fac42d85a45a612b696a461784d9fa537c81e5c816040518082815260200191505060405180910390a15056fea265627a7a72305820e22b049858b33291cbe67eeaece0c5f64333e439d27032ea8337d08b1de18fe864736f6c634300050a0032",
}

// Overload is an auto generated Go binding around an Ethereum contract.
type Overload struct {
	abi abi.ABI
}

// NewOverload creates a new instance of Overload.
func NewOverload() (*Overload, error) {
	parsed, err := OverloadMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Overload{abi: *parsed}, nil
}

func (overload *Overload) PackConstructor() []byte {
	res, _ := overload.abi.Pack("")
	return res
}

// Foo is a free data retrieval call binding the contract method 0x04bc52f8.
//
// Solidity: function foo(uint256 i, uint256 j) returns()
func (overload *Overload) PackFoo(I *big.Int, J *big.Int) ([]byte, error) {
	return overload.abi.Pack("foo", I, J)
}

// Foo0 is a free data retrieval call binding the contract method 0x2fbebd38.
//
// Solidity: function foo(uint256 i) returns()
func (overload *Overload) PackFoo0(I *big.Int) ([]byte, error) {
	return overload.abi.Pack("foo0", I)
}

// OverloadBar represents a Bar event raised by the Overload contract.
type OverloadBar struct {
	I   *big.Int
	Raw *types.Log // Blockchain specific contextual infos
}

const OverloadBarEventName = "bar"

func (overload *Overload) UnpackBarEvent(log *types.Log) (*OverloadBar, error) {
	event := "bar"
	if log.Topics[0] != overload.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(OverloadBar)
	if len(log.Data) > 0 {
		if err := overload.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range overload.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// OverloadBar0 represents a Bar0 event raised by the Overload contract.
type OverloadBar0 struct {
	I   *big.Int
	J   *big.Int
	Raw *types.Log // Blockchain specific contextual infos
}

const OverloadBar0EventName = "bar0"

func (overload *Overload) UnpackBar0Event(log *types.Log) (*OverloadBar0, error) {
	event := "bar0"
	if log.Topics[0] != overload.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(OverloadBar0)
	if len(log.Data) > 0 {
		if err := overload.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range overload.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}