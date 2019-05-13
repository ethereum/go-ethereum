// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// PublicResolverABI is the input ABI used to generate the binding from.
const PublicResolverABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"interfaceID\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"pure\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"key\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"string\"}],\"name\":\"setText\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"contentTypes\",\"type\":\"uint256\"}],\"name\":\"ABI\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"x\",\"type\":\"bytes32\"},{\"name\":\"y\",\"type\":\"bytes32\"}],\"name\":\"setPubkey\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"hash\",\"type\":\"bytes\"}],\"name\":\"setContenthash\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"addr\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"key\",\"type\":\"string\"}],\"name\":\"text\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"contentType\",\"type\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"setABI\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"name\",\"type\":\"string\"}],\"name\":\"setName\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"contenthash\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"}],\"name\":\"pubkey\",\"outputs\":[{\"name\":\"x\",\"type\":\"bytes32\"},{\"name\":\"y\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"node\",\"type\":\"bytes32\"},{\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"setAddr\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"ensAddr\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"a\",\"type\":\"address\"}],\"name\":\"AddrChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"}],\"name\":\"NameChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":true,\"name\":\"contentType\",\"type\":\"uint256\"}],\"name\":\"ABIChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"x\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"y\",\"type\":\"bytes32\"}],\"name\":\"PubkeyChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"indexedKey\",\"type\":\"string\"},{\"indexed\":false,\"name\":\"key\",\"type\":\"string\"}],\"name\":\"TextChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"node\",\"type\":\"bytes32\"},{\"indexed\":false,\"name\":\"hash\",\"type\":\"bytes\"}],\"name\":\"ContenthashChanged\",\"type\":\"event\"}]"

