// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package posminer

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

// PosminerABI is the input ABI used to generate the binding from.
const PosminerABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"MinerPool\",\"type\":\"address\"},{\"name\":\"RegisterFingure\",\"type\":\"string\"}],\"name\":\"MinerRegistry\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"MPools\",\"outputs\":[{\"name\":\"status\",\"type\":\"uint256\"},{\"name\":\"AppAddr\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"Manager\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"Registers\",\"outputs\":[{\"name\":\"MinerPool\",\"type\":\"address\"},{\"name\":\"RegistryTime\",\"type\":\"uint256\"},{\"name\":\"PayTime\",\"type\":\"uint256\"},{\"name\":\"Register\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"MinerPool\",\"type\":\"address\"},{\"name\":\"status\",\"type\":\"uint256\"},{\"name\":\"AppAddr\",\"type\":\"string\"}],\"name\":\"MinerPoolSetting\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"ActiveUsers\",\"outputs\":[{\"name\":\"LastTime\",\"type\":\"uint256\"},{\"name\":\"ActiveNum\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newManager\",\"type\":\"address\"}],\"name\":\"transferManagement\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// PosminerBin is the compiled bytecode used for deploying new contracts.
const PosminerBin = `{
	"linkReferences": {},
	"object": "6060604052341561000f57600080fd5b336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610a228061005e6000396000f300606060405260043610610083576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680635b86b6c814610088578063654ca9d91461011157806378357e53146101e75780637b3d32ce1461023c578063a8bfb8c61461034c578063ccf7fd8d146103d1578063e4edf85214610401575b600080fd5b6100f7600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509190505061043a565b604051808215151515815260200191505060405180910390f35b341561011c57600080fd5b610148600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190505061072c565b60405180838152602001806020018281038252838181546001816001161561010002031660029004815260200191508054600181600116156101000203166002900480156101d75780601f106101ac576101008083540402835291602001916101d7565b820191906000526020600020905b8154815290600101906020018083116101ba57829003601f168201915b5050935050505060405180910390f35b34156101f257600080fd5b6101fa61074f565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b341561024757600080fd5b610273600480803573ffffffffffffffffffffffffffffffffffffffff16906020019091905050610774565b604051808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018481526020018381526020018060200182810382528381815460018160011615610100020316600290048152602001915080546001816001161561010002031660029004801561033a5780601f1061030f5761010080835404028352916020019161033a565b820191906000526020600020905b81548152906001019060200180831161031d57829003601f168201915b50509550505050505060405180910390f35b341561035757600080fd5b6103cf600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190803590602001909190803590602001908201803590602001908080601f016020809104026020016040519081016040528093929190818152602001838380828437820191505050505050919050506107c3565b005b34156103dc57600080fd5b6103e46108a1565b604051808381526020018281526020019250505060405180910390f35b341561040c57600080fd5b610438600480803573ffffffffffffffffffffffffffffffffffffffff169060200190919050506108b3565b005b60006001600260008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600001541415156104915760009050610726565b4262015180600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600201540110156105ed573373ffffffffffffffffffffffffffffffffffffffff166108fc6001600360010154013073ffffffffffffffffffffffffffffffffffffffff163181151561052457fe5b049081150290604051600060405180830381858888f19350505050151561054a57600080fd5b42600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206002018190555042620151806003600001540110156105d75760408051908101604052804281526020016001815250600360008201518160000155602082015181600101559050506105ec565b60016003600101600082825401925050819055505b5b6080604051908101604052808473ffffffffffffffffffffffffffffffffffffffff168152602001428152602001600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060020154815260200183815250600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008201518160000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506020820151816001015560408201518160020155606082015181600301908051906020019061071d929190610951565b50905050600190505b92915050565b600260205280600052604060002060009150905080600001549080600101905082565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60016020528060005260406000206000915090508060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169080600101549080600201549080600301905084565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561081e57600080fd5b604080519081016040528083815260200182815250600260008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082015181600001556020820151816001019080519060200190610898929190610951565b50905050505050565b60038060000154908060010154905082565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561090e57600080fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061099257805160ff19168380011785556109c0565b828001600101855582156109c0579182015b828111156109bf5782518255916020019190600101906109a4565b5b5090506109cd91906109d1565b5090565b6109f391905b808211156109ef5760008160009055506001016109d7565b5090565b905600a165627a7a72305820325342257ec0d1a0612a67caf8a6c7f3f3042f95625e591222db3fafdf8e0f900029",
	"opcodes": "PUSH1 0x60 PUSH1 0x40 MSTORE CALLVALUE ISZERO PUSH2 0xF JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST CALLER PUSH1 0x0 DUP1 PUSH2 0x100 EXP DUP2 SLOAD DUP2 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF MUL NOT AND SWAP1 DUP4 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND MUL OR SWAP1 SSTORE POP PUSH2 0xA22 DUP1 PUSH2 0x5E PUSH1 0x0 CODECOPY PUSH1 0x0 RETURN STOP PUSH1 0x60 PUSH1 0x40 MSTORE PUSH1 0x4 CALLDATASIZE LT PUSH2 0x83 JUMPI PUSH1 0x0 CALLDATALOAD PUSH29 0x100000000000000000000000000000000000000000000000000000000 SWAP1 DIV PUSH4 0xFFFFFFFF AND DUP1 PUSH4 0x5B86B6C8 EQ PUSH2 0x88 JUMPI DUP1 PUSH4 0x654CA9D9 EQ PUSH2 0x111 JUMPI DUP1 PUSH4 0x78357E53 EQ PUSH2 0x1E7 JUMPI DUP1 PUSH4 0x7B3D32CE EQ PUSH2 0x23C JUMPI DUP1 PUSH4 0xA8BFB8C6 EQ PUSH2 0x34C JUMPI DUP1 PUSH4 0xCCF7FD8D EQ PUSH2 0x3D1 JUMPI DUP1 PUSH4 0xE4EDF852 EQ PUSH2 0x401 JUMPI JUMPDEST PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0xF7 PUSH1 0x4 DUP1 DUP1 CALLDATALOAD PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP3 ADD DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP1 DUP1 PUSH1 0x1F ADD PUSH1 0x20 DUP1 SWAP2 DIV MUL PUSH1 0x20 ADD PUSH1 0x40 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 SWAP4 SWAP3 SWAP2 SWAP1 DUP2 DUP2 MSTORE PUSH1 0x20 ADD DUP4 DUP4 DUP1 DUP3 DUP5 CALLDATACOPY DUP3 ADD SWAP2 POP POP POP POP POP POP SWAP2 SWAP1 POP POP PUSH2 0x43A JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP3 ISZERO ISZERO ISZERO ISZERO DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE ISZERO PUSH2 0x11C JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x148 PUSH1 0x4 DUP1 DUP1 CALLDATALOAD PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 POP POP PUSH2 0x72C JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP4 DUP2 MSTORE PUSH1 0x20 ADD DUP1 PUSH1 0x20 ADD DUP3 DUP2 SUB DUP3 MSTORE DUP4 DUP2 DUP2 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP DUP1 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV DUP1 ISZERO PUSH2 0x1D7 JUMPI DUP1 PUSH1 0x1F LT PUSH2 0x1AC JUMPI PUSH2 0x100 DUP1 DUP4 SLOAD DIV MUL DUP4 MSTORE SWAP2 PUSH1 0x20 ADD SWAP2 PUSH2 0x1D7 JUMP JUMPDEST DUP3 ADD SWAP2 SWAP1 PUSH1 0x0 MSTORE PUSH1 0x20 PUSH1 0x0 KECCAK256 SWAP1 JUMPDEST DUP2 SLOAD DUP2 MSTORE SWAP1 PUSH1 0x1 ADD SWAP1 PUSH1 0x20 ADD DUP1 DUP4 GT PUSH2 0x1BA JUMPI DUP3 SWAP1 SUB PUSH1 0x1F AND DUP3 ADD SWAP2 JUMPDEST POP POP SWAP4 POP POP POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE ISZERO PUSH2 0x1F2 JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x1FA PUSH2 0x74F JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP3 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE ISZERO PUSH2 0x247 JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x273 PUSH1 0x4 DUP1 DUP1 CALLDATALOAD PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 POP POP PUSH2 0x774 JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP6 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD DUP5 DUP2 MSTORE PUSH1 0x20 ADD DUP4 DUP2 MSTORE PUSH1 0x20 ADD DUP1 PUSH1 0x20 ADD DUP3 DUP2 SUB DUP3 MSTORE DUP4 DUP2 DUP2 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV DUP2 MSTORE PUSH1 0x20 ADD SWAP2 POP DUP1 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV DUP1 ISZERO PUSH2 0x33A JUMPI DUP1 PUSH1 0x1F LT PUSH2 0x30F JUMPI PUSH2 0x100 DUP1 DUP4 SLOAD DIV MUL DUP4 MSTORE SWAP2 PUSH1 0x20 ADD SWAP2 PUSH2 0x33A JUMP JUMPDEST DUP3 ADD SWAP2 SWAP1 PUSH1 0x0 MSTORE PUSH1 0x20 PUSH1 0x0 KECCAK256 SWAP1 JUMPDEST DUP2 SLOAD DUP2 MSTORE SWAP1 PUSH1 0x1 ADD SWAP1 PUSH1 0x20 ADD DUP1 DUP4 GT PUSH2 0x31D JUMPI DUP3 SWAP1 SUB PUSH1 0x1F AND DUP3 ADD SWAP2 JUMPDEST POP POP SWAP6 POP POP POP POP POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE ISZERO PUSH2 0x357 JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x3CF PUSH1 0x4 DUP1 DUP1 CALLDATALOAD PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP3 ADD DUP1 CALLDATALOAD SWAP1 PUSH1 0x20 ADD SWAP1 DUP1 DUP1 PUSH1 0x1F ADD PUSH1 0x20 DUP1 SWAP2 DIV MUL PUSH1 0x20 ADD PUSH1 0x40 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 SWAP4 SWAP3 SWAP2 SWAP1 DUP2 DUP2 MSTORE PUSH1 0x20 ADD DUP4 DUP4 DUP1 DUP3 DUP5 CALLDATACOPY DUP3 ADD SWAP2 POP POP POP POP POP POP SWAP2 SWAP1 POP POP PUSH2 0x7C3 JUMP JUMPDEST STOP JUMPDEST CALLVALUE ISZERO PUSH2 0x3DC JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x3E4 PUSH2 0x8A1 JUMP JUMPDEST PUSH1 0x40 MLOAD DUP1 DUP4 DUP2 MSTORE PUSH1 0x20 ADD DUP3 DUP2 MSTORE PUSH1 0x20 ADD SWAP3 POP POP POP PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 RETURN JUMPDEST CALLVALUE ISZERO PUSH2 0x40C JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH2 0x438 PUSH1 0x4 DUP1 DUP1 CALLDATALOAD PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 PUSH1 0x20 ADD SWAP1 SWAP2 SWAP1 POP POP PUSH2 0x8B3 JUMP JUMPDEST STOP JUMPDEST PUSH1 0x0 PUSH1 0x1 PUSH1 0x2 PUSH1 0x0 DUP6 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP1 DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x0 KECCAK256 PUSH1 0x0 ADD SLOAD EQ ISZERO ISZERO PUSH2 0x491 JUMPI PUSH1 0x0 SWAP1 POP PUSH2 0x726 JUMP JUMPDEST TIMESTAMP PUSH3 0x15180 PUSH1 0x1 PUSH1 0x0 CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP1 DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x0 KECCAK256 PUSH1 0x2 ADD SLOAD ADD LT ISZERO PUSH2 0x5ED JUMPI CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH2 0x8FC PUSH1 0x1 PUSH1 0x3 PUSH1 0x1 ADD SLOAD ADD ADDRESS PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND BALANCE DUP2 ISZERO ISZERO PUSH2 0x524 JUMPI INVALID JUMPDEST DIV SWAP1 DUP2 ISZERO MUL SWAP1 PUSH1 0x40 MLOAD PUSH1 0x0 PUSH1 0x40 MLOAD DUP1 DUP4 SUB DUP2 DUP6 DUP9 DUP9 CALL SWAP4 POP POP POP POP ISZERO ISZERO PUSH2 0x54A JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST TIMESTAMP PUSH1 0x1 PUSH1 0x0 CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP1 DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x0 KECCAK256 PUSH1 0x2 ADD DUP2 SWAP1 SSTORE POP TIMESTAMP PUSH3 0x15180 PUSH1 0x3 PUSH1 0x0 ADD SLOAD ADD LT ISZERO PUSH2 0x5D7 JUMPI PUSH1 0x40 DUP1 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 TIMESTAMP DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x1 DUP2 MSTORE POP PUSH1 0x3 PUSH1 0x0 DUP3 ADD MLOAD DUP2 PUSH1 0x0 ADD SSTORE PUSH1 0x20 DUP3 ADD MLOAD DUP2 PUSH1 0x1 ADD SSTORE SWAP1 POP POP PUSH2 0x5EC JUMP JUMPDEST PUSH1 0x1 PUSH1 0x3 PUSH1 0x1 ADD PUSH1 0x0 DUP3 DUP3 SLOAD ADD SWAP3 POP POP DUP2 SWAP1 SSTORE POP JUMPDEST JUMPDEST PUSH1 0x80 PUSH1 0x40 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 DUP5 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD TIMESTAMP DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x1 PUSH1 0x0 CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP1 DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x0 KECCAK256 PUSH1 0x2 ADD SLOAD DUP2 MSTORE PUSH1 0x20 ADD DUP4 DUP2 MSTORE POP PUSH1 0x1 PUSH1 0x0 CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP1 DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x0 KECCAK256 PUSH1 0x0 DUP3 ADD MLOAD DUP2 PUSH1 0x0 ADD PUSH1 0x0 PUSH2 0x100 EXP DUP2 SLOAD DUP2 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF MUL NOT AND SWAP1 DUP4 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND MUL OR SWAP1 SSTORE POP PUSH1 0x20 DUP3 ADD MLOAD DUP2 PUSH1 0x1 ADD SSTORE PUSH1 0x40 DUP3 ADD MLOAD DUP2 PUSH1 0x2 ADD SSTORE PUSH1 0x60 DUP3 ADD MLOAD DUP2 PUSH1 0x3 ADD SWAP1 DUP1 MLOAD SWAP1 PUSH1 0x20 ADD SWAP1 PUSH2 0x71D SWAP3 SWAP2 SWAP1 PUSH2 0x951 JUMP JUMPDEST POP SWAP1 POP POP PUSH1 0x1 SWAP1 POP JUMPDEST SWAP3 SWAP2 POP POP JUMP JUMPDEST PUSH1 0x2 PUSH1 0x20 MSTORE DUP1 PUSH1 0x0 MSTORE PUSH1 0x40 PUSH1 0x0 KECCAK256 PUSH1 0x0 SWAP2 POP SWAP1 POP DUP1 PUSH1 0x0 ADD SLOAD SWAP1 DUP1 PUSH1 0x1 ADD SWAP1 POP DUP3 JUMP JUMPDEST PUSH1 0x0 DUP1 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 JUMP JUMPDEST PUSH1 0x1 PUSH1 0x20 MSTORE DUP1 PUSH1 0x0 MSTORE PUSH1 0x40 PUSH1 0x0 KECCAK256 PUSH1 0x0 SWAP2 POP SWAP1 POP DUP1 PUSH1 0x0 ADD PUSH1 0x0 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND SWAP1 DUP1 PUSH1 0x1 ADD SLOAD SWAP1 DUP1 PUSH1 0x2 ADD SLOAD SWAP1 DUP1 PUSH1 0x3 ADD SWAP1 POP DUP5 JUMP JUMPDEST PUSH1 0x0 DUP1 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND EQ ISZERO ISZERO PUSH2 0x81E JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST PUSH1 0x40 DUP1 MLOAD SWAP1 DUP2 ADD PUSH1 0x40 MSTORE DUP1 DUP4 DUP2 MSTORE PUSH1 0x20 ADD DUP3 DUP2 MSTORE POP PUSH1 0x2 PUSH1 0x0 DUP6 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND DUP2 MSTORE PUSH1 0x20 ADD SWAP1 DUP2 MSTORE PUSH1 0x20 ADD PUSH1 0x0 KECCAK256 PUSH1 0x0 DUP3 ADD MLOAD DUP2 PUSH1 0x0 ADD SSTORE PUSH1 0x20 DUP3 ADD MLOAD DUP2 PUSH1 0x1 ADD SWAP1 DUP1 MLOAD SWAP1 PUSH1 0x20 ADD SWAP1 PUSH2 0x898 SWAP3 SWAP2 SWAP1 PUSH2 0x951 JUMP JUMPDEST POP SWAP1 POP POP POP POP POP JUMP JUMPDEST PUSH1 0x3 DUP1 PUSH1 0x0 ADD SLOAD SWAP1 DUP1 PUSH1 0x1 ADD SLOAD SWAP1 POP DUP3 JUMP JUMPDEST PUSH1 0x0 DUP1 SWAP1 SLOAD SWAP1 PUSH2 0x100 EXP SWAP1 DIV PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND CALLER PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND EQ ISZERO ISZERO PUSH2 0x90E JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST DUP1 PUSH1 0x0 DUP1 PUSH2 0x100 EXP DUP2 SLOAD DUP2 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF MUL NOT AND SWAP1 DUP4 PUSH20 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF AND MUL OR SWAP1 SSTORE POP POP JUMP JUMPDEST DUP3 DUP1 SLOAD PUSH1 0x1 DUP2 PUSH1 0x1 AND ISZERO PUSH2 0x100 MUL SUB AND PUSH1 0x2 SWAP1 DIV SWAP1 PUSH1 0x0 MSTORE PUSH1 0x20 PUSH1 0x0 KECCAK256 SWAP1 PUSH1 0x1F ADD PUSH1 0x20 SWAP1 DIV DUP2 ADD SWAP3 DUP3 PUSH1 0x1F LT PUSH2 0x992 JUMPI DUP1 MLOAD PUSH1 0xFF NOT AND DUP4 DUP1 ADD OR DUP6 SSTORE PUSH2 0x9C0 JUMP JUMPDEST DUP3 DUP1 ADD PUSH1 0x1 ADD DUP6 SSTORE DUP3 ISZERO PUSH2 0x9C0 JUMPI SWAP2 DUP3 ADD JUMPDEST DUP3 DUP2 GT ISZERO PUSH2 0x9BF JUMPI DUP3 MLOAD DUP3 SSTORE SWAP2 PUSH1 0x20 ADD SWAP2 SWAP1 PUSH1 0x1 ADD SWAP1 PUSH2 0x9A4 JUMP JUMPDEST JUMPDEST POP SWAP1 POP PUSH2 0x9CD SWAP2 SWAP1 PUSH2 0x9D1 JUMP JUMPDEST POP SWAP1 JUMP JUMPDEST PUSH2 0x9F3 SWAP2 SWAP1 JUMPDEST DUP1 DUP3 GT ISZERO PUSH2 0x9EF JUMPI PUSH1 0x0 DUP2 PUSH1 0x0 SWAP1 SSTORE POP PUSH1 0x1 ADD PUSH2 0x9D7 JUMP JUMPDEST POP SWAP1 JUMP JUMPDEST SWAP1 JUMP STOP LOG1 PUSH6 0x627A7A723058 KECCAK256 ORIGIN MSTORE8 TIMESTAMP 0x25 PUSH31 0xC0D1A0612A67CAF8A6C7F3F3042F95625E591222DB3FAFDF8E0F9000290000 ",
	"sourceMap": "28:2509:0:-;;;844:69;;;;;;;;894:10;886:7;;:18;;;;;;;;;;;;;;;;;;28:2509;;;;;;"
}`

