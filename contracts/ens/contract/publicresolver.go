// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// PublicResolverABI is the input ABI used to generate the binding from.
const PublicResolverABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"interfaceID\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"key\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"string\"}],\"name\":\"setText\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"contentTypes\",\"type\":\"uint256\"}],\"name\":\"ABI\",\"outputs\":[{\"name\":\"contentType\",\"type\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"x\",\"type\":\"bytes32\"},{\"name\":\"y\",\"type\":\"bytes32\"}],\"name\":\"setPubkey\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"content\",\"outputs\":[{\"name\":\"ret\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"addr\",\"outputs\":[{\"name\":\"ret\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"key\",\"type\":\"string\"}],\"name\":\"text\",\"outputs\":[{\"name\":\"ret\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"contentType\",\"type\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"setABI\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"name\",\"outputs\":[{\"name\":\"ret\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"name\",\"type\":\"string\"}],\"name\":\"setName\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"hash\",\"type\":\"bytes32\"}],\"name\":\"setContent\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"pubkey\",\"outputs\":[{\"name\":\"x\",\"type\":\"bytes32\"},{\"name\":\"y\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"setAddr\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"ensAddr\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"a\",\"type\":\"address\"}],\"name\":\"AddrChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"hash\",\"type\":\"bytes32\"}],\"name\":\"ContentChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"}],\"name\":\"NameChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"contentType\",\"type\":\"uint256\"}],\"name\":\"ABIChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"x\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"y\",\"type\":\"bytes32\"}],\"name\":\"PubkeyChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"indexedKey\",\"type\":\"string\"},{\"indexed\":false,\"name\":\"key\",\"type\":\"string\"}],\"name\":\"TextChanged\",\"type\":\"event\"}]"

