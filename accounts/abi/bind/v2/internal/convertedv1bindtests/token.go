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
// TokenMetaData contains all meta data concerning the Token contract.
var TokenMetaData = &bind.MetaData{
	ABI:     "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"},{\"name\":\"_extraData\",\"type\":\"bytes\"}],\"name\":\"approveAndCall\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"}],\"name\":\"spentAllowance\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"type\":\"function\"},{\"inputs\":[{\"name\":\"initialSupply\",\"type\":\"uint256\"},{\"name\":\"tokenName\",\"type\":\"string\"},{\"name\":\"decimalUnits\",\"type\":\"uint8\"},{\"name\":\"tokenSymbol\",\"type\":\"string\"}],\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"}]",
	Pattern: "1317f51c845ce3bfb7c268e5337a825f12",
	Bin:     "0x60606040526040516107fd3803806107fd83398101604052805160805160a05160c051929391820192909101600160a060020a0333166000908152600360209081526040822086905581548551838052601f6002600019610100600186161502019093169290920482018390047f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e56390810193919290918801908390106100e857805160ff19168380011785555b506101189291505b8082111561017157600081556001016100b4565b50506002805460ff19168317905550505050610658806101a56000396000f35b828001600101855582156100ac579182015b828111156100ac5782518260005055916020019190600101906100fa565b50508060016000509080519060200190828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061017557805160ff19168380011785555b506100c89291506100b4565b5090565b82800160010185558215610165579182015b8281111561016557825182600050559160200191906001019061018756606060405236156100775760e060020a600035046306fdde03811461007f57806323b872dd146100dc578063313ce5671461010e57806370a082311461011a57806395d89b4114610132578063a9059cbb1461018e578063cae9ca51146101bd578063dc3080f21461031c578063dd62ed3e14610341575b610365610002565b61036760008054602060026001831615610100026000190190921691909104601f810182900490910260809081016040526060828152929190828280156104eb5780601f106104c0576101008083540402835291602001916104eb565b6103d5600435602435604435600160a060020a038316600090815260036020526040812054829010156104f357610002565b6103e760025460ff1681565b6103d560043560036020526000908152604090205481565b610367600180546020600282841615610100026000190190921691909104601f810182900490910260809081016040526060828152929190828280156104eb5780601f106104c0576101008083540402835291602001916104eb565b610365600435602435600160a060020a033316600090815260036020526040902054819010156103f157610002565b60806020604435600481810135601f8101849004909302840160405260608381526103d5948235946024803595606494939101919081908382808284375094965050505050505060006000836004600050600033600160a060020a03168152602001908152602001600020600050600087600160a060020a031681526020019081526020016000206000508190555084905080600160a060020a0316638f4ffcb1338630876040518560e060020a0281526004018085600160a060020a0316815260200184815260200183600160a060020a03168152602001806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156102f25780820380516001836020036101000a031916815260200191505b50955050505050506000604051808303816000876161da5a03f11561000257505050509392505050565b6005602090815260043560009081526040808220909252602435815220546103d59081565b60046020818152903560009081526040808220909252602435815220546103d59081565b005b60405180806020018281038252838181518152602001915080519060200190808383829060006004602084601f0104600f02600301f150905090810190601f1680156103c75780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60408051918252519081900360200190f35b6060908152602090f35b600160a060020a03821660009081526040902054808201101561041357610002565b806003600050600033600160a060020a03168152602001908152602001600020600082828250540392505081905550806003600050600084600160a060020a0316815260200190815260200160002060008282825054019250508190555081600160a060020a031633600160a060020a03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a35050565b820191906000526020600020905b8154815290600101906020018083116104ce57829003601f168201915b505050505081565b600160a060020a03831681526040812054808301101561051257610002565b600160a060020a0380851680835260046020908152604080852033949094168086529382528085205492855260058252808520938552929052908220548301111561055c57610002565b816003600050600086600160a060020a03168152602001908152602001600020600082828250540392505081905550816003600050600085600160a060020a03168152602001908152602001600020600082828250540192505081905550816005600050600086600160a060020a03168152602001908152602001600020600050600033600160a060020a0316815260200190815260200160002060008282825054019250508190555082600160a060020a031633600160a060020a03167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040518082815260200191505060405180910390a3939250505056",
}

