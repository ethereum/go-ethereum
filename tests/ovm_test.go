package tests

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var chainConfig params.ChainConfig

func init() {
	chainConfig = params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      new(big.Int),
		ByzantiumBlock:      new(big.Int),
		ConstantinopleBlock: new(big.Int),
		DAOForkBlock:        new(big.Int),
		DAOForkSupport:      false,
		EIP150Block:         new(big.Int),
		EIP155Block:         new(big.Int),
		EIP158Block:         new(big.Int),
	}
}

const GAS_LIMIT = 15000000

var ZERO_ADDRESS = common.HexToAddress("0000000000000000000000000000000000000000")
var OTHER_FROM_ADDR = common.HexToAddress("8888888888888888888888888888888888888888")

func TestContractCreationAndSimpleStorageTxs(t *testing.T) {
	currentState := newState()

	// Next we've got to generate & apply a transaction which calls the EM to deploy a new contract
	initCode, _ := hex.DecodeString("608060405234801561001057600080fd5b5060405161026b38038061026b8339818101604052602081101561003357600080fd5b8101908080519060200190929190505050806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550506101d7806100946000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80633408f73a1461003b578063d3404b6d14610045575b600080fd5b61004361004f565b005b61004d6100fa565b005b600060e060405180807f6f766d534c4f4144282900000000000000000000000000000000000000000000815250600a0190506040518091039020901c905060008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905060405136600082378260181c81538260101c60018201538260081c60028201538260038201536040516207a1208136846000875af160008114156100f657600080fd5b3d82f35b600060e060405180807f6f766d5353544f52452829000000000000000000000000000000000000000000815250600b0190506040518091039020901c905060008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905060405136600082378260181c81538260101c60018201538260081c600282015382600382015360008036836000865af1600081141561019c57600080fd5b5050505056fea265627a7a72315820311a406c97055eec367b660092882e1a174e14333416a3de384439293b7b129264736f6c6343000510003200000000000000000000000000000000000000000000000000000000dead0000")

	log.Debug("\n\nApplying CREATE SIMPLE STORAGE Tx to State.")
	applyMessageToState(currentState, OTHER_FROM_ADDR, ZERO_ADDRESS, GAS_LIMIT, initCode)
	log.Debug("Complete.")

	log.Debug("\n\nApplying CALL SIMPLE STORAGE Tx to State.")
	newContractAddr := common.HexToAddress("65486c8ec9167565eBD93c94ED04F0F71d1b5137")
	setStorageInnerCalldata, _ := hex.DecodeString("d3404b6d99999999999999999999999999999999999999999999999999999999999999990101010101010101010101010101010101010101010101010101010101010101")
	getStorageInnerCalldata, _ := hex.DecodeString("3408f73a9999999999999999999999999999999999999999999999999999999999999999")

	log.Debug("\n\nApplying `set()` SIMPLE STORAGE Tx to State.")
	applyMessageToState(currentState, OTHER_FROM_ADDR, newContractAddr, GAS_LIMIT, setStorageInnerCalldata)
	log.Debug("\n\nApplying `get()` SIMPLE STORAGE Tx to State.")
	returnValue, _, _, _ := applyMessageToState(currentState, OTHER_FROM_ADDR, newContractAddr, GAS_LIMIT, getStorageInnerCalldata)
	log.Debug("Complete.")

	expectedReturnValue, _ := hex.DecodeString("0101010101010101010101010101010101010101010101010101010101010101")
	if !bytes.Equal(returnValue[:], expectedReturnValue) {
		t.Errorf("Expected %020x; got %020x", returnValue[:], expectedReturnValue)
	}
}

func TestSloadAndStore(t *testing.T) {
	rawStateManagerAbi, _ := ioutil.ReadFile("./StateManagerABI.json")
	stateManagerAbi, _ := abi.JSON(strings.NewReader(string(rawStateManagerAbi)))
	state := newState()

	address := common.HexToAddress("9999999999999999999999999999999999999999")
	key := [32]byte{}
	value := [32]byte{}
	copy(key[:], []byte("hello"))
	copy(value[:], []byte("world"))

	storeCalldata, _ := stateManagerAbi.Pack("setStorage", address, key, value)
	getCalldata, _ := stateManagerAbi.Pack("getStorage", address, key)

	call(t, state, vm.StateManagerAddress, storeCalldata)
	getStorageReturnValue, _ := call(t, state, vm.StateManagerAddress, getCalldata)

	if !bytes.Equal(value[:], getStorageReturnValue) {
		t.Errorf("Expected %020x; got %020x", value[:], getStorageReturnValue)
	}
}