// PublicResolverBin is the compiled bytecode used for deploying new contracts.
const PublicResolverBin = `0x6060604052341561000f57600080fd5b6040516020806111b28339810160405280805160008054600160a060020a03909216600160a060020a0319909216919091179055505061115e806100546000396000f3006060604052600436106100ab5763ffffffff60e060020a60003504166301ffc9a781146100b057806310f13a8c146100e45780632203ab561461017e57806329cd62ea146102155780632dff6941146102315780633b3b57de1461025957806359d1d43c1461028b578063623195b014610358578063691f3431146103b457806377372213146103ca578063c3d014d614610420578063c869023314610439578063d5fa2b0014610467575b600080fd5b34156100bb57600080fd5b6100d0600160e060020a031960043516610489565b604051901515815260200160405180910390f35b34156100ef57600080fd5b61017c600480359060446024803590810190830135806020601f8201819004810201604051908101604052818152929190602084018383808284378201915050505050509190803590602001908201803590602001908080601f0160208091040260200160405190810160405281815292919060208401838380828437509496506105f695505050505050565b005b341561018957600080fd5b610197600435602435610807565b60405182815260406020820181815290820183818151815260200191508051906020019080838360005b838110156101d95780820151838201526020016101c1565b50505050905090810190601f1680156102065780820380516001836020036101000a031916815260200191505b50935050505060405180910390f35b341561022057600080fd5b61017c600435602435604435610931565b341561023c57600080fd5b610247600435610a30565b60405190815260200160405180910390f35b341561026457600080fd5b61026f600435610a46565b604051600160a060020a03909116815260200160405180910390f35b341561029657600080fd5b6102e1600480359060446024803590810190830135806020601f82018190048102016040519081016040528181529291906020840183838082843750949650610a6195505050505050565b60405160208082528190810183818151815260200191508051906020019080838360005b8381101561031d578082015183820152602001610305565b50505050905090810190601f16801561034a5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561036357600080fd5b61017c600480359060248035919060649060443590810190830135806020601f82018190048102016040519081016040528181529291906020840183838082843750949650610b8095505050505050565b34156103bf57600080fd5b6102e1600435610c7c565b34156103d557600080fd5b61017c600480359060446024803590810190830135806020601f82018190048102016040519081016040528181529291906020840183838082843750949650610d4295505050505050565b341561042b57600080fd5b61017c600435602435610e8c565b341561044457600080fd5b61044f600435610f65565b60405191825260208201526040908101905180910390f35b341561047257600080fd5b61017c600435600160a060020a0360243516610f82565b6000600160e060020a031982167f3b3b57de0000000000000000000000000000000000000000000000000000000014806104ec5750600160e060020a031982167fd8389dc500000000000000000000000000000000000000000000000000000000145b806105205750600160e060020a031982167f691f343100000000000000000000000000000000000000000000000000000000145b806105545750600160e060020a031982167f2203ab5600000000000000000000000000000000000000000000000000000000145b806105885750600160e060020a031982167fc869023300000000000000000000000000000000000000000000000000000000145b806105bc5750600160e060020a031982167f59d1d43c00000000000000000000000000000000000000000000000000000000145b806105f05750600160e060020a031982167f01ffc9a700000000000000000000000000000000000000000000000000000000145b92915050565b600080548491600160a060020a033381169216906302571be39084906040516020015260405160e060020a63ffffffff84160281526004810191909152602401602060405180830381600087803b151561064f57600080fd5b6102c65a03f1151561066057600080fd5b50505060405180519050600160a060020a031614151561067f57600080fd5b6000848152600160205260409081902083916005909101908590518082805190602001908083835b602083106106c65780518252601f1990920191602091820191016106a7565b6001836020036101000a038019825116818451168082178552505050505050905001915050908152602001604051809103902090805161070a929160200190611085565b50826040518082805190602001908083835b6020831061073b5780518252601f19909201916020918201910161071c565b6001836020036101000a0380198251168184511617909252505050919091019250604091505051908190039020847fd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a75508560405160208082528190810183818151815260200191508051906020019080838360005b838110156107c75780820151838201526020016107af565b50505050905090810190601f1680156107f45780820380516001836020036101000a031916815260200191505b509250505060405180910390a350505050565b6000610811611103565b60008481526001602081905260409091209092505b838311610924578284161580159061085f5750600083815260068201602052604081205460026000196101006001841615020190911604115b15610919578060060160008481526020019081526020016000208054600181600116156101000203166002900480601f01602080910402602001604051908101604052809291908181526020018280546001816001161561010002031660029004801561090d5780601f106108e25761010080835404028352916020019161090d565b820191906000526020600020905b8154815290600101906020018083116108f057829003601f168201915b50505050509150610929565b600290920291610826565b600092505b509250929050565b600080548491600160a060020a033381169216906302571be39084906040516020015260405160e060020a63ffffffff84160281526004810191909152602401602060405180830381600087803b151561098a57600080fd5b6102c65a03f1151561099b57600080fd5b50505060405180519050600160a060020a03161415156109ba57600080fd5b6040805190810160409081528482526020808301859052600087815260019091522060030181518155602082015160019091015550837f1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46848460405191825260208201526040908101905180910390a250505050565b6000908152600160208190526040909120015490565b600090815260016020526040902054600160a060020a031690565b610a69611103565b60008381526001602052604090819020600501908390518082805190602001908083835b60208310610aac5780518252601f199092019160209182019101610a8d565b6001836020036101000a03801982511681845116808217855250505050505090500191505090815260200160405180910390208054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610b735780601f10610b4857610100808354040283529160200191610b73565b820191906000526020600020905b815481529060010190602001808311610b5657829003601f168201915b5050505050905092915050565b600080548491600160a060020a033381169216906302571be39084906040516020015260405160e060020a63ffffffff84160281526004810191909152602401602060405180830381600087803b1515610bd957600080fd5b6102c65a03f11515610bea57600080fd5b50505060405180519050600160a060020a0316141515610c0957600080fd5b6000198301831615610c1a57600080fd5b60008481526001602090815260408083208684526006019091529020828051610c47929160200190611085565b5082847faa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe360405160405180910390a350505050565b610c84611103565b6001600083600019166000191681526020019081526020016000206002018054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610d365780601f10610d0b57610100808354040283529160200191610d36565b820191906000526020600020905b815481529060010190602001808311610d1957829003601f168201915b50505050509050919050565b600080548391600160a060020a033381169216906302571be39084906040516020015260405160e060020a63ffffffff84160281526004810191909152602401602060405180830381600087803b1515610d9b57600080fd5b6102c65a03f11515610dac57600080fd5b50505060405180519050600160a060020a0316141515610dcb57600080fd5b6000838152600160205260409020600201828051610ded929160200190611085565b50827fb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f78360405160208082528190810183818151815260200191508051906020019080838360005b83811015610e4d578082015183820152602001610e35565b50505050905090810190601f168015610e7a5780820380516001836020036101000a031916815260200191505b509250505060405180910390a2505050565b600080548391600160a060020a033381169216906302571be39084906040516020015260405160e060020a63ffffffff84160281526004810191909152602401602060405180830381600087803b1515610ee557600080fd5b6102c65a03f11515610ef657600080fd5b50505060405180519050600160a060020a0316141515610f1557600080fd5b6000838152600160208190526040918290200183905583907f0424b6fe0d9c3bdbece0e7879dc241bb0c22e900be8b6c168b4ee08bd9bf83bc9084905190815260200160405180910390a2505050565b600090815260016020526040902060038101546004909101549091565b600080548391600160a060020a033381169216906302571be39084906040516020015260405160e060020a63ffffffff84160281526004810191909152602401602060405180830381600087803b1515610fdb57600080fd5b6102c65a03f11515610fec57600080fd5b50505060405180519050600160a060020a031614151561100b57600080fd5b60008381526001602052604090819020805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a03851617905583907f52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd290849051600160a060020a03909116815260200160405180910390a2505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106110c657805160ff19168380011785556110f3565b828001600101855582156110f3579182015b828111156110f35782518255916020019190600101906110d8565b506110ff929150611115565b5090565b60206040519081016040526000815290565b61112f91905b808211156110ff576000815560010161111b565b905600a165627a7a72305820691c9aa3f737ab0ca25e23bc35cc10d4b93067d8a1fc5c9266b66365e32ed85a0029`