// Token is an auto generated Go binding around an Ethereum contract.
type Token struct {
	abi abi.ABI
}

// NewToken creates a new instance of Token.
func NewToken() (*Token, error) {
	parsed, err := TokenMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return &Token{abi: *parsed}, nil
}

func (token *Token) PackConstructor(initialSupply *big.Int, tokenName string, decimalUnits uint8, tokenSymbol string) []byte {
	res, _ := token.abi.Pack("", initialSupply, tokenName, decimalUnits, tokenSymbol)
	return res
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address , address ) returns(uint256)
func (token *Token) PackAllowance(Arg0 common.Address, Arg1 common.Address) ([]byte, error) {
	return token.abi.Pack("allowance", Arg0, Arg1)
}

func (token *Token) UnpackAllowance(data []byte) (*big.Int, error) {
	out, err := token.abi.Unpack("allowance", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// ApproveAndCall is a free data retrieval call binding the contract method 0xcae9ca51.
//
// Solidity: function approveAndCall(address _spender, uint256 _value, bytes _extraData) returns(bool success)
func (token *Token) PackApproveAndCall(Spender common.Address, Value *big.Int, ExtraData []byte) ([]byte, error) {
	return token.abi.Pack("approveAndCall", Spender, Value, ExtraData)
}

func (token *Token) UnpackApproveAndCall(data []byte) (bool, error) {
	out, err := token.abi.Unpack("approveAndCall", data)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address ) returns(uint256)
func (token *Token) PackBalanceOf(Arg0 common.Address) ([]byte, error) {
	return token.abi.Pack("balanceOf", Arg0)
}

func (token *Token) UnpackBalanceOf(data []byte) (*big.Int, error) {
	out, err := token.abi.Unpack("balanceOf", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() returns(uint8)
func (token *Token) PackDecimals() ([]byte, error) {
	return token.abi.Pack("decimals")
}

func (token *Token) UnpackDecimals(data []byte) (uint8, error) {
	out, err := token.abi.Unpack("decimals", data)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() returns(string)
func (token *Token) PackName() ([]byte, error) {
	return token.abi.Pack("name")
}

func (token *Token) UnpackName(data []byte) (string, error) {
	out, err := token.abi.Unpack("name", data)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// SpentAllowance is a free data retrieval call binding the contract method 0xdc3080f2.
//
// Solidity: function spentAllowance(address , address ) returns(uint256)
func (token *Token) PackSpentAllowance(Arg0 common.Address, Arg1 common.Address) ([]byte, error) {
	return token.abi.Pack("spentAllowance", Arg0, Arg1)
}

func (token *Token) UnpackSpentAllowance(data []byte) (*big.Int, error) {
	out, err := token.abi.Unpack("spentAllowance", data)

	if err != nil {
		return new(big.Int), err
	}

	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() returns(string)
func (token *Token) PackSymbol() ([]byte, error) {
	return token.abi.Pack("symbol")
}

func (token *Token) UnpackSymbol(data []byte) (string, error) {
	out, err := token.abi.Unpack("symbol", data)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Transfer is a free data retrieval call binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address _to, uint256 _value) returns()
func (token *Token) PackTransfer(To common.Address, Value *big.Int) ([]byte, error) {
	return token.abi.Pack("transfer", To, Value)
}

// TransferFrom is a free data retrieval call binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address _from, address _to, uint256 _value) returns(bool success)
func (token *Token) PackTransferFrom(From common.Address, To common.Address, Value *big.Int) ([]byte, error) {
	return token.abi.Pack("transferFrom", From, To, Value)
}

func (token *Token) UnpackTransferFrom(data []byte) (bool, error) {
	out, err := token.abi.Unpack("transferFrom", data)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// TokenTransfer represents a Transfer event raised by the Token contract.
type TokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   *types.Log // Blockchain specific contextual infos
}

const TokenTransferEventName = "Transfer"

func (token *Token) UnpackTransferEvent(log *types.Log) (*TokenTransfer, error) {
	event := "Transfer"
	if log.Topics[0] != token.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(TokenTransfer)
	if len(log.Data) > 0 {
		if err := token.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range token.abi.Events[event].Inputs {
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