// DeployPosminer deploys a new Ethereum contract, binding an instance of Posminer to it.
func DeployPosminer(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Posminer, error) {
	parsed, err := abi.JSON(strings.NewReader(PosminerABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(PosminerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Posminer{PosminerCaller: PosminerCaller{contract: contract}, PosminerTransactor: PosminerTransactor{contract: contract}}, nil
}

// Posminer is an auto generated Go binding around an Ethereum contract.
type Posminer struct {
	PosminerCaller     // Read-only binding to the contract
	PosminerTransactor // Write-only binding to the contract
}

// PosminerCaller is an auto generated read-only Go binding around an Ethereum contract.
type PosminerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PosminerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PosminerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PosminerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PosminerSession struct {
	Contract     *Posminer         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PosminerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PosminerCallerSession struct {
	Contract *PosminerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// PosminerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PosminerTransactorSession struct {
	Contract     *PosminerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// PosminerRaw is an auto generated low-level Go binding around an Ethereum contract.
type PosminerRaw struct {
	Contract *Posminer // Generic contract binding to access the raw methods on
}

// PosminerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PosminerCallerRaw struct {
	Contract *PosminerCaller // Generic read-only contract binding to access the raw methods on
}

// PosminerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PosminerTransactorRaw struct {
	Contract *PosminerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPosminer creates a new instance of Posminer, bound to a specific deployed contract.
func NewPosminer(address common.Address, backend bind.ContractBackend) (*Posminer, error) {
	contract, err := bindPosminer(address, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Posminer{PosminerCaller: PosminerCaller{contract: contract}, PosminerTransactor: PosminerTransactor{contract: contract}}, nil
}

// NewPosminerCaller creates a new read-only instance of Posminer, bound to a specific deployed contract.
func NewPosminerCaller(address common.Address, caller bind.ContractCaller) (*PosminerCaller, error) {
	contract, err := bindPosminer(address, caller, nil)
	if err != nil {
		return nil, err
	}
	return &PosminerCaller{contract: contract}, nil
}

// NewPosminerTransactor creates a new write-only instance of Posminer, bound to a specific deployed contract.
func NewPosminerTransactor(address common.Address, transactor bind.ContractTransactor) (*PosminerTransactor, error) {
	contract, err := bindPosminer(address, nil, transactor)
	if err != nil {
		return nil, err
	}
	return &PosminerTransactor{contract: contract}, nil
}

// bindPosminer binds a generic wrapper to an already deployed contract.
func bindPosminer(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PosminerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Posminer *PosminerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Posminer.Contract.PosminerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Posminer *PosminerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Posminer.Contract.PosminerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Posminer *PosminerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Posminer.Contract.PosminerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Posminer *PosminerCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Posminer.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Posminer *PosminerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Posminer.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Posminer *PosminerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Posminer.Contract.contract.Transact(opts, method, params...)
}

// ActiveUsers is a free data retrieval call binding the contract method 0xccf7fd8d.
//
// Solidity: function ActiveUsers() constant returns(LastTime uint256, ActiveNum uint256)
func (_Posminer *PosminerCaller) ActiveUsers(opts *bind.CallOpts) (struct {
	LastTime  *big.Int
	ActiveNum *big.Int
}, error) {
	ret := new(struct {
		LastTime  *big.Int
		ActiveNum *big.Int
	})
	out := ret
	err := _Posminer.contract.Call(opts, out, "ActiveUsers")
	return *ret, err
}

// ActiveUsers is a free data retrieval call binding the contract method 0xccf7fd8d.
//
// Solidity: function ActiveUsers() constant returns(LastTime uint256, ActiveNum uint256)
func (_Posminer *PosminerSession) ActiveUsers() (struct {
	LastTime  *big.Int
	ActiveNum *big.Int
}, error) {
	return _Posminer.Contract.ActiveUsers(&_Posminer.CallOpts)
}

// ActiveUsers is a free data retrieval call binding the contract method 0xccf7fd8d.
//
// Solidity: function ActiveUsers() constant returns(LastTime uint256, ActiveNum uint256)
func (_Posminer *PosminerCallerSession) ActiveUsers() (struct {
	LastTime  *big.Int
	ActiveNum *big.Int
}, error) {
	return _Posminer.Contract.ActiveUsers(&_Posminer.CallOpts)
}

// MPools is a free data retrieval call binding the contract method 0x654ca9d9.
//
// Solidity: function MPools( address) constant returns(status uint256, AppAddr string)
func (_Posminer *PosminerCaller) MPools(opts *bind.CallOpts, arg0 common.Address) (struct {
	Status  *big.Int
	AppAddr string
}, error) {
	ret := new(struct {
		Status  *big.Int
		AppAddr string
	})
	out := ret
	err := _Posminer.contract.Call(opts, out, "MPools", arg0)
	return *ret, err
}

// MPools is a free data retrieval call binding the contract method 0x654ca9d9.
//
// Solidity: function MPools( address) constant returns(status uint256, AppAddr string)
func (_Posminer *PosminerSession) MPools(arg0 common.Address) (struct {
	Status  *big.Int
	AppAddr string
}, error) {
	return _Posminer.Contract.MPools(&_Posminer.CallOpts, arg0)
}

// MPools is a free data retrieval call binding the contract method 0x654ca9d9.
//
// Solidity: function MPools( address) constant returns(status uint256, AppAddr string)
func (_Posminer *PosminerCallerSession) MPools(arg0 common.Address) (struct {
	Status  *big.Int
	AppAddr string
}, error) {
	return _Posminer.Contract.MPools(&_Posminer.CallOpts, arg0)
}

// Manager is a free data retrieval call binding the contract method 0x78357e53.
//
// Solidity: function Manager() constant returns(address)
func (_Posminer *PosminerCaller) Manager(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Posminer.contract.Call(opts, out, "Manager")
	return *ret0, err
}

// Manager is a free data retrieval call binding the contract method 0x78357e53.
//
// Solidity: function Manager() constant returns(address)
func (_Posminer *PosminerSession) Manager() (common.Address, error) {
	return _Posminer.Contract.Manager(&_Posminer.CallOpts)
}

// Manager is a free data retrieval call binding the contract method 0x78357e53.
//
// Solidity: function Manager() constant returns(address)
func (_Posminer *PosminerCallerSession) Manager() (common.Address, error) {
	return _Posminer.Contract.Manager(&_Posminer.CallOpts)
}

// Registers is a free data retrieval call binding the contract method 0x7b3d32ce.
//
// Solidity: function Registers( address) constant returns(MinerPool address, RegistryTime uint256, PayTime uint256, Register string)
func (_Posminer *PosminerCaller) Registers(opts *bind.CallOpts, arg0 common.Address) (struct {
	MinerPool    common.Address
	RegistryTime *big.Int
	PayTime      *big.Int
	Register     string
}, error) {
	ret := new(struct {
		MinerPool    common.Address
		RegistryTime *big.Int
		PayTime      *big.Int
		Register     string
	})
	out := ret
	err := _Posminer.contract.Call(opts, out, "Registers", arg0)
	return *ret, err
}

// Registers is a free data retrieval call binding the contract method 0x7b3d32ce.
//
// Solidity: function Registers( address) constant returns(MinerPool address, RegistryTime uint256, PayTime uint256, Register string)
func (_Posminer *PosminerSession) Registers(arg0 common.Address) (struct {
	MinerPool    common.Address
	RegistryTime *big.Int
	PayTime      *big.Int
	Register     string
}, error) {
	return _Posminer.Contract.Registers(&_Posminer.CallOpts, arg0)
}

// Registers is a free data retrieval call binding the contract method 0x7b3d32ce.
//
// Solidity: function Registers( address) constant returns(MinerPool address, RegistryTime uint256, PayTime uint256, Register string)
func (_Posminer *PosminerCallerSession) Registers(arg0 common.Address) (struct {
	MinerPool    common.Address
	RegistryTime *big.Int
	PayTime      *big.Int
	Register     string
}, error) {
	return _Posminer.Contract.Registers(&_Posminer.CallOpts, arg0)
}

// MinerPoolSetting is a paid mutator transaction binding the contract method 0xa8bfb8c6.
//
// Solidity: function MinerPoolSetting(MinerPool address, status uint256, AppAddr string) returns()
func (_Posminer *PosminerTransactor) MinerPoolSetting(opts *bind.TransactOpts, MinerPool common.Address, status *big.Int, AppAddr string) (*types.Transaction, error) {
	return _Posminer.contract.Transact(opts, "MinerPoolSetting", MinerPool, status, AppAddr)
}

// MinerPoolSetting is a paid mutator transaction binding the contract method 0xa8bfb8c6.
//
// Solidity: function MinerPoolSetting(MinerPool address, status uint256, AppAddr string) returns()
func (_Posminer *PosminerSession) MinerPoolSetting(MinerPool common.Address, status *big.Int, AppAddr string) (*types.Transaction, error) {
	return _Posminer.Contract.MinerPoolSetting(&_Posminer.TransactOpts, MinerPool, status, AppAddr)
}

// MinerPoolSetting is a paid mutator transaction binding the contract method 0xa8bfb8c6.
//
// Solidity: function MinerPoolSetting(MinerPool address, status uint256, AppAddr string) returns()
func (_Posminer *PosminerTransactorSession) MinerPoolSetting(MinerPool common.Address, status *big.Int, AppAddr string) (*types.Transaction, error) {
	return _Posminer.Contract.MinerPoolSetting(&_Posminer.TransactOpts, MinerPool, status, AppAddr)
}

// MinerRegistry is a paid mutator transaction binding the contract method 0x5b86b6c8.
//
// Solidity: function MinerRegistry(MinerPool address, RegisterFingure string) returns(success bool)
func (_Posminer *PosminerTransactor) MinerRegistry(opts *bind.TransactOpts, MinerPool common.Address, RegisterFingure string) (*types.Transaction, error) {
	return _Posminer.contract.Transact(opts, "MinerRegistry", MinerPool, RegisterFingure)
}

// MinerRegistry is a paid mutator transaction binding the contract method 0x5b86b6c8.
//
// Solidity: function MinerRegistry(MinerPool address, RegisterFingure string) returns(success bool)
func (_Posminer *PosminerSession) MinerRegistry(MinerPool common.Address, RegisterFingure string) (*types.Transaction, error) {
	return _Posminer.Contract.MinerRegistry(&_Posminer.TransactOpts, MinerPool, RegisterFingure)
}

// MinerRegistry is a paid mutator transaction binding the contract method 0x5b86b6c8.
//
// Solidity: function MinerRegistry(MinerPool address, RegisterFingure string) returns(success bool)
func (_Posminer *PosminerTransactorSession) MinerRegistry(MinerPool common.Address, RegisterFingure string) (*types.Transaction, error) {
	return _Posminer.Contract.MinerRegistry(&_Posminer.TransactOpts, MinerPool, RegisterFingure)
}

// TransferManagement is a paid mutator transaction binding the contract method 0xe4edf852.
//
// Solidity: function transferManagement(newManager address) returns()
func (_Posminer *PosminerTransactor) TransferManagement(opts *bind.TransactOpts, newManager common.Address) (*types.Transaction, error) {
	return _Posminer.contract.Transact(opts, "transferManagement", newManager)
}

// TransferManagement is a paid mutator transaction binding the contract method 0xe4edf852.
//
// Solidity: function transferManagement(newManager address) returns()
func (_Posminer *PosminerSession) TransferManagement(newManager common.Address) (*types.Transaction, error) {
	return _Posminer.Contract.TransferManagement(&_Posminer.TransactOpts, newManager)
}

// TransferManagement is a paid mutator transaction binding the contract method 0xe4edf852.
//
// Solidity: function transferManagement(newManager address) returns()
func (_Posminer *PosminerTransactorSession) TransferManagement(newManager common.Address) (*types.Transaction, error) {
	return _Posminer.Contract.TransferManagement(&_Posminer.TransactOpts, newManager)
}