// DeployPublicResolver deploys a new Ethereum contract, binding an instance of PublicResolver to it.
func DeployPublicResolver(auth *bind.TransactOpts, backend bind.ContractBackend, ensAddr common.Address) (common.Address, *types.Transaction, *PublicResolver, error) {
	parsed, err := abi.JSON(strings.NewReader(PublicResolverABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(PublicResolverBin), backend, ensAddr)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PublicResolver{PublicResolverCaller: PublicResolverCaller{contract: contract}, PublicResolverTransactor: PublicResolverTransactor{contract: contract}}, nil
}

// PublicResolver is an auto generated Go binding around an Ethereum contract.
type PublicResolver struct {
	PublicResolverCaller     // Read-only binding to the contract
	PublicResolverTransactor // Write-only binding to the contract
}

// PublicResolverCaller is an auto generated read-only Go binding around an Ethereum contract.
type PublicResolverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PublicResolverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PublicResolverSession struct {
	Contract     *PublicResolver   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PublicResolverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PublicResolverCallerSession struct {
	Contract *PublicResolverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// PublicResolverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PublicResolverTransactorSession struct {
	Contract     *PublicResolverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// PublicResolverRaw is an auto generated low-level Go binding around an Ethereum contract.
type PublicResolverRaw struct {
	Contract *PublicResolver // Generic contract binding to access the raw methods on
}

// PublicResolverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PublicResolverCallerRaw struct {
	Contract *PublicResolverCaller // Generic read-only contract binding to access the raw methods on
}

// PublicResolverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PublicResolverTransactorRaw struct {
	Contract *PublicResolverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPublicResolver creates a new instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolver(address common.Address, backend bind.ContractBackend) (*PublicResolver, error) {
	contract, err := bindPublicResolver(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PublicResolver{PublicResolverCaller: PublicResolverCaller{contract: contract}, PublicResolverTransactor: PublicResolverTransactor{contract: contract}}, nil
}

// NewPublicResolverCaller creates a new read-only instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverCaller(address common.Address, caller bind.ContractCaller) (*PublicResolverCaller, error) {
	contract, err := bindPublicResolver(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &PublicResolverCaller{contract: contract}, nil
}

// NewPublicResolverTransactor creates a new write-only instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverTransactor(address common.Address, transactor bind.ContractTransactor) (*PublicResolverTransactor, error) {
	contract, err := bindPublicResolver(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &PublicResolverTransactor{contract: contract}, nil
}

// bindPublicResolver binds a generic wrapper to an already deployed contract.
func bindPublicResolver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PublicResolverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PublicResolver *PublicResolverRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PublicResolver.Contract.PublicResolverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PublicResolver *PublicResolverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PublicResolver.Contract.PublicResolverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PublicResolver *PublicResolverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PublicResolver.Contract.PublicResolverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PublicResolver *PublicResolverCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _PublicResolver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PublicResolver *PublicResolverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PublicResolver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PublicResolver *PublicResolverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PublicResolver.Contract.contract.Transact(opts, method, params...)
}

// ABI is a free data retrieval call binding the contract method 0x2203ab56.
//
// Solidity: function ABI(node bytes32, contentTypes uint256) constant returns(contentType uint256, data bytes)
func (_PublicResolver *PublicResolverCaller) ABI(opts *bind.CallOpts, node [32]byte, contentTypes *big.Int) (struct {
	ContentType *big.Int
	Data        []byte
}, error) {
	ret := new(struct {
		ContentType *big.Int
		Data        []byte
	})
	out := ret
	err := _PublicResolver.contract.Call(opts, out, "ABI", node, contentTypes)
	return *ret, err
}

// ABI is a free data retrieval call binding the contract method 0x2203ab56.
//
// Solidity: function ABI(node bytes32, contentTypes uint256) constant returns(contentType uint256, data bytes)
func (_PublicResolver *PublicResolverSession) ABI(node [32]byte, contentTypes *big.Int) (struct {
	ContentType *big.Int
	Data        []byte
}, error) {
	return _PublicResolver.Contract.ABI(&_PublicResolver.CallOpts, node, contentTypes)
}

// ABI is a free data retrieval call binding the contract method 0x2203ab56.
//
// Solidity: function ABI(node bytes32, contentTypes uint256) constant returns(contentType uint256, data bytes)
func (_PublicResolver *PublicResolverCallerSession) ABI(node [32]byte, contentTypes *big.Int) (struct {
	ContentType *big.Int
	Data        []byte
}, error) {
	return _PublicResolver.Contract.ABI(&_PublicResolver.CallOpts, node, contentTypes)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_PublicResolver *PublicResolverCaller) Addr(opts *bind.CallOpts, node [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "addr", node)
	return *ret0, err
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_PublicResolver *PublicResolverSession) Addr(node [32]byte) (common.Address, error) {
	return _PublicResolver.Contract.Addr(&_PublicResolver.CallOpts, node)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(node bytes32) constant returns(ret address)
func (_PublicResolver *PublicResolverCallerSession) Addr(node [32]byte) (common.Address, error) {
	return _PublicResolver.Contract.Addr(&_PublicResolver.CallOpts, node)
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_PublicResolver *PublicResolverCaller) Content(opts *bind.CallOpts, node [32]byte) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "content", node)
	return *ret0, err
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_PublicResolver *PublicResolverSession) Content(node [32]byte) ([32]byte, error) {
	return _PublicResolver.Contract.Content(&_PublicResolver.CallOpts, node)
}

// Content is a free data retrieval call binding the contract method 0x2dff6941.
//
// Solidity: function content(node bytes32) constant returns(ret bytes32)
func (_PublicResolver *PublicResolverCallerSession) Content(node [32]byte) ([32]byte, error) {
	return _PublicResolver.Contract.Content(&_PublicResolver.CallOpts, node)
}

// Name is a free data retrieval call binding the contract method 0x691f3431.
//
// Solidity: function name(node bytes32) constant returns(ret string)
func (_PublicResolver *PublicResolverCaller) Name(opts *bind.CallOpts, node [32]byte) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "name", node)
	return *ret0, err
}

