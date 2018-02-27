// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package MinerPoolManagement

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

	//定义POSMINER访问客户端
	var(
		PosConn *ethclient.Client
	)	
	
// MinerPoolManagementABI is the input ABI used to generate the binding from.
const MinerPoolManagementABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"MinerPool\",\"type\":\"address\"},{\"name\":\"status\",\"type\":\"bool\"},{\"name\":\"AppAddr\",\"type\":\"string\"}],\"name\":\"MinerPoolSetting\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"MPools\",\"outputs\":[{\"name\":\"status\",\"type\":\"bool\"},{\"name\":\"AppAddr\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"Manager\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"RegistryPoolNum\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newManager\",\"type\":\"address\"}],\"name\":\"transferManagement\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// MinerPoolManagementBin is the compiled bytecode used for deploying new contracts.
const MinerPoolManagementBin = `{
	"linkReferences": {},
	"object": "6060604052341561000f57600080fd5b336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506105e68061005e6000396000f30060606040526004361061006d576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636293c7ba14610072578063654ca9d9146100f957806378357e53146101d3578063a138452214610228578063e4edf85214610251575b600080fd5b341561007d57600080fd5b6100f7600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091908035151590602001909190803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509190505061028a565b005b341561010457600080fd5b610130600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190505061041c565b6040518083151515158152602001806020018281038252838181546001816001161561010002031660029004815260200191508054600181600116156101000203166002900480156101c35780601f10610198576101008083540402835291602001916101c3565b820191906000526020600020905b8154815290600101906020018083116101a657829003601f168201915b5050935050505060405180910390f35b34156101de57600080fd5b6101e661044c565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b341561023357600080fd5b61023b610471565b6040518082815260200191505060405180910390f35b341561025c57600080fd5b610288600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610477565b005b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156102e557600080fd5b60011515600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060000160009054906101000a900460ff1615151415156103545760016002600082825401925050819055505b6040805190810160405280831515815260200182815250600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008201518160000160006101000a81548160ff02191690831515021790555060208201518160010190805190602001906103e6929190610515565b509050506000151582151514801561040057506001600254115b156104175760016002600082825403925050819055505b505050565b60016020528060005260406000206000915090508060000160009054906101000a900460ff169080600101905082565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60025481565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156104d257600080fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061055657805160ff1916838001178555610584565b82800160010185558215610584579182015b82811115610583578251825591602001919060010190610568565b5b5090506105919190610595565b5090565b6105b791905b808211156105b357600081600090555060010161059b565b5090565b905600a165627a7a72305820f63c742bf30bb7050f986fc484783b5a2b7b0fd9575e79311b1e81ad2484064c0029",
	"opcodes": "PUSH1 0x60 PUSH1 0x40 MSTORE CALLVALUE ISZERO PUSH2 0xF JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST CALLER PUSH1 0x0 DUP1 PUSH2 0x100 EXP DUP2 SLOAD DUP2 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF MUL NOT AND SWAP1 DUP4 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND MUL OR SWAP1 SSTORE POP PUSH2 0x5E6 DUP1 PUSH2 0x5E PUSH1 0x0 CODECOPY PUSH1 0x0 RETURN STOP PUSH1 0x60 PUSH1 0x40 MSTORE PUSH1 0x4 CALLDATASIZE LT PUSH2 0x6D JUMPI PUSH1 0x0 CALLDATALOAD PUSH29 0x100000000000000000000000000000000000000000000000000000000 SWAP1 DIV PUSH4 0xFFFFFFFF AND DUP1 PUSH4 0x6293C7BA EQ PUSH2 0x72 JUMPI DUP1 PUSH4 0x654CA9D9 EQ PUSH2 0xF9 JUMPI DUP1 PUSH4 0x78357E53 EQ PUSH2 0x1D3 JUMPI DUP1 PUSH4 0xA1384522 EQ PUSH2 0x228 JUMPI DUP1 PUSH4 0xE4EDF852 EQ PUSH2 0x251 JUMPI JUMPDEST PUSH1 0x0 DUP1 REVERT JUMPDEST CALLVALUE ISZERO PUSH2 0x7D JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0xF7 PUSH1 0x4 DUP1 DUP1 CALLDATALOAD PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 DUP1 CALLDATALOAD ISZERO ISZERO SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP3 ADD DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP1 DUP1 PUSH1 0x1F ADD PUSH1 0x20 DUP1 SWAP2 DIV MUL PUSH1 0x20 ADD PUSH1 0x40 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 SWAP4 SWAP3 SWAP2 SWAP1 DUP2 DUP2 MSTORE PUSH1 0x20 ADD DUP4 DUP4 DUP1 DUP3 DUP5 CALLDATACOPY DUP3 ADD SWAP2 POP POP POP POP POP POP SWAP2 SWAP1 POP POP PUSH2 0x28A JUMP JUMPDEST STOP JUMPDEST CALLVALUE ISZERO PUSH2 0x104 JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x130 PUSH1 0x4 DUP1 DUP1 CALLDATALOAD PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 POP POP PUSH2 0x41C JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP4 ISZERO ISZERO ISZERO ISZERO DUP2 MSTORE PUSH1 0x20 ADD DUP1 PUSH1 0x20 ADD DUP3 DUP2 SUB DUP3 MSTORE DUP4 DUP2 DUP2 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP DUP1 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV DUP1 ISZERO PUSH2 0x1C3 JUMPI DUP1 PUSH1 0x1F LT PUSH2 0x198 JUMPI PUSH2 0x100 DUP1 DUP4 SLOAD DIV MUL DUP4 MSTORE SWAP2 PUSH1 0x20 ADD SWAP2 PUSH2 0x1C3 JUMP JUMPDEST DUP3 ADD SWAP2 SWAP1 PUSH1 0x0 MSTORE PUSH1 0x20 PUSH1 0x0 KECCAK256 SWAP1 JUMPDEST DUP2 SLOAD DUP2 MSTORE SWAP1 PUSH1 0x1 ADD SWAP1 PUSH1 0x20 ADD DUP1 DUP4 GT PUSH2 0x1A6 JUMPI DUP3 SWAP1 SUB PUSH1 0x1F AND DUP3 ADD SWAP2 JUMPDEST POP POP SWAP4 POP POP POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE ISZERO PUSH2 0x1DE JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x1E6 PUSH2 0x44C JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP3 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE ISZERO PUSH2 0x233 JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x23B PUSH2 0x471 JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP3 DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE ISZERO PUSH2 0x25C JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x288 PUSH1 0x4 DUP1 DUP1 CALLDATALOAD PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 POP POP PUSH2 0x477 JUMP JUMPDEST STOP JUMPDEST PUSH1 0x0 DUP1 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND EQ ISZERO ISZERO PUSH2 0x2E5 JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH1 0x1 ISZERO ISZERO PUSH1 0x1 PUSH1 0x0 DUP6 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP1 DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x0 KECCAK256 PUSH1 0x0 ADD PUSH1 0x0 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH1 0xFF AND ISZERO ISZERO EQ ISZERO ISZERO PUSH2 0x354 JUMPI PUSH1 0x1 PUSH1 0x2 PUSH1 0x0 DUP3 DUP3 SLOAD ADD SWAP3 POP POP DUP2 SWAP1 SSTORE POP JUMPDEST PUSH1 0x40 DUP1 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 DUP4 ISZERO ISZERO DUP2 MSTORE PUSH1 0x20 ADD DUP3 DUP2 MSTORE POP PUSH1 0x1 PUSH1 0x0 DUP6 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP1 DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x0 KECCAK256 PUSH1 0x0 DUP3 ADD MLOAD DUP2 PUSH1 0x0 ADD PUSH1 0x0 PUSH2 0x100 EXP DUP2 SLOAD DUP2 PUSH1 0xFF MUL NOT AND SWAP1 DUP4 ISZERO ISZERO MUL OR SWAP1 SSTORE POP PUSH1 0x20 DUP3 ADD MLOAD DUP2 PUSH1 0x1 ADD SWAP1 DUP1 MLOAD SWAP1 PUSH1 0x20 ADD SWAP1 PUSH2 0x3E6 SWAP3 SWAP2 SWAP1 PUSH2 0x515 JUMP JUMPDEST POP SWAP1 POP POP PUSH1 0x0 ISZERO ISZERO DUP3 ISZERO ISZERO EQ DUP1 ISZERO PUSH2 0x400 JUMPI POP PUSH1 0x1 PUSH1 0x2 SLOAD GT JUMPDEST ISZERO PUSH2 0x417 JUMPI PUSH1 0x1 PUSH1 0x2 PUSH1 0x0 DUP3 DUP3 SLOAD SUB SWAP3 POP POP DUP2 SWAP1 SSTORE POP JUMPDEST POP POP POP JUMP JUMPDEST PUSH1 0x1 PUSH1 0x20 MSTORE DUP1 PUSH1 0x0 MSTORE PUSH1 0x40 PUSH1 0x0 KECCAK256 PUSH1 0x0 SWAP2 POP SWAP1 POP DUP1 PUSH1 0x0 ADD PUSH1 0x0 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH1 0xFF AND SWAP1 DUP1 PUSH1 0x1 ADD SWAP1 POP DUP3 JUMP JUMPDEST PUSH1 0x0 DUP1 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 JUMP JUMPDEST PUSH1 0x2 SLOAD DUP2 JUMP JUMPDEST PUSH1 0x0 DUP1 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND EQ ISZERO ISZERO PUSH2 0x4D2 JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST DUP1 PUSH1 0x0 DUP1 PUSH2 0x100 EXP DUP2 SLOAD DUP2 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF MUL NOT AND SWAP1 DUP4 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND MUL OR SWAP1 SSTORE POP POP JUMP JUMPDEST DUP3 DUP1 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV SWAP1 PUSH1 0x0 MSTORE PUSH1 0x20 PUSH1 0x0 KECCAK256 SWAP1 PUSH1 0x1F ADD PUSH1 0x20 SWAP1 DIV DUP2 ADD SWAP3 DUP3 PUSH1 0x1F LT PUSH2 0x556 JUMPI DUP1 MLOAD PUSH1 0xFF NOT AND DUP4 DUP1 ADD OR DUP6 SSTORE PUSH2 0x584 JUMP JUMPDEST DUP3 DUP1 ADD PUSH1 0x1 ADD DUP6 SSTORE DUP3 ISZERO PUSH2 0x584 JUMPI SWAP2 DUP3 ADD JUMPDEST DUP3 DUP2 GT ISZERO PUSH2 0x583 JUMPI DUP3 MLOAD DUP3 SSTORE SWAP2 PUSH1 0x20 ADD SWAP2 SWAP1 PUSH1 0x1 ADD SWAP1 PUSH2 0x568 JUMP JUMPDEST JUMPDEST POP SWAP1 POP PUSH2 0x591 SWAP2 SWAP1 PUSH2 0x595 JUMP JUMPDEST POP SWAP1 JUMP JUMPDEST PUSH2 0x5B7 SWAP2 SWAP1 JUMPDEST DUP1 DUP3 GT ISZERO PUSH2 0x5B3 JUMPI PUSH1 0x0 DUP2 PUSH1 0x0 SWAP1 SSTORE POP PUSH1 0x1 ADD PUSH2 0x59B JUMP JUMPDEST POP SWAP1 JUMP JUMPDEST SWAP1 JUMP STOP LOG1 PUSH6 0x627A7A723058 KECCAK256 0xf6 EXTCODECOPY PUSH21 0x2BF30BB7050F986FC484783B5A2B7B0FD9575E7931 0x1b 0x1e DUP2 0xad 0x24 DUP5 MOD 0x4c STOP 0x29 ",
	"sourceMap": "28:1107:0:-;;;501:81;;;;;;;;563:10;555:7;;:18;;;;;;;;;;;;;;;;;;28:1107;;;;;;"
}`