// PublicResolverBin is the compiled bytecode used for deploying new contracts.
const PublicResolverBin = `0x608060405234801561001057600080fd5b506040516020806112ce8339810180604052602081101561003057600080fd5b505160008054600160a060020a03909216600160a060020a031990921691909117905561126c806100626000396000f3fe6080604052600436106100c45763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166301ffc9a781146100c957806310f13a8c146101115780632203ab56146101e957806329cd62ea14610298578063304e6ade146102ce5780633b3b57de1461035257806359d1d43c14610398578063623195b014610491578063691f34311461051a5780637737221314610544578063bc1c58d1146105c8578063c8690233146105f2578063d5fa2b0014610635575b600080fd5b3480156100d557600080fd5b506100fd600480360360208110156100ec57600080fd5b5035600160e060020a03191661066e565b604080519115158252519081900360200190f35b34801561011d57600080fd5b506101e76004803603606081101561013457600080fd5b8135919081019060408101602082013564010000000081111561015657600080fd5b82018360208201111561016857600080fd5b8035906020019184600183028401116401000000008311171561018a57600080fd5b9193909290916020810190356401000000008111156101a857600080fd5b8201836020820111156101ba57600080fd5b803590602001918460018302840111640100000000831117156101dc57600080fd5b5090925090506107db565b005b3480156101f557600080fd5b506102196004803603604081101561020c57600080fd5b508035906020013561094d565b6040518083815260200180602001828103825283818151815260200191508051906020019080838360005b8381101561025c578181015183820152602001610244565b50505050905090810190601f1680156102895780820380516001836020036101000a031916815260200191505b50935050505060405180910390f35b3480156102a457600080fd5b506101e7600480360360608110156102bb57600080fd5b5080359060208101359060400135610a65565b3480156102da57600080fd5b506101e7600480360360408110156102f157600080fd5b8135919081019060408101602082013564010000000081111561031357600080fd5b82018360208201111561032557600080fd5b8035906020019184600183028401116401000000008311171561034757600080fd5b509092509050610b65565b34801561035e57600080fd5b5061037c6004803603602081101561037557600080fd5b5035610c7b565b60408051600160a060020a039092168252519081900360200190f35b3480156103a457600080fd5b5061041c600480360360408110156103bb57600080fd5b813591908101906040810160208201356401000000008111156103dd57600080fd5b8201836020820111156103ef57600080fd5b8035906020019184600183028401116401000000008311171561041157600080fd5b509092509050610c96565b6040805160208082528351818301528351919283929083019185019080838360005b8381101561045657818101518382015260200161043e565b50505050905090810190601f1680156104835780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561049d57600080fd5b506101e7600480360360608110156104b457600080fd5b8135916020810135918101906060810160408201356401000000008111156104db57600080fd5b8201836020820111156104ed57600080fd5b8035906020019184600183028401116401000000008311171561050f57600080fd5b509092509050610d60565b34801561052657600080fd5b5061041c6004803603602081101561053d57600080fd5b5035610e5f565b34801561055057600080fd5b506101e76004803603604081101561056757600080fd5b8135919081019060408101602082013564010000000081111561058957600080fd5b82018360208201111561059b57600080fd5b803590602001918460018302840111640100000000831117156105bd57600080fd5b509092509050610f01565b3480156105d457600080fd5b5061041c600480360360208110156105eb57600080fd5b5035611018565b3480156105fe57600080fd5b5061061c6004803603602081101561061557600080fd5b5035611084565b6040805192835260208301919091528051918290030190f35b34801561064157600080fd5b506101e76004803603604081101561065857600080fd5b5080359060200135600160a060020a03166110a1565b6000600160e060020a031982167f3b3b57de0000000000000000000000000000000000000000000000000000000014806106d15750600160e060020a031982167f691f343100000000000000000000000000000000000000000000000000000000145b806107055750600160e060020a031982167f2203ab5600000000000000000000000000000000000000000000000000000000145b806107395750600160e060020a031982167fc869023300000000000000000000000000000000000000000000000000000000145b8061076d5750600160e060020a031982167f59d1d43c00000000000000000000000000000000000000000000000000000000145b806107a15750600160e060020a031982167fbc1c58d100000000000000000000000000000000000000000000000000000000145b806107d55750600160e060020a031982167f01ffc9a700000000000000000000000000000000000000000000000000000000145b92915050565b6000546040805160e060020a6302571be302815260048101889052905187923392600160a060020a03909116916302571be391602480820192602092909190829003018186803b15801561082e57600080fd5b505afa158015610842573d6000803e3d6000fd5b505050506040513d602081101561085857600080fd5b5051600160a060020a03161461086d57600080fd5b8282600160008981526020019081526020016000206004018787604051808383808284378083019250505092505050908152602001604051809103902091906108b79291906111a5565b50857fd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a7550868688886040518080602001806020018381038352878782818152602001925080828437600083820152601f01601f191690910184810383528581526020019050858580828437600083820152604051601f909101601f19169092018290039850909650505050505050a2505050505050565b600082815260016020819052604082206060915b848111610a53578085161580159061099a5750600081815260058301602052604081205460026000196101006001841615020190911604115b15610a4b57600081815260058301602090815260409182902080548351601f6002600019610100600186161502019093169290920491820184900484028101840190945280845284939192839190830182828015610a395780601f10610a0e57610100808354040283529160200191610a39565b820191906000526020600020905b815481529060010190602001808311610a1c57829003601f168201915b50505050509050935093505050610a5e565b600202610961565b506000925060609150505b9250929050565b6000546040805160e060020a6302571be302815260048101869052905185923392600160a060020a03909116916302571be391602480820192602092909190829003018186803b158015610ab857600080fd5b505afa158015610acc573d6000803e3d6000fd5b505050506040513d6020811015610ae257600080fd5b5051600160a060020a031614610af757600080fd5b604080518082018252848152602080820185815260008881526001835284902092516002840155516003909201919091558151858152908101849052815186927f1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46928290030190a250505050565b6000546040805160e060020a6302571be302815260048101869052905185923392600160a060020a03909116916302571be391602480820192602092909190829003018186803b158015610bb857600080fd5b505afa158015610bcc573d6000803e3d6000fd5b505050506040513d6020811015610be257600080fd5b5051600160a060020a031614610bf757600080fd5b6000848152600160205260409020610c139060060184846111a5565b50837fe379c1624ed7e714cc0937528a32359d69d5281337765313dba4e081b72d7578848460405180806020018281038252848482818152602001925080828437600083820152604051601f909101601f19169092018290039550909350505050a250505050565b600090815260016020526040902054600160a060020a031690565b6060600160008581526020019081526020016000206004018383604051808383808284379190910194855250506040805160209481900385018120805460026001821615610100026000190190911604601f81018790048702830187019093528282529094909350909150830182828015610d525780601f10610d2757610100808354040283529160200191610d52565b820191906000526020600020905b815481529060010190602001808311610d3557829003601f168201915b505050505090509392505050565b6000546040805160e060020a6302571be302815260048101879052905186923392600160a060020a03909116916302571be391602480820192602092909190829003018186803b158015610db357600080fd5b505afa158015610dc7573d6000803e3d6000fd5b505050506040513d6020811015610ddd57600080fd5b5051600160a060020a031614610df257600080fd5b6000198401841615610e0357600080fd5b60008581526001602090815260408083208784526005019091529020610e2a9084846111a5565b50604051849086907faa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe390600090a35050505050565b600081815260016020818152604092839020820180548451600294821615610100026000190190911693909304601f81018390048302840183019094528383526060939091830182828015610ef55780601f10610eca57610100808354040283529160200191610ef5565b820191906000526020600020905b815481529060010190602001808311610ed857829003601f168201915b50505050509050919050565b6000546040805160e060020a6302571be302815260048101869052905185923392600160a060020a03909116916302571be391602480820192602092909190829003018186803b158015610f5457600080fd5b505afa158015610f68573d6000803e3d6000fd5b505050506040513d6020811015610f7e57600080fd5b5051600160a060020a031614610f9357600080fd5b6000848152600160208190526040909120610fb0910184846111a5565b50837fb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f7848460405180806020018281038252848482818152602001925080828437600083820152604051601f909101601f19169092018290039550909350505050a250505050565b60008181526001602081815260409283902060060180548451600294821615610100026000190190911693909304601f81018390048302840183019094528383526060939091830182828015610ef55780601f10610eca57610100808354040283529160200191610ef5565b600090815260016020526040902060028101546003909101549091565b6000546040805160e060020a6302571be302815260048101859052905184923392600160a060020a03909116916302571be391602480820192602092909190829003018186803b1580156110f457600080fd5b505afa158015611108573d6000803e3d6000fd5b505050506040513d602081101561111e57600080fd5b5051600160a060020a03161461113357600080fd5b600083815260016020908152604091829020805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0386169081179091558251908152915185927f52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd292908290030190a2505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106111e65782800160ff19823516178555611213565b82800160010185558215611213579182015b828111156112135782358255916020019190600101906111f8565b5061121f929150611223565b5090565b61123d91905b8082111561121f5760008155600101611229565b9056fea165627a7a7230582047f310fc746ab2e282cf63ba794d20abb361f9284c6c5f2a2e26151e5b7fab600029`

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
	return address, tx, &PublicResolver{PublicResolverCaller: PublicResolverCaller{contract: contract}, PublicResolverTransactor: PublicResolverTransactor{contract: contract}, PublicResolverFilterer: PublicResolverFilterer{contract: contract}}, nil
}