// Name is a free data retrieval call binding the contract method 0x691f3431.
//
// Solidity: function name(node bytes32) constant returns(ret string)
func (_PublicResolver *PublicResolverSession) Name(node [32]byte) (string, error) {
	return _PublicResolver.Contract.Name(&_PublicResolver.CallOpts, node)
}

// Name is a free data retrieval call binding the contract method 0x691f3431.
//
// Solidity: function name(node bytes32) constant returns(ret string)
func (_PublicResolver *PublicResolverCallerSession) Name(node [32]byte) (string, error) {
	return _PublicResolver.Contract.Name(&_PublicResolver.CallOpts, node)
}

// Pubkey is a free data retrieval call binding the contract method 0xc8690233.
//
// Solidity: function pubkey(node bytes32) constant returns(x bytes32, y bytes32)
func (_PublicResolver *PublicResolverCaller) Pubkey(opts *bind.CallOpts, node [32]byte) (struct {
	X [32]byte
	Y [32]byte
}, error) {
	ret := new(struct {
		X [32]byte
		Y [32]byte
	})
	out := ret
	err := _PublicResolver.contract.Call(opts, out, "pubkey", node)
	return *ret, err
}

// Pubkey is a free data retrieval call binding the contract method 0xc8690233.
//
// Solidity: function pubkey(node bytes32) constant returns(x bytes32, y bytes32)
func (_PublicResolver *PublicResolverSession) Pubkey(node [32]byte) (struct {
	X [32]byte
	Y [32]byte
}, error) {
	return _PublicResolver.Contract.Pubkey(&_PublicResolver.CallOpts, node)
}

