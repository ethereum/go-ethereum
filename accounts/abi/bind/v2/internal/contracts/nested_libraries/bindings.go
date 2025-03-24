// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package nested_libraries

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = bytes.Equal
	_ = errors.New
	_ = big.NewInt
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// C1MetaData contains all meta data concerning the C1 contract.
var C1MetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"v1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"v2\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"Do\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"res\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "ae26158f1824f3918bd66724ee8b6eb7c9",
	Bin: "0x6080604052348015600e575f5ffd5b506040516103983803806103988339818101604052810190602e91906066565b5050609d565b5f5ffd5b5f819050919050565b6048816038565b81146051575f5ffd5b50565b5f815190506060816041565b92915050565b5f5f6040838503121560795760786034565b5b5f6084858286016054565b92505060206093858286016054565b9150509250929050565b6102ee806100aa5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c80632ad112721461002d575b5f5ffd5b6100476004803603810190610042919061019e565b61005d565b60405161005491906101d8565b60405180910390f35b5f600173__$ffc1393672b8ed81d0c8093ffcb0e7fbe8$__632ad112725f6040518263ffffffff1660e01b81526004016100979190610200565b602060405180830381865af41580156100b2573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100d6919061022d565b73__$5f33a1fab8ea7d932b4bc8c5e7dcd90bc2$__632ad11272856040518263ffffffff1660e01b815260040161010d9190610200565b602060405180830381865af4158015610128573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061014c919061022d565b6101569190610285565b6101609190610285565b9050919050565b5f5ffd5b5f819050919050565b61017d8161016b565b8114610187575f5ffd5b50565b5f8135905061019881610174565b92915050565b5f602082840312156101b3576101b2610167565b5b5f6101c08482850161018a565b91505092915050565b6101d28161016b565b82525050565b5f6020820190506101eb5f8301846101c9565b92915050565b6101fa8161016b565b82525050565b5f6020820190506102135f8301846101f1565b92915050565b5f8151905061022781610174565b92915050565b5f6020828403121561024257610241610167565b5b5f61024f84828501610219565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61028f8261016b565b915061029a8361016b565b92508282019050808211156102b2576102b1610258565b5b9291505056fea26469706673582212205d4715a8d20a3a0a43113e268ec8868b3c3ce24f7cbdb8735b4eeeebf0b5565164736f6c634300081c0033",
	Deps: []*bind.MetaData{
		&L1MetaData,
		&L4MetaData,
	},
}

// C1 is an auto generated Go binding around an Ethereum contract.
type C1 struct {
	abi abi.ABI
}