// PublicResolver is an auto generated Go binding around an Ethereum contract.
type PublicResolver struct {
	PublicResolverCaller     // Read-only binding to the contract
	PublicResolverTransactor // Write-only binding to the contract
	PublicResolverFilterer   // Log filterer for contract events
}

// PublicResolverCaller is an auto generated read-only Go binding around an Ethereum contract.
type PublicResolverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PublicResolverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PublicResolverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PublicResolverFilterer struct {
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
	contract, err := bindPublicResolver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PublicResolver{PublicResolverCaller: PublicResolverCaller{contract: contract}, PublicResolverTransactor: PublicResolverTransactor{contract: contract}, PublicResolverFilterer: PublicResolverFilterer{contract: contract}}, nil
}

// NewPublicResolverCaller creates a new read-only instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverCaller(address common.Address, caller bind.ContractCaller) (*PublicResolverCaller, error) {
	contract, err := bindPublicResolver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PublicResolverCaller{contract: contract}, nil
}

// NewPublicResolverTransactor creates a new write-only instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverTransactor(address common.Address, transactor bind.ContractTransactor) (*PublicResolverTransactor, error) {
	contract, err := bindPublicResolver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PublicResolverTransactor{contract: contract}, nil
}

// NewPublicResolverFilterer creates a new log filterer instance of PublicResolver, bound to a specific deployed contract.
func NewPublicResolverFilterer(address common.Address, filterer bind.ContractFilterer) (*PublicResolverFilterer, error) {
	contract, err := bindPublicResolver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PublicResolverFilterer{contract: contract}, nil
}