func TestCreate(t *testing.T) {
	currentState := newState()
	initCode, _ := hex.DecodeString("608060405234801561001057600080fd5b5060405161026b38038061026b8339818101604052602081101561003357600080fd5b8101908080519060200190929190505050806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550506101d7806100946000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80633408f73a1461003b578063d3404b6d14610045575b600080fd5b61004361004f565b005b61004d6100fa565b005b600060e060405180807f6f766d534c4f4144282900000000000000000000000000000000000000000000815250600a0190506040518091039020901c905060008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905060405136600082378260181c81538260101c60018201538260081c60028201538260038201536040516207a1208136846000875af160008114156100f657600080fd5b3d82f35b600060e060405180807f6f766d5353544f52452829000000000000000000000000000000000000000000815250600b0190506040518091039020901c905060008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905060405136600082378260181c81538260101c60018201538260081c600282015382600382015360008036836000865af1600081141561019c57600080fd5b5050505056fea265627a7a72315820311a406c97055eec367b660092882e1a174e14333416a3de384439293b7b129264736f6c6343000510003200000000000000000000000000000000000000000000000000000000dead0000")
	applyMessageToState(currentState, OTHER_FROM_ADDR, ZERO_ADDRESS, GAS_LIMIT, initCode)

	deployedBytecode := currentState.GetCode(crypto.CreateAddress(OTHER_FROM_ADDR, 0))

	// Just make sure the deployed bytecode exists at that address
	if len(deployedBytecode) == 0 {
		t.Errorf("Deployed bytecode not found at expected address!")
	}
}

func TestGetAndIncrementNonce(t *testing.T) {
	rawStateManagerAbi, _ := ioutil.ReadFile("./StateManagerABI.json")
	stateManagerAbi, _ := abi.JSON(strings.NewReader(string(rawStateManagerAbi)))
	state := newState()

	address := common.HexToAddress("9999999999999999999999999999999999999999")

	getNonceCalldata, _ := stateManagerAbi.Pack("getOvmContractNonce", address)
	incrementNonceCalldata, _ := stateManagerAbi.Pack("incrementOvmContractNonce", address)

	getStorageReturnValue1, _ := call(t, state, vm.StateManagerAddress, getNonceCalldata)

	expectedReturnValue1 := makeUint256WithUint64(0)
	if !bytes.Equal(getStorageReturnValue1, expectedReturnValue1) {
		t.Errorf("Expected %020x; got %020x", expectedReturnValue1, getStorageReturnValue1)
	}

	call(t, state, vm.StateManagerAddress, incrementNonceCalldata)
	getStorageReturnValue2, _ := call(t, state, vm.StateManagerAddress, getNonceCalldata)

	expectedReturnValue2 := makeUint256WithUint64(1)
	if !bytes.Equal(getStorageReturnValue2, expectedReturnValue2) {
		t.Errorf("Expected %020x; got %020x", expectedReturnValue2, getStorageReturnValue2)
	}
}

func TestGetCodeContractAddressSucceedsForNormalContract(t *testing.T) {
	rawStateManagerAbi, _ := ioutil.ReadFile("./StateManagerABI.json")
	stateManagerAbi, _ := abi.JSON(strings.NewReader(string(rawStateManagerAbi)))
	state := newState()

	address := common.HexToAddress("9999999999999999999999999999999999999999")

	getCodeContractAddressCalldata, _ := stateManagerAbi.Pack("getCodeContractAddressFromOvmAddress", address)

	getCodeContractAddressReturnValue, _ := call(t, state, vm.StateManagerAddress, getCodeContractAddressCalldata)

	if !bytes.Equal(getCodeContractAddressReturnValue[12:], address.Bytes()) {
		t.Errorf("Expected %020x; got %020x", getCodeContractAddressReturnValue[12:], address.Bytes())
	}
}

func TestGetCodeContractAddressFailsForDeadContract(t *testing.T) {
	rawStateManagerAbi, _ := ioutil.ReadFile("./StateManagerABI.json")
	stateManagerAbi, _ := abi.JSON(strings.NewReader(string(rawStateManagerAbi)))
	state := newState()

	deadAddress := common.HexToAddress("00000000000000000000000000000000dead9999")

	getCodeContractAddressCalldata, _ := stateManagerAbi.Pack("getCodeContractAddressFromOvmAddress", deadAddress)

	_, err := call(t, state, vm.StateManagerAddress, getCodeContractAddressCalldata)

	if err == nil {
		t.Errorf("Expected error to be thrown accessing dead address!")
	}
}

func TestAssociateCodeContract(t *testing.T) {
	rawStateManagerAbi, _ := ioutil.ReadFile("./StateManagerABI.json")
	stateManagerAbi, _ := abi.JSON(strings.NewReader(string(rawStateManagerAbi)))
	state := newState()

	address := common.HexToAddress("9999999999999999999999999999999999999999")

	getCodeContractAddressCalldata, _ := stateManagerAbi.Pack("associateCodeContract", address, address)

	_, err := call(t, state, vm.StateManagerAddress, getCodeContractAddressCalldata)
	if err != nil {
		t.Errorf("Failed to call associateCodeContract: %s", err)
	}
}