// DeployMinerPoolManagement deploys a new Ethereum contract, binding an instance of MinerPoolManagement to it.
func DeployMinerPoolManagement(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MinerPoolManagement, error) {
	parsed, err := abi.JSON(strings.NewReader(MinerPoolManagementABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(MinerPoolManagementBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MinerPoolManagement{MinerPoolManagementCaller: MinerPoolManagementCaller{contract: contract}, MinerPoolManagementTransactor: MinerPoolManagementTransactor{contract: contract}}, nil
}

// MinerPoolManagement is an auto generated Go binding around an Ethereum contract.
type MinerPoolManagement struct {
	MinerPoolManagementCaller     // Read-only binding to the contract
	MinerPoolManagementTransactor // Write-only binding to the contract
}

// MinerPoolManagementCaller is an auto generated read-only Go binding around an Ethereum contract.
type MinerPoolManagementCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MinerPoolManagementTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MinerPoolManagementTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MinerPoolManagementSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MinerPoolManagementSession struct {
	Contract     *MinerPoolManagement // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// MinerPoolManagementCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MinerPoolManagementCallerSession struct {
	Contract *MinerPoolManagementCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// MinerPoolManagementTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MinerPoolManagementTransactorSession struct {
	Contract     *MinerPoolManagementTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// MinerPoolManagementRaw is an auto generated low-level Go binding around an Ethereum contract.
type MinerPoolManagementRaw struct {
	Contract *MinerPoolManagement // Generic contract binding to access the raw methods on
}

// MinerPoolManagementCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MinerPoolManagementCallerRaw struct {
	Contract *MinerPoolManagementCaller // Generic read-only contract binding to access the raw methods on
}

// MinerPoolManagementTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MinerPoolManagementTransactorRaw struct {
	Contract *MinerPoolManagementTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMinerPoolManagement creates a new instance of MinerPoolManagement, bound to a specific deployed contract.
func NewMinerPoolManagement(address common.Address, backend bind.ContractBackend) (*MinerPoolManagement, error) {
	contract, err := bindMinerPoolManagement(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MinerPoolManagement{MinerPoolManagementCaller: MinerPoolManagementCaller{contract: contract}, MinerPoolManagementTransactor: MinerPoolManagementTransactor{contract: contract}}, nil
}

// NewMinerPoolManagementCaller creates a new read-only instance of MinerPoolManagement, bound to a specific deployed contract.
func NewMinerPoolManagementCaller(address common.Address, caller bind.ContractCaller) (*MinerPoolManagementCaller, error) {
	contract, err := bindMinerPoolManagement(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &MinerPoolManagementCaller{contract: contract}, nil
}

// NewMinerPoolManagementTransactor creates a new write-only instance of MinerPoolManagement, bound to a specific deployed contract.
func NewMinerPoolManagementTransactor(address common.Address, transactor bind.ContractTransactor) (*MinerPoolManagementTransactor, error) {
	contract, err := bindMinerPoolManagement(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &MinerPoolManagementTransactor{contract: contract}, nil
}

// bindMinerPoolManagement binds a generic wrapper to an already deployed contract.
func bindMinerPoolManagement(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MinerPoolManagementABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MinerPoolManagement *MinerPoolManagementRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _MinerPoolManagement.Contract.MinerPoolManagementCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MinerPoolManagement *MinerPoolManagementRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MinerPoolManagement.Contract.MinerPoolManagementTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MinerPoolManagement *MinerPoolManagementRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MinerPoolManagement.Contract.MinerPoolManagementTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MinerPoolManagement *MinerPoolManagementCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _MinerPoolManagement.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MinerPoolManagement *MinerPoolManagementTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MinerPoolManagement.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MinerPoolManagement *MinerPoolManagementTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MinerPoolManagement.Contract.contract.Transact(opts, method, params...)
}

// MPools is a free data retrieval call binding the contract method 0x654ca9d9.
//
// Solidity: function MPools( address) constant returns(status bool, AppAddr string)
func (_MinerPoolManagement *MinerPoolManagementCaller) MPools(opts *bind.CallOpts, arg0 common.Address) (struct {
	Status  bool
	AppAddr string
}, error) {
	ret := new(struct {
		Status  bool
		AppAddr string
	})
	out := ret
	err := _MinerPoolManagement.contract.Call(opts, out, "MPools", arg0)
	return *ret, err
}

// MPools is a free data retrieval call binding the contract method 0x654ca9d9.
//
// Solidity: function MPools( address) constant returns(status bool, AppAddr string)
func (_MinerPoolManagement *MinerPoolManagementSession) MPools(arg0 common.Address) (struct {
	Status  bool
	AppAddr string
}, error) {
	return _MinerPoolManagement.Contract.MPools(&_MinerPoolManagement.CallOpts, arg0)
}

// MPools is a free data retrieval call binding the contract method 0x654ca9d9.
//
// Solidity: function MPools( address) constant returns(status bool, AppAddr string)
func (_MinerPoolManagement *MinerPoolManagementCallerSession) MPools(arg0 common.Address) (struct {
	Status  bool
	AppAddr string
}, error) {
	return _MinerPoolManagement.Contract.MPools(&_MinerPoolManagement.CallOpts, arg0)
}

// Manager is a free data retrieval call binding the contract method 0x78357e53.
//
// Solidity: function Manager() constant returns(address)
func (_MinerPoolManagement *MinerPoolManagementCaller) Manager(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _MinerPoolManagement.contract.Call(opts, out, "Manager")
	return *ret0, err
}

// Manager is a free data retrieval call binding the contract method 0x78357e53.
//
// Solidity: function Manager() constant returns(address)
func (_MinerPoolManagement *MinerPoolManagementSession) Manager() (common.Address, error) {
	return _MinerPoolManagement.Contract.Manager(&_MinerPoolManagement.CallOpts)
}

// Manager is a free data retrieval call binding the contract method 0x78357e53.
//
// Solidity: function Manager() constant returns(address)
func (_MinerPoolManagement *MinerPoolManagementCallerSession) Manager() (common.Address, error) {
	return _MinerPoolManagement.Contract.Manager(&_MinerPoolManagement.CallOpts)
}

// RegistryPoolNum is a free data retrieval call binding the contract method 0xa1384522.
//
// Solidity: function RegistryPoolNum() constant returns(uint256)
func (_MinerPoolManagement *MinerPoolManagementCaller) RegistryPoolNum(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _MinerPoolManagement.contract.Call(opts, out, "RegistryPoolNum")
	return *ret0, err
}

// RegistryPoolNum is a free data retrieval call binding the contract method 0xa1384522.
//
// Solidity: function RegistryPoolNum() constant returns(uint256)
func (_MinerPoolManagement *MinerPoolManagementSession) RegistryPoolNum() (*big.Int, error) {
	return _MinerPoolManagement.Contract.RegistryPoolNum(&_MinerPoolManagement.CallOpts)
}

// RegistryPoolNum is a free data retrieval call binding the contract method 0xa1384522.
//
// Solidity: function RegistryPoolNum() constant returns(uint256)
func (_MinerPoolManagement *MinerPoolManagementCallerSession) RegistryPoolNum() (*big.Int, error) {
	return _MinerPoolManagement.Contract.RegistryPoolNum(&_MinerPoolManagement.CallOpts)
}

// MinerPoolSetting is a paid mutator transaction binding the contract method 0x6293c7ba.
//
// Solidity: function MinerPoolSetting(MinerPool address, status bool, AppAddr string) returns()
func (_MinerPoolManagement *MinerPoolManagementTransactor) MinerPoolSetting(opts *bind.TransactOpts, MinerPool common.Address, status bool, AppAddr string) (*types.Transaction, error) {
	return _MinerPoolManagement.contract.Transact(opts, "MinerPoolSetting", MinerPool, status, AppAddr)
}

// MinerPoolSetting is a paid mutator transaction binding the contract method 0x6293c7ba.
//
// Solidity: function MinerPoolSetting(MinerPool address, status bool, AppAddr string) returns()
func (_MinerPoolManagement *MinerPoolManagementSession) MinerPoolSetting(MinerPool common.Address, status bool, AppAddr string) (*types.Transaction, error) {
	return _MinerPoolManagement.Contract.MinerPoolSetting(&_MinerPoolManagement.TransactOpts, MinerPool, status, AppAddr)
}

// MinerPoolSetting is a paid mutator transaction binding the contract method 0x6293c7ba.
//
// Solidity: function MinerPoolSetting(MinerPool address, status bool, AppAddr string) returns()
func (_MinerPoolManagement *MinerPoolManagementTransactorSession) MinerPoolSetting(MinerPool common.Address, status bool, AppAddr string) (*types.Transaction, error) {
	return _MinerPoolManagement.Contract.MinerPoolSetting(&_MinerPoolManagement.TransactOpts, MinerPool, status, AppAddr)
}

// TransferManagement is a paid mutator transaction binding the contract method 0xe4edf852.
//
// Solidity: function transferManagement(newManager address) returns()
func (_MinerPoolManagement *MinerPoolManagementTransactor) TransferManagement(opts *bind.TransactOpts, newManager common.Address) (*types.Transaction, error) {
	return _MinerPoolManagement.contract.Transact(opts, "transferManagement", newManager)
}

// TransferManagement is a paid mutator transaction binding the contract method 0xe4edf852.
//
// Solidity: function transferManagement(newManager address) returns()
func (_MinerPoolManagement *MinerPoolManagementSession) TransferManagement(newManager common.Address) (*types.Transaction, error) {
	return _MinerPoolManagement.Contract.TransferManagement(&_MinerPoolManagement.TransactOpts, newManager)
}

// TransferManagement is a paid mutator transaction binding the contract method 0xe4edf852.
//
// Solidity: function transferManagement(newManager address) returns()
func (_MinerPoolManagement *MinerPoolManagementTransactorSession) TransferManagement(newManager common.Address) (*types.Transaction, error) {
	return _MinerPoolManagement.Contract.TransferManagement(&_MinerPoolManagement.TransactOpts, newManager)
}