// bindPublicResolver binds a generic wrapper to an already deployed contract.
func bindPublicResolver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PublicResolverABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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
// Solidity: function ABI(bytes32 node, uint256 contentTypes) constant returns(uint256, bytes)
func (_PublicResolver *PublicResolverCaller) ABI(opts *bind.CallOpts, node [32]byte, contentTypes *big.Int) (*big.Int, []byte, error) {
	var (
		ret0 = new(*big.Int)
		ret1 = new([]byte)
	)
	out := &[]interface{}{
		ret0,
		ret1,
	}
	err := _PublicResolver.contract.Call(opts, out, "ABI", node, contentTypes)
	return *ret0, *ret1, err
}

// ABI is a free data retrieval call binding the contract method 0x2203ab56.
//
// Solidity: function ABI(bytes32 node, uint256 contentTypes) constant returns(uint256, bytes)
func (_PublicResolver *PublicResolverSession) ABI(node [32]byte, contentTypes *big.Int) (*big.Int, []byte, error) {
	return _PublicResolver.Contract.ABI(&_PublicResolver.CallOpts, node, contentTypes)
}

// ABI is a free data retrieval call binding the contract method 0x2203ab56.
//
// Solidity: function ABI(bytes32 node, uint256 contentTypes) constant returns(uint256, bytes)
func (_PublicResolver *PublicResolverCallerSession) ABI(node [32]byte, contentTypes *big.Int) (*big.Int, []byte, error) {
	return _PublicResolver.Contract.ABI(&_PublicResolver.CallOpts, node, contentTypes)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(bytes32 node) constant returns(address)
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
// Solidity: function addr(bytes32 node) constant returns(address)
func (_PublicResolver *PublicResolverSession) Addr(node [32]byte) (common.Address, error) {
	return _PublicResolver.Contract.Addr(&_PublicResolver.CallOpts, node)
}

// Addr is a free data retrieval call binding the contract method 0x3b3b57de.
//
// Solidity: function addr(bytes32 node) constant returns(address)
func (_PublicResolver *PublicResolverCallerSession) Addr(node [32]byte) (common.Address, error) {
	return _PublicResolver.Contract.Addr(&_PublicResolver.CallOpts, node)
}

// Contenthash is a free data retrieval call binding the contract method 0xbc1c58d1.
//
// Solidity: function contenthash(bytes32 node) constant returns(bytes)
func (_PublicResolver *PublicResolverCaller) Contenthash(opts *bind.CallOpts, node [32]byte) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _PublicResolver.contract.Call(opts, out, "contenthash", node)
	return *ret0, err
}

// Contenthash is a free data retrieval call binding the contract method 0xbc1c58d1.
//
// Solidity: function contenthash(bytes32 node) constant returns(bytes)
func (_PublicResolver *PublicResolverSession) Contenthash(node [32]byte) ([]byte, error) {
	return _PublicResolver.Contract.Contenthash(&_PublicResolver.CallOpts, node)
}

// Contenthash is a free data retrieval call binding the contract method 0xbc1c58d1.
//
// Solidity: function contenthash(bytes32 node) constant returns(bytes)
func (_PublicResolver *PublicResolverCallerSession) Contenthash(node [32]byte) ([]byte, error) {
	return _PublicResolver.Contract.Contenthash(&_PublicResolver.CallOpts, node)
}