func TestGetCodeContractBytecode(t *testing.T) {
	state := newState()
	initCode, _ := hex.DecodeString("6080604052348015600f57600080fd5b5060b28061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80639b0b0fda14602d575b600080fd5b606060048036036040811015604157600080fd5b8101908080359060200190929190803590602001909291905050506062565b005b8060008084815260200190815260200160002081905550505056fea265627a7a7231582053ac32a8b70d1cf87fb4ebf5a538ea9d9e773351e6c8afbc4bf6a6c273187f4a64736f6c63430005110032")
	applyMessageToState(state, OTHER_FROM_ADDR, ZERO_ADDRESS, GAS_LIMIT, initCode)

	deployedBytecode := state.GetCode(crypto.CreateAddress(OTHER_FROM_ADDR, 0))
	expectedDeployedByteCode := common.FromHex("6080604052348015600f57600080fd5b506004361060285760003560e01c80639b0b0fda14602d575b600080fd5b606060048036036040811015604157600080fd5b8101908080359060200190929190803590602001909291905050506062565b005b8060008084815260200190815260200160002081905550505056fea265627a7a7231582053ac32a8b70d1cf87fb4ebf5a538ea9d9e773351e6c8afbc4bf6a6c273187f4a64736f6c63430005110032")
	if !bytes.Equal(expectedDeployedByteCode, deployedBytecode) {
		t.Errorf("Expected %020x; got %020x", expectedDeployedByteCode, deployedBytecode)
	}
}

func TestGetCodeContractHash(t *testing.T) {
	state := newState()
	initCode, _ := hex.DecodeString("6080604052348015600f57600080fd5b5060b28061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c80639b0b0fda14602d575b600080fd5b606060048036036040811015604157600080fd5b8101908080359060200190929190803590602001909291905050506062565b005b8060008084815260200190815260200160002081905550505056fea265627a7a7231582053ac32a8b70d1cf87fb4ebf5a538ea9d9e773351e6c8afbc4bf6a6c273187f4a64736f6c63430005110032")
	applyMessageToState(state, OTHER_FROM_ADDR, ZERO_ADDRESS, GAS_LIMIT, initCode)

	rawStateManagerAbi, _ := ioutil.ReadFile("./StateManagerABI.json")
	stateManagerAbi, _ := abi.JSON(strings.NewReader(string(rawStateManagerAbi)))
	getCodeContractBytecodeCalldata, _ := stateManagerAbi.Pack("getCodeContractHash", crypto.CreateAddress(OTHER_FROM_ADDR, 0))
	getCodeContractBytecodeReturnValue, _ := call(t, state, vm.StateManagerAddress, getCodeContractBytecodeCalldata)
	expectedCreatedCodeHash := crypto.Keccak256(common.FromHex("6080604052348015600f57600080fd5b506004361060285760003560e01c80639b0b0fda14602d575b600080fd5b606060048036036040811015604157600080fd5b8101908080359060200190929190803590602001909291905050506062565b005b8060008084815260200190815260200160002081905550505056fea265627a7a7231582053ac32a8b70d1cf87fb4ebf5a538ea9d9e773351e6c8afbc4bf6a6c273187f4a64736f6c63430005110032"))
	if !bytes.Equal(getCodeContractBytecodeReturnValue, expectedCreatedCodeHash) {
		t.Errorf("Expected %020x; got %020x", getCodeContractBytecodeReturnValue, expectedCreatedCodeHash)
	}
}

func makeUint256WithUint64(num uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, num)
	val := append(make([]byte, 24), b[:]...)
	return val
}

func newState() *state.StateDB {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	state, _ := state.New(common.Hash{}, db)
	core.ApplyOvmStateToState(state)
	return state
}

func applyMessageToState(currentState *state.StateDB, from common.Address, to common.Address, gasLimit uint64, data []byte) ([]byte, uint64, bool, error) {
	header := &types.Header{
		Number:     big.NewInt(0),
		Difficulty: big.NewInt(0),
		Time:       1,
	}
	gasPool := core.GasPool(100000000)
	// Generate the message
	var message types.Message
	if to == ZERO_ADDRESS {
		// Check if to the ZERO_ADDRESS, if so, make it nil
		message = types.NewMessage(
			from,
			nil,
			currentState.GetNonce(from),
			big.NewInt(0),
			gasLimit,
			big.NewInt(0),
			data,
			false,
			&ZERO_ADDRESS,
			nil,
		)
	} else {
		// Otherwise we actually use the `to` field!
		message = types.NewMessage(
			from,
			&to,
			currentState.GetNonce(from),
			big.NewInt(0),
			gasLimit,
			big.NewInt(0),
			data,
			false,
			&ZERO_ADDRESS,
			nil,
		)
	}

	context := core.NewEVMContext(message, header, nil, &from)
	evm := vm.NewEVM(context, currentState, &chainConfig, vm.Config{})

	returnValue, gasUsed, failed, err := core.ApplyMessage(evm, message, &gasPool)
	log.Debug("Return val: [HIDDEN]", "Gas used:", gasUsed, "Failed:", failed, "Error:", err)

	commitHash, commitErr := currentState.Commit(false)
	log.Debug("Commit hash:", commitHash, "Commit err:", commitErr)

	return returnValue, gasUsed, failed, err
}

func call(t *testing.T, currentState *state.StateDB, address common.Address, callData []byte) ([]byte, error) {
	returnValue, _, err := runtime.Call(address, callData, &runtime.Config{
		State:       currentState,
		ChainConfig: &chainConfig,
	})

	return returnValue, err
}