// NewC1 creates a new instance of C1.
func NewC1() *C1 {
	parsed, err := C1MetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &C1{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *C1) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackConstructor is the Go binding used to pack the parameters required for
// contract deployment.
//
// Solidity: constructor(uint256 v1, uint256 v2) returns()
func (c1 *C1) PackConstructor(v1 *big.Int, v2 *big.Int) []byte {
	enc, err := c1.abi.Pack("", v1, v2)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackDo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256 res)
func (c1 *C1) PackDo(val *big.Int) []byte {
	enc, err := c1.abi.Pack("Do", val)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDo is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256 res)
func (c1 *C1) UnpackDo(data []byte) (*big.Int, error) {
	out, err := c1.abi.Unpack("Do", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// C2MetaData contains all meta data concerning the C2 contract.
var C2MetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"v1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"v2\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"Do\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"res\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "78ef2840de5b706112ca2dbfa765501a89",
	Bin: "0x6080604052348015600e575f5ffd5b506040516103983803806103988339818101604052810190602e91906066565b5050609d565b5f5ffd5b5f819050919050565b6048816038565b81146051575f5ffd5b50565b5f815190506060816041565b92915050565b5f5f6040838503121560795760786034565b5b5f6084858286016054565b92505060206093858286016054565b9150509250929050565b6102ee806100aa5f395ff3fe608060405234801561000f575f5ffd5b5060043610610029575f3560e01c80632ad112721461002d575b5f5ffd5b6100476004803603810190610042919061019e565b61005d565b60405161005491906101d8565b60405180910390f35b5f600173__$ffc1393672b8ed81d0c8093ffcb0e7fbe8$__632ad112725f6040518263ffffffff1660e01b81526004016100979190610200565b602060405180830381865af41580156100b2573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100d6919061022d565b73__$6070639404c39b5667691bb1f9177e1eac$__632ad11272856040518263ffffffff1660e01b815260040161010d9190610200565b602060405180830381865af4158015610128573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061014c919061022d565b6101569190610285565b6101609190610285565b9050919050565b5f5ffd5b5f819050919050565b61017d8161016b565b8114610187575f5ffd5b50565b5f8135905061019881610174565b92915050565b5f602082840312156101b3576101b2610167565b5b5f6101c08482850161018a565b91505092915050565b6101d28161016b565b82525050565b5f6020820190506101eb5f8301846101c9565b92915050565b6101fa8161016b565b82525050565b5f6020820190506102135f8301846101f1565b92915050565b5f8151905061022781610174565b92915050565b5f6020828403121561024257610241610167565b5b5f61024f84828501610219565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61028f8261016b565b915061029a8361016b565b92508282019050808211156102b2576102b1610258565b5b9291505056fea2646970667358221220dd394981f1e9fefa4d88bac1c4f1da4131779c7d3bd4189958d278e57e96d96f64736f6c634300081c0033",
	Deps: []*bind.MetaData{
		&L1MetaData,
		&L4bMetaData,
	},
}

// C2 is an auto generated Go binding around an Ethereum contract.
type C2 struct {
	abi abi.ABI
}

// NewC2 creates a new instance of C2.
func NewC2() *C2 {
	parsed, err := C2MetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &C2{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *C2) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackConstructor is the Go binding used to pack the parameters required for
// contract deployment.
//
// Solidity: constructor(uint256 v1, uint256 v2) returns()
func (c2 *C2) PackConstructor(v1 *big.Int, v2 *big.Int) []byte {
	enc, err := c2.abi.Pack("", v1, v2)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackDo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256 res)
func (c2 *C2) PackDo(val *big.Int) []byte {
	enc, err := c2.abi.Pack("Do", val)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDo is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256 res)
func (c2 *C2) UnpackDo(data []byte) (*big.Int, error) {
	out, err := c2.abi.Unpack("Do", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// L1MetaData contains all meta data concerning the L1 contract.
var L1MetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"Do\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "ffc1393672b8ed81d0c8093ffcb0e7fbe8",
	Bin: "0x61011c61004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe73000000000000000000000000000000000000000030146080604052600436106032575f3560e01c80632ad11272146036575b5f5ffd5b604c600480360381019060489190609c565b6060565b6040516057919060cf565b60405180910390f35b5f60019050919050565b5f5ffd5b5f819050919050565b607e81606e565b81146087575f5ffd5b50565b5f813590506096816077565b92915050565b5f6020828403121560ae5760ad606a565b5b5f60b984828501608a565b91505092915050565b60c981606e565b82525050565b5f60208201905060e05f83018460c2565b9291505056fea26469706673582212200161c5f22d130a2b7ec6cf22e0910e42e32c2881fa4a8a01455f524f63cf218d64736f6c634300081c0033",
}

// L1 is an auto generated Go binding around an Ethereum contract.
type L1 struct {
	abi abi.ABI
}

// NewL1 creates a new instance of L1.
func NewL1() *L1 {
	parsed, err := L1MetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &L1{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *L1) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackDo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l1 *L1) PackDo(val *big.Int) []byte {
	enc, err := l1.abi.Pack("Do", val)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDo is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l1 *L1) UnpackDo(data []byte) (*big.Int, error) {
	out, err := l1.abi.Unpack("Do", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// L2MetaData contains all meta data concerning the L2 contract.
var L2MetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"Do\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "2ce896a6dd38932d354f317286f90bc675",
	Bin: "0x61025161004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe7300000000000000000000000000000000000000003014608060405260043610610034575f3560e01c80632ad1127214610038575b5f5ffd5b610052600480360381019061004d9190610129565b610068565b60405161005f9190610163565b60405180910390f35b5f600173__$ffc1393672b8ed81d0c8093ffcb0e7fbe8$__632ad11272846040518263ffffffff1660e01b81526004016100a29190610163565b602060405180830381865af41580156100bd573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100e19190610190565b6100eb91906101e8565b9050919050565b5f5ffd5b5f819050919050565b610108816100f6565b8114610112575f5ffd5b50565b5f81359050610123816100ff565b92915050565b5f6020828403121561013e5761013d6100f2565b5b5f61014b84828501610115565b91505092915050565b61015d816100f6565b82525050565b5f6020820190506101765f830184610154565b92915050565b5f8151905061018a816100ff565b92915050565b5f602082840312156101a5576101a46100f2565b5b5f6101b28482850161017c565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6101f2826100f6565b91506101fd836100f6565b9250828201905080821115610215576102146101bb565b5b9291505056fea264697066735822122026999f96e14b0e279909ca5972343113c358e93a904569409a86866e2064f0fa64736f6c634300081c0033",
	Deps: []*bind.MetaData{
		&L1MetaData,
	},
}

// L2 is an auto generated Go binding around an Ethereum contract.
type L2 struct {
	abi abi.ABI
}

// NewL2 creates a new instance of L2.
func NewL2() *L2 {
	parsed, err := L2MetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &L2{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *L2) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackDo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l2 *L2) PackDo(val *big.Int) []byte {
	enc, err := l2.abi.Pack("Do", val)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDo is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l2 *L2) UnpackDo(data []byte) (*big.Int, error) {
	out, err := l2.abi.Unpack("Do", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// L2bMetaData contains all meta data concerning the L2b contract.
var L2bMetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"Do\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "fd1474cf57f7ed48491e8bfdfd0d172adf",
	Bin: "0x61025161004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe7300000000000000000000000000000000000000003014608060405260043610610034575f3560e01c80632ad1127214610038575b5f5ffd5b610052600480360381019061004d9190610129565b610068565b60405161005f9190610163565b60405180910390f35b5f600173__$ffc1393672b8ed81d0c8093ffcb0e7fbe8$__632ad11272846040518263ffffffff1660e01b81526004016100a29190610163565b602060405180830381865af41580156100bd573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100e19190610190565b6100eb91906101e8565b9050919050565b5f5ffd5b5f819050919050565b610108816100f6565b8114610112575f5ffd5b50565b5f81359050610123816100ff565b92915050565b5f6020828403121561013e5761013d6100f2565b5b5f61014b84828501610115565b91505092915050565b61015d816100f6565b82525050565b5f6020820190506101765f830184610154565b92915050565b5f8151905061018a816100ff565b92915050565b5f602082840312156101a5576101a46100f2565b5b5f6101b28482850161017c565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6101f2826100f6565b91506101fd836100f6565b9250828201905080821115610215576102146101bb565b5b9291505056fea2646970667358221220d6e7078682642d273736fd63baaa28538fe72495816c810fa0e77034de385dc564736f6c634300081c0033",
	Deps: []*bind.MetaData{
		&L1MetaData,
	},
}

// L2b is an auto generated Go binding around an Ethereum contract.
type L2b struct {
	abi abi.ABI
}

// NewL2b creates a new instance of L2b.
func NewL2b() *L2b {
	parsed, err := L2bMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &L2b{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *L2b) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackDo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l2b *L2b) PackDo(val *big.Int) []byte {
	enc, err := l2b.abi.Pack("Do", val)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDo is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l2b *L2b) UnpackDo(data []byte) (*big.Int, error) {
	out, err := l2b.abi.Unpack("Do", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// L3MetaData contains all meta data concerning the L3 contract.
var L3MetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"Do\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "d03b97f5e1a564374023a72ac7d1806773",
	Bin: "0x61011c61004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe73000000000000000000000000000000000000000030146080604052600436106032575f3560e01c80632ad11272146036575b5f5ffd5b604c600480360381019060489190609c565b6060565b6040516057919060cf565b60405180910390f35b5f60019050919050565b5f5ffd5b5f819050919050565b607e81606e565b81146087575f5ffd5b50565b5f813590506096816077565b92915050565b5f6020828403121560ae5760ad606a565b5b5f60b984828501608a565b91505092915050565b60c981606e565b82525050565b5f60208201905060e05f83018460c2565b9291505056fea264697066735822122094cfcb0ce039318885cc58f6d8e609e6e4bec575e1a046d3d15ea2e01e97241e64736f6c634300081c0033",
}

// L3 is an auto generated Go binding around an Ethereum contract.
type L3 struct {
	abi abi.ABI
}

// NewL3 creates a new instance of L3.
func NewL3() *L3 {
	parsed, err := L3MetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &L3{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *L3) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackDo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l3 *L3) PackDo(val *big.Int) []byte {
	enc, err := l3.abi.Pack("Do", val)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDo is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l3 *L3) UnpackDo(data []byte) (*big.Int, error) {
	out, err := l3.abi.Unpack("Do", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// L4MetaData contains all meta data concerning the L4 contract.
var L4MetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"Do\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "5f33a1fab8ea7d932b4bc8c5e7dcd90bc2",
	Bin: "0x6102d161004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe7300000000000000000000000000000000000000003014608060405260043610610034575f3560e01c80632ad1127214610038575b5f5ffd5b610052600480360381019061004d91906101a9565b610068565b60405161005f91906101e3565b60405180910390f35b5f600173__$d03b97f5e1a564374023a72ac7d1806773$__632ad11272846040518263ffffffff1660e01b81526004016100a291906101e3565b602060405180830381865af41580156100bd573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100e19190610210565b73__$2ce896a6dd38932d354f317286f90bc675$__632ad11272856040518263ffffffff1660e01b815260040161011891906101e3565b602060405180830381865af4158015610133573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101579190610210565b6101619190610268565b61016b9190610268565b9050919050565b5f5ffd5b5f819050919050565b61018881610176565b8114610192575f5ffd5b50565b5f813590506101a38161017f565b92915050565b5f602082840312156101be576101bd610172565b5b5f6101cb84828501610195565b91505092915050565b6101dd81610176565b82525050565b5f6020820190506101f65f8301846101d4565b92915050565b5f8151905061020a8161017f565b92915050565b5f6020828403121561022557610224610172565b5b5f610232848285016101fc565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61027282610176565b915061027d83610176565b92508282019050808211156102955761029461023b565b5b9291505056fea2646970667358221220531485f0b9ff78ba5ef06ef345aaddccec3ad15d1460014ccd7c2a58d36d0d4464736f6c634300081c0033",
	Deps: []*bind.MetaData{
		&L2MetaData,
		&L3MetaData,
	},
}

// L4 is an auto generated Go binding around an Ethereum contract.
type L4 struct {
	abi abi.ABI
}

// NewL4 creates a new instance of L4.
func NewL4() *L4 {
	parsed, err := L4MetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &L4{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *L4) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackDo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l4 *L4) PackDo(val *big.Int) []byte {
	enc, err := l4.abi.Pack("Do", val)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDo is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l4 *L4) UnpackDo(data []byte) (*big.Int, error) {
	out, err := l4.abi.Unpack("Do", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// L4bMetaData contains all meta data concerning the L4b contract.
var L4bMetaData = bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"val\",\"type\":\"uint256\"}],\"name\":\"Do\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"}]",
	ID:  "6070639404c39b5667691bb1f9177e1eac",
	Bin: "0x61025161004d600b8282823980515f1a6073146041577f4e487b71000000000000000000000000000000000000000000000000000000005f525f60045260245ffd5b305f52607381538281f3fe7300000000000000000000000000000000000000003014608060405260043610610034575f3560e01c80632ad1127214610038575b5f5ffd5b610052600480360381019061004d9190610129565b610068565b60405161005f9190610163565b60405180910390f35b5f600173__$fd1474cf57f7ed48491e8bfdfd0d172adf$__632ad11272846040518263ffffffff1660e01b81526004016100a29190610163565b602060405180830381865af41580156100bd573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100e19190610190565b6100eb91906101e8565b9050919050565b5f5ffd5b5f819050919050565b610108816100f6565b8114610112575f5ffd5b50565b5f81359050610123816100ff565b92915050565b5f6020828403121561013e5761013d6100f2565b5b5f61014b84828501610115565b91505092915050565b61015d816100f6565b82525050565b5f6020820190506101765f830184610154565b92915050565b5f8151905061018a816100ff565b92915050565b5f602082840312156101a5576101a46100f2565b5b5f6101b28482850161017c565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f6101f2826100f6565b91506101fd836100f6565b9250828201905080821115610215576102146101bb565b5b9291505056fea264697066735822122008a2478fd2427f180ace529e137b69337cb655dc21d6426de37054c32e821c6a64736f6c634300081c0033",
	Deps: []*bind.MetaData{
		&L2bMetaData,
	},
}

// L4b is an auto generated Go binding around an Ethereum contract.
type L4b struct {
	abi abi.ABI
}

// NewL4b creates a new instance of L4b.
func NewL4b() *L4b {
	parsed, err := L4bMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &L4b{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *L4b) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackDo is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l4b *L4b) PackDo(val *big.Int) []byte {
	enc, err := l4b.abi.Pack("Do", val)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackDo is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2ad11272.
//
// Solidity: function Do(uint256 val) pure returns(uint256)
func (l4b *L4b) UnpackDo(data []byte) (*big.Int, error) {
	out, err := l4b.abi.Unpack("Do", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}