// Name is a free data retrieval call binding the contract method 0x691f3431.
//
// Solidity: function name(bytes32 node) constant returns(string)
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
// Solidity: function name(bytes32 node) constant returns(string)
func (_PublicResolver *PublicResolverSession) Name(node [32]byte) (string, error) {
	return _PublicResolver.Contract.Name(&_PublicResolver.CallOpts, node)
}

// Name is a free data retrieval call binding the contract method 0x691f3431.
//
// Solidity: function name(bytes32 node) constant returns(string)
func (_PublicResolver *PublicResolverCallerSession) Name(node [32]byte) (string, error) {
	return _PublicResolver.Contract.Name(&_PublicResolver.CallOpts, node)
}

// Pubkey is a free data retrieval call binding the contract method 0xc8690233.
//
// Solidity: function pubkey(bytes32 node) constant returns(bytes32 x, bytes32 y)
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
// Solidity: function pubkey(bytes32 node) constant returns(bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverSession) Pubkey(node [32]byte) (struct {
	X [32]byte
	Y [32]byte
}, error) {
	return _PublicResolver.Contract.Pubkey(&_PublicResolver.CallOpts, node)
}

// Pubkey is a free data retrieval call binding the contract method 0xc8690233.
//
// Solidity: function pubkey(bytes32 node) constant returns(bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverCallerSession) Pubkey(node [32]byte) (struct {
	X [32]byte
	Y [32]byte
}, error) {
	return _PublicResolver.Contract.Pubkey(&_PublicResolver.CallOpts, node)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceID) constant returns(bool)
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
// Solidity: function supportsInterface(bytes4 interfaceID) constant returns(bool)
func (_PublicResolver *PublicResolverSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _PublicResolver.Contract.SupportsInterface(&_PublicResolver.CallOpts, interfaceID)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceID) constant returns(bool)
func (_PublicResolver *PublicResolverCallerSession) SupportsInterface(interfaceID [4]byte) (bool, error) {
	return _PublicResolver.Contract.SupportsInterface(&_PublicResolver.CallOpts, interfaceID)
}

// Text is a free data retrieval call binding the contract method 0x59d1d43c.
//
// Solidity: function text(bytes32 node, string key) constant returns(string)
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
// Solidity: function text(bytes32 node, string key) constant returns(string)
func (_PublicResolver *PublicResolverSession) Text(node [32]byte, key string) (string, error) {
	return _PublicResolver.Contract.Text(&_PublicResolver.CallOpts, node, key)
}

