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

// Oraclerequest is an auto generated low-level Go binding around an user-defined struct.
type Oraclerequest struct {
	Data  []byte
	Data0 []byte
}

// TODO: convert this type to value type after everything works.
// NameConflictMetaData contains all meta data concerning the NameConflict contract.
var NameConflictMetaData = &bind.MetaData{
	ABI:     "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"msg\",\"type\":\"int256\"},{\"indexed\":false,\"internalType\":\"int256\",\"name\":\"_msg\",\"type\":\"int256\"}],\"name\":\"log\",\"type\":\"event\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"internalType\":\"structoracle.request\",\"name\":\"req\",\"type\":\"tuple\"}],\"name\":\"addRequest\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getRequest\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"internalType\":\"structoracle.request\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	Pattern: "8f6e2703b307244ae6bd61ed94ce959cf9",
	Bin:     "0x608060405234801561001057600080fd5b5061042b806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c8063c2bb515f1461003b578063cce7b04814610059575b600080fd5b610043610075565b60405161005091906101af565b60405180910390f35b610073600480360381019061006e91906103ac565b6100b5565b005b61007d6100b8565b604051806040016040528060405180602001604052806000815250815260200160405180602001604052806000815250815250905090565b50565b604051806040016040528060608152602001606081525090565b600081519050919050565b600082825260208201905092915050565b60005b8381101561010c5780820151818401526020810190506100f1565b8381111561011b576000848401525b50505050565b6000601f19601f8301169050919050565b600061013d826100d2565b61014781856100dd565b93506101578185602086016100ee565b61016081610121565b840191505092915050565b600060408301600083015184820360008601526101888282610132565b915050602083015184820360208601526101a28282610132565b9150508091505092915050565b600060208201905081810360008301526101c9818461016b565b905092915050565b6000604051905090565b600080fd5b600080fd5b600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b61022282610121565b810181811067ffffffffffffffff82111715610241576102406101ea565b5b80604052505050565b60006102546101d1565b90506102608282610219565b919050565b600080fd5b600080fd5b600080fd5b600067ffffffffffffffff82111561028f5761028e6101ea565b5b61029882610121565b9050602081019050919050565b82818337600083830152505050565b60006102c76102c284610274565b61024a565b9050828152602081018484840111156102e3576102e261026f565b5b6102ee8482856102a5565b509392505050565b600082601f83011261030b5761030a61026a565b5b813561031b8482602086016102b4565b91505092915050565b60006040828403121561033a576103396101e5565b5b610344604061024a565b9050600082013567ffffffffffffffff81111561036457610363610265565b5b610370848285016102f6565b600083015250602082013567ffffffffffffffff81111561039457610393610265565b5b6103a0848285016102f6565b60208301525092915050565b6000602082840312156103c2576103c16101db565b5b600082013567ffffffffffffffff8111156103e0576103df6101e0565b5b6103ec84828501610324565b9150509291505056fea264697066735822122033bca1606af9b6aeba1673f98c52003cec19338539fb44b86690ce82c51483b564736f6c634300080e0033",
}

// NameConflict is an auto generated Go binding around an Ethereum contract.
type NameConflict struct {
	abi abi.ABI
}

// NewNameConflict creates a new instance of NameConflict.
func NewNameConflict() (*NameConflict, error) {
	parsed, err := NameConflictMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &NameConflict{abi: *parsed}, nil
}

func (nameConflict *NameConflict) PackConstructor() []byte {
	res, _ := nameConflict.abi.Pack("")
	return res
}

// AddRequest is a free data retrieval call binding the contract method 0xcce7b048.
//
// Solidity: function addRequest((bytes,bytes) req) pure returns()
func (nameConflict *NameConflict) PackAddRequest(Req Oraclerequest) ([]byte, error) {
	return nameConflict.abi.Pack("addRequest", Req)
}

// GetRequest is a free data retrieval call binding the contract method 0xc2bb515f.
//
// Solidity: function getRequest() pure returns((bytes,bytes))
func (nameConflict *NameConflict) PackGetRequest() ([]byte, error) {
	return nameConflict.abi.Pack("getRequest")
}

func (nameConflict *NameConflict) UnpackGetRequest(data []byte) (Oraclerequest, error) {
	out, err := nameConflict.abi.Unpack("getRequest", data)

	if err != nil {
		return *new(Oraclerequest), err
	}

	out0 := *abi.ConvertType(out[0], new(Oraclerequest)).(*Oraclerequest)

	return out0, err

}

// NameConflictLog represents a Log event raised by the NameConflict contract.
type NameConflictLog struct {
	Msg  *big.Int
	Msg0 *big.Int
	Raw  *types.Log // Blockchain specific contextual infos
}

const NameConflictLogEventName = "log"

func (nameConflict *NameConflict) UnpackLogEvent(log *types.Log) (*NameConflictLog, error) {
	event := "log"
	if log.Topics[0] != nameConflict.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(NameConflictLog)
	if len(log.Data) > 0 {
		if err := nameConflict.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range nameConflict.abi.Events[event].Inputs {
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