// Pubkey is a free data retrieval call binding the contract method 0xc8690233.
//
// Solidity: function pubkey(node bytes32) constant returns(x bytes32, y bytes32)
func (_PublicResolver *PublicResolverCallerSession) Pubkey(node [32]byte) (struct {
	X [32]byte
	Y [32]byte
}, error) {
	return _PublicResolver.Contract.Pubkey(&_PublicResolver.CallOpts, node)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_PublicResolver *PublicResolverCaller) SupportsInterface(opts *bind.CallOpts, interfaceID [4]byte) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "supportsInterface", interfaceID)
	return *ret0, err
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_PublicResolver *PublicResolverSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _PublicResolver.Contract.SupportsInterface(&_PublicResolver.CallOpts, interfaceID)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(interfaceID bytes4) constant returns(bool)
func (_PublicResolver *PublicResolverCallerSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _PublicResolver.Contract.SupportsInterface(&_PublicResolver.CallOpts, interfaceID)
}

// Text is a free data retrieval call binding the contract method 0x59d1d43c.
//
// Solidity: function text(node bytes32, key string) constant returns(ret string)
func (_PublicResolver *PublicResolverCaller) Text(opts *bind.CallOpts, node [32]byte, key string) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "text", node, key)
	return *ret0, err
}

// Text is a free data retrieval call binding the contract method 0x59d1d43c.
//
// Solidity: function text(node bytes32, key string) constant returns(ret string)
func (_PublicResolver *PublicResolverSession) Text(node [32]byte, key string) (string, error) {
	return _PublicResolver.Contract.Text(&_PublicResolver.CallOpts, node, key)
}

// Text is a free data retrieval call binding the contract method 0x59d1d43c.
//
// Solidity: function text(node bytes32, key string) constant returns(ret string)
func (_PublicResolver *PublicResolverCallerSession) Text(node [32]byte, key string) (string, error) {
	return _PublicResolver.Contract.Text(&_PublicResolver.CallOpts, node, key)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(node bytes32, contentType uint256, data bytes) returns()
func (_PublicResolver *PublicResolverTransactor) SetABI(opts *bind.TransactOpts, node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setABI", node, contentType, data)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(node bytes32, contentType uint256, data bytes) returns()
func (_PublicResolver *PublicResolverSession) SetABI(node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetABI(&_PublicResolver.TransactOpts, node, contentType, data)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(node bytes32, contentType uint256, data bytes) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetABI(node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetABI(&_PublicResolver.TransactOpts, node, contentType, data)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(node bytes32, addr address) returns()
func (_PublicResolver *PublicResolverTransactor) SetAddr(opts *bind.TransactOpts, node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setAddr", node, addr)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(node bytes32, addr address) returns()
func (_PublicResolver *PublicResolverSession) SetAddr(node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetAddr(&_PublicResolver.TransactOpts, node, addr)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(node bytes32, addr address) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetAddr(node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetAddr(&_PublicResolver.TransactOpts, node, addr)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(node bytes32, hash bytes32) returns()
func (_PublicResolver *PublicResolverTransactor) SetContent(opts *bind.TransactOpts, node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setContent", node, hash)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(node bytes32, hash bytes32) returns()
func (_PublicResolver *PublicResolverSession) SetContent(node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetContent(&_PublicResolver.TransactOpts, node, hash)
}

// SetContent is a paid mutator transaction binding the contract method 0xc3d014d6.
//
// Solidity: function setContent(node bytes32, hash bytes32) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetContent(node [32]byte, hash [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetContent(&_PublicResolver.TransactOpts, node, hash)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(node bytes32, name string) returns()
func (_PublicResolver *PublicResolverTransactor) SetName(opts *bind.TransactOpts, node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setName", node, name)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(node bytes32, name string) returns()
func (_PublicResolver *PublicResolverSession) SetName(node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetName(&_PublicResolver.TransactOpts, node, name)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(node bytes32, name string) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetName(node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetName(&_PublicResolver.TransactOpts, node, name)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(node bytes32, x bytes32, y bytes32) returns()
func (_PublicResolver *PublicResolverTransactor) SetPubkey(opts *bind.TransactOpts, node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setPubkey", node, x, y)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(node bytes32, x bytes32, y bytes32) returns()
func (_PublicResolver *PublicResolverSession) SetPubkey(node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetPubkey(&_PublicResolver.TransactOpts, node, x, y)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(node bytes32, x bytes32, y bytes32) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetPubkey(node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetPubkey(&_PublicResolver.TransactOpts, node, x, y)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(node bytes32, key string, value string) returns()
func (_PublicResolver *PublicResolverTransactor) SetText(opts *bind.TransactOpts, node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setText", node, key, value)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(node bytes32, key string, value string) returns()
func (_PublicResolver *PublicResolverSession) SetText(node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetText(&_PublicResolver.TransactOpts, node, key, value)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(node bytes32, key string, value string) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetText(node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetText(&_PublicResolver.TransactOpts, node, key, value)
}