// Text is a free data retrieval call binding the contract method 0x59d1d43c.
//
// Solidity: function text(bytes32 node, string key) constant returns(string)
func (_PublicResolver *PublicResolverCallerSession) Text(node [32]byte, key string) (string, error) {
	return _PublicResolver.Contract.Text(&_PublicResolver.CallOpts, node, key)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(bytes32 node, uint256 contentType, bytes data) returns()
func (_PublicResolver *PublicResolverTransactor) SetABI(opts *bind.TransactOpts, node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setABI", node, contentType, data)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(bytes32 node, uint256 contentType, bytes data) returns()
func (_PublicResolver *PublicResolverSession) SetABI(node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetABI(&_PublicResolver.TransactOpts, node, contentType, data)
}

// SetABI is a paid mutator transaction binding the contract method 0x623195b0.
//
// Solidity: function setABI(bytes32 node, uint256 contentType, bytes data) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetABI(node [32]byte, contentType *big.Int, data []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetABI(&_PublicResolver.TransactOpts, node, contentType, data)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(bytes32 node, address addr) returns()
func (_PublicResolver *PublicResolverTransactor) SetAddr(opts *bind.TransactOpts, node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setAddr", node, addr)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(bytes32 node, address addr) returns()
func (_PublicResolver *PublicResolverSession) SetAddr(node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetAddr(&_PublicResolver.TransactOpts, node, addr)
}

// SetAddr is a paid mutator transaction binding the contract method 0xd5fa2b00.
//
// Solidity: function setAddr(bytes32 node, address addr) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetAddr(node [32]byte, addr common.Address) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetAddr(&_PublicResolver.TransactOpts, node, addr)
}

// SetContenthash is a paid mutator transaction binding the contract method 0x304e6ade.
//
// Solidity: function setContenthash(bytes32 node, bytes hash) returns()
func (_PublicResolver *PublicResolverTransactor) SetContenthash(opts *bind.TransactOpts, node [32]byte, hash []byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setContenthash", node, hash)
}

// SetContenthash is a paid mutator transaction binding the contract method 0x304e6ade.
//
// Solidity: function setContenthash(bytes32 node, bytes hash) returns()
func (_PublicResolver *PublicResolverSession) SetContenthash(node [32]byte, hash []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetContenthash(&_PublicResolver.TransactOpts, node, hash)
}

// SetContenthash is a paid mutator transaction binding the contract method 0x304e6ade.
//
// Solidity: function setContenthash(bytes32 node, bytes hash) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetContenthash(node [32]byte, hash []byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetContenthash(&_PublicResolver.TransactOpts, node, hash)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(bytes32 node, string name) returns()
func (_PublicResolver *PublicResolverTransactor) SetName(opts *bind.TransactOpts, node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setName", node, name)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(bytes32 node, string name) returns()
func (_PublicResolver *PublicResolverSession) SetName(node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetName(&_PublicResolver.TransactOpts, node, name)
}

// SetName is a paid mutator transaction binding the contract method 0x77372213.
//
// Solidity: function setName(bytes32 node, string name) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetName(node [32]byte, name string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetName(&_PublicResolver.TransactOpts, node, name)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(bytes32 node, bytes32 x, bytes32 y) returns()
func (_PublicResolver *PublicResolverTransactor) SetPubkey(opts *bind.TransactOpts, node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setPubkey", node, x, y)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(bytes32 node, bytes32 x, bytes32 y) returns()
func (_PublicResolver *PublicResolverSession) SetPubkey(node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetPubkey(&_PublicResolver.TransactOpts, node, x, y)
}

// SetPubkey is a paid mutator transaction binding the contract method 0x29cd62ea.
//
// Solidity: function setPubkey(bytes32 node, bytes32 x, bytes32 y) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetPubkey(node [32]byte, x [32]byte, y [32]byte) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetPubkey(&_PublicResolver.TransactOpts, node, x, y)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(bytes32 node, string key, string value) returns()
func (_PublicResolver *PublicResolverTransactor) SetText(opts *bind.TransactOpts, node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.contract.Transact(opts, "setText", node, key, value)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(bytes32 node, string key, string value) returns()
func (_PublicResolver *PublicResolverSession) SetText(node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetText(&_PublicResolver.TransactOpts, node, key, value)
}

// SetText is a paid mutator transaction binding the contract method 0x10f13a8c.
//
// Solidity: function setText(bytes32 node, string key, string value) returns()
func (_PublicResolver *PublicResolverTransactorSession) SetText(node [32]byte, key string, value string) (*types.Transaction, error) {
	return _PublicResolver.Contract.SetText(&_PublicResolver.TransactOpts, node, key, value)
}

// PublicResolverABIChangedIterator is returned from FilterABIChanged and is used to iterate over the raw logs and unpacked data for ABIChanged events raised by the PublicResolver contract.
type PublicResolverABIChangedIterator struct {
	Event *PublicResolverABIChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PublicResolverABIChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverABIChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PublicResolverABIChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PublicResolverABIChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverABIChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverABIChanged represents a ABIChanged event raised by the PublicResolver contract.
type PublicResolverABIChanged struct {
	Node        [32]byte
	ContentType *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterABIChanged is a free log retrieval operation binding the contract event 0xaa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe3.
//
// Solidity: event ABIChanged(bytes32 indexed node, uint256 indexed contentType)
func (_PublicResolver *PublicResolverFilterer) FilterABIChanged(opts *bind.FilterOpts, node [][32]byte, contentType []*big.Int) (*PublicResolverABIChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}
	var contentTypeRule []interface{}
	for _, contentTypeItem := range contentType {
		contentTypeRule = append(contentTypeRule, contentTypeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "ABIChanged", nodeRule, contentTypeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverABIChangedIterator{contract: _PublicResolver.contract, event: "ABIChanged", logs: logs, sub: sub}, nil
}

// WatchABIChanged is a free log subscription operation binding the contract event 0xaa121bbeef5f32f5961a2a28966e769023910fc9479059ee3495d4c1a696efe3.
//
// Solidity: event ABIChanged(bytes32 indexed node, uint256 indexed contentType)
func (_PublicResolver *PublicResolverFilterer) WatchABIChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverABIChanged, node [][32]byte, contentType []*big.Int) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}
	var contentTypeRule []interface{}
	for _, contentTypeItem := range contentType {
		contentTypeRule = append(contentTypeRule, contentTypeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "ABIChanged", nodeRule, contentTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverABIChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "ABIChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// PublicResolverAddrChangedIterator is returned from FilterAddrChanged and is used to iterate over the raw logs and unpacked data for AddrChanged events raised by the PublicResolver contract.
type PublicResolverAddrChangedIterator struct {
	Event *PublicResolverAddrChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PublicResolverAddrChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverAddrChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PublicResolverAddrChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PublicResolverAddrChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverAddrChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverAddrChanged represents a AddrChanged event raised by the PublicResolver contract.
type PublicResolverAddrChanged struct {
	Node [32]byte
	A    common.Address
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterAddrChanged is a free log retrieval operation binding the contract event 0x52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd2.
//
// Solidity: event AddrChanged(bytes32 indexed node, address a)
func (_PublicResolver *PublicResolverFilterer) FilterAddrChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverAddrChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "AddrChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverAddrChangedIterator{contract: _PublicResolver.contract, event: "AddrChanged", logs: logs, sub: sub}, nil
}

// WatchAddrChanged is a free log subscription operation binding the contract event 0x52d7d861f09ab3d26239d492e8968629f95e9e318cf0b73bfddc441522a15fd2.
//
// Solidity: event AddrChanged(bytes32 indexed node, address a)
func (_PublicResolver *PublicResolverFilterer) WatchAddrChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverAddrChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "AddrChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverAddrChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "AddrChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// PublicResolverContenthashChangedIterator is returned from FilterContenthashChanged and is used to iterate over the raw logs and unpacked data for ContenthashChanged events raised by the PublicResolver contract.
type PublicResolverContenthashChangedIterator struct {
	Event *PublicResolverContenthashChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PublicResolverContenthashChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverContenthashChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PublicResolverContenthashChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PublicResolverContenthashChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverContenthashChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverContenthashChanged represents a ContenthashChanged event raised by the PublicResolver contract.
type PublicResolverContenthashChanged struct {
	Node [32]byte
	Hash []byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterContenthashChanged is a free log retrieval operation binding the contract event 0xe379c1624ed7e714cc0937528a32359d69d5281337765313dba4e081b72d7578.
//
// Solidity: event ContenthashChanged(bytes32 indexed node, bytes hash)
func (_PublicResolver *PublicResolverFilterer) FilterContenthashChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverContenthashChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "ContenthashChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverContenthashChangedIterator{contract: _PublicResolver.contract, event: "ContenthashChanged", logs: logs, sub: sub}, nil
}

// WatchContenthashChanged is a free log subscription operation binding the contract event 0xe379c1624ed7e714cc0937528a32359d69d5281337765313dba4e081b72d7578.
//
// Solidity: event ContenthashChanged(bytes32 indexed node, bytes hash)
func (_PublicResolver *PublicResolverFilterer) WatchContenthashChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverContenthashChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "ContenthashChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverContenthashChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "ContenthashChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// PublicResolverNameChangedIterator is returned from FilterNameChanged and is used to iterate over the raw logs and unpacked data for NameChanged events raised by the PublicResolver contract.
type PublicResolverNameChangedIterator struct {
	Event *PublicResolverNameChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PublicResolverNameChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverNameChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PublicResolverNameChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PublicResolverNameChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverNameChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverNameChanged represents a NameChanged event raised by the PublicResolver contract.
type PublicResolverNameChanged struct {
	Node [32]byte
	Name string
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterNameChanged is a free log retrieval operation binding the contract event 0xb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f7.
//
// Solidity: event NameChanged(bytes32 indexed node, string name)
func (_PublicResolver *PublicResolverFilterer) FilterNameChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverNameChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "NameChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverNameChangedIterator{contract: _PublicResolver.contract, event: "NameChanged", logs: logs, sub: sub}, nil
}

// WatchNameChanged is a free log subscription operation binding the contract event 0xb7d29e911041e8d9b843369e890bcb72c9388692ba48b65ac54e7214c4c348f7.
//
// Solidity: event NameChanged(bytes32 indexed node, string name)
func (_PublicResolver *PublicResolverFilterer) WatchNameChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverNameChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "NameChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverNameChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "NameChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// PublicResolverPubkeyChangedIterator is returned from FilterPubkeyChanged and is used to iterate over the raw logs and unpacked data for PubkeyChanged events raised by the PublicResolver contract.
type PublicResolverPubkeyChangedIterator struct {
	Event *PublicResolverPubkeyChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PublicResolverPubkeyChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverPubkeyChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PublicResolverPubkeyChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PublicResolverPubkeyChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverPubkeyChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverPubkeyChanged represents a PubkeyChanged event raised by the PublicResolver contract.
type PublicResolverPubkeyChanged struct {
	Node [32]byte
	X    [32]byte
	Y    [32]byte
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterPubkeyChanged is a free log retrieval operation binding the contract event 0x1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46.
//
// Solidity: event PubkeyChanged(bytes32 indexed node, bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverFilterer) FilterPubkeyChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverPubkeyChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "PubkeyChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverPubkeyChangedIterator{contract: _PublicResolver.contract, event: "PubkeyChanged", logs: logs, sub: sub}, nil
}

// WatchPubkeyChanged is a free log subscription operation binding the contract event 0x1d6f5e03d3f63eb58751986629a5439baee5079ff04f345becb66e23eb154e46.
//
// Solidity: event PubkeyChanged(bytes32 indexed node, bytes32 x, bytes32 y)
func (_PublicResolver *PublicResolverFilterer) WatchPubkeyChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverPubkeyChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "PubkeyChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverPubkeyChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "PubkeyChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// PublicResolverTextChangedIterator is returned from FilterTextChanged and is used to iterate over the raw logs and unpacked data for TextChanged events raised by the PublicResolver contract.
type PublicResolverTextChangedIterator struct {
	Event *PublicResolverTextChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PublicResolverTextChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PublicResolverTextChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PublicResolverTextChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PublicResolverTextChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PublicResolverTextChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PublicResolverTextChanged represents a TextChanged event raised by the PublicResolver contract.
type PublicResolverTextChanged struct {
	Node       [32]byte
	IndexedKey string
	Key        string
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTextChanged is a free log retrieval operation binding the contract event 0xd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a7550.
//
// Solidity: event TextChanged(bytes32 indexed node, string indexedKey, string key)
func (_PublicResolver *PublicResolverFilterer) FilterTextChanged(opts *bind.FilterOpts, node [][32]byte) (*PublicResolverTextChangedIterator, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.FilterLogs(opts, "TextChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return &PublicResolverTextChangedIterator{contract: _PublicResolver.contract, event: "TextChanged", logs: logs, sub: sub}, nil
}

// WatchTextChanged is a free log subscription operation binding the contract event 0xd8c9334b1a9c2f9da342a0a2b32629c1a229b6445dad78947f674b44444a7550.
//
// Solidity: event TextChanged(bytes32 indexed node, string indexedKey, string key)
func (_PublicResolver *PublicResolverFilterer) WatchTextChanged(opts *bind.WatchOpts, sink chan<- *PublicResolverTextChanged, node [][32]byte) (event.Subscription, error) {

	var nodeRule []interface{}
	for _, nodeItem := range node {
		nodeRule = append(nodeRule, nodeItem)
	}

	logs, sub, err := _PublicResolver.contract.WatchLogs(opts, "TextChanged", nodeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PublicResolverTextChanged)
				if err := _PublicResolver.contract.UnpackLog(event, "TextChